package fail_test

import (
	"errors"
	"testing"

	"fail"
)

type TestMapper struct{}

func (m *TestMapper) Name() string                { return "TestMapper" }
func (m *TestMapper) Priority() int               { return 100 }
func (m *TestMapper) Map(err error) (error, bool) { return nil, false }
func (m *TestMapper) MapToFail(err error) (*fail.Error, bool) {
	if err.Error() == "map_me" {
		return fail.New(CoreTestID).Msg("mapped"), true // Reuse ID from core_test
	}
	return nil, false
}
func (m *TestMapper) MapFromFail(err *fail.Error) (error, bool) { return nil, false }

func TestMappers_Custom(t *testing.T) {
	// Ensure ID is registered
	fail.Register(fail.ErrorDefinition{ID: CoreTestID})

	fail.RegisterMapper(&TestMapper{})

	// Test mapping
	err := errors.New("map_me")
	res := fail.From(err)

	if res.ID != CoreTestID {
		t.Error("Mapper failed to map ID")
	}
	if res.Message != "mapped" {
		t.Error("Mapper failed to map message")
	}

	// Test fallback
	err2 := errors.New("dont_map_me")
	res2 := fail.From(err2)
	if res2.ID.Domain() != "FAIL" {
		t.Errorf("Fallback failed, expected FAIL domain got %s", res2.ID.Domain())
	}
}
