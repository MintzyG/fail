package fail_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MintzyG/fail"
)

var BuilderID = fail.ID(0, "BUILD", 1, false, "BuilderTestError")

func init() {
	fail.Register(fail.ErrorDefinition{ID: BuilderID, DefaultMessage: "base"})
}

func TestBuilder_Methods(t *testing.T) {
	cause := errors.New("underlying")

	err := fail.New(BuilderID).
		Msg("overridden").
		Msgf("formatted %d", 1).
		Internal("secret log").
		Internalf("secret %d", 2).
		With(cause).
		WithMeta(map[string]any{"a": 1}).
		AddMeta("b", 2).
		Trace("trace1").
		Traces("trace2", "trace3").
		Debug("debug1").
		Validation("field", "bad").
		Validations([]fail.ValidationError{{Field: "f2", Message: "bad2"}}).
		System() // Set to true

	// Checks
	if err.Message != "formatted 1" {
		t.Errorf("Msgf failed: %s", err.Message)
	}
	if err.InternalMessage != "secret 2" {
		t.Errorf("Internalf failed")
	}
	if !errors.Is(cause, err.Cause) {
		t.Errorf("With cause failed")
	}

	// Meta checks
	if err.Meta["a"] != 1 || err.Meta["b"] != 2 {
		t.Errorf("Meta storage failed")
	}

	// Trace checks
	traces, ok := fail.GetTraces(err)
	if !ok || len(traces) != 3 {
		t.Errorf("Traces failed: %v", traces)
	}

	// Validation checks
	validations, ok := fail.GetValidations(err)
	if !ok || len(validations) != 2 {
		t.Errorf("Validations failed: %v", validations)
	}
	if validations[0].Field != "field" || validations[1].Field != "f2" {
		t.Errorf("Validation content wrong")
	}

	// System check
	if !fail.IsSystem(err) {
		t.Error("System() didn't set IsSystem")
	}

	// Domain flip
	_ = err.Domain()
	if fail.IsSystem(err) {
		t.Error("Domain() didn't unset IsSystem")
	}
}

func TestConstructors(t *testing.T) {
	// Fast
	e1 := fail.Fast(BuilderID, "fast")
	if e1.Message != "fast" || e1.ID != BuilderID {
		t.Error("Fast failed")
	}

	// Wrap
	cause := errors.New("c")
	e2 := fail.Wrap(BuilderID, cause)
	if !errors.Is(cause, e2.Cause) {
		t.Error("Wrap failed")
	}

	// WrapMsg
	e3 := fail.WrapMsg(BuilderID, "wrapmsg", cause)
	if !errors.Is(cause, e3.Cause) || e3.Message != "wrapmsg" {
		t.Error("WrapMsg failed")
	}

	// FromWithMsg
	e4 := fail.FromWithMsg(cause, "frommsg")
	if !errors.Is(cause, e4.Cause) || e4.Message != "frommsg" {
		t.Error("FromWithMsg failed")
	}
}

// Mock logger/tracer for testing Log/Record
type mockLog struct {
	lastErr *fail.Error
	ctx     context.Context
}

func (m *mockLog) Log(e *fail.Error)                         { m.lastErr = e }
func (m *mockLog) LogCtx(ctx context.Context, e *fail.Error) { m.lastErr = e; m.ctx = ctx }

type mockTrace struct {
	called bool
}

func (m *mockTrace) Trace(_ string, fn func() error) error { m.called = true; return fn() }
func (m *mockTrace) TraceCtx(ctx context.Context, _ string, fn func(context.Context) error) error {
	m.called = true
	return fn(ctx)
}

func TestBuilder_Observability(t *testing.T) {
	ml := &mockLog{}
	mt := &mockTrace{}

	fail.SetLogger(ml)
	fail.SetTracer(mt)

	err := fail.New(BuilderID).Log().Record()

	if !errors.Is(err, ml.lastErr) {
		t.Error("Log() did not trigger logger")
	}
	if !mt.called {
		t.Error("Record() did not trigger tracer")
	}

	// Test Ctx variants
	ctx := context.Background()
	_ = err.LogAndRecordCtx(ctx)

	if ml.ctx != ctx {
		t.Error("LogCtx context missing")
	}
}
