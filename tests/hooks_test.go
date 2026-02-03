package fail_test

import (
	"errors"
	"testing"

	"github.com/MintzyG/fail/v3"
)

var HookID = fail.ID(0, "HOOKS", 0, true, "HooksLifecycleError")

func TestHooks_Lifecycle(t *testing.T) {
	// Hooks are global, so we must be careful.
	// We'll register hooks that flip flags.

	created := false
	logged := false
	matched := false

	fail.OnCreate(func(e *fail.Error, data map[string]any) {
		if e.ID == HookID {
			created = true
		}
	})

	fail.OnLog(func(e *fail.Error, data map[string]any) {
		logged = true
	})

	fail.OnMatch(func(e *fail.Error, data map[string]any) {
		matched = true
	})

	// Trigger Create
	fail.Register(fail.ErrorDefinition{ID: HookID})
	err := fail.New(HookID)

	if !created {
		t.Error("OnCreate hook failed")
	}

	// Trigger Log
	_ = err.Log()
	if !logged {
		t.Error("OnLog hook failed")
	}

	// Trigger Match
	fail.Match(err).Case(HookID, func(e *fail.Error) {})
	if !matched {
		t.Error("OnMatch hook failed")
	}
}

func TestHooks_MapFrom(t *testing.T) {
	mapped := false
	fail.OnFromFail(func(orig error) {
		mapped = true
	})

	_ = fail.From(errors.New("triggers map"))
	if !mapped {
		t.Error("OnFrom hook failed")
	}
}

func TestHooks_PanicRecovery(t *testing.T) {
	panicked := false
	afterPanic := false

	fail.OnCreate(func(e *fail.Error, data map[string]any) {
		panicked = true
		panic("hook boom")
	})

	fail.OnCreate(func(e *fail.Error, data map[string]any) {
		afterPanic = true
	})

	// Triggering hooks
	_ = fail.New(HookID)

	if !panicked {
		t.Error("First hook was not called")
	}
	if !afterPanic {
		t.Error("Second hook was not called after first hook panicked")
	}
}
