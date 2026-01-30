package fail

import (
	"errors"
	"fmt"
)

// Must panics if the error is not nil
// Useful for initialization code
func Must(err error) {
	if err != nil {
		panic(err)
	}
}

// MustNew creates an error and panics if it's not registered
func MustNew(id ErrorID) *Error {
	err := New(id)
	if !err.trusted {
		panic(fmt.Sprintf("error ID %s not registered", id))
	}
	return err
}

// Is checks if the target error is an Error with the specified ID
func Is(err error, id ErrorID) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.ID.String() == id.String()
	}
	return false
}

// As extracts an Error from any error
func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// IsSystem checks if an error is a system error
func IsSystem(err error) bool {
	if e, ok := As(err); ok {
		return e.IsSystem
	}
	return false // Unknown errors default to non-system
}

// IsDomain checks if an error is a domain error
func IsDomain(err error) bool {
	if e, ok := As(err); ok {
		return !e.IsSystem
	}
	return false
}

// GetID extracts the error ID from any error
func GetID(err error) (ErrorID, bool) {
	if e, ok := As(err); ok {
		return e.ID, true
	}
	return ErrorID{}, false
}

// GetMessage extracts the user-facing message from any error
func GetMessage(err error) string {
	if e, ok := As(err); ok {
		return e.Message
	}
	return err.Error()
}

// GetInternalMessage extracts the internal message from an error
func GetInternalMessage(err error) string {
	if e, ok := As(err); ok {
		return e.InternalMessage
	}
	return ""
}

// GetMeta extracts metadata from an error
func GetMeta(err error, key string) (any, bool) {
	if e, ok := As(err); ok && e.Meta != nil {
		val, exists := e.Meta[key]
		return val, exists
	}
	return nil, false
}

// GetValidations extracts validation errors from an error
func GetValidations(err error) ([]ValidationError, bool) {
	if e, ok := As(err); ok && e.Meta != nil {
		if validations, ok := e.Meta["validations"].([]ValidationError); ok {
			return validations, true
		}
	}
	return nil, false
}

// GetTraces extracts trace information from an error
func GetTraces(err error) ([]string, bool) {
	if e, ok := As(err); ok && e.Meta != nil {
		if traces, ok := e.Meta["traces"].([]string); ok {
			return traces, true
		}
	}
	return nil, false
}

// GetDebug extracts debug information from an error
func GetDebug(err error) ([]string, bool) {
	if e, ok := As(err); ok && e.Meta != nil {
		if debug, ok := e.Meta["debug"].([]string); ok {
			return debug, true
		}
	}
	return nil, false
}
