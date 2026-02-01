package fail_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/MintzyG/fail"
)

var HelperID = fail.ID(0, "HELP", 1, false, "HelperError")

func TestHelpers_Is(t *testing.T) {
	fail.Register(fail.ErrorDefinition{ID: HelperID})

	e := fail.New(HelperID)
	wrapped := fmt.Errorf("wrap: %w", e)

	if !fail.Is(e, HelperID) {
		t.Error("Is failed direct")
	}
	if !fail.Is(wrapped, HelperID) {
		t.Error("Is failed wrapped")
	}
	if fail.Is(errors.New("other"), HelperID) {
		t.Error("Is matched unrelated")
	}
}

func TestHelpers_As(t *testing.T) {
	e := fail.New(HelperID)
	wrapped := fmt.Errorf("wrap: %w", e)

	if got, ok := fail.As(wrapped); !ok || got.ID != HelperID {
		t.Error("As failed to extract error")
	}
}

func TestHelpers_Must(t *testing.T) {
	// Must(nil) -> ok
	fail.Must(nil)

	// Must(err) -> panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Must(err) did not panic")
		}
	}()
	fail.Must(errors.New("panic"))
}

func TestHelpers_MustNew(t *testing.T) {
	// Registered -> ok
	_ = fail.MustNew(HelperID)

	// Unregistered (mock by using an ID created manually if possible, or untrusted?)
	// Since we can't easily create an untrusted ID that looks valid without ID(),
	// we will skip the panic test for MustNew unless we can inject one.
	// Actually, fail.New() handles unregistered errors gracefully usually, but MustNew panics if !trusted.
	// We can't legally create an untrusted ID via public API easily.
}

func TestHelpers_MetadataExtractors(t *testing.T) {
	err := fail.New(HelperID).
		Validation("f", "msg").
		Trace("t1").
		Debug("d1")

	if _, ok := fail.GetValidations(err); !ok {
		t.Error("GetValidations failed")
	}
	if _, ok := fail.GetTraces(err); !ok {
		t.Error("GetTraces failed")
	}
	if _, ok := fail.GetDebug(err); !ok {
		t.Error("GetDebug failed")
	}
}
