package fail

import (
	"fmt"
)

// Constructors for common patterns

// Fast creates a simple error with just an ID and custom message
func Fast(id ErrorID, message string) *Error {
	return New(id).Msg(message)
}

// Wrap creates an error that wraps another error
func Wrap(id ErrorID, cause error) *Error {
	return New(id).With(cause)
}

// WrapMsg creates an error with a custom message that wraps another error
func WrapMsg(id ErrorID, message string, cause error) *Error {
	return New(id).Msg(message).With(cause)
}

// FromWithMsg ingests a generic error and adds a custom message
func FromWithMsg(err error, message string) *Error {
	return From(err).Msg(message)
}

// Builder methods for Error - these all return *Error for easy chaining

// Clone lets you safely clone a fail.Error
func (e *Error) Clone() *Error {
	clone := *e
	if e.Meta != nil {
		clone.Meta = make(map[string]any, len(e.Meta))
		for k, v := range e.Meta {
			clone.Meta[k] = v
		}
	}
	return &clone
}

// System marks this error as a system error
func (e *Error) System() *Error {
	if e.checkStatic("System") {
		return e
	}
	e.IsSystem = true
	return e
}

// Domain marks this error as a domain error
func (e *Error) Domain() *Error {
	if e.checkStatic("Domain") {
		return e
	}
	e.IsSystem = false
	return e
}

// Msg sets or overrides the error message (for Dynamic errors)
func (e *Error) Msg(msg string) *Error {
	if e.checkStatic("Msg") {
		return e
	}
	e.Message = msg
	return e
}

// Msgf sets the error message using format string
func (e *Error) Msgf(format string, args ...any) *Error {
	if e.checkStatic("Msgf") {
		return e
	}
	e.Message = fmt.Sprintf(format, args...)
	return e
}

// Internal sets or overrides the internal message
func (e *Error) Internal(msg string) *Error {
	if e.checkStatic("Internal") {
		return e
	}
	e.InternalMessage = msg
	return e
}

// Internalf sets the internal message using format string
func (e *Error) Internalf(format string, args ...any) *Error {
	if e.checkStatic("Internalf") {
		return e
	}
	e.InternalMessage = fmt.Sprintf(format, args...)
	return e
}

// With sets the cause of the error
func (e *Error) With(cause error) *Error {
	if e.checkStatic("With") {
		return e
	}
	e.Cause = cause

	// Get registry first (with fallback)
	reg := e.registry
	if reg == nil {
		reg = global
	}
	reg.hooks.runWrap(e, cause)

	return e
}

// WithLocale sets the target locale for this error
func (e *Error) WithLocale(locale string) *Error {
	if e.checkStatic("WithLocale") {
		return e
	}
	e.Locale = locale
	return e
}

// WithArgs sets the arguments for template formatting
func (e *Error) WithArgs(args ...any) *Error {
	if e.checkStatic("WithArgs") {
		return e
	}
	e.Args = args
	return e
}

// WithMeta sets the metadata to data, it replaces existing metadata to merge use MergeMeta
func (e *Error) WithMeta(data map[string]any) *Error {
	if e.checkStatic("WithMeta") {
		return e
	}
	e.Meta = data
	return e
}

// AddMeta sets a metadata value
func (e *Error) AddMeta(key string, value any) *Error {
	if e.checkStatic("AddMeta") {
		return e
	}
	if e.Meta == nil {
		e.Meta = make(map[string]any)
	}
	e.Meta[key] = value
	return e
}

// MergeMeta merges a map into the metadata
func (e *Error) MergeMeta(data map[string]any) *Error {
	if e.checkStatic("MergeMeta") {
		return e
	}
	if e.Meta == nil {
		e.Meta = make(map[string]any, len(data))
	}
	for k, v := range data {
		e.Meta[k] = v
	}
	return e
}

// Trace adds trace information to metadata
func (e *Error) Trace(trace string) *Error {
	if e.checkStatic("Trace") {
		return e
	}
	return e.addToSliceMeta("traces", trace)
}

// Traces adds each trace information to metadata
func (e *Error) Traces(trace ...string) *Error {
	if e.checkStatic("Traces") {
		return e
	}
	for _, t := range trace {
		_ = e.addToSliceMeta("debug", t)
	}
	return e
}

// Debug adds debug information to metadata
func (e *Error) Debug(debug string) *Error {
	if e.checkStatic("Debug") {
		return e
	}
	return e.addToSliceMeta("debug", debug)
}

// Debugs adds each debug information to metadata
func (e *Error) Debugs(debug ...string) *Error {
	if e.checkStatic("Debugs") {
		return e
	}
	for _, t := range debug {
		_ = e.addToSliceMeta("debug", t)
	}
	return e
}

// ValidationError represents a field validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// NewValidationError is a helper for initializing a validation error
func NewValidationError(field, msg string) ValidationError {
	return ValidationError{Field: field, Message: msg}
}

// Validation adds a validation error to metadata
func (e *Error) Validation(field, message string) *Error {
	if e.checkStatic("Validation") {
		return e
	}
	if e.Meta == nil {
		e.Meta = make(map[string]any)
	}

	validations, exists := e.Meta["validations"]
	if !exists {
		validations = make([]ValidationError, 0, 1)
	}

	validationList, ok := validations.([]ValidationError)
	if !ok {
		validations = make([]ValidationError, 0, 1)
		validationList = validations.([]ValidationError)
	}

	validationList = append(validationList, ValidationError{
		Field:   field,
		Message: message,
	})

	e.Meta["validations"] = validationList
	return e
}

// Validations adds multiple validation errors at once
func (e *Error) Validations(errs []ValidationError) *Error {
	if e.checkStatic("Validations") {
		return e
	}
	if e.Meta == nil {
		e.Meta = make(map[string]any)
	}

	validations, exists := e.Meta["validations"]
	if !exists {
		e.Meta["validations"] = errs
		return e
	}

	validationList, ok := validations.([]ValidationError)
	if !ok {
		validations = make([]ValidationError, 0, 1)
		validationList = validations.([]ValidationError)
	}

	validationList = append(validationList, errs...)
	e.Meta["validations"] = validationList
	return e
}

// Helper to add items to slice metadata
func (e *Error) addToSliceMeta(key string, value string) *Error {
	if e.Meta == nil {
		e.Meta = make(map[string]any)
	}

	slice, exists := e.Meta[key]
	if !exists {
		e.Meta[key] = []string{value}
		return e
	}

	stringSlice := slice.([]string)
	e.Meta[key] = append(stringSlice, value)
	return e
}
