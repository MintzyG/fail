# 🔥 FAIL - Failure Abstraction & Instrumentation Layer

**Production-grade, type-safe error handling for Go with explicit numbering, validation, and rich metadata.**

FAIL provides a revolutionary approach to error management with **explicitly numbered IDs**, automatic validation, localization support, and beautiful ergonomics.

---

## ✨ Why FAIL?

### 🎯 Explicitly Numbered, Stable IDs

Error IDs use **explicit numbering** that's stable across versions:

```go
// Define errors with explicit numbers - they NEVER change!
var (
    AuthInvalidCredentials = fail.ID(0, "AUTH", 0, true, "AuthInvalidCredentials")  // 0_AUTH_0000_S
    AuthTokenExpired       = fail.ID(0, "AUTH", 1, true, "AuthTokenExpired")        // 0_AUTH_0001_S
    UserNotFound           = fail.ID(0, "USER", 0, true, "UserNotFound")            // 0_USER_0000_S
)

// Numbers are explicitly assigned and validated
// No surprises, no auto-generation, complete control 🎯
```

### 🛡️ Built-in Validation

FAIL validates IDs at package initialization time:

```go
// ✅ Valid - name starts with domain, sequential numbering
var Good = fail.ID(0, "AUTH", 0, true, "AuthInvalidCredentials")

// ❌ PANIC - name doesn't start with domain  
var Bad1 = fail.ID(0, "AUTH", 1, true, "InvalidCredentials")

// ❌ PANIC - duplicate number
var Bad2 = fail.ID(0, "AUTH", 0, true, "AuthDuplicateNumber")

// ❌ PANIC - too similar name (Levenshtein distance < 3)
var Bad3 = fail.ID(0, "AUTH", 2, true, "AuthInvalidCredential")

// ❌ PANIC - gap in numbering (skipped from 0 to 2)
var Bad4 = fail.ID(0, "AUTH", 2, true, "AuthSkippedNumber")
```

### 🌍 First-Class Localization

Built-in support for multi-language applications:

```go
var ErrUserNotFound = fail.Form(UserNotFound, "user %s not found", false, nil).
    AddLocalizations(map[string]string{
        "pt-BR": "usuário %s não encontrado",
        "es-ES": "usuario %s no encontrado",
        "fr-FR": "utilisateur %s introuvable",
    })

// Use with automatic rendering
err := fail.New(UserNotFound).
    WithLocale("pt-BR").
    WithArgs("alice@example.com")

// err.Error() outputs: [0_USER_0000_S] usuário alice@example.com não encontrado
```

### 🔗 Fluent Builder API

Chain methods for expressive error construction:

```go
err := fail.New(UserValidationFailed).
    WithLocale("en-US").
    WithArgs("admin").
    Msg("custom message override").
    With(cause).
    Internal("debug: validation failed on email field").
    Trace("step 1: email validation").
    Validation("email", "invalid format").
    AddMeta("request_id", "abc-123").
    LogAndRecord()
```

---

## 🚀 Quick Start

### 1. Install

```bash
go get github.com/MintzyG/fail/v3
```

### 2. Define Error IDs (Centralized)

**IMPORTANT:** Define all your error IDs in one package to ensure proper sequential validation.

Create `errors/errors.go`:

```go
package errors

import "github.com/MintzyG/fail/v3"

// Auth domain errors - names MUST start with "Auth"
var (
    AuthInvalidCredentials = fail.ID(0, "AUTH", 0, true, "AuthInvalidCredentials")
    AuthTokenExpired       = fail.ID(0, "AUTH", 1, true, "AuthTokenExpired")
    AuthValidationFailed   = fail.ID(0, "AUTH", 0, false, "AuthValidationFailed") // Dynamic!
)

// User domain errors - names MUST start with "User"
var (
    UserNotFound      = fail.ID(0, "USER", 0, true, "UserNotFound")
    UserAlreadyExists = fail.ID(0, "USER", 1, true, "UserAlreadyExists")
)

// Database domain errors
var (
    DatabaseConnectionFailed = fail.ID(1, "DATABASE", 0, true, "DatabaseConnectionFailed")
    DatabaseQueryTimeout     = fail.ID(1, "DATABASE", 1, true, "DatabaseQueryTimeout")
)
```

### 3. Register Errors (Optional but Recommended)

```go
// Option A: Simple registration
func init() {
    fail.Register(fail.ErrorDefinition{
        ID:             UserNotFound,
        DefaultMessage: "user not found",
        IsSystem:       false,
    })
}

// Option B: Form() - one-liner with sentinel + localization
var ErrUserNotFound = fail.Form(
    UserNotFound, 
    "user %s not found", 
    false, 
    nil,
).AddLocalizations(map[string]string{
    "pt-BR": "usuário %s não encontrado",
    "es-ES": "usuario %s no encontrado",
})
```

### 4. Use Everywhere

```go
func GetUser(email string) (*User, error) {
    user, err := db.GetUser(email)
    if err != nil {
        return nil, fail.New(UserNotFound).
            WithArgs(email).
            With(err). // Wrap underlying error
            Internal(fmt.Sprintf("database query failed for %s", email))
    }
    return user, nil
}

func HandleRequest(w http.ResponseWriter, r *http.Request) {
    user, err := GetUser("alice@example.com")
    if err != nil {
        // err.Error() automatically localizes and renders
        log.Println(err.Error())
        // Output: [0_USER_0000_S] user alice@example.com not found
    }
}
```

---

## 📖 Core Concepts

### Error ID Structure

Format: `LEVEL_DOMAIN_NUMBER_TYPE`

- **LEVEL**: Severity level (0-9, where 0 = info, 9 = critical)
- **DOMAIN**: Error category (e.g., AUTH, USER, DATABASE)
- **NUMBER**: Sequential number within domain+type (0000-9999)
- **TYPE**: S (Static - message won't change) or D (Dynamic - message varies)

Examples:
- `0_AUTH_0000_S` - Low severity, Auth domain, first static error
- `1_DATABASE_0001_S` - Medium severity, Database domain, second static error
- `0_USER_0000_D` - Low severity, User domain, first dynamic error

### Static vs Dynamic Errors

**Static Errors** - Message is the same every time:
```go
var AuthTokenExpired = fail.ID(0, "AUTH", 1, true, "AuthTokenExpired")
// Always: "token expired"
```

**Dynamic Errors** - Message varies per occurrence:
```go
var UserValidationFailed = fail.ID(0, "USER", 0, false, "UserValidationFailed")
// Varies: "validation failed: email invalid", "validation failed: name too short", etc.
```

### Sequential Numbering

Within each `DOMAIN + TYPE` combination, numbers must be sequential starting from 0:

```go
// ✅ Correct - sequential within AUTH static
var (
    AuthError1 = fail.ID(0, "AUTH", 0, true, "AuthError1")  // 0_AUTH_0000_S
    AuthError2 = fail.ID(0, "AUTH", 1, true, "AuthError2")  // 0_AUTH_0001_S
    AuthError3 = fail.ID(0, "AUTH", 2, true, "AuthError3")  // 0_AUTH_0002_S
)

// ✅ Correct - AUTH dynamic has separate sequence
var (
    AuthDynamic1 = fail.ID(0, "AUTH", 0, false, "AuthDynamic1")  // 0_AUTH_0000_D
    AuthDynamic2 = fail.ID(0, "AUTH", 1, false, "AuthDynamic2")  // 0_AUTH_0001_D
)

// ❌ PANIC - gap in numbering (skipped number 1)
var (
    AuthError1 = fail.ID(0, "AUTH", 0, true, "AuthError1")
    AuthError3 = fail.ID(0, "AUTH", 2, true, "AuthError3")  // Missing number 1!
)
```

---

## 🎨 Features

### 🌍 Localization & Rendering

FAIL provides first-class support for multi-language applications.

#### Basic Localization

```go
// Set global default locale
fail.SetDefaultLocale("en-US")

// Add translations
var ErrUserNotFound = fail.Form(UserNotFound, "user %s not found", false, nil).
    AddLocalizations(map[string]string{
        "pt-BR": "usuário %s não encontrado",
        "es-ES": "usuario %s no encontrado",
        "fr-FR": "utilisateur %s introuvable",
    })

// Use per-error locale
err := fail.New(UserNotFound).
    WithLocale("fr-FR").
    WithArgs("bob@example.com")

msg := err.GetRendered()  // "utilisateur bob@example.com introuvable"
```

#### Template Rendering

Use standard `fmt` placeholders:

```go
err := fail.New(UserNotFound).
    WithArgs("alice@example.com").
    Render()  // Formats template with args

// Or get rendered message directly
msg := err.GetRendered()
```

#### Localization Plugin

For advanced use cases, implement the `Localizer` interface:

```go
import "github.com/MintzyG/fail/v3/plugins/localization"

localizer := localization.New()
fail.SetLocalizer(localizer)

// Bulk register translations
fail.RegisterLocalizations("pt-BR", map[fail.ErrorID]string{
    UserNotFound:      "usuário %s não encontrado",
    AuthTokenExpired:  "token expirado",
})
```

### 🔄 Robust Retry Logic

Built-in retry mechanism with configurable backoff strategies.

```go
// Basic retry (uses global config)
err := fail.Retry(func() error {
    return db.Connect()
})

// Configure global retry behavior
fail.SetRetryConfig(&fail.RetryConfig{
    MaxAttempts: 5,
    ShouldRetry: fail.IsRetryableDefault,
    Delay: fail.BackoffExponential(100 * time.Millisecond),
})

// Advanced retry with custom config
cfg := fail.RetryConfig{
    MaxAttempts: 3,
    Delay: fail.WithJitter(
        fail.BackoffExponential(100*time.Millisecond), 
        0.3, // 30% jitter
    ),
    ShouldRetry: func(err error) bool {
        return fail.IsSystem(err)
    },
}

err := fail.RetryCFG(cfg, func() error {
    return remoteAPI.Call()
})

// Retry with return value
user, err := fail.RetryValue(func() (*User, error) {
    return repo.GetUser(id)
})
```

#### Backoff Strategies

```go
// Constant backoff
fail.BackoffConstant(500 * time.Millisecond)

// Linear backoff
fail.BackoffLinear(200 * time.Millisecond)

// Exponential backoff
fail.BackoffExponential(100 * time.Millisecond)

// With jitter (recommended for distributed systems)
fail.WithJitter(
    fail.BackoffExponential(100 * time.Millisecond),
    0.3, // 30% jitter
)
```

#### Marking Errors as Retryable

```go
err := fail.New(DatabaseTimeout).
    AddMeta("retryable", true)

// Will be retried by default ShouldRetry function
fail.Retry(func() error {
    return err
})
```

### 🔗 Error Chaining

Fluent chain API for executing steps with automatic error handling.

```go
err := fail.Chain(validateRequest).
    Then(checkPermissions).
    ThenCtx("database", saveData).       // Adds context to error
    ThenIf(shouldNotify, sendEmail).     // Conditional execution
    OnError(func(e *fail.Error) {
        log.Error("chain failed", e)     // Side effects on error
    }).
    Catch(func(e *fail.Error) *fail.Error {
        // Transform error
        return e.AddMeta("caught", true)
    }).
    Finally(func() {
        cleanup()                         // Always executes
    }).
    Error()  // Returns *fail.Error or nil

// Check chain status
if err != nil {
    step := err.Error().Step()  // Get number of successful steps
}
```

### 📦 Error Groups

Collect multiple errors thread-safely (perfect for parallel validation).

```go
group := fail.NewErrorGroup(10)

// Add errors safely from goroutines
var wg sync.WaitGroup
for _, item := range items {
    wg.Add(1)
    go func(i Item) {
        defer wg.Done()
        if err := validate(i); err != nil {
            group.Add(err)
        }
    }(item)
}
wg.Wait()

// Convert to single error
if group.HasErrors() {
    return group.ToError()  // Returns MultipleErrors with all errors in metadata
}

// Access individual errors
for _, err := range group.Errors() {
    log.Println(err)
}
```

### 🪝 Hooks & Lifecycle Events

Hook into error lifecycle events for monitoring, logging, or metrics.

```go
// Global hooks
fail.OnCreate(func(e *fail.Error, data map[string]any) {
    metrics.Increment("errors.created", map[string]string{
        "domain": e.ID.Domain(),
    })
})

fail.OnLog(func(e *fail.Error, data map[string]any) {
    // Called when e.Log() is used
})

fail.OnTrace(func(e *fail.Error, data map[string]any) {
    // Called when e.Record() is used
})

fail.OnWrap(func(wrapper *fail.Error, wrapped error) {
    // Called when .With() is used
})

fail.OnMatch(func(e *fail.Error, data map[string]any) {
    // Called when error is matched in fail.Match()
})

// Available hooks:
// - HookCreate: When fail.New() is called
// - HookLog: When .Log() or .LogCtx() is called
// - HookTrace: When .Record() or .RecordCtx() is called
// - HookWrap: When .With() wraps another error
// - HookFromSuccess: When fail.From() successfully maps an error
// - HookFromFail: When fail.From() fails to map an error
// - HookForm: When fail.Form() creates a sentinel
// - HookTranslate: When error is translated
// - HookMatch: When error matches in pattern matching
```

### 📊 Observability

Integrate with your logging and tracing infrastructure.

#### Logging

```go
// Implement the Logger interface
type MyLogger struct {
    logger *zap.Logger
}

func (l *MyLogger) Log(err *fail.Error) {
    l.logger.Error("error occurred",
        zap.String("id", err.ID.String()),
        zap.String("domain", err.ID.Domain()),
        zap.String("message", err.GetRendered()),
        zap.Bool("is_system", err.IsSystem),
    )
}

func (l *MyLogger) LogCtx(ctx context.Context, err *fail.Error) {
    // Extract context values, trace IDs, etc.
    l.logger.Error("error occurred", 
        zap.String("trace_id", getTraceID(ctx)),
        zap.String("id", err.ID.String()),
    )
}

// Register your logger
fail.SetLogger(&MyLogger{logger: zapLogger})

// Use in code
err := fail.New(AuthTokenExpired).Log()  // Automatically logged
```

#### Tracing

```go
// Implement the Tracer interface
type MyTracer struct {
    tracer trace.Tracer
}

func (t *MyTracer) Record(err *fail.Error) *fail.Error {
    span := t.tracer.StartSpan("error")
    defer span.End()
    
    span.SetAttributes(
        attribute.String("error.id", err.ID.String()),
        attribute.String("error.domain", err.ID.Domain()),
    )
    return err
}

func (t *MyTracer) RecordCtx(ctx context.Context, err *fail.Error) *fail.Error {
    span := trace.SpanFromContext(ctx)
    span.RecordError(err)
    return err
}

// Register your tracer
fail.SetTracer(&MyTracer{tracer: otelTracer})

// Use in code
err := fail.New(DatabaseTimeout).Record()  // Automatically traced
```

#### OpenTelemetry Plugin

FAIL includes an OpenTelemetry plugin for easy integration:

```go
import "github.com/MintzyG/fail/v3/plugins/otel"

// Create OpenTelemetry tracer with configuration
tracer := otel.New(
    otel.WithMode(otel.RecordSmart),       // Smart mode: events for domain, status for system
    otel.WithStackTrace(),                 // Include stack traces
    otel.WithAttributePrefix("app.error"), // Custom attribute prefix
)

fail.SetTracer(tracer)

// Errors are automatically recorded in spans
err := fail.New(DatabaseTimeout).RecordCtx(ctx)
```

### 🔄 Generic Error Mapping

Map external library errors to your domain errors automatically.

```go
// Implement the Mapper interface
type PostgresMapper struct{}

func (m *PostgresMapper) Name() string {
    return "postgres"
}

func (m *PostgresMapper) Priority() int {
    return 100  // Higher priority = checked first
}

func (m *PostgresMapper) Map(err error) (*fail.Error, bool) {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        switch pgErr.Code {
        case "23505": // unique_violation
            return fail.New(UserAlreadyExists), true
        case "23503": // foreign_key_violation
            return fail.New(DatabaseForeignKeyViolation), true
        }
    }
    return nil, false
}

// Register mapper
fail.RegisterMapper(&PostgresMapper{})

// Use automatically
dbErr := db.Insert(user)
return fail.From(dbErr)  // Automatically mapped to UserAlreadyExists!
```

### 🔀 Error Translation

Convert FAIL errors to other formats (HTTP responses, gRPC status, CLI output).

```go
// Implement the Translator interface
type HTTPTranslator struct{}

func (t *HTTPTranslator) Name() string {
    return "http"
}

func (t *HTTPTranslator) Supports(err *fail.Error) error {
    // Check if error can be translated
    return nil
}

func (t *HTTPTranslator) Translate(err *fail.Error) (any, error) {
    statusCode := 500
    if !err.IsSystem {
        statusCode = 400
    }
    
    return map[string]any{
        "error": map[string]any{
            "code":    err.ID.String(),
            "message": err.GetRendered(),
            "domain":  err.ID.Domain(),
        },
        "status": statusCode,
    }, nil
}

// Register translator
fail.RegisterTranslator(&HTTPTranslator{})

// Use in HTTP handlers
func handler(w http.ResponseWriter, r *http.Request) {
    user, err := getUser(r.Context())
    if err != nil {
        if failErr, ok := fail.As(err); ok {
            resp, _ := fail.To(failErr, "http")
            json.NewEncoder(w).Encode(resp)
            return
        }
    }
}

// Type-safe translation
resp, err := fail.ToAs[HTTPResponse](failErr, "http")
```

### 🎯 Pattern Matching

Match errors elegantly without nested if-statements.

```go
fail.Match(err).
    Case(AuthInvalidCredentials, func(e *fail.Error) {
        log.Info("invalid credentials attempt")
    }).
    CaseAny(func(e *fail.Error) {
        log.Warn("authentication error")
    }, AuthTokenExpired, AuthSessionExpired).
    CaseSystem(func(e *fail.Error) {
        alert.PagerDuty(e)  // Alert on-call for system errors
    }).
    CaseDomain(func(e *fail.Error) {
        // Handle expected business logic errors
    }).
    Default(func(err error) {
        // Handle unknown errors
        log.Error("unexpected error", err)
    })
```

### 🛠️ Helper Functions

```go
// Quick constructors
err := fail.Fast(AuthTokenExpired, "custom message")
err := fail.Wrap(DatabaseQueryFailed, dbErr)
err := fail.WrapMsg(DatabaseQueryFailed, "query failed for user", dbErr)
err := fail.FromWithMsg(genericErr, "additional context")

// Panic helpers (for initialization)
fail.Must(err)
fail.MustNew(AuthTokenExpired)

// Type checking
if fail.Is(err, AuthTokenExpired) {
    // Handle specifically
}

if fail.IsSystem(err) {
    // System/infrastructure error
}

if fail.IsDomain(err) {
    // Expected business logic error
}

// Extract information
if failErr, ok := fail.As(err); ok {
    id := failErr.ID
    msg := fail.GetMessage(err)
    internal := fail.GetInternalMessage(err)
    validations, _ := fail.GetValidations(err)
}

// Metadata helpers
meta, exists := fail.GetMeta(err, "request_id")
traces, ok := fail.GetTraces(err)
debug, ok := fail.GetDebug(err)
```

### 🗂️ Metadata & Context

Attach rich metadata to errors:

```go
err := fail.New(UserValidationFailed).
    AddMeta("request_id", "abc-123").
    AddMeta("user_ip", "192.168.1.1").
    Validation("email", "invalid format").
    Validation("password", "too weak").
    Trace("step 1: validation").
    Debug("validation context: signup form")

// Extract metadata
if validations, ok := fail.GetValidations(err); ok {
    for _, v := range validations {
        fmt.Printf("Field %s: %s\n", v.Field, v.Message)
    }
}
```

---

## 🏗️ Advanced Usage

### Custom Registries

Create isolated registries for testing or multi-tenant applications:

```go
// Create custom registry
registry := fail.MustNewRegistry("tenant-a")

// Register errors to custom registry
registry.Register(&fail.Error{
    ID:       TenantSpecificError,
    Message:  "tenant specific error",
    IsSystem: false,
})

// Use custom registry
err := registry.New(TenantSpecificError)

// Check which registry an error belongs to
if err.FromRegistry(registry) {
    // Handle tenant-specific error
}
```

### ID Validation

```go
// Validate all IDs after initialization (optional but recommended)
func main() {
    fail.ValidateIDs()  // Panics if gaps or duplicates found
    
    // Your application code
}
```

### Export Error Catalog

Generate documentation from your errors:

```go
// Export all registered error IDs as JSON
data, err := fail.ExportIDList()
if err != nil {
    log.Fatal(err)
}

// Write to file for documentation
os.WriteFile("errors.json", data, 0644)

// Output format:
// [
//   {
//     "name": "AuthInvalidCredentials",
//     "domain": "AUTH",
//     "static": true,
//     "level": 0,
//     "number": 0,
//     "id": "0_AUTH_0000_S"
//   },
//   ...
// ]
```

### Configuration

```go
// Allow internal library logging (useful for debugging)
fail.AllowInternalLogs(true)

// Control static error mutation behavior
// Default: mutations silently fail
fail.AllowStaticMutations(false, false)

// Panic on static mutation attempts (strict mode)
fail.AllowStaticMutations(false, true)

// Allow mutations (not recommended)
fail.AllowStaticMutations(true, false)

// Control runtime panics
fail.AllowRuntimePanics(true)  // Panic on programming errors
```

---

## 🎓 Best Practices

### 1. Centralize Error Definitions

Define all error IDs in a single package:

```go
// errors/errors.go
package errors

import "github.com/MintzyG/fail/v3"

// All application error IDs
var (
    // Auth domain
    AuthInvalidCredentials = fail.ID(0, "AUTH", 0, true, "AuthInvalidCredentials")
    AuthTokenExpired       = fail.ID(0, "AUTH", 1, true, "AuthTokenExpired")
    
    // User domain
    UserNotFound      = fail.ID(0, "USER", 0, true, "UserNotFound")
    UserAlreadyExists = fail.ID(0, "USER", 1, true, "UserAlreadyExists")
)
```

### 2. Use Static for Predictable Messages

```go
// ✅ Good - message is always the same
var AuthTokenExpired = fail.ID(0, "AUTH", 1, true, "AuthTokenExpired")

// ❌ Bad - message varies, should be dynamic
var UserValidationFailed = fail.ID(0, "USER", 0, true, "UserValidationFailed")

// ✅ Good - dynamic for varying messages
var UserValidationFailed = fail.ID(0, "USER", 0, false, "UserValidationFailed")
```

### 3. Register with Localization

```go
var ErrUserNotFound = fail.Form(
    UserNotFound,
    "user %s not found",
    false,
    nil,
).AddLocalizations(map[string]string{
    "es-ES": "usuario %s no encontrado",
    "pt-BR": "usuário %s não encontrado",
})
```

### 4. Use Appropriate Severity Levels

```go
// Level 0: Info/expected errors
var UserNotFound = fail.ID(0, "USER", 0, true, "UserNotFound")

// Level 1: Warnings
var RateLimitExceeded = fail.ID(1, "RATE", 0, true, "RateLimitExceeded")

// Level 2-3: Errors
var DatabaseTimeout = fail.ID(2, "DATABASE", 0, true, "DatabaseTimeout")

// Level 4-5: Critical
var DatabaseConnectionLost = fail.ID(4, "DATABASE", 1, true, "DatabaseConnectionLost")
```

### 5. Wrap External Errors

```go
user, err := db.GetUser(id)
if err != nil {
    return nil, fail.New(DatabaseQueryFailed).
        With(err).  // Preserve original error
        Internal(fmt.Sprintf("failed to get user %s", id)).
        AddMeta("user_id", id)
}
```

### 6. Add Context Early

```go
if err := validateEmail(email); err != nil {
    return fail.From(err).
        AddMeta("email", email).
        AddMeta("request_id", requestID).
        Trace("email validation")
}
```

---

## 🔌 Plugins

FAIL supports optional plugins for enhanced functionality:

### Localization Plugin

```go
import "github.com/MintzyG/fail/v3/plugins/localization"

localizer := localization.New()
fail.SetLocalizer(localizer)
```

### OpenTelemetry Plugin

```go
import "github.com/MintzyG/fail/v3/plugins/otel"

tracer := otel.New(
    otel.WithMode(otel.RecordSmart),
    otel.WithStackTrace(),
)
fail.SetTracer(tracer)
```

---

## 📚 Examples

### Complete Example: User Service

```go
package main

import (
    "context"
    "github.com/MintzyG/fail/v3"
)

// Define error IDs
var (
    UserNotFound      = fail.ID(0, "USER", 0, true, "UserNotFound")
    UserAlreadyExists = fail.ID(0, "USER", 1, true, "UserAlreadyExists")
    UserValidation    = fail.ID(0, "USER", 0, false, "UserValidation")
    DatabaseError     = fail.ID(2, "DATABASE", 0, true, "DatabaseError")
)

// Register with localization
var (
    ErrUserNotFound = fail.Form(UserNotFound, "user %s not found", false, nil).
        AddLocalizations(map[string]string{
            "pt-BR": "usuário %s não encontrado",
        })
)

type UserService struct {
    db *Database
}

func (s *UserService) CreateUser(ctx context.Context, email string) error {
    // Validate input
    if err := validateEmail(email); err != nil {
        return fail.New(UserValidation).
            Validation("email", "invalid format").
            With(err)
    }
    
    // Check if exists
    existing, err := s.db.GetUser(ctx, email)
    if err == nil {
        return fail.New(UserAlreadyExists).
            WithArgs(email).
            AddMeta("existing_id", existing.ID)
    }
    
    // Create user
    if err := s.db.Insert(ctx, email); err != nil {
        return fail.From(err).  // Maps DB errors automatically
            Internal(fmt.Sprintf("failed to insert user %s", email)).
            AddMeta("email", email).
            LogAndRecordCtx(ctx)
    }
    
    return nil
}

func main() {
    // Setup
    fail.SetDefaultLocale("en-US")
    fail.SetLogger(myLogger)
    fail.SetTracer(myTracer)
    fail.RegisterMapper(&PostgresMapper{})
    
    // Validate IDs
    fail.ValidateIDs()
    
    // Run application
    service := &UserService{db: db}
    if err := service.CreateUser(ctx, "test@example.com"); err != nil {
        fail.Match(err).
            Case(UserAlreadyExists, func(e *fail.Error) {
                log.Info("user already exists")
            }).
            CaseSystem(func(e *fail.Error) {
                log.Error("system error", e)
            }).
            Default(func(err error) {
                log.Error("unexpected error", err)
            })
    }
}
```

---

## 🤝 Contributing

Contributions are welcome! Please open an issue or PR.

---

## 📄 License

MIT License

---

**FAIL - Because production-grade error handling shouldn't be a failure! 🔥**