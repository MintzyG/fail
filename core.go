package fail

import (
	"fmt"
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
	err.Message = fmt.Sprintf(format, args...)
	return err
}

// From ingests a generic error and transforms it to an Error
func From(err error) *Error {
	return global.From(err)
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
	}

	r.Register(tmpl)
	global.hooks.runForm(id, tmpl)

	return r.New(id)
}

// Error is the core error type that all domain errors implement
type Error struct {
	// Required fields
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
	trusted  bool // Whether this error was registered in the hub and should be trusted
	registry *Registry
}

// Error() uses GetRendered() for the final message
func (e *Error) Error() string {
	msg := e.GetRendered()

	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.ID.String(), msg, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.ID.String(), msg)
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
	})
}

var UnknownError = internalID(0, 7, false, "FailUnknownError")
var ErrUnknownError = Form(UnknownError, "unknown error", true, nil)
