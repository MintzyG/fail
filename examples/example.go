package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/MintzyG/fail"
)

// ============================================================================
// 1. Error Definitions (using Form in var space for Sentinels)
// ============================================================================

var (
	// Auth Domain

	AuthInvalidCredentials = fail.ID(0, "AUTH", 0, true, "AuthInvalidCredentials")
	ErrInvalidCreds        = fail.Form(AuthInvalidCredentials, "invalid username or password", false, nil).
				AddLocalizations(map[string]string{
			"pt-BR": "usuário ou senha inválidos",
			"es-ES": "usuario o contraseña inválidos",
			"zh-CN": "用户名或密码无效",
		})

	AuthTokenExpired = fail.ID(0, "AUTH", 1, true, "AuthTokenExpired")
	// Sentinel for generic expiration

	_ = fail.Form(AuthTokenExpired, "authentication token has expired", false, nil)

	_ = fail.ID(1, "AUTH", 0, false, "AuthRateLimitExceeded")

	// User Domain

	UserNotFound = fail.ID(0, "USER", 0, true, "UserNotFound")
	_            = fail.Form(UserNotFound, "user was not found in the system", false, nil)

	UserValidationFailed = fail.ID(0, "USER", 0, false, "UserValidationFailed")

	// DB Domain

	DBConnectionFailed    = fail.ID(2, "DB", 0, true, "DBConnectionFailed")
	ErrDBConnectionFailed = fail.Form(DBConnectionFailed, "database connection failed", true, nil)

	DBQueryFailed = fail.ID(2, "DB", 0, false, "DBQueryFailed")
)

// ============================================================================
// 2. Custom Components (Logger, Tracer, Mapper, Translator)
// ============================================================================

type MyLogger struct{}

func (l *MyLogger) Log(e *fail.Error) {
	fmt.Printf("[LOG] %s: %s (System=%v)\n", e.ID, e.Message, e.IsSystem)
}
func (l *MyLogger) LogCtx(_ context.Context, e *fail.Error) {
	fmt.Printf("[LOG+CTX] %s: %s\n", e.ID, e.Message)
}

type MyTracer struct{}

func (t *MyTracer) Trace(op string, fn func() error) error {
	fmt.Printf("[TRACE] Starting %s...\n", op)
	defer fmt.Printf("[TRACE] Ended %s\n", op)
	return fn()
}
func (t *MyTracer) TraceCtx(ctx context.Context, op string, fn func(context.Context) error) error {
	return t.Trace(op, func() error { return fn(ctx) })
}

type MyMapper struct{}

func (m *MyMapper) Name() string                            { return "MyMapper" }
func (m *MyMapper) Priority() int                           { return 100 }
func (m *MyMapper) Map(_ error) (error, bool)               { return nil, false }
func (m *MyMapper) MapFromFail(_ *fail.Error) (error, bool) { return nil, false }
func (m *MyMapper) MapToFail(err error) (*fail.Error, bool) {
	if err.Error() == "sql: connection refused" {
		// Use sentinel for connection failures as they are usually static
		return ErrDBConnectionFailed.With(err), true
	}
	return nil, false
}

type HTTPResponse struct {
	Status int    `json:"status"`
	Error  string `json:"error"`
	Ref    string `json:"ref"`
}

type HTTPTranslator struct{}

func (t *HTTPTranslator) Name() string                 { return "http" }
func (t *HTTPTranslator) Supports(_ *fail.Error) error { return nil }
func (t *HTTPTranslator) Translate(e *fail.Error) (any, error) {
	status := 500
	if !e.IsSystem {
		switch e.ID.Domain() {
		case "AUTH":
			status = 401
		case "USER":
			status = 400
		}
	}
	return HTTPResponse{
			Status: status,
			Error:  e.Message,
			Ref:    e.ID.String(),
		},
		nil
}

// ============================================================================
// 3. Application Logic
// ============================================================================

func setup() {
	fail.SetLogger(&MyLogger{})
	fail.SetTracer(&MyTracer{})
	fail.RegisterMapper(&MyMapper{})
	if err := fail.RegisterTranslator(&HTTPTranslator{}); err != nil {
		log.Fatalf("fail register translator: %v", err)
	}

	fail.OnCreate(func(e *fail.Error, data map[string]any) {
		_ = e.AddMeta("timestamp", time.Now().Unix())
	})
}

func simulateAuth(token string) error {
	if token == "expired" {
		// Use New(ID) for dynamic messages to avoid mutating global sentinels
		return fail.New(AuthTokenExpired).
			Msg("token expired 5 mins ago").
			AddMeta("user_id", 42).
			Log()
	}
	if token == "invalid" {
		// Use sentinel directly for static errors
		return ErrInvalidCreds
	}
	return nil
}

func simulateDBAction() error {
	cfg := fail.RetryConfig{
		MaxAttempts: 3,
		Delay:       fail.BackoffConstant(10 * time.Millisecond),
		ShouldRetry: func(err error) bool {
			return fail.IsSystem(err)
		},
	}

	attempts := 0
	return fail.RetryCFG(cfg, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("sql: connection refused")
		}
		return nil
	})
}

func simulateParallelValidation() error {
	g := fail.NewErrorGroup(2)
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		// Use New(ID) for thread-safe dynamic errors
		_ = g.Add(fail.New(UserValidationFailed).Msg("email is invalid"))
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = g.Add(fail.New(UserValidationFailed).Msg("password is too short"))
	}()

	wg.Wait()

	if g.HasErrors() {
		return g.ToError()
	}
	return nil
}

func simulateWorkflow(_ context.Context) error {
	return fail.Chain(func() error {
		return simulateAuth("valid_token")
	}).
		Then(func() error {
			// Use New(ID) for context-rich errors
			return fail.New(DBQueryFailed).
				Msg("failed to fetch preferences").
				WithMeta(map[string]any{"query": "SELECT * FROM prefs"})
		}).
		Catch(func(e *fail.Error) *fail.Error {
			return e.Msg("workflow failed: " + e.Message)
		}).
		Finally(func() {
			fmt.Println("Workflow cleanup...")
		}).
		Error()
}

func main() {
	fmt.Println("=== FAIL Library Complete Example ===")
	setup()

	// 1. Basic Error & Translation
	fmt.Println("\n--- 1. Basic Error & Translation ---")
	err := simulateAuth("expired")
	if err != nil {
		resp, _ := fail.TranslateAs[HTTPResponse](fail.From(err), "http")
		fmt.Printf("HTTP Response: %+v\n", resp)
	}

	// 2. Pattern Matching
	fmt.Println("\n--- 2. Pattern Matching ---")
	err = simulateAuth("invalid")
	fail.Match(err).
		Case(AuthInvalidCredentials, func(e *fail.Error) {
			fmt.Println("Matched: Invalid Credentials!")
		}).
		CaseDomain(func(e *fail.Error) {
			fmt.Println("Matched: Generic Domain Error")
		}).
		Default(func(err error) {
			fmt.Println("Matched: Unknown")
		})

	// 3. Retry & Mapping
	fmt.Println("\n--- 3. Retry & Mapping ---")
	err = simulateDBAction()
	if err == nil {
		fmt.Println("DB Action succeeded after retry!")
	} else {
		fErr := fail.From(err)
		fmt.Printf("DB Action failed: %s (Original: %v)\n", fErr.ID, err)
	}

	// 4. Error Groups (Validation)
	fmt.Println("\n--- 4. Error Groups ---")
	err = simulateParallelValidation()
	if err != nil {
		if e, ok := fail.As(err); ok {
			if gErr, ok := e.Meta["errors"].([]*fail.Error); ok {
				fmt.Printf("Validation failed with %d errors:\n", len(gErr))
				for _, subErr := range gErr {
					fmt.Printf(" - %s\n", subErr.Message)
				}
			}
		}
	}

	// 5. Chaining & Trace
	fmt.Println("\n--- 5. Chaining & Trace ---")
	err = simulateWorkflow(context.Background())
	if err != nil {
		_ = err.(*fail.Error).LogAndRecord()
	}

	// 6. Documentation
	fmt.Println("\n--- 6. Export Documentation ---")
	docs, _ := fail.ExportIDList()
	fmt.Printf("Registered IDs (JSON snippet):\n%s\n", string(docs))

	// 7. Localization
	fmt.Println("\n--- 7. Localization ---")
	locErr := fail.New(AuthInvalidCredentials)

	// Default (English)
	fmt.Printf("En-US: %s\n", locErr.Localize().Message)

	// Portuguese
	locErr.Locale = "pt-BR"
	fmt.Printf("Pt-BR: %s\n", locErr.Localize().Message)

	// Spanish
	locErr.Locale = "es-ES"
	fmt.Printf("Es-ES: %s\n", locErr.Localize().Message)

	// Chinese
	locErr.Locale = "zh-CN"
	fmt.Printf("Zh-CN: %s\n", locErr.Localize().Message)
}
