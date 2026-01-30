package fail

import (
	"fmt"
	"sync"
)

// Registry holds all registered error definitions and mappers
type Registry struct {
	mu             sync.RWMutex
	errors         map[string]*Error // Keyed by ID.String()
	genericMappers []GenericMapper
	translators    map[string]Translator

	// Hooks for automatic behavior
	onErrorCreated []func(*Error)
	tracer         Tracer
	logger         Logger
}

// Global registry - users can also create their own
var global = &Registry{
	errors:         make(map[string]*Error),
	genericMappers: make([]GenericMapper, 0),
	translators:    make(map[string]Translator),
	onErrorCreated: make([]func(*Error), 0),
}

// NewRegistry creates a new isolated registry (for testing or multi-app scenarios)
func NewRegistry() *Registry {
	return &Registry{
		errors:         make(map[string]*Error),
		genericMappers: make([]GenericMapper, 0),
		translators:    make(map[string]Translator),
		onErrorCreated: make([]func(*Error), 0),
	}
}

// Register adds an error definition to this registry
func (r *Registry) Register(err Error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Verify the ErrorID is trusted
	if !err.ID.IsTrusted() {
		panic(fmt.Sprintf("cannot register untrusted error ID: %s (must use fail.ID() to create)", err.ID))
	}

	r.errors[err.ID.String()] = &err
}

// RegisterMany registers multiple error definitions at once
func RegisterMany(defs ...ErrorDefinition) {
	global.RegisterMany(defs...)
}

func (r *Registry) RegisterMany(defs ...ErrorDefinition) {
	for _, def := range defs {
		r.Register(Error{
			ID:       def.ID,
			Message:  def.DefaultMessage,
			IsSystem: def.IsSystem,
			Meta:     def.Meta,
		})
	}
}

func (r *Registry) New(id ErrorID) *Error {
	// Verify the ErrorID is trusted
	if !id.IsTrusted() {
		panic(fmt.Sprintf("cannot create error with untrusted ID: %s (must use fail.ID() to create)", id))
	}

	r.mu.RLock()
	def, exists := r.errors[id.String()]
	r.mu.RUnlock()

	if !exists {
		// Return an unregistered error with a warning
		return &Error{
			ID:              id,
			Message:         "unregistered error",
			InternalMessage: fmt.Sprintf("error ID %s not found in registry", id),
			IsSystem:        true,
			trusted:         false,
		}
	}

	err := &Error{
		ID:       def.ID,
		Message:  def.Message,
		IsSystem: def.IsSystem,
		trusted:  true,
	}

	// Copy default meta if present
	if len(def.Meta) > 0 {
		err.Meta = make(map[string]any, len(def.Meta))
		for k, v := range def.Meta {
			err.Meta[k] = v
		}
	}

	// Run onCreate hooks
	r.runOnCreate(err)

	return err
}

func (r *Registry) From(err error) *Error {
	if err == nil {
		return nil
	}

	// Already an Error? Return as-is
	if e, ok := err.(*Error); ok {
		return e
	}

	r.mu.RLock()
	mappers := r.genericMappers
	r.mu.RUnlock()

	// Try each mapper in priority order
	for _, mapper := range mappers {
		if mapper.Matcher(err) {
			transformed := mapper.Transform(err)
			r.runOnCreate(transformed)
			return transformed
		}
	}

	// No mapper matched - create a generic system error
	// Note: This uses a hardcoded untrusted ID since it's unmapped
	return &Error{
		ID:              ErrorID{domain: "UNMAPPED", number: 0, isStatic: false, trusted: false},
		Message:         "an unexpected error occurred",
		InternalMessage: err.Error(),
		Cause:           err,
		IsSystem:        true,
		trusted:         false,
	}
}
