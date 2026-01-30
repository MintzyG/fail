package mappers

import (
	"database/sql"
	"errors"
	"fail"
)

// Common error IDs for generic mappers
var (
	SQLNotFound             = fail.ID("SQLNotFound", "SQL", true)
	SQLUniqueViolation      = fail.ID("SQLUniqueViolation", "SQL", true)
	SQLForeignKey           = fail.ID("SQLForeignKey", "SQL", true)
	SQLNotNull              = fail.ID("SQLNotNull", "SQL", true)
	SQLValueTooLong         = fail.ID("SQLValueTooLong", "SQL", true)
	SQLCheckViolation       = fail.ID("SQLCheckViolation", "SQL", true)
	SQLSerializationFailure = fail.ID("SQLSerializationFailure", "SQL", true)
	SQLConnectionError      = fail.ID("SQLConnectionError", "SQL", true)
	SQLUnknownError         = fail.ID("SQLUnknownError", "SQL", false)
)

// RegisterSQLMappers registers mappers for database/sql and pgx errors
func RegisterSQLMappers() {
	// Register error definitions
	fail.RegisterMany(
		fail.ErrorDefinition{
			ID:             SQLNotFound,
			DefaultMessage: "resource not found",
			IsSystem:       false,
		},
		fail.ErrorDefinition{
			ID:             SQLUniqueViolation,
			DefaultMessage: "resource already exists",
			IsSystem:       false,
		},
		fail.ErrorDefinition{
			ID:             SQLForeignKey,
			DefaultMessage: "invalid reference",
			IsSystem:       false,
		},
		fail.ErrorDefinition{
			ID:             SQLNotNull,
			DefaultMessage: "required field missing",
			IsSystem:       false,
		},
		fail.ErrorDefinition{
			ID:             SQLValueTooLong,
			DefaultMessage: "value too long",
			IsSystem:       false,
		},
		fail.ErrorDefinition{
			ID:             SQLCheckViolation,
			DefaultMessage: "invalid value",
			IsSystem:       false,
		},
		fail.ErrorDefinition{
			ID:             SQLSerializationFailure,
			DefaultMessage: "transaction conflict, please retry",
			IsSystem:       false,
		},
		fail.ErrorDefinition{
			ID:             SQLConnectionError,
			DefaultMessage: "database connection error",
			IsSystem:       true,
		},
		fail.ErrorDefinition{
			ID:             SQLUnknownError,
			DefaultMessage: "database error",
			IsSystem:       true,
		},
	)

	// Register sql.ErrNoRows mapper
	fail.RegisterMapper(fail.GenericMapper{
		Name:     "sql.ErrNoRows",
		Priority: 100,
		Matcher: func(err error) bool {
			return errors.Is(err, sql.ErrNoRows)
		},
		Transform: func(err error) *fail.Error {
			return fail.New(SQLNotFound).With(err).Debug(err.Error())
		},
	})
}
