# 🔥 FAIL - Failure Abstraction & Instrumentation Layer

**Deterministic, type-safe, compilation-order-independent error handling for Go.**

FAIL provides a revolutionary approach to error handling with **name-based deterministic IDs**, automatic validation, and beautiful ergonomics.

## ✨ What Makes FAIL Revolutionary

### 🎯 Name-Based Deterministic IDs

Error IDs are **hash-based** and **compilation-order independent**:

```go
// Define with NAMES - numbers are deterministic!
var (
    AuthInvalidCredentials = fail.ID(0, "AUTH", 0, true, "AuthInvalidCredentials")  // 0_AUTH_0000_S
    AuthTokenExpired       = fail.ID(0, "AUTH", 1, true, "AuthTokenExpired")        // 0_AUTH_0001_S
    UserNotFound           = fail.ID(0, "USER", 0, true, "UserNotFound")            // 0_USER_0000_S
)

// Numbers are based on explicit assignment per domain
// They NEVER change unless you change the number
// No more file-order issues! 🎉
```

### 🛡️ Built-in Validation

FAIL validates at runtime (via `fail.ID` or `ValidateIDs`):

```go
// ✅ Valid - name starts with domain
var Good1 = fail.ID(0, "AUTH", 0, true, "AuthInvalidCredentials")

// ❌ PANIC - name doesn't start with domain  
var Bad1 = fail.ID(0, "AUTH", 1, true, "InvalidCredentials")

// ❌ PANIC - duplicate name
var Bad2 = fail.ID(0, "AUTH", 0, true, "AuthInvalidCredentials")

// ❌ PANIC - too similar (Levenshtein distance < 3)
var Bad3 = fail.ID(0, "AUTH", 2, true, "AuthInvalidCredential")  // Too close!
```

### 🎨 One-Line Error Registration

```go
// Form() creates sentinel, registers it, and supports translations/default args!
var ErrUserNotFound = fail.Form(UserNotFound, "user %s not found", false, nil).
    AddLocalizations(map[string]string{
        "pt-BR": "usuário %s não encontrado",
        "es-ES": "usuario %s no encontrado",
    })

// Registration is IDEMPOTENT - first register wins! 🛡️
```

### 📋 Auto-Documentation

```go
// Export all your errors as JSON for documentation
fail.ExportIDList()

// Output:
// [
//   {"name": "AuthInvalidCredentials", "domain": "AUTH", "static": true, "id": "0_AUTH_0000_S"},
//   {"name": "UserNotFound", "domain": "USER", "static": true, "id": "0_USER_0000_S"},
//   ...
// ]
```

## 🚀 Quick Start

### 1. Define Error IDs

Create `errors.go`:

```go
package myapp

import "your-module/fail"

// Auth domain - names MUST start with "Auth"
var (
    AuthInvalidCredentials = fail.ID(0, "AUTH", 0, true, "AuthInvalidCredentials")
    AuthTokenExpired       = fail.ID(0, "AUTH", 1, true, "AuthTokenExpired")
    AuthValidationFailed   = fail.ID(0, "AUTH", 0, false, "AuthValidationFailed") // Dynamic!
)
```

### 2. Register Errors with Localization

```go
// Form() - one-liner sentinels with translations! 🎉
var (
    ErrUserNotFound = fail.Form(UserNotFound, "user %s not found", false, nil).
        AddLocalizations(map[string]string{
            "pt-BR": "usuário %s não encontrado",
            "es-ES": "usuario %s no encontrado",
        })
)
```

### 3. Use Everywhere

```go
func GetUser(email string) (*User, error) {
    user, err := db.GetUser(email)
    if err != nil {
        // Localization + Rendering
        return nil, fail.New(UserNotFound).
            WithLocale("pt-BR").
            WithArgs(email)
    }
    return user, nil
}
```

### 4. Render & Handle

```go
func HandleRequest(w http.ResponseWriter, r *http.Request) {
    _, err := GetUser("alice@example.com")
    if err != nil {
        // err.Error() automatically calls Render()!
        fmt.Println(err.Error()) 
        // Output: [0_USER_0000_S] usuário alice@example.com não encontrado
    }
}
```

## 🌐 Localization & Rendering

FAIL provides first-class support for multi-language applications and dynamic message templates.

### Template Rendering
Use standard `fmt` placeholders in your messages:
```go
err := fail.New(UserNotFound).WithArgs("bob@fail.com")
msg := err.Render().Message // "user bob@fail.com not found"
```

### Switching Locales
```go
// Global fallback
fail.SetDefaultLocale("en-US")

// Per-error instance
err.WithLocale("es-ES").Localize()
```

## 🎨 Core Features

### Fluent Builder API

```go
err := fail.New(UserValidationFailed).
    WithLocale("fr-FR").            // Set target language
    WithArgs("admin").              // Set template arguments
    Localize().                     // Resolve translation
    Render().                       // Format message
    Msg("override message").        // Manual override
    With(cause).
    Internal("debug info").
    Trace("step 1").
    Validation("email", "invalid").
    LogAndRecord()
```

### 🔄 Robust Retry Logic

Built-in retry mechanism with configurable backoff strategies and jitter.

```go
// Basic retry (uses default config)
err := fail.Retry(func() error {
    return db.Connect()
})

// Advanced retry with exponential backoff + jitter
cfg := fail.RetryConfig{
    MaxAttempts: 5,
    Delay: fail.WithJitter(
        fail.BackoffExponential(100*time.Millisecond), 
        0.3, // 30% jitter
    ),
    ShouldRetry: func(err error) bool {
        // Custom retry logic
        return fail.IsSystem(err)
    },
}

err := fail.RetryCFG(cfg, func() error {
    return remoteAPI.Call()
})

// Retry with return value
val, err := fail.RetryValue(func() (*User, error) {
    return repo.GetUser(id)
})
```

### 🔗 Advanced Error Chaining

Fluent chain API for executing steps and handling errors cleanly.

```go
err := fail.Chain(validateRequest).
    Then(checkPermissions).
    ThenCtx("database", saveData).       // Adds context to error if fails
    ThenIf(shouldNotify, sendEmail).     // Conditional execution
    Catch(func(e *fail.Error) *fail.Error {
        // Transform error if needed
        return e.AddMeta("caught", true)
    }).
    Finally(func() {
        cleanup()
    }).
    Error() // Returns *fail.Error or nil
```

### 📦 Error Groups

Collect multiple errors thread-safely (e.g., parallel validation).

```go
group := fail.NewErrorGroup(10)

// Add errors safely from goroutines
group.Add(err1)
group.Addf(ValidationFailed, "field %s invalid", "email")

if group.HasErrors() {
    // Returns a single error containing all others
    return group.ToError() 
}
```

### 🪝 Hooks & Lifecycle

Hook into error events for global monitoring, logging, or metrics.

```go
// Register global hooks
fail.OnCreate(func(e *fail.Error) {
    // Called when fail.New() is used
})

fail.OnLog(func(e *fail.Error, data map[string]any) {
    // Called when e.Log() is called
})

fail.OnMatch(func(e *fail.Error, data map[string]any) {
    // Called when fail.Match() succeeds
})
```

### 🔍 Observability

Integrate with your favorite tracing and logging libraries.

```go
// 1. Implement fail.Tracer and fail.Logger interfaces
type MyTracer struct { ... }
type MyLogger struct { ... }

// 2. Register them
fail.SetTracer(&MyTracer{})
fail.SetLogger(&MyLogger{})

// 3. Use in code
err := fail.New(AuthTokenExpired).
    Record(). // Traces error
    Log()     // Logs error
```

### 🔄 Generic Error Mapping

Map external errors (DB, libraries) to your domain errors.

```go
// Implement Mapper interface
type MyMapper struct{}
func (m *MyMapper) MapToFail(err error) (*fail.Error, bool) {
    if isPostgresDuplicateKey(err) {
        return fail.New(UserEmailExists), true
    }
    return nil, false
}

// Register with priority
fail.RegisterMapper(&MyMapper{})

// Usage:
err := db.Query(...)
return fail.From(err) // Automatically mapped!
```

### 🌐 Translation

Convert errors to other formats (HTTP, gRPC, CLI).

```go
// Implement Translator
type HTTPTranslator struct{}
func (t *HTTPTranslator) Translate(e *fail.Error) (any, error) {
    return HTTPResponse{Code: 400, Msg: e.Message}, nil
}

fail.RegisterTranslator(&HTTPTranslator{})

// Usage
resp, _ := fail.Translate(err, "http")
```

## 🔍 Pattern Matching

Match errors elegantly without nested if-statements.

```go
fail.Match(err).
    Case(AuthInvalidCredentials, func(e *fail.Error) {
        log.Info("invalid credentials")
    }).
    CaseDomain(func(e *fail.Error) {
        // Handle any domain error
    }).
    CaseSystem(func(e *fail.Error) {
        // Handle system/unexpected errors
        alert.Ops(e)
    }).
    Default(func(err error) {
        // Unknown error
    })
```

## 🎁 Helper Functions

```go
// Quick constructors
fail.Fast(AuthTokenExpired, "custom msg")
fail.Wrap(DBQueryFailed, dbErr)
fail.WrapMsg(DBQueryFailed, "query failed", dbErr)

// Panic on error (for init)
fail.Must(err)
fail.MustNew(AuthTokenExpired)

// Checkers
fail.Is(err, AuthTokenExpired)
fail.IsSystem(err)
fail.IsTrusted(err)
```

## 🚀 Why FAIL?

- **Deterministic**: Hash-based IDs never change unless you rename
- **Type-Safe**: ErrorID is a struct, impossible to typo
- **Validated**: Name, domain, similarity all checked at creation
- **Documented**: Export JSON list for automatic documentation
- **Ergonomic**: Beautiful fluent API with Form() helper
- **Framework-Agnostic**: Works with any HTTP framework
- **Observable**: Built-in logging and tracing hooks
- **Fun**: Actually enjoyable to use!

## 📦 Installation

```bash
go get your-module/fail
```

## 📄 License

MIT

---

**FAIL - Because deterministic error handling shouldn't be a failure! 🔥**
