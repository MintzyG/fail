package main

import (
	"errors"
	"fmt"

	"github.com/MintzyG/fail/v2"
	"github.com/jackc/pgconn"
)

var (
	SQLUniqueViolation    = fail.ID(0, "SQL", 0, true, "SQLUniqueViolation")
	ErrSQLUniqueViolation = fail.Form(SQLUniqueViolation, "unique violation", false, nil)

	SQLForeignKey    = fail.ID(0, "SQL", 1, true, "SQLForeignKey")
	ErrSQLForeignKey = fail.Form(SQLForeignKey, "foreign key violation", false, nil)

	SQLUnknownError    = fail.ID(0, "SQL", 0, false, "SQLUnknownError")
	ErrSQLUnknownError = fail.Form(SQLUnknownError, "unknown error", false, nil)
)

type PGXMapper struct{}

func (m *PGXMapper) Name() string  { return "pgx" }
func (m *PGXMapper) Priority() int { return 100 }

func (m *PGXMapper) Map(err error) (error, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return nil, false
	}

	switch pgErr.Code {
	case "23505":
		return ErrSQLUniqueViolation, true
	case "23503":
		return ErrSQLForeignKey, true
	default:
		return ErrSQLUnknownError, true
	}
}

func (m *PGXMapper) MapFromFail(fe *fail.Error) (error, bool) {
	return errors.New(fe.Message), true
}

func (m *PGXMapper) Map(err error) (*fail.Error, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return nil, false
	}

	// Use fail.New(ID) to create a fresh error instance to wrap the cause
	// Using sentinels (ErrSQLUniqueViolation.With) would mutate the global sentinel!
	switch pgErr.Code {
	case "23505":
		return fail.New(SQLUniqueViolation).With(err), true
	case "23503":
		return fail.New(SQLForeignKey).With(err), true
	default:
		return fail.New(SQLUnknownError).With(err), true
	}
}

func main() {
	fail.RegisterMapper(&PGXMapper{})

	fmt.Println("=== PGX Mapper Example ===")

	pgErr := &pgconn.PgError{
		Code:    "23505",
		Message: "duplicate key value violates unique constraint",
	}

	err := fail.From(pgErr)

	fmt.Printf("Original: %v\n", pgErr)
	fmt.Printf("Mapped:   %s [%s]\n", err.Message, err.ID)

	if err.ID == SQLUniqueViolation {
		fmt.Println("✅ Successfully mapped to SQLUniqueViolation")
	}
}
