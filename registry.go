package fail

import (
	"log"
	"sync"
)

// Registry holds all registered error definitions and mappers
type Registry struct {
	name           string
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

	allowInternalLogs      bool
	allowStaticMutations   bool
	panicOnStaticMutations bool
}

var allowRuntimePanics bool

// Global registry - users can also create their own
var global = &Registry{
	name:           "global",
	errors:         make(map[string]*Error),
	genericMappers: NewMapperList(),
	translators:    make(map[string]Translator),
	hooks:          Hooks{},
}

var (
	userRegistries   = map[string]bool{}
	userRegistriesMu sync.RWMutex
)

// MustNewRegistry creates a new isolated registry (for testing or multi-app scenarios)
// Panics if a registry of the same name is already registered
func MustNewRegistry(name string) *Registry {
	if registry, err := NewRegistry(name); err != nil {
		panic(err)
	} else {
		return registry
	}
}

// NewRegistry creates a new isolated registry (for testing or multi-app scenarios)
func NewRegistry(name string) (*Registry, error) {
	userRegistriesMu.Lock()
	defer userRegistriesMu.Unlock()

	if userRegistries[name] {
		return nil, New(RegistryAlreadyRegistered).WithArgs(name).Render()
	}
	userRegistries[name] = true

	return &Registry{
		name:           name,
		errors:         make(map[string]*Error),
		genericMappers: NewMapperList(),
		translators:    make(map[string]Translator),
		hooks:          Hooks{},
	}, nil
}

// Register adds an error definition to this registry
func (r *Registry) Register(err *Error) *Error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Verify the ErrorID is trusted
	if !err.ID.IsRegistered() {
		if r.allowInternalLogs {
			log.Printf("cannot register error with unregistered ID: %s (must use fail.ID() to create)\n", err.ID)
		}
		return New(UnregisteredIDError).WithArgs(err.ID).Render()
	}

	// First register wins (idempotent)
	if _, exists := r.errors[err.ID.String()]; exists {
		return nil
	}

	r.errors[err.ID.String()] = err
	return nil
}

// RegisterMany registers multiple error definitions at once
func RegisterMany(defs ...*ErrorDefinition) *Error {
	return global.RegisterMany(defs...)
}

func (r *Registry) RegisterMany(defs ...*ErrorDefinition) *Error {
	failures := make(map[string]*Error, len(defs))

	for _, def := range defs {
		if err := r.Register(&Error{
			ID:       def.ID,
			Message:  def.DefaultMessage,
			IsSystem: def.IsSystem,
			Meta:     def.Meta,
			isStatic: def.ID.IsStatic(),
		}); err != nil {
			failures[err.ID.String()] = err
		}
	}

	// Only return error if there were failures
	if len(failures) > 0 {
		return New(RegisterManyError).
			AddMeta("failures", failures).
			AddMeta("failure_count", len(failures)).
			AddMeta("total_count", len(defs))
	}

	return nil // All succeeded
}

func (r *Registry) New(id ErrorID) *Error {
	// Verify the ErrorID is trusted
	if !id.IsRegistered() {
		if r.allowInternalLogs {
			log.Printf("cannot New() an error with unregistered ID: %s (must use fail.ID() to register the id first)\n", id)
		}
		return New(UnregisteredIDError).WithArgs(id).Render()
	}

	r.mu.RLock()
	def, exists := r.errors[id.String()]
	r.mu.RUnlock()

	if !exists {
		return New(UnregisteredError).WithArgs(id.String()).Render()
	}

	err := &Error{
		ID:           def.ID,
		Message:      def.Message,
		IsSystem:     def.IsSystem,
		isRegistered: true,
		registry:     r,
		isStatic:     id.IsStatic(),
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
