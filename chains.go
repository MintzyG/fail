package fail

// ErrorChain enables fluent error handling with *Error as first-class citizen internally
// Accepts standard Go error functions externally, converts immediately to *Error
type ErrorChain struct {
	err  *Error
	step int
}

// Chain starts a new error chain, immediately executing the first step
// The error is normalized to *Error via From() on entry
//
// Example:
//
//	err := fail.Chain(validateRequest).
//		Then(checkPermissions).
//		ThenCtx("database", saveData).
//		Error()
func Chain(fn func() error) *ErrorChain {
	return (&ErrorChain{}).Then(fn)
}

// ChainCtx starts a chain with immediate named context
func ChainCtx(stepName string, fn func() error) *ErrorChain {
	return (&ErrorChain{}).ThenCtx(stepName, fn)
}

// Then executes the next step if no error has occurred
// Automatically converts returned error to *Error via From()
func (c *ErrorChain) Then(fn func() error) *ErrorChain {
	if c.err != nil {
		return c
	}
	if err := fn(); err != nil {
		c.err = From(err).MergeMeta(map[string]any{
			"chain_step_index": c.step,
		})
	} else {
		c.step++
	}
	return c
}

// ThenCtx executes with named step context for observability
func (c *ErrorChain) ThenCtx(stepName string, fn func() error) *ErrorChain {
	if c.err != nil {
		return c
	}
	if err := fn(); err != nil {
		c.err = From(err).MergeMeta(map[string]any{
			"chain_step":       stepName,
			"chain_step_index": c.step,
		})
	} else {
		c.step++
	}
	return c
}

// ThenIf executes conditionally only if condition is true and no prior error
func (c *ErrorChain) ThenIf(condition bool, fn func() error) *ErrorChain {
	if !condition || c.err != nil {
		return c
	}
	return c.Then(fn)
}

// ThenCtxIf executes named step conditionally
func (c *ErrorChain) ThenCtxIf(condition bool, stepName string, fn func() error) *ErrorChain {
	if !condition || c.err != nil {
		return c
	}
	return c.ThenCtx(stepName, fn)
}

// OnError executes callback if chain has failed (for logging/metrics/cleanup)
func (c *ErrorChain) OnError(fn func(*Error)) *ErrorChain {
	if c.err != nil {
		fn(c.err)
	}
	return c
}

// Catch transforms the error, enabling recovery or enrichment
// fn receives the current *Error and returns modified *Error
func (c *ErrorChain) Catch(fn func(*Error) *Error) *ErrorChain {
	if c.err != nil {
		c.err = fn(c.err)
	}
	return c
}

// Finally executes cleanup regardless of error state
func (c *ErrorChain) Finally(fn func()) *ErrorChain {
	fn()
	return c
}

// Valid returns true if no errors occurred (all steps succeeded)
func (c *ErrorChain) Valid() bool {
	return c.err == nil
}

// Error returns the final *Error or nil
// Returns concrete type for rich error handling (ID, Meta, Cause, etc.)
func (c *ErrorChain) Error() *Error {
	return c.err
}

// Unwrap implements error interface for compatibility
func (c *ErrorChain) Unwrap() error {
	return c.err
}

// Step returns count of successfully completed steps
func (c *ErrorChain) Step() int {
	return c.step
}
