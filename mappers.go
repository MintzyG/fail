package fail

import (
	"container/list"
	"errors"
	"log"
	"sync"
)

// Mapper converts errors in any direction: generic->fail, fail->generic, fail->fail, etc.
//
// IMPORTANT: Map must return errors created via fail.New() or fail.From(),
// not hand-crafted *fail.Error structs. Hand-crafted errors will be unregistered and
// may cause issues with translators and other components that require registered errors.
type Mapper interface {
	Name() string
	Priority() int

	// Map : should map generic errors to fail.Error type
	Map(err error) (*Error, bool)
}

// RegisterMapper adds a generic error mapper
func RegisterMapper(mapper Mapper) {
	global.RegisterMapper(mapper)
}

func (r *Registry) RegisterMapper(mapper Mapper) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Insert in priority order (higher first)
	r.genericMappers.Add(mapper)
}

// MapperList keeps mappers sorted by priority using container/list
type MapperList struct {
	mu      sync.RWMutex
	mappers *list.List // *list.Element.Value will be Mapper
}

// NewMapperList creates a new MapperList. If includeDefault is true, adds the default mapper with priority -1
func NewMapperList() *MapperList {
	ml := &MapperList{
		mappers: list.New(),
	}

	return ml
}

// Add inserts a mapper into the list by descending priority
func (ml *MapperList) Add(m Mapper) {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	priority := m.Priority()
	for e := ml.mappers.Front(); e != nil; e = e.Next() {
		existing := e.Value.(Mapper)
		if priority > existing.Priority() {
			ml.mappers.InsertBefore(m, e)
			return
		}
	}
	// If we didn't insert yet, add at the end
	ml.mappers.PushBack(m)
}

// Map maps to *fail.Error
func (ml *MapperList) Map(err error) (*Error, string, bool) {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	for mapper := ml.mappers.Front(); mapper != nil; mapper = mapper.Next() {
		if fe, ok := mapper.Value.(Mapper).Map(err); ok {
			return fe, mapper.Value.(Mapper).Name(), true
		}
	}
	return nil, "", false
}

// From ingests a generic error and maps it to an Error
func From(err error) *Error {
	return global.From(err)
}

// From ingests a generic error and maps it to an Error
func (r *Registry) From(err error) *Error {
	if err == nil {
		return nil
	}

	var e *Error
	if errors.As(err, &e) {
		// Same registry, just return it
		if e.registry == r {
			return e
		}
		// Different registry warn
		if r.allowInternalLogs {
			log.Printf("[fail] From() received error from different registry")
		}
		return e
	}

	// Need to map
	if r.genericMappers != nil {
		if fe, mapperName, ok := r.genericMappers.Map(err); ok {
			if !fe.IsRegistered() && r.allowInternalLogs {
				log.Printf("[fail] WARNING: mapper '%s' returned unregistered error ID(%s) - mapper should use fail.New()",
					mapperName,
					fe.ID.String())
			}
			r.hooks.runFromSuccess(err, fe)
			return fe
		}
	} else {
		if r.allowInternalLogs {
			log.Printf("[fail] tried to map with no mappers registered")
		}
		result := New(NoMapperRegistered).With(err)
		r.hooks.runFromFail(err)
		return result
	}

	if r.allowInternalLogs {
		log.Printf("[fail] No mapper matched error: %T, msg=%q", err, err.Error())
	}

	r.hooks.runFromFail(err)
	return New(NotMatchedInAnyMapper).With(err)
}
