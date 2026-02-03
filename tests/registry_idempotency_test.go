package fail_test

import (
	"testing"

	"github.com/MintzyG/fail/v3"
)

var IdemID = fail.ID(0, "IDEM", 0, true, "IdemTestError")

func TestRegistry_Idempotency(t *testing.T) {
	reg := fail.MustNewRegistry("test_registry_idem")

	// First registration
	reg.Register(&fail.Error{
		ID:      IdemID,
		Message: "first message",
	})

	// Second registration (should be ignored)
	reg.Register(&fail.Error{
		ID:      IdemID,
		Message: "second message",
	})

	err := reg.New(IdemID)
	if err.Message != "first message" {
		t.Errorf("Idempotency failed: expected 'first message', got '%s'", err.Message)
	}
}

func TestForm_Idempotency(t *testing.T) {
	reg := fail.MustNewRegistry("test_registry_form_idem")

	// First Form call
	_ = reg.Form(IdemID, "form first", false, nil)

	// Second Form call (should be ignored for registration purposes)
	_ = reg.Form(IdemID, "form second", true, nil)

	err := reg.New(IdemID)
	if err.Message != "form first" {
		t.Errorf("Form idempotency failed: expected 'form first', got '%s'", err.Message)
	}

	if err.IsSystem {
		t.Error("Form idempotency failed: IsSystem was overwritten")
	}
}
