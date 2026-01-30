package fail

import (
	"math/rand"
	"sync/atomic"
	"time"
)

// RetryConfig helper for transient errors
type RetryConfig struct {
	MaxAttempts int
	ShouldRetry func(error) bool

	// Delay returns how long to wait BEFORE the next attempt.
	// attempt starts at 1 for the first retry (not the first call).
	Delay func(attempt int) time.Duration
}

var retryConfig atomic.Pointer[RetryConfig]

func init() {
	retryConfig.Store(&RetryConfig{
		MaxAttempts: 5,
		ShouldRetry: IsRetryableDefault,
	})
}

func SetRetryConfig(config *RetryConfig) {
	if config == nil {
		return
	}
	if config.ShouldRetry == nil {
		config.ShouldRetry = IsRetryableDefault
	}
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = 1
	}

	retryConfig.Store(config)
}

func getRetryConfig() *RetryConfig {
	return retryConfig.Load()
}

func IsRetryableDefault(err error) bool {
	if err == nil {
		return false
	}

	// not a fail.Error? not retryable
	if e, ok := err.(*Error); !ok {
		return false
	} else {
		if v, ok := e.Meta["retryable"].(bool); ok {
			return v
		}
		return false
	}
}

// Retry executes a function with retries
func Retry(fn func() error) error {
	var lastErr error

	cfg := getRetryConfig()

	for i := 0; i < cfg.MaxAttempts; i++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if e, ok := As(err); ok {
			if !cfg.ShouldRetry(e) {
				return err
			}
		} else {
			return err
		}

		if cfg.Delay != nil && i < cfg.MaxAttempts-1 {
			time.Sleep(cfg.Delay(i + 1))
		}
	}

	return lastErr
}

// RetryCFG executes a function with retries using the passed config
func RetryCFG(config RetryConfig, fn func() error) error {
	normalizeConfig(&config)

	var lastErr error

	for i := 0; i < config.MaxAttempts; i++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		if e, ok := As(err); ok {
			if !config.ShouldRetry(e) {
				return err
			}
		} else {
			return err
		}

		if config.Delay != nil && i < config.MaxAttempts-1 {
			time.Sleep(config.Delay(i + 1))
		}
	}

	return lastErr
}

// RetryValue retries a function that returns (T, error)
func RetryValue[T any](fn func() (T, error)) (T, error) {
	var zero T
	var lastErr error

	cfg := getRetryConfig()

	for i := 0; i < cfg.MaxAttempts; i++ {
		v, err := fn()
		if err == nil {
			return v, nil
		}

		lastErr = err

		// Only retry fail.Error that is retryable
		if e, ok := As(err); ok {
			if !cfg.ShouldRetry(e) {
				return zero, err
			}
		} else {
			// Non-fail errors are not retryable
			return zero, err
		}

		if cfg.Delay != nil && i < cfg.MaxAttempts-1 {
			time.Sleep(cfg.Delay(i + 1))
		}
	}

	return zero, lastErr
}

func RetryValueCFG[T any](config RetryConfig, fn func() (T, error)) (T, error) {
	normalizeConfig(&config)

	var zero T
	var lastErr error

	for i := 0; i < config.MaxAttempts; i++ {
		v, err := fn()
		if err == nil {
			return v, nil
		}

		lastErr = err

		if e, ok := As(err); ok {
			if !config.ShouldRetry(e) {
				return zero, err
			}
		} else {
			return zero, err
		}

		if config.Delay != nil && i < config.MaxAttempts-1 {
			time.Sleep(config.Delay(i + 1))
		}
	}

	return zero, lastErr
}

func BackoffConstant(d time.Duration) func(int) time.Duration {
	return func(int) time.Duration { return d }
}

func BackoffLinear(step time.Duration) func(int) time.Duration {
	return func(attempt int) time.Duration {
		return time.Duration(attempt) * step
	}
}

func BackoffExponential(base time.Duration) func(int) time.Duration {
	return func(attempt int) time.Duration {
		return base * (1 << (attempt - 1)) // base * 2^(attempt-1)
	}
}

func WithJitter(delay func(int) time.Duration, jitterFraction float64) func(int) time.Duration {
	return func(attempt int) time.Duration {
		d := delay(attempt)
		if jitterFraction <= 0 {
			return d
		}

		jitter := time.Duration(rand.Float64() * jitterFraction * float64(d))
		return d - jitter/2 + jitter // +/- jitter/2
	}
}

func normalizeConfig(c *RetryConfig) {
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 1
	}
	if c.ShouldRetry == nil {
		c.ShouldRetry = IsRetryableDefault
	}
}

// ----------------------------------------------------
// ---------------- Retry Usage Examples --------------
// ----------------------------------------------------

/*
Basic retry with global config:

	err := fail.Retry(func() error {
		return saveToDatabase()
	})
*/

/*
Retry with a returned value:

	user, err := fail.RetryValue(func() (User, error) {
		return repo.GetUser(id)
	})
*/

/*
Override retry behavior per call:

	cfg := fail.RetryConfig{
		MaxAttempts: 3,
		ShouldRetry: fail.IsRetryableDefault,
	}

	err := fail.RetryCFG(cfg, func() error {
		return callExternalAPI()
	})
*/

/*
Retry with constant backoff:

	cfg := fail.NewRetryConfig(
		fail.WithMaxAttempts(4),
		fail.WithBackoff(fail.BackoffConstant(500 * time.Millisecond)),
	)

	err := fail.RetryCFG(cfg, doWork)
*/

/*
Retry with linear backoff:

	cfg := fail.NewRetryConfig(
		fail.WithBackoff(fail.BackoffLinear(200 * time.Millisecond)),
	)

	err := fail.RetryCFG(cfg, doWork)
*/

/*
Retry with exponential backoff:

	cfg := fail.NewRetryConfig(
		fail.WithBackoff(fail.BackoffExponential(100 * time.Millisecond)),
	)

	err := fail.RetryCFG(cfg, doWork)
*/

/*
Retry with exponential backoff + jitter (recommended for distributed systems):

	cfg := fail.NewRetryConfig(
		fail.WithMaxAttempts(5),
		fail.WithBackoff(
			fail.WithJitter(
				fail.BackoffExponential(100*time.Millisecond),
				0.3, // 30% jitter
			),
		),
	)

	err := fail.RetryCFG(cfg, doWork)
*/

/*
Retry a function that returns a value with backoff:

	cfg := fail.NewRetryConfig(
		fail.WithBackoff(fail.BackoffLinear(300 * time.Millisecond)),
	)

	result, err := fail.RetryValueCFG(cfg, func() (Result, error) {
		return fetchSomething()
	})
*/
