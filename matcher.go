package fail

// ErrorMatcher provides pattern matching on error IDs
type ErrorMatcher struct {
	err     error
	failErr *Error
	matched bool
}

// Match starts a new error matcher
func Match(err error) *ErrorMatcher {
	fe, _ := As(err)
	return &ErrorMatcher{
		err:     err,
		failErr: fe,
	}
}

// Case checks if the error matches the given ID and executes the handler
func (m *ErrorMatcher) Case(id ErrorID, handler func(*Error)) *ErrorMatcher {
	if m.matched || m.failErr == nil {
		return m
	}

	if m.failErr.ID == id {
		handler(m.failErr)
		m.matched = true
	}
	return m
}

func (m *ErrorMatcher) CaseAny(handler func(*Error), ids ...ErrorID) *ErrorMatcher {
	if m.matched || m.failErr == nil {
		return m
	}

	for _, id := range ids {
		if m.failErr.ID == id {
			handler(m.failErr)
			m.matched = true
			return m
		}
	}
	return m
}

// CaseSystem handles any system error
func (m *ErrorMatcher) CaseSystem(handler func(*Error)) *ErrorMatcher {
	if m.matched || m.failErr == nil {
		return m
	}

	if m.failErr.IsSystem {
		handler(m.failErr)
		m.matched = true
	}
	return m
}

// CaseDomain handles any domain error
func (m *ErrorMatcher) CaseDomain(handler func(*Error)) *ErrorMatcher {
	if m.matched || m.failErr == nil {
		return m
	}

	if !m.failErr.IsSystem {
		handler(m.failErr)
		m.matched = true
	}
	return m
}

// Default handles any unmatched error
func (m *ErrorMatcher) Default(handler func(error)) {
	if !m.matched {
		handler(m.err)
	}
}

// Example usage of Match:
// Match(err).
//     Case("AUTH_001_S", func(e *Error) { /* handle auth error */ }).
//     Case("USER_001_S", func(e *Error) { /* handle user error */ }).
//     CaseSystem(func(e *Error) { /* handle any system error */ }).
//     Default(func(err error) { /* handle unknown error */ })
