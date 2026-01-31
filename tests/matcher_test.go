package fail_test

import (
	"errors"
	"testing"

	"fail"
)

var MatchID1 = fail.ID(0, "MATC", 1, true, "MatchAlpha")
var MatchID2 = fail.ID(0, "MATC", 2, true, "MatchBeta")

func TestMatcher(t *testing.T) {
	fail.RegisterMany(
		fail.ErrorDefinition{ID: MatchID1},
		fail.ErrorDefinition{ID: MatchID2},
	)

	tests := []struct {
		name     string
		err      error
		check    func(*fail.ErrorMatcher)
		expected string // "case1", "case2", "default", "system", "domain"
	}{
		{
			name: "Match ID1",
			err:  fail.New(MatchID1),
			check: func(m *fail.ErrorMatcher) {
				m.Case(MatchID1, func(e *fail.Error) { panic("case1") })
			},
			expected: "case1",
		},
		{
			name: "Match ID2",
			err:  fail.New(MatchID2),
			check: func(m *fail.ErrorMatcher) {
				m.Case(MatchID1, func(e *fail.Error) {}).
					Case(MatchID2, func(e *fail.Error) { panic("case2") })
			},
			expected: "case2",
		},
		{
			name: "Match System",
			err:  fail.New(MatchID1).System(),
			check: func(m *fail.ErrorMatcher) {
				m.CaseSystem(func(e *fail.Error) { panic("system") })
			},
			expected: "system",
		},
		{
			name: "Match Default",
			err:  errors.New("other"),
			check: func(m *fail.ErrorMatcher) {
				m.Case(MatchID1, func(e *fail.Error) {}).
					Default(func(err error) { panic("default") })
			},
			expected: "default",
		},
		{
			name: "Match Any",
			err:  fail.New(MatchID1),
			check: func(m *fail.ErrorMatcher) {
				m.CaseAny(func(e *fail.Error) { panic("any") }, MatchID2, MatchID1)
			},
			expected: "any",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r != tt.expected {
					t.Errorf("Expected panic '%s', got '%v'", tt.expected, r)
				}
			}()
			tt.check(fail.Match(tt.err))
		})
	}
}
