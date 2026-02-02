package fail_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/MintzyG/fail/v2"
)

var (
	CoreTestID  = fail.ID(0, "CORE", 0, true, "CoreTestError")
	CoreTestID2 = fail.ID(0, "CORE", 0, false, "CoreDynamicError")
)

func TestNew_And_Format(t *testing.T) {
	// Register first
	fail.Register(fail.ErrorDefinition{
		ID:             CoreTestID,
		DefaultMessage: "default core message",
		IsSystem:       false,
	})

	// Test New
	err := fail.New(CoreTestID)
	if err == nil {
		t.Fatal("New returned nil")
	}
	if err.ID != CoreTestID {
		t.Errorf("ID mismatch")
	}
	if err.Message != "default core message" {
		t.Errorf("Message mismatch: %s", err.Message)
	}

	// Test Error() string
	str := err.Error()
	expected := fmt.Sprintf("[%s] %s", CoreTestID, "default core message")
	if str != expected {
		t.Errorf("Error string mismatch. Got: %s, Want: %s", str, expected)
	}

	// Test Newf
	err2 := fail.Newf(CoreTestID2, "custom %s", "msg")
	if err2.Message != "custom msg" {
		t.Errorf("Newf message mismatch: %s", err2.Message)
	}
}

func TestForm(t *testing.T) {
	// Form registers and creates sentinel
	var SentinelErr = fail.Form(CoreTestID2, "sentinel message", true, map[string]any{"foo": "bar"})

	if SentinelErr.Message != "sentinel message" {
		t.Errorf("Form message wrong")
	}
	if !SentinelErr.IsSystem {
		t.Errorf("Form IsSystem wrong")
	}
	if SentinelErr.Meta["foo"] != "bar" {
		t.Errorf("Form meta wrong")
	}

	// Verify it's in registry by calling New
	errFromReg := fail.New(CoreTestID2)
	if errFromReg.Message != "sentinel message" {
		t.Errorf("Registry didn't capture Form message")
	}
}

func TestFrom(t *testing.T) {
	// 1. From nil
	if fail.From(nil) != nil {
		t.Error("From(nil) should return nil")
	}

	// 2. From existing *fail.Error
	existing := fail.New(CoreTestID)
	if !errors.Is(existing, fail.From(existing)) {
		t.Error("From(*fail.Error) should return identity")
	}

	// 3. From generic error (should map to generic system error if no mapper)
	stdErr := errors.New("std error")
	fErr := fail.From(stdErr)
	if !errors.Is(stdErr, fErr.Cause) {
		fmt.Printf("From(stdErr) lost cause. Got: %v, Want: %v\n", fErr.Cause, stdErr)
		t.Error("From(stdErr) lost cause")
	}
	if !fErr.IsSystem {
		t.Error("Generic error should be IsSystem=true by default")
	}
	if fErr.ID.Domain() != "FAIL" {
		t.Errorf("Expected FAIL domain, got %s", fErr.ID.Domain())
	}
}

func TestUnwrap(t *testing.T) {
	cause := errors.New("root cause")
	err := fail.Wrap(CoreTestID2, cause)

	if !errors.Is(cause, errors.Unwrap(err)) {
		t.Error("Unwrap failed to return cause")
	}

	if !errors.Is(err, cause) {
		t.Error("errors.Is failed to match cause")
	}
}
