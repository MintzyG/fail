package fail_test

import (
	"errors"
	"testing"

	"github.com/MintzyG/fail/v2"
)

func TestChain_Flow(t *testing.T) {
	step1 := false
	step2 := false

	err := fail.Chain(func() error {
		step1 = true
		return nil
	}).Then(func() error {
		step2 = true
		return nil
	}).Error()

	if err != nil {
		t.Errorf("Expected nil error, got %v", err)
	}
	if !step1 || !step2 {
		t.Error("Steps not executed")
	}
}

func TestChain_StopOnError(t *testing.T) {
	step1 := false
	step2 := false
	expectedErr := errors.New("boom")

	fErr := fail.Chain(func() error {
		step1 = true
		return expectedErr
	}).Then(func() error {
		step2 = true // Should skip
		return nil
	}).Error()

	if fErr == nil || !errors.Is(expectedErr, fErr.Cause) {
		t.Error("Error not propagated")
	}
	if !step1 {
		t.Error("Step 1 not executed")
	}
	if step2 {
		t.Error("Step 2 should have been skipped")
	}
}

func TestChain_Conditionals(t *testing.T) {
	executed := false

	fail.Chain(func() error { return nil }).
		ThenIf(false, func() error {
			executed = true
			return nil
		})

	if executed {
		t.Error("ThenIf(false) executed")
	}

	fail.Chain(func() error { return nil }).
		ThenIf(true, func() error {
			executed = true
			return nil
		})

	if !executed {
		t.Error("ThenIf(true) failed to execute")
	}
}

func TestChain_Ctx_And_Catch(t *testing.T) {
	caught := false

	err := fail.ChainCtx("step1", func() error {
		return errors.New("github.com/MintzyG/fail")
	}).Catch(func(e *fail.Error) *fail.Error {
		caught = true
		step, _ := fail.GetMeta(e, "chain_step")
		if step != "step1" {
			t.Errorf("Chain context missing, got %v", step)
		}
		return e.Msg("caught")
	}).Error()

	if !caught {
		t.Error("Catch not executed")
	}
	if err.Message != "caught" {
		t.Error("Error modification in catch failed")
	}
}

func TestChain_Finally(t *testing.T) {
	finalized := false
	fail.Chain(func() error { return errors.New("err") }).
		Finally(func() { finalized = true })

	if !finalized {
		t.Error("Finally not executed on error")
	}

	finalized = false
	fail.Chain(func() error { return nil }).
		Finally(func() { finalized = true })

	if !finalized {
		t.Error("Finally not executed on success")
	}
}

func TestChain_OnError(t *testing.T) {
	called := false
	fail.Chain(func() error { return errors.New("err") }).
		OnError(func(e *fail.Error) { called = true })

	if !called {
		t.Error("OnError not called")
	}
}

func TestChain_Valid(t *testing.T) {
	c := fail.Chain(func() error { return nil })
	if !c.Valid() {
		t.Error("Valid() false on success")
	}

	c = fail.Chain(func() error { return errors.New("e") })
	if c.Valid() {
		t.Error("Valid() true on error")
	}
}
