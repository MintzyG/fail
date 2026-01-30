package fail

// ErrorChain allows chaining multiple error checks
type ErrorChain struct {
	err error
}

// Chain starts a new error chain
func Chain() *ErrorChain {
	return &ErrorChain{}
}

// Then executes the function if no error has occurred yet
func (c *ErrorChain) Then(fn func() error) *ErrorChain {
	if c.err != nil {
		return c
	}
	c.err = fn()
	return c
}

// ThenIf executes the function conditionally
func (c *ErrorChain) ThenIf(condition bool, fn func() error) *ErrorChain {
	if c.err != nil || !condition {
		return c
	}
	c.err = fn()
	return c
}

// Error returns the final error or nil
func (c *ErrorChain) Error() error {
	return c.err
}

// Example usage:
// err := Chain().
//     Then(validateInput).
//     Then(checkPermissions).
//     Then(saveToDatabase).
//     Error()
