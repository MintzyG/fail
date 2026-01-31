package fail_test

import (
	"errors"
	"strings"
	"testing"

	"fail"
)

var GroupID = fail.ID(0, "GROUP", 1, false, "GroupError")

func TestErrorGroup_Basics(t *testing.T) {
	g := fail.NewErrorGroup(2)

	if g.HasErrors() {
		t.Error("New group shouldn't have errors")
	}

	g.Add(nil) // Should ignore
	if g.Len() != 0 {
		t.Error("Add(nil) increased length")
	}

	g.Add(errors.New("e1"))
	g.Addf(GroupID, "e%d", 2)

	if g.Len() != 2 {
		t.Errorf("Expected 2 errors, got %d", g.Len())
	}

	if g.HasErrors() == false {
		t.Error("HasErrors false")
	}
}

func TestErrorGroup_ToError(t *testing.T) {
	// 0 errors
	g0 := fail.NewErrorGroup(0)
	if g0.ToError() != nil {
		t.Error("ToError should be nil for empty group")
	}

	// 1 error -> returns that error directly
	g1 := fail.NewErrorGroup(1)
	baseErr := fail.New(GroupID)
	g1.Add(baseErr)
	if g1.ToError() != baseErr {
		t.Error("ToError with 1 error should return it directly")
	}

	// Multiple errors -> Aggregate
	g2 := fail.NewErrorGroup(2)
	g2.Add(errors.New("err1"))
	g2.Add(errors.New("err2"))

	agg := g2.ToError()
	if agg.ID.Name() != "FailMultipleErrors" {
		t.Errorf("Expected FailMultipleErrors, got %s", agg.ID.Name())
	}

	// Check text
	if !strings.Contains(agg.Message, "2 errors occurred") {
		t.Error("Message should mention count")
	}
}

func TestErrorGroup_Concurrency(t *testing.T) {
	g := fail.NewErrorGroup(100)
	done := make(chan bool)

	for i := 0; i < 100; i++ {
		go func() {
			g.Add(errors.New("err"))
			done <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	if g.Len() != 100 {
		t.Errorf("Race condition suspected? Len=%d", g.Len())
	}
}

func TestErrorGroup_Unwrap(t *testing.T) {
	g := fail.NewErrorGroup(2)
	e1 := errors.New("match_me")
	e2 := errors.New("other")
	g.Add(e1)
	g.Add(e2)

	// Go 1.20+ errors.Is support via Unwrap() []error
	// Since we are inside the test, we can check if it works with errors.Is
	// assuming the test runner is on a recent Go version.
	// But strictly, we can test Unwrap() return value.

	errs := g.Unwrap()
	if len(errs) != 2 {
		t.Error("Unwrap returned wrong count")
	}
}
