package fail_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/MintzyG/fail/v2" // assuming the module name is 'fail' or replaced by go.mod context
)

// Helper to handle panics in tests
func expectPanic(t *testing.T, expectedSnippet string, fn func()) {
	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("Expected panic containing '%s', but code did not panic", expectedSnippet)
			return
		}
		str := fmt.Sprintf("%v", r)
		if !strings.Contains(str, expectedSnippet) {
			t.Errorf("Expected panic containing '%s', got '%s'", expectedSnippet, str)
		}
	}()
	fn()
}

func TestID_ValidRegistration(t *testing.T) {
	fail.OverrideAllowIDRuntimeRegistrationForTestingOnly(true)
	defer fail.OverrideAllowIDRuntimeRegistrationForTestingOnly(false)

	// We need to be careful with global state.
	// In a real scenario, we might want to reset the registry, but ID() is designed for init() time.
	// We'll use a unique domain for this test to avoid conflicts.

	id := fail.ID(0, "TEST", 0, true, "TestValidID")

	if id.String() != "0_TEST_0000_S" {
		t.Errorf("Expected 0_TEST_0000_S, got %s", id.String())
	}
	if id.Name() != "TestValidID" {
		t.Errorf("Expected TestValidID, got %s", id.Name())
	}
	if id.Domain() != "TEST" {
		t.Errorf("Expected TEST, got %s", id.Domain())
	}
	if !id.IsStatic() {
		t.Errorf("Expected IsStatic true")
	}
	if !id.IsRegistered() {
		t.Errorf("Expected IsRegistered true")
	}
}

func TestID_ValidationPanics(t *testing.T) {
	fail.OverrideAllowIDRuntimeRegistrationForTestingOnly(true)
	defer fail.OverrideAllowIDRuntimeRegistrationForTestingOnly(false)

	// 1. Reserved Domain
	expectPanic(t, "reserved for internal errors", func() {
		fail.ID(0, "FAIL", 0, true, "FailSomething")
	})

	// 2. Prefix Mismatch
	expectPanic(t, "must start with domain", func() {
		fail.ID(0, "AUTH", 0, true, "UserLoginFailed")
	})

	// 3. Duplicate Name (We need a unique one first)
	uniqueName := "AuthDuplicateCheck"
	fail.ID(0, "AUTH", 0, true, uniqueName)
	expectPanic(t, "already registered", func() {
		fail.ID(0, "AUTH", 0, true, uniqueName)
	})

	// 4. Similarity
	// Register base if not exists (might from other tests, so use unique domain/name combo)
	// Using a specific domain for similarity tests to ensure isolation
	fail.ID(0, "SIM", 0, true, "SimUserNotFound")

	expectPanic(t, "too similar", func() {
		fail.ID(0, "SIM", 1, true, "SimUserNotFounds") // Distance 1
	})

	// 5. Duplicate Number
	fail.ID(0, "NUM", 0, true, "NumErrorTen")
	expectPanic(t, "number 0 already used", func() {
		fail.ID(0, "NUM", 0, true, "NumErrorTenDuplicate")
	})
}

func TestExportIDList(t *testing.T) {
	fail.OverrideAllowIDRuntimeRegistrationForTestingOnly(true)
	defer fail.OverrideAllowIDRuntimeRegistrationForTestingOnly(false)

	// Register a known ID to check export
	fail.ID(1, "EXPORT", 0, false, "ExportTestError")

	jsonBytes, err := fail.ExportIDList()
	if err != nil {
		t.Fatalf("ExportIDList failed: %v", err)
	}

	var exported []map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &exported); err != nil {
		t.Fatalf("Failed to unmarshal export: %v", err)
	}

	found := false
	for _, item := range exported {
		if item["name"] == "ExportTestError" {
			found = true
			if item["domain"] != "EXPORT" {
				t.Errorf("Exported domain mismatch")
			}
			if item["id"] != "1_EXPORT_0000_D" {
				t.Errorf("Exported ID mismatch: %v", item["id"])
			}
		}
	}

	if !found {
		t.Error("Did not find registered ID in export")
	}
}

func TestID_RuntimeBlocking(t *testing.T) {
	// Ensure override is off
	fail.OverrideAllowIDRuntimeRegistrationForTestingOnly(false)
	fail.AllowRuntimePanics(false)

	// This should return the invalid ID and log a critical error
	id := fail.ID(0, "BLCK", 0, true, "BlockedTest")

	if id.Name() != "FailRuntimeIDInvalid" {
		t.Errorf("Expected invalid ID name, got %s", id.Name())
	}
	// It's an internal library-generated ID, so it is trusted.
	if !id.IsRegistered() {
		t.Error("Invalid ID sentinel should be trusted")
	}
}

func TestID_RuntimePanic(t *testing.T) {
	fail.OverrideAllowIDRuntimeRegistrationForTestingOnly(false)
	fail.AllowRuntimePanics(true)
	defer fail.AllowRuntimePanics(false)

	expectPanic(t, "All error IDs must be defined at package initialization time", func() {
		fail.ID(0, "PANIC", 0, true, "PanicTest")
	})
}

// Note: ValidateIDs panic logic is hard to test without polluting global state with "bad" gaps,
// ensuring the test order doesn't break others. We'll skip forcing a gap panic on the global registry.
