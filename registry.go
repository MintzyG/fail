package fail

import (
	"errors"
	"fmt"
	"log"
	"sync"
)

// Registry holds all registered error definitions and mappers
type Registry struct {
	mu             sync.RWMutex
	errors         map[string]*Error // Keyed by ID.String()
	definitions    map[ErrorID]ErrorDefinition
	genericMappers *MapperList
	translators    map[string]Translator

	defaultLocale string
	localization  TranslationRegistry

	// Hooks for automatic behavior
	hooks Hooks

	tracer Tracer
	logger Logger

	allowInternalLogs bool
}

// Global registry - users can also create their own
var global = &Registry{
	errors:         make(map[string]*Error),
	genericMappers: NewMapperList(),
	translators:    make(map[string]Translator),
	hooks:          Hooks{},
}

// NewRegistry creates a new isolated registry (for testing or multi-app scenarios)
func NewRegistry() *Registry {
	return &Registry{
		errors:         make(map[string]*Error),
		genericMappers: NewMapperList(),
		translators:    make(map[string]Translator),
		hooks:          Hooks{},
	}
}

// Register adds an error definition to this registry
func (r *Registry) Register(err *Error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Verify the ErrorID is trusted
	if !err.ID.IsTrusted() {
		panic(fmt.Sprintf("cannot register untrusted error ID: %s (must use fail.ID() to create)", err.ID))
	}

	// First register wins (idempotent)
	if _, exists := r.errors[err.ID.String()]; exists {
		return
	}

	r.errors[err.ID.String()] = err
}

// RegisterMany registers multiple error definitions at once
func RegisterMany(defs ...*ErrorDefinition) {
	global.RegisterMany(defs...)
}

func (r *Registry) RegisterMany(defs ...*ErrorDefinition) {
	for _, def := range defs {
		r.Register(&Error{
			ID:       def.ID,
			Message:  def.DefaultMessage,
			IsSystem: def.IsSystem,
			Meta:     def.Meta,
		})
	}
}

var UnregisteredError = internalID(0, 0, false, "FailUnregisteredError")
var ErrUnregisteredError = Form(UnregisteredError, "error with ID(%s) is not registered in the registry", true, nil, "ID NOT SET")

func (r *Registry) New(id ErrorID) *Error {
	// Verify the ErrorID is trusted
	if !id.IsTrusted() {
		panic(fmt.Sprintf("cannot create error with untrusted ID: %s (must use fail.ID() to create)", id))
	}

	r.mu.RLock()
	def, exists := r.errors[id.String()]
	r.mu.RUnlock()

	if !exists {
		return New(UnregisteredError).WithArgs(id.String())
	}

	err := &Error{
		ID:       def.ID,
		Message:  def.Message,
		IsSystem: def.IsSystem,
		trusted:  true,
		registry: r,
	}

	// Copy default meta if present
	if len(def.Meta) > 0 {
		err.Meta = make(map[string]any, len(def.Meta))
		for k, v := range def.Meta {
			err.Meta[k] = v
		}
	}

	// Run onCreate hooks
	r.hooks.runCreate(err, map[string]any{"create": def.ID.String()})

	return err
}

var NotMatchedInAnyMapper = internalID(0, 11, false, "FailNotMatchedInAnyMapper")
var ErrNotMatchedInAnyMapper = Form(NotMatchedInAnyMapper, "error wasn't matched/mapped by any mapper", true, nil)

// FIXME Implement and enforce registry names

var NoMapperRegistered = internalID(0, 12, false, "FailNoMapperRegistered")
var ErrNoMapperRegistered = Form(NoMapperRegistered, "no mapper is registered in the registry", true, nil)

func (r *Registry) From(err error) *Error {
	if err == nil {
		return nil
	}

	if r.allowInternalLogs {
		log.Printf("[fail] From() called with: %T, msg=%q", err, err.Error())
	}

	var e *Error
	if errors.As(err, &e) {
		if e.createdByFrom {
			if r.allowInternalLogs {
				log.Printf("[fail] From() called on already-processed error: ID(%s)", e.ID.String())
			}
			return e
		} else {
			if r.allowInternalLogs {
				log.Printf("[fail] From() called on already defined fail.Error with ID(%s), consider removing redundant From() call", e.ID.String())
			}
			r.hooks.runFromSuccess(err, e)
			return e
		}
	}

	r.mu.RLock()
	mappers := r.genericMappers
	allowLogs := r.allowInternalLogs
	r.mu.RUnlock()

	if mappers != nil {
		if fe, ok := mappers.MapToFail(err); ok {
			fe.createdByFrom = true
			fe.registry = r
			fe.trusted = true
			r.hooks.runFromSuccess(err, fe)
			return fe
		}
		if allowLogs {
			log.Printf("[fail] No mapper matched error: %T, msg=%q", err, err.Error())
		}
	} else {
		if allowLogs {
			log.Printf("[fail] No mappers registered")
		}
		result := New(NoMapperRegistered).With(err)
		r.hooks.runFromFail(err)
		return result
	}

	result := New(NotMatchedInAnyMapper).With(err)
	r.hooks.runFromFail(err)
	return result
}
