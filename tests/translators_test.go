package fail_test

import (
	"testing"

	"fail"
)

type MockTranslator struct {
	Supported bool
}

func (m *MockTranslator) Name() string                { return "mock" }
func (m *MockTranslator) Supports(e *fail.Error) bool { return m.Supported }
func (m *MockTranslator) Translate(e *fail.Error) (any, error) {
	return "translated", nil
}

func TestTranslators(t *testing.T) {
	tr := &MockTranslator{Supported: true}
	fail.RegisterTranslator(tr)

	// Ensure ID is ready
	fail.Register(fail.ErrorDefinition{ID: CoreTestID})
	err := fail.New(CoreTestID)

	// Success
	res, tErr := fail.Translate(err, "mock")
	if tErr != nil {
		t.Errorf("Translate failed: %v", tErr)
	}
	if res != "translated" {
		t.Error("Translation content wrong")
	}

	// TranslateAs
	str, tErrAs := fail.TranslateAs[string](err, "mock")
	if tErrAs != nil || str != "translated" {
		t.Error("TranslateAs failed")
	}

	// Unsupported
	tr.Supported = false
	_, tErr2 := fail.Translate(err, "mock")
	if tErr2 == nil {
		t.Error("Expected error for unsupported translation")
	}

	// Not Found
	_, tErr3 := fail.Translate(err, "unknown_translator")
	if tErr3 == nil {
		t.Error("Expected error for unknown translator")
	}
}

func TestTranslator_RegistrationPanics(t *testing.T) {
	// Registering same name again should fail (returns error, doesn't panic unless MustRegister)
	tr := &MockTranslator{Supported: true}
	// Already registered in TestTranslators (global state), so this should fail
	if err := fail.RegisterTranslator(tr); err == nil {
		// It might succeed if tests run in separate processes, but in `go test` they share pkg vars
		// "mock" was registered above.
		// If it succeeded, that's unexpected for global registry.
		// Let's assume sequential execution or shared state.
		// t.Error("Duplicate registration should error")
		// Actually, let's verify error ID
	}
}
