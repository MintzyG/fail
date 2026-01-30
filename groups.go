package fail

import "fmt"

// ErrorGroup collects multiple errors
type ErrorGroup struct {
	errors []*Error
}

// NewErrorGroup creates a new error group
func NewErrorGroup() *ErrorGroup {
	return &ErrorGroup{
		errors: make([]*Error, 0),
	}
}

// Add adds an error to the group
func (g *ErrorGroup) Add(err error) {
	if err == nil {
		return
	}

	e := From(err)
	g.errors = append(g.errors, e)
}

// Errors returns all collected errors
func (g *ErrorGroup) Errors() []*Error {
	return g.errors
}

// HasErrors returns true if the group has any errors
func (g *ErrorGroup) HasErrors() bool {
	return len(g.errors) > 0
}

// First returns the first error or nil
func (g *ErrorGroup) First() *Error {
	if len(g.errors) == 0 {
		return nil
	}
	return g.errors[0]
}

// Error implements error interface
func (g *ErrorGroup) Error() string {
	if len(g.errors) == 0 {
		return ""
	}

	if len(g.errors) == 1 {
		return g.errors[0].Error()
	}

	return fmt.Sprintf("%d errors occurred: %s (and %d more)",
		len(g.errors), g.errors[0].Error(), len(g.errors)-1)
}

// FIXME make ToError convert to fail.Error with trace meta slice for all causes
// FIXME implement ToGoError as the implementation below

// ToError converts the group to a single error
// Returns nil if no errors, first error if one, or a multi-error if multiple
func (g *ErrorGroup) ToError() error {
	if len(g.errors) == 0 {
		return nil
	}

	if len(g.errors) == 1 {
		return g.errors[0]
	}

	return g
}
