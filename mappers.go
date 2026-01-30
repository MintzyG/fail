package fail

// GenericMapper transforms generic errors into Error instances
type GenericMapper struct {
	Name      string
	Priority  int // Higher priority runs first
	Matcher   func(error) bool
	Transform func(error) *Error
}

// NewMapper is a helper function to quickly create a mapper
func NewMapper(name string, priority int, matcher func(error) bool, errorID ErrorID) GenericMapper {
	return GenericMapper{
		Name:     name,
		Priority: priority,
		Matcher:  matcher,
		Transform: func(err error) *Error {
			return New(errorID).With(err).Debug(err.Error())
		},
	}
}

// RegisterMapper adds a generic error mapper
func RegisterMapper(mapper GenericMapper) {
	global.RegisterMapper(mapper)
}

func (r *Registry) RegisterMapper(mapper GenericMapper) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Insert in priority order (higher first)
	inserted := false
	for i, existing := range r.genericMappers {
		if mapper.Priority > existing.Priority {
			r.genericMappers = append(r.genericMappers[:i], append([]GenericMapper{mapper}, r.genericMappers[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		r.genericMappers = append(r.genericMappers, mapper)
	}
}

// ErrorMatcher provides pattern matching on error IDs
type ErrorMatcher struct {
	err     error
	matched bool
}

// Match starts a new error matcher
func Match(err error) *ErrorMatcher {
	return &ErrorMatcher{err: err}
}

// Case checks if the error matches the given ID and executes the handler
func (m *ErrorMatcher) Case(id ErrorID, handler func(*Error)) *ErrorMatcher {
	if m.matched {
		return m
	}

	if Is(m.err, id) {
		if e, ok := As(m.err); ok {
			handler(e)
			m.matched = true
		}
	}
	return m
}

// CaseSystem handles any system error
func (m *ErrorMatcher) CaseSystem(handler func(*Error)) *ErrorMatcher {
	if m.matched {
		return m
	}

	if IsSystem(m.err) {
		if e, ok := As(m.err); ok {
			handler(e)
			m.matched = true
		}
	}
	return m
}

// CaseDomain handles any domain error
func (m *ErrorMatcher) CaseDomain(handler func(*Error)) *ErrorMatcher {
	if m.matched {
		return m
	}

	if IsDomain(m.err) {
		if e, ok := As(m.err); ok {
			handler(e)
			m.matched = true
		}
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
