# 🔥 FAIL - Failure Abstraction & Instrumentation Layer

**Deterministic, type-safe, compilation-order-independent error handling for Go.**

FAIL provides a revolutionary approach to error handling with **name-based deterministic IDs**, automatic validation, and beautiful ergonomics.

## ✨ What Makes FAIL Revolutionary

### 🎯 Name-Based Deterministic IDs

Error IDs are **hash-based** and **compilation-order independent**:

```go
// Define with NAMES - numbers are deterministic!
var (
    AuthInvalidCredentials = fail.ID("AuthInvalidCredentials", "AUTH", true)  // AUTH_4721_S
    AuthTokenExpired       = fail.ID("AuthTokenExpired", "AUTH", true)        // AUTH_8392_S
    UserNotFound           = fail.ID("UserNotFound", "USER", true)            // USER_1847_S
)

// Numbers are based on SHA-256 hash of the name
// They NEVER change unless you rename the error
// No more file-order issues! 🎉
```

### 🛡️ Built-in Validation

FAIL validates at compile time (via `init()` or first use):

```go
// ✅ Valid - name starts with domain
var Good1 = fail.ID("AuthInvalidCredentials", "AUTH", true)

// ❌ PANIC - name doesn't start with domain  
var Bad1 = fail.ID("InvalidCredentials", "AUTH", true)

// ❌ PANIC - duplicate name
var Bad2 = fail.ID("AuthInvalidCredentials", "AUTH", true)

// ❌ PANIC - too similar (Levenshtein distance < 3)
var Bad3 = fail.ID("AuthInvalidCredential", "AUTH", true)  // Too close!
var Bad4 = fail.ID("UserNotFounds", "USER", true)         // Too close to UserNotFound!
```

### 🎨 One-Line Error Registration

```go
// Old way - verbose
fail.Register(fail.ErrorDefinition{
    ID:             UserNotFound,
    DefaultMessage: "user not found",
    IsSystem:       false,
})

// New way - Form() creates sentinel and registers in one line!
var ErrUserNotFound = fail.Form(UserNotFound, "user not found", false, nil)
```

### 📋 Auto-Documentation

```go
// Export all your errors as JSON for documentation
fail.ExportIDList()

// Output:
// [
//   {"name": "AuthInvalidCredentials", "domain": "AUTH", "static": true, "id": "AUTH_4721_S"},
//   {"name": "UserNotFound", "domain": "USER", "static": true, "id": "USER_1847_S"},
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
    AuthInvalidCredentials = fail.ID("AuthInvalidCredentials", "AUTH", true)
    AuthUserNotFound       = fail.ID("AuthUserNotFound", "AUTH", true)
    AuthTokenExpired       = fail.ID("AuthTokenExpired", "AUTH", true)
    AuthValidationFailed   = fail.ID("AuthValidationFailed", "AUTH", false) // Dynamic!
)

// User domain - names MUST start with "User"
var (
    UserNotFound         = fail.ID("UserNotFound", "USER", true)
    UserEmailExists      = fail.ID("UserEmailExists", "USER", true)
    UserValidationFailed = fail.ID("UserValidationFailed", "USER", false)
)
```

### 2. Register Errors

```go
// Option 1: Traditional registration
func init() {
    fail.RegisterMany(
        fail.ErrorDefinition{
            ID:             AuthInvalidCredentials,
            DefaultMessage: "invalid credentials",
            IsSystem:       false,
        },
        fail.ErrorDefinition{
            ID:             UserNotFound,
            DefaultMessage: "user not found",
            IsSystem:       false,
        },
    )
}

// Option 2: Form() - one-liner sentinels! 🎉
var (
    ErrUserNotFound    = fail.Form(UserNotFound, "user not found", false, nil)
    ErrUserEmailExists = fail.Form(UserEmailExists, "email already registered", false, nil)
)
```

### 3. Use Everywhere

```go
func Login(email, password string) error {
    // Static error
    if !valid {
        return fail.New(AuthInvalidCredentials)
    }
    
    // Dynamic error
    return fail.New(AuthValidationFailed).
        Msg("authentication failed").
        Validation("email", "invalid format").
        Trace("checked credentials")
}

func GetUser(id int) (*User, error) {
    user, err := db.GetUser(id)
    if err != nil {
        // Using sentinel
        return nil, ErrUserNotFound
    }
    return user, nil
}
```

### 4. Handle Errors

```go
func HandleLogin(w http.ResponseWriter, r *http.Request) {
    err := Login(email, password)
    if err != nil {
        resp, _ := fail.Translate(fail.From(err), "http")
        httpResp := resp.(fail.HTTPResponse)
        
        w.WriteHeader(httpResp.StatusCode)
        json.NewEncoder(w).Encode(httpResp)
        return
    }
    
    w.WriteHeader(200)
}
```

## 🎯 ID Format

Format: `DOMAIN_NNNN_(S|D)`

- **DOMAIN**: Error category (AUTH, USER, DB, etc.)
- **NNNN**: 4-digit hash-based number (0000-9999)
- **S/D**: Static or Dynamic message

**Hash-Based Numbering:**
```go
// Numbers are deterministic based on name hash
AuthInvalidCredentials // AUTH_4721_S - always this number!
AuthTokenExpired       // AUTH_8392_S - always this number!
UserNotFound           // USER_1847_S - always this number!

// Even if you reorder files or change import order, IDs stay the same!
```

**Name Requirements:**
```go
// ✅ Valid - name starts with domain
fail.ID("AuthInvalidCredentials", "AUTH", true)
fail.ID("UserNotFound", "USER", true)
fail.ID("DBConnectionFailed", "DB", true)

// ❌ Invalid - name doesn't start with domain
fail.ID("InvalidCredentials", "AUTH", true) // PANIC!
fail.ID("NotFound", "USER", true)           // PANIC!
```

**Similarity Protection:**
```go
// ✅ Valid - names are sufficiently different
fail.ID("UserNotFound", "USER", true)
fail.ID("UserNotActive", "USER", true)      // Distance = 7, OK!

// ❌ Invalid - names too similar (distance < 3)
fail.ID("UserNotFound", "USER", true)
fail.ID("UserNotFounds", "USER", true)      // Distance = 1, PANIC!
fail.ID("UserNot Found", "USER", true)      // Distance = 1, PANIC!
```

## 🎨 Core Features

### Fluent Builder API

```go
err := fail.New(UserValidationFailed).
    Msg("validation failed").
    Msgf("failed for user %s", email).
    With(cause).
    Internal("debug info").
    Trace("step 1").
    Debug("SQL: SELECT...").
    Validation("email", "invalid").
    WithMeta("key", value).
    System().
    LogAndRecord()
```

### Form() Convenience

```go
// Create and register in one line
var (
    ErrUserNotFound    = fail.Form(UserNotFound, "user not found", false, nil)
    ErrDBConnFailed    = fail.Form(DBConnectionFailed, "db connection failed", true, nil)
    ErrInvalidInput    = fail.Form(ValidationFailed, "invalid input", false, map[string]any{
        "category": "validation",
    })
)

// Use directly
return ErrUserNotFound
return ErrUserNotFound.WithMeta("user_id", 123)
```

### Auto-Documentation

```go
// Generate JSON documentation of all errors
fail.ExportIDList()

// Output format:
// [
//   {
//     "name": "AuthInvalidCredentials",
//     "domain": "AUTH",
//     "static": true,
//     "id": "AUTH_4721_S"
//   },
//   {
//     "name": "UserValidationFailed",
//     "domain": "USER",
//     "static": false,
//     "id": "USER_3892_D"
//   }
// ]

// Pipe to file for documentation
// go run main.go > errors.json
```

### HTTP Translation

```go
// Setup once
fail.RegisterTranslator(fail.DefaultHTTPTranslator())

// Use everywhere
err := fail.New(AuthInvalidCredentials)
resp, _ := fail.Translate(err, "http")
httpResp := resp.(fail.HTTPResponse)

// HTTP status codes inferred from domain:
// AUTH    -> 401 Unauthorized
// USER    -> 400 Bad Request
// PERM    -> 403 Forbidden
// DB      -> 500 Internal Server Error (if IsSystem=true)
```

### Validation Errors

```go
err := fail.New(UserValidationFailed).
    Msg("registration failed").
    Validation("email", "invalid format").
    Validation("password", "too short").
    Validation("age", "must be 18+")

// Clean JSON response:
// {
//   "error_id": "USER_3892_D",
//   "message": "registration failed",
//   "validations": [
//     {"field": "email", "message": "invalid format"},
//     {"field": "password", "message": "too short"},
//     {"field": "age", "message": "must be 18+"}
//   ]
// }
```

### Error Checking

```go
// Type-safe checking
if fail.Is(err, AuthInvalidCredentials) {
    // Handle auth error
}

if fail.IsSystem(err) {
    // Alert ops
}

// Pattern matching
fail.Match(err).
    Case(AuthInvalidCredentials, func(e *fail.Error) {
        log.Info("invalid credentials")
    }).
    Case(UserNotFound, func(e *fail.Error) {
        log.Info("user not found")
    }).
    CaseSystem(func(e *fail.Error) {
        alert.Ops(e)
    }).
    Default(func(err error) {
        log.Error("unknown error")
    })
```

### Generic Error Mapping

```go
// Auto-map stdlib/library errors
fail.RegisterSQLMappers()

err := db.Query(...)
return fail.From(err) // Auto-mapped to SQLNotFound, SQLUniqueViolation, etc.

// Custom mappers
var UserEmailExists = fail.ID("UserEmailExists", "USER", true)

mapper := fail.NewPGXMapper().
    MapUniqueConstraint("users_email_key", UserEmailExists)
fail.RegisterMapper(mapper.ToGenericMapper())
```

### Observability

```go
// Setup once
fail.QuickSetupOTel(span)
fail.QuickSetupSlog(logger)

// All errors auto-traced and logged
err := fail.New(AuthTokenExpired).Record().Log()

// Custom hooks
fail.OnErrorCreated(func(e *fail.Error) {
    if e.IsSystem {
        sentry.CaptureError(e)
    }
})
```

## 🔍 Real-World Example

```go
// errors.go
package myapp

import "your-module/fail"

var (
    // Define IDs
    AuthInvalidCredentials = fail.ID("AuthInvalidCredentials", "AUTH", true)
    AuthTokenExpired       = fail.ID("AuthTokenExpired", "AUTH", true)
    UserNotFound           = fail.ID("UserNotFound", "USER", true)
    UserEmailExists        = fail.ID("UserEmailExists", "USER", true)
    
    // Create sentinels
    ErrAuthInvalidCreds = fail.Form(AuthInvalidCredentials, "invalid credentials", false, nil)
    ErrUserNotFound     = fail.Form(UserNotFound, "user not found", false, nil)
    ErrUserEmailExists  = fail.Form(UserEmailExists, "email already registered", false, nil)
)

// service.go
func (s *Service) Login(ctx context.Context, email, password string) error {
    user, err := s.db.GetUserByEmail(ctx, email)
    if err != nil {
        if fail.Is(err, SQLNotFound) {
            return ErrUserNotFound
        }
        return fail.From(err).Trace("fetching user")
    }
    
    if !s.validatePassword(password, user.Hash) {
        return ErrAuthInvalidCreds
    }
    
    return nil
}

func (s *Service) Register(ctx context.Context, email, password string) error {
    err := s.db.CreateUser(ctx, email, password)
    if err != nil {
        // Auto-mapped to UserEmailExists if constraint violated!
        return fail.From(err).WithMeta("email", email)
    }
    return nil
}

// handler.go
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
    err := h.service.Login(r.Context(), email, password)
    if err != nil {
        resp, _ := fail.Translate(fail.From(err), "http")
        httpResp := resp.(fail.HTTPResponse)
        
        w.WriteHeader(httpResp.StatusCode)
        json.NewEncoder(w).Encode(httpResp)
        return
    }
    
    w.WriteHeader(200)
}

// main.go - Generate docs
func main() {
    fail.ExportIDList() // Print JSON to stdout
}
```

## 🎁 Helper Functions

```go
// Quick constructors
fail.Quick(AuthTokenExpired, "custom msg")
fail.Wrap(DBQueryFailed, dbErr)
fail.WrapMsg(DBQueryFailed, "query failed", dbErr)

// Error groups
group := fail.NewErrorGroup()
group.Add(err1)
group.Add(err2)
return group.ToError()

// Error chains
err := fail.Chain().
    Then(validate).
    Then(process).
    Then(save).
    Error()
```

## 🚀 Why FAIL?

- **Deterministic**: Hash-based IDs never change unless you rename
- **Type-Safe**: ErrorID is a struct, impossible to typo
- **Validated**: Name, domain, similarity all checked at creation
- **Documented**: Export JSON list for automatic documentation
- **Ergonomic**: Beautiful fluent API with Form() helper
- **Framework-Agnostic**: Works with any HTTP framework
- **Observable**: Built-in logging and tracing
- **Fun**: Actually enjoyable to use!

## 📦 Installation

```bash
go get your-module/fail
```

## 📄 License

MIT

---

**FAIL - Because deterministic error handling shouldn't be a failure! 🔥**