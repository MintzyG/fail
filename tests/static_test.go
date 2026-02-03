package fail_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/MintzyG/fail/v3"
)

func TestStaticErrorBuilders(t *testing.T) {
	fail.OverrideAllowIDRuntimeRegistrationForTestingOnly(true)
	defer fail.OverrideAllowIDRuntimeRegistrationForTestingOnly(false)

	// Ensure clean state
	fail.AllowStaticMutations(false, false)
	fail.AllowRuntimePanics(false)

	staticID := fail.ID(0, "STAT", 0, true, "StatStaticError")
	dynamicID := fail.ID(0, "STAT", 0, false, "StatDynamicError")

	fail.Register(fail.ErrorDefinition{
		ID:             staticID,
		DefaultMessage: "static message",
	})
	fail.Register(fail.ErrorDefinition{
		ID:             dynamicID,
		DefaultMessage: "dynamic message",
	})

	t.Run("Static error should ignore Msg by default", func(t *testing.T) {
		err := fail.New(staticID)
		_ = err.Msg("new message")
		if err.Message != "static message" {
			t.Errorf("expected message to remain 'static message', got '%s'", err.Message)
		}
	})

	t.Run("Static error should ignore Newf by default", func(t *testing.T) {
		err := fail.Newf(staticID, "formatted %d", 1)
		if err.Message != "static message" {
			t.Errorf("expected message to remain 'static message', got '%s'", err.Message)
		}
	})

	t.Run("Dynamic error should allow Msg", func(t *testing.T) {
		err := fail.New(dynamicID)
		_ = err.Msg("new message")
		if err.Message != "new message" {
			t.Errorf("expected message to be 'new message', got '%s'", err.Message)
		}
	})

	t.Run("Dynamic error should allow Newf", func(t *testing.T) {
		err := fail.Newf(dynamicID, "formatted %d", 1)
		if err.Message != "formatted 1" {
			t.Errorf("expected message to be 'formatted 1', got '%s'", err.Message)
		}
	})

	t.Run("PanicOnStaticMutations should panic when RuntimePanics enabled", func(t *testing.T) {
		// Configure to panic
		fail.AllowStaticMutations(false, true)
		fail.AllowRuntimePanics(true)
		defer func() {
			fail.AllowStaticMutations(false, false)
			fail.AllowRuntimePanics(false)
		}()

		defer func() {
			r := recover()
			if r == nil {
				t.Error("Expected panic, but did not panic")
			} else {
				// Verify panic message
				str := fmt.Sprintf("%v", r)
				if !strings.Contains(str, "modifications to static errors are discouraged") {
					t.Errorf("Unexpected panic message: %s", str)
				}
			}
		}()

		// Trigger panic
		_ = fail.New(staticID).Msg("boom")
	})

	t.Run("PanicOnStaticMutations should NOT panic if RuntimePanics disabled", func(t *testing.T) {
		// Configure to panic but disable runtime panics
		fail.AllowStaticMutations(false, true)
		fail.AllowRuntimePanics(false) // Default
		defer fail.AllowStaticMutations(false, false)

		// Should not panic, just log (and ignore mutation)
		err := fail.New(staticID).Msg("boom")

		if err.Message != "static message" {
			t.Error("Mutation should still be ignored")
		}
	})
}
