package fail

import (
	"fmt"
	"strings"
	"sync"
)

// MultipleErrors is raised when multiple errors are aggregated in ErrorGroup
var MultipleErrors = internalID("FailMultipleErrors", false, 2) // Level 2: warning/moderate severity
var ErrMultipleErrors = Form(MultipleErrors, "multiple errors occurred", false, nil)

// MultipleErrorChild is used for errors that are added to an ErrorGroup generically and not as Error
var MultipleErrorChild = internalID("FailMultipleErrorChild", false, 0)
var ErrMultipleErrorChild = Form(MultipleErrorChild, "no message set yet", false, nil)

// ErrorGroup collects multiple errors thread-safely
type ErrorGroup struct {
	mu     sync.RWMutex
	errors []*Error
}

// NewErrorGroup creates a new error group
func NewErrorGroup(capacity int) *ErrorGroup {
	return &ErrorGroup{errors: make([]*Error, 0, capacity)}
}

// Add adds an error to the group (nil-safe, concurrency-safe)
func (g *ErrorGroup) Add(err error) *ErrorGroup {
	if err == nil {
		return g
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	e := From(err)
	g.errors = append(g.errors, e)
	return g
}

// Addf adds a formatted error string as a dynamic error (convenience method)
func (g *ErrorGroup) Addf(id ErrorID, format string, args ...interface{}) *ErrorGroup {
	err := Newf(id, format, args)
	return g.Add(err)
}

// Len returns the number of errors collected (concurrency-safe)
func (g *ErrorGroup) Len() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.errors)
}

// Errors returns a copy of all collected errors (concurrency-safe)
func (g *ErrorGroup) Errors() []*Error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.errors) == 0 {
		return nil
	}
	// Return copy to prevent external modification
	out := make([]*Error, len(g.errors))
	copy(out, g.errors)
	return out
}

// HasErrors returns true if the group has any errors
func (g *ErrorGroup) HasErrors() bool {
	return g.Len() > 0
}

// First returns the first error or nil (concurrency-safe)
func (g *ErrorGroup) First() *Error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.errors) == 0 {
		return nil
	}
	return g.errors[0]
}

// Last returns the last error or nil (useful for "most recent error" scenarios)
func (g *ErrorGroup) Last() *Error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if len(g.errors) == 0 {
		return nil
	}
	return g.errors[len(g.errors)-1]
}

func (g *ErrorGroup) Any(match func(*Error) bool) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, e := range g.errors {
		if match(e) {
			return true
		}
	}
	return false
}

// Error implements error interface
func (g *ErrorGroup) Error() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	count := g.Len()

	switch count {
	case 0:
		return "no errors"
	case 1:
		return g.First().Error()
	default:
		first := g.First()
		return fmt.Sprintf("%d errors occurred: %s (and %d more)",
			count, first.Error(), count-1)
	}
}

// Unwrap implements the Go 1.20+ multiple error unwrapping interface
// This allows errors.Is() and errors.As() to check against any error in the group
func (g *ErrorGroup) Unwrap() []error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	errs := make([]error, len(g.errors))
	for i, e := range g.errors {
		errs[i] = e
	}
	return errs
}

// ToError converts the group to a single *Error
// Returns nil if no errors, the first error if only one,
// or a FailMultipleErrors error containing all errors in meta if multiple
func (g *ErrorGroup) ToError() *Error {
	count := g.Len()

	if count == 0 {
		return nil
	}

	if count == 1 {
		return g.First()
	}

	// Multiple errors: create aggregated error with full context
	errs := g.Errors()

	// Build human-readable summary of error IDs/names if available
	var summary strings.Builder
	for i, e := range errs {
		if i > 0 {
			summary.WriteString(", ")
		}
		if e.IsTrusted() {
			summary.WriteString(e.ID.String())
		} else {
			summary.WriteString(e.Error())
		}
		if i >= 2 && count > 3 {
			fmt.Fprintf(&summary, " ... (+%d more)", count-i-1)
			break
		}
	}

	msg := fmt.Sprintf("%d errors occurred (first: %s)", count, errs[0].Error())

	return New(MultipleErrors).
		MergeMeta(map[string]interface{}{
			"errors":      errs,             // Full []*Error slice for programmatic access
			"error_count": count,            // Convenience counter
			"error_ids":   extractIDs(errs), // Just the IDs for quick scanning
			"summary":     summary.String(), // Human-readable list
		}).Msg(msg)
}

// Helper to extract ID strings for meta
func extractIDs(errs []*Error) []string {
	ids := make([]string, len(errs))
	for i, e := range errs {
		if e.IsTrusted() {
			ids[i] = e.ID.String()
		} else {
			ids[i] = "UNTRUSTED"
		}
	}
	return ids
}

// Reset clears all errors (useful for pooling/reuse scenarios)
func (g *ErrorGroup) Reset() *ErrorGroup {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.errors = g.errors[:0]
	return g // Keep capacity, zero length
}
