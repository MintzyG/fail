package main

import (
	"fail"
	"fail/examples/mappers"
	"fail/examples/translators"
	"fmt"
	"log"

	"github.com/jackc/pgconn"
)

// Auth domain errors
var (
	AuthValidationFailed   = fail.ID("AuthValidationFailed", "AUTH", false, 0)
	AuthInvalidCredentials = fail.ID("AuthInvalidCredentials", "AUTH", true, 0)
	AuthTokenExpired       = fail.ID("AuthTokenExpired", "AUTH", true, 0)
	AuthTokenInvalid       = fail.ID("AuthTokenInvalid", "AUTH", false, 0)
	AuthUserNotFound       = fail.ID("AuthUserNotFound", "AUTH", true, 0)
)

// User domain errors
var (
	UserEmailExists      = fail.ID("UserEmailExists", "USER", true, 0)
	UserValidationFailed = fail.ID("UserValidationFailed", "USER", false, 0)
	UserNotFound         = fail.ID("UserNotFound", "USER", true, 0)
	UserUsernameExists   = fail.ID("UserUsernameExists", "USER", true, 0) // USER_0001_S
)

// Database domain errors
var (
	DBConnectionFailed = fail.ID("DBConnectionFailed", "DB", true, 1)
	DBQueryFailed      = fail.ID("DBQueryFailed", "DB", false, 3)
)

var (
	ContextCanceled = fail.ID("ContextCanceled", "CONTEXT", true, 2) // CONTEXT_0000_S
	ContextDeadline = fail.ID("ContextDeadline", "CONTEXT", true, 5) // CONTEXT_0001_S
)

// ============================================================================
// REGISTERING ERRORS - Two Ways
// ============================================================================

// Option 1: Traditional - Register separately
func registerTraditional() {
	fail.RegisterMany(
		fail.ErrorDefinition{
			ID:             AuthInvalidCredentials,
			DefaultMessage: "invalid credentials",
			IsSystem:       false,
		},
		fail.ErrorDefinition{
			ID:             AuthUserNotFound,
			DefaultMessage: "user not found",
			IsSystem:       false,
		},
		fail.ErrorDefinition{
			ID:             AuthTokenExpired,
			DefaultMessage: "token has expired",
			IsSystem:       false,
		},
	)
}

// Option 2: Form() - Create sentinel and register in one line! 🎉
var (
	ErrUserEmailExists      = fail.Form(UserEmailExists, "email address already registered", false, nil)
	ErrUsernameExists       = fail.Form(UserUsernameExists, "username already taken", false, nil)
	ErrUserValidationFailed = fail.Form(UserValidationFailed, "user validation failed", false, nil)
	ErrDBConnectionFailed   = fail.Form(DBConnectionFailed, "database connection failed", true, nil)
	ErrContextCanceled      = fail.Form(ContextCanceled, "operation canceled", true, nil)
	ErrContextDeadline      = fail.Form(ContextDeadline, "operation timed out", true, nil)
)

// ============================================================================
// VALIDATION EXAMPLES - These would PANIC at startup!
// ============================================================================

func showValidationExamples() {
	fmt.Println("🛡️  Built-in Validation Examples:")
	fmt.Println("")

	// ❌ PANIC: Name doesn't start with domain
	fmt.Println("❌ fail.ID(\"InvalidName\", \"AUTH\", true)")
	fmt.Println("   PANIC: error name 'InvalidName' must start with domain 'AUTH'")
	fmt.Println("")

	// ❌ PANIC: Duplicate name
	fmt.Println("❌ fail.ID(\"AuthInvalidCredentials\", \"AUTH\", true)")
	fmt.Println("   PANIC: error name 'AuthInvalidCredentials' already registered")
	fmt.Println("")

	// ❌ PANIC: Too similar (distance < 3)
	fmt.Println("❌ fail.ID(\"AuthInvalidCredential\", \"AUTH\", true)")
	fmt.Println("   PANIC: too similar to 'AuthInvalidCredentials' (distance: 1, must be >= 3)")
	fmt.Println("")

	fmt.Println("❌ fail.ID(\"UserNotFounds\", \"USER\", true)")
	fmt.Println("   PANIC: too similar to 'UserNotFound' (distance: 1, must be >= 3)")
	fmt.Println("")
}

// ============================================================================
// USAGE IN CODE
// ============================================================================

func Login(email, password string) error {
	// Static error - message from definition
	if email == "" {
		return fail.New(AuthInvalidCredentials)
	}

	// Dynamic error - customize message
	if len(password) < 8 {
		return fail.New(AuthTokenInvalid).
			Msgf("password must be at least 8 characters, got %d", len(password))
	}

	// Rich error with context
	return fail.New(AuthValidationFailed).
		Msg("authentication failed").
		Trace("validated email format").
		Trace("checked password strength").
		Debug("login attempt from IP: 192.168.1.1").
		AddMeta("attempt", 3)
}

func Register(email, password string) error {
	// Using Form() sentinel
	if emailExists(email) {
		return ErrUserEmailExists.
			AddMeta("email", email).
			Trace("checked uniqueness")
	}

	// Validation errors
	if !validateInput(email, password) {
		return fail.New(UserValidationFailed).
			Msg("registration validation failed").
			Validation("email", "invalid format").
			Validation("password", "too short")
	}

	return nil
}

func validateInput(email, password string) bool {
	return len(email) > 0 && len(password) >= 8
}

func emailExists(email string) bool {
	return email == "taken@example.com"
}

// ============================================================================
// MAIN - DEMONSTRATION
// ============================================================================

func main() {
	registerTraditional()
	if err := fail.RegisterTranslator(translators.HTTPResponseTranslator()); err != nil {
		log.Fatalf("register translator failed: %v", err)
	}

	fail.RegisterMapper(&mappers.PGXMapper{})

	fmt.Println("🔥 FAIL - Failure Abstraction & Instrumentation Layer")
	fmt.Println("=" + "=============================================================")
	fmt.Println("")

	// 1. Show deterministic IDs
	fmt.Println("1️⃣  Deterministic Hash-Based IDs")
	fmt.Println("-------------------------------------")
	fmt.Printf("%-25s = %-13s (name: %s)\n", "AuthInvalidCredentials", AuthInvalidCredentials, AuthInvalidCredentials.Name())
	fmt.Printf("%-25s = %-13s (name: %s)\n", "AuthUserNotFound", AuthUserNotFound, AuthUserNotFound.Name())
	fmt.Printf("%-25s = %-13s (name: %s)\n", "AuthTokenExpired", AuthTokenExpired, AuthTokenExpired.Name())
	fmt.Printf("%-25s = %-13s (name: %s)\n", "UserNotFound", UserNotFound, UserNotFound.Name())
	fmt.Printf("%-25s = %-13s (name: %s)\n", "UserEmailExists", UserEmailExists, UserEmailExists.Name())
	fmt.Println("")
	fmt.Println("✅ Numbers are based on name hash - compilation order independent!")
	fmt.Println("")

	// 2. Show validation
	showValidationExamples()

	// 3. Simple error
	fmt.Println("2️⃣  Creating Errors")
	fmt.Println("-------------------------------------")
	err1 := fail.New(AuthInvalidCredentials)
	fmt.Printf("Error:   %s\n", err1)
	fmt.Printf("ID:      %s\n", err1.ID)
	fmt.Printf("Name:    %s\n", err1.ID.Name())
	fmt.Printf("Message: %s\n", err1.Message)
	fmt.Println("")

	// 4. Using Form() sentinel
	fmt.Println("3️⃣  Using Form() Sentinels")
	fmt.Println("-------------------------------------")
	fmt.Printf("Sentinel: %s\n", ErrUserEmailExists)
	fmt.Printf("ID:       %s\n", ErrUserEmailExists.ID)
	fmt.Printf("Name:     %s\n", ErrUserEmailExists.ID.Name())
	fmt.Printf("Message:  %s\n", ErrUserEmailExists.Message)
	fmt.Println("")

	// 5. Rich error
	fmt.Println("4️⃣  Rich Error with Validation")
	fmt.Println("-------------------------------------")
	err2 := Register("bad@email", "short")
	if err2 != nil {
		e := fail.From(err2)
		fmt.Printf("Error: %s\n", e)
		if validations, ok := e.Meta["validations"]; ok {
			fmt.Printf("Validations:\n")
			for _, v := range validations.([]fail.ValidationError) {
				fmt.Printf("  - %s: %s\n", v.Field, v.Message)
			}
		}
	}
	fmt.Println("")

	// 6. HTTP translation
	fmt.Println("5️⃣  HTTP Translation")
	fmt.Println("-------------------------------------")
	err3 := fail.New(AuthInvalidCredentials).
		Trace("checking credentials").
		Debug("login attempt #3")

	resp, _ := fail.Translate(err3, "http")
	httpResp := resp.(translators.HTTPResponse)
	fmt.Printf("Status:   %d\n", httpResp.StatusCode)
	fmt.Printf("Error ID: %s\n", httpResp.ErrorID)
	fmt.Printf("Message:  %s\n", httpResp.Message)
	fmt.Println("")

	// 6️⃣ Database Error Mapping (PGX)
	fmt.Println("6️⃣  Database Error Mapping")
	fmt.Println("-------------------------------------")

	// Simulate a pgx unique violation
	pgErr := &pgconn.PgError{
		Code:           "23505", // unique_violation
		ConstraintName: "users_email_key",
		Message:        "duplicate key value violates unique constraint",
	}

	mapped := fail.From(pgErr)

	fmt.Printf("Mapped Error: %s\n", mapped)
	fmt.Printf("ID:           %s\n", mapped.ID)
	fmt.Printf("Name:         %s\n", mapped.ID.Name())
	fmt.Printf("Message:      %s\n", mapped.Message)
	fmt.Println("")

	// 7. Type-safe checking
	fmt.Println("7️⃣ Type-Safe Error Checking")
	fmt.Println("-------------------------------------")
	err4 := fail.New(AuthTokenExpired)

	if fail.Is(err4, AuthTokenExpired) {
		fmt.Printf("✅ Matched: %s\n", AuthTokenExpired.Name())
	}

	fail.Match(err4).
		Case(AuthInvalidCredentials, func(e *fail.Error) {
			fmt.Println("Won't match")
		}).
		Case(AuthTokenExpired, func(e *fail.Error) {
			fmt.Printf("✅ Pattern matched: %s\n", e.ID.Name())
		}).
		Default(func(err error) {
			fmt.Println("Default")
		})
	fmt.Println("")

	// 8. Export ID list
	fmt.Println("8️⃣  Export ID List (for docs)")
	fmt.Println("-------------------------------------")
	fmt.Println("JSON output of all registered errors:")
	data, err := fail.ExportIDList()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(data))
	fmt.Println("")

	// Summary
	fmt.Println("🎉 Key Benefits")
	fmt.Println("-------------------------------------")
	fmt.Println("✅ Deterministic IDs - hash-based, not file-order dependent")
	fmt.Println("✅ Name validation - must start with domain")
	fmt.Println("✅ Duplicate detection - can't register same name twice")
	fmt.Println("✅ Similarity check - prevents typos (distance >= 3)")
	fmt.Println("✅ Type-safe - ErrorID is a struct, impossible to typo")
	fmt.Println("✅ Exportable - generate JSON docs automatically")
	fmt.Println("✅ Form() helper - one-line sentinel creation")
	fmt.Println("")
	fmt.Println("🚀 Run with 'go run example_main.go' to see it in action!")
}
