package fail

import (
	"fmt"
	"log"
)

// ErrorType indicates whether the error message is static or dynamic
type ErrorType string

const (
	Static  ErrorType = "S" // Message won't change between occurrences
	Dynamic ErrorType = "D" // Message can vary per occurrence
)

// New returns a new Error from a registered definition
func New(id ErrorID) *Error {
	return global.New(id)
}

// Newf returns a new Error from a registered definition with a new formatted message
func Newf(id ErrorID, format string, args ...interface{}) *Error {
	err := New(id)
	if err.checkStatic("Newf") {
		return err
	}
	err.Message = fmt.Sprintf(format, args...)
	return err
}

// Form creates, registers, and returns an error in one call
// This is a convenience function for defining error sentinels
//
// WARNING Only use package level sentinel errors that are created by Form in non-concurrent environments
// For concurrent environment prefer calling New with the error ID
//
// Example:
//
//	var ErrUserNotFound = fail.Form(UserNotFound, "user not found", false, nil)
//
// This is equivalent to:
//
//	fail.Register(fail.ErrorDefinition{
//	    ID:             UserNotFound,
//	    DefaultMessage: "user not found",
//	    IsSystem:       false,
//	})
//	var ErrUserNotFound = fail.New(UserNotFound)
func Form(id ErrorID, defaultMsg string, isSystem bool, meta map[string]any, defaultArgs ...any) *Error {
	return global.Form(id, defaultMsg, isSystem, meta, defaultArgs...)
}

// FIXME, like ID Form should only be called at package level, and should panic if called after or in main

func (r *Registry) Form(id ErrorID, defaultMsg string, isSystem bool, meta map[string]any, defaultArgs ...any) *Error {
	def := ErrorDefinition{
		ID:             id,
		DefaultMessage: defaultMsg,
		IsSystem:       isSystem,
		Meta:           meta,
		DefaultArgs:    defaultArgs,
	}

	r.mu.Lock()
	if r.definitions == nil {
		r.definitions = make(map[ErrorID]ErrorDefinition)
	}
	r.definitions[id] = def
	r.mu.Unlock()

	// Create template error
	tmpl := &Error{
		ID:       id,
		Message:  defaultMsg,
		IsSystem: isSystem,
		Meta:     meta,
		Args:     defaultArgs,
		registry: r,
		isStatic: id.IsStatic(),
	}

	r.Register(tmpl)
	global.hooks.runForm(id, tmpl)

	return r.New(id)
}

// Error is the core error type that all domain errors implement
type Error struct {
	// Required fields
	// FIXME ID should be private to not let users perform 'surgery on errors'
	ID              ErrorID // Unique trusted identifier
	Message         string  // User-facing message
	InternalMessage string  // Internal/debug message (optional but recommended)
	Cause           error   // The underlying error that caused this
	IsSystem        bool    // true = infrastructure/unexpected, false = domain/expected

	Args   []any  // Captured arguments for localization
	Locale string // Target locale for this error instance

	// Optional structured data
	Meta map[string]any // Arbitrary metadata (traces, validation errors, etc.)

	// Internal tracking
	isRegistered  bool // Whether this error was registered in the hub
	registry      *Registry
	createdByFrom bool
	isStatic      bool
}

// Error() uses GetRendered() for the final message
func (e *Error) Error() string {
	msg := e.GetRendered()

	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.ID.String(), msg, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.ID.String(), msg)
}

func (e *Error) Dump() map[string]any {
	return map[string]any{
		"id":               e.ID,
		"message":          e.Message,
		"internal_message": e.InternalMessage,
		"cause":            e.Cause,
		"is_system":        e.IsSystem,
		"args":             e.Args,
		"locale":           e.Locale,
		"meta":             e.Meta,
		"is_registered":    e.isRegistered,
	}
}

// AsFail ensures the error is a *fail.Error.
// If err is already a *fail.Error, returns it as-is.
// If err is a generic error, converts via From() (uses mappers).
// If err is nil, returns nil.
//
// This is useful at boundaries where you have generic errors
// but need *fail.Error for translation, hooks, or metadata access.
//
// Example:
//
//	resp, _ := fail.To(fail.AsFail(err), "http")
func AsFail(err error) *Error {
	if err == nil {
		return nil
	}

	// Already a fail.Error? Return as-is
	if fe, ok := As(err); ok {
		return fe
	}

	// Convert via From() (uses mappers)
	return From(err)
}

// checkStatic verifies if the error is static and handles mutation attempts.
// It should only ever be called by builder methods.
//
// Returns true if the error is static AND mutations should be prevented (the builder should abort).
// Returns false if the error is not static, or if static mutations are allowed.
//
// Behavior depends on registry settings:
//   - panicOnStaticMutations AND allowRuntimePanics: both must be true to panic.
//     This prevents accidental panics if a developer disables runtime panics globally
//     but forgets to also disable panicOnStaticMutations.
//   - allowInternalLogs: if true and panics are disabled, logs warnings
//   - allowStaticMutations: if true, allows the mutation but may log a warning
func (e *Error) checkStatic(builderName string) bool {
	if !e.isStatic {
		return false
	}

	reg := e.registry
	if reg == nil {
		reg = global
	}

	// Only panic if explicitly enabled
	if reg.panicOnStaticMutations && allowRuntimePanics {
		panic(fmt.Sprintf("[fail] error: builder %s() called on static error with ID(%s), modifications to static errors are discouraged\n", builderName, e.ID.String()))
	}

	if reg.allowInternalLogs {
		log.Printf("[fail] warning: %s() called on static error ID(%s)", builderName, e.ID)
	}

	// Silently fail, we are never allowed to mutate static errors on the global registry
	// On an upcoming update custom user registries will have a toggle to allow static mutation EWWWW
	return true
}

// Unwrap implements error unwrapping for errors.Is/As
func (e *Error) Unwrap() error {
	if e.Cause != nil {
		return e.Cause
	}
	return nil
}

// ErrorDefinition is the blueprint for creating errors
type ErrorDefinition struct {
	ID             ErrorID
	DefaultMessage string // Used for Static errors or as fallback
	IsSystem       bool
	Meta           map[string]any // Default metadata to include
	DefaultArgs    []any
}

// Register adds an error definition to the global registry
func Register(def ErrorDefinition) {
	global.Register(&Error{
		ID:       def.ID,
		Message:  def.DefaultMessage,
		IsSystem: def.IsSystem,
		Meta:     def.Meta,
		isStatic: def.ID.IsStatic(),
	})
}

// AllowInternalLogs enables or disables internal library logging.
// When enabled, the library will log warnings and debug information
// about error processing, such as:
//   - Double calls to From()
//   - Unmapped errors
//   - Mapper registration issues
//
// This is useful for debugging integration issues but should be disabled
// in production to avoid log spam. Default is false.
//
// Example:
//
//	fail.AllowInternalLogs(true)  // Enable logging
//	fail.AllowInternalLogs(false) // Disable logging (default)
func AllowInternalLogs(allow bool) {
	global.AllowInternalLogs(allow)
}

// AllowInternalLogs enables or disables internal library logging for this registry.
// See AllowInternalLogs for details.
func (r *Registry) AllowInternalLogs(allow bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.allowInternalLogs = allow
}

// AllowStaticMutations controls whether static errors in the global registry
// can be mutated. When set to false (default), builder methods on static errors
// will silently return the original error without modifications. When set to true,
// shoudlPanic is ignored and mutations are allowed but warnings are logged
// if internal logging is enabled.
//
// If shouldPanic is true and allow is false, attempts to modify static errors will
// panic instead of failing silently. This overrides the default silent behavior.
//
// This is a convenience wrapper for global.AllowStaticMutations(allow, shouldPanic).
func AllowStaticMutations(allow bool, shouldPanic bool) {
	global.AllowStaticMutations(allow, shouldPanic)
}

// AllowStaticMutations controls whether static errors in this registry can be
// mutated. When allow is false (default), builder methods on static errors return
// the original error unchanged. When allow is true shoudlPanic is ignored, mutations are permitted but
// may log warnings if internal logging is enabled.
//
// If shouldPanic is true and allow is false, any builder method called on a static
// error will panic immediately with a descriptive message. This is useful for
// catching programming errors during development.
//
// This method is safe for concurrent use.
func (r *Registry) AllowStaticMutations(allow bool, shouldPanic bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if allow {
		r.allowStaticMutations = allow
		r.panicOnStaticMutations = false
		return
	}
	r.panicOnStaticMutations = shouldPanic
	r.allowStaticMutations = false
}

// AllowRuntimePanics controls whether the library may panic at runtime.
// When true, operations that would normally log warnings or fail
// silently will instead panic immediately (e.g., modifying static errors).
// When false (default), the library avoids panics and uses error logging or
// silent failures where appropriate.
//
// Note: The ID() method will always panic regardless of this setting.
//
// Enabling this is recommended during development and testing to catch
// programming errors early, but should typically be disabled in production.
func AllowRuntimePanics(allow bool) {
	allowRuntimePanics = allow
}
