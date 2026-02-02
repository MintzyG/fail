package fail_test

import (
	"testing"
	"time"

	"github.com/MintzyG/fail/v2"
)

func TestRetry_Success(t *testing.T) {
	attempts := 0
	err := fail.Retry(func() error {
		attempts++
		return nil
	})
	if err != nil {
		t.Error("Retry returned error on success")
	}
	if attempts != 1 {
		t.Error("Retry called multiple times on success")
	}
}

func TestRetry_FailAttempts(t *testing.T) {
	attempts := 0
	cfg := fail.RetryConfig{
		MaxAttempts: 3,
		Delay:       fail.BackoffConstant(1 * time.Millisecond),
	}

	err := fail.RetryCFG(cfg, func() error {
		attempts++
		return fail.New(CoreTestID2).AddMeta("retryable", true)
	})

	if err == nil {
		t.Error("RetryCFG returned nil on failure")
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetry_Value(t *testing.T) {
	attempts := 0
	val, err := fail.RetryValue(func() (int, error) {
		attempts++
		if attempts < 2 {
			// Return a retryable error (need fail.Error with retryable meta or default?)
			// IsRetryableDefault checks fail.Error.Meta["retryable"]
			return 0, fail.New(CoreTestID2).AddMeta("retryable", true)
		}
		return 42, nil
	})

	if err != nil {
		t.Errorf("RetryValue failed: %v", err)
	}
	if val != 42 {
		t.Errorf("RetryValue returned wrong value: %d", val)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}

func TestBackoffStrategies(t *testing.T) {
	// Just verify they return expected durations
	if fail.BackoffConstant(10)(1) != 10 {
		t.Error("Constant backoff wrong")
	}
	if fail.BackoffLinear(10)(2) != 20 {
		t.Error("Linear backoff wrong")
	}
	if fail.BackoffExponential(10)(3) != 40 { // 10 * 2^(3-1) = 40
		t.Error("Exponential backoff wrong")
	}
}
