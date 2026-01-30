package mappers

import (
	"database/sql"
	"errors"
	"fail"
)

// Common error IDs for generic mappers
var (
	SQLNotFound             = fail.ID("SQLNotFound", "SQL", true, 0)
	SQLUniqueViolation      = fail.ID("SQLUniqueViolation", "SQL", true, 0)
	SQLForeignKey           = fail.ID("SQLForeignKey", "SQL", true, 0)
	SQLNotNull              = fail.ID("SQLNotNull", "SQL", true, 0)
	SQLValueTooLong         = fail.ID("SQLValueTooLong", "SQL", true, 0)
	SQLCheckViolation       = fail.ID("SQLCheckViolation", "SQL", true, 0)
	SQLSerializationFailure = fail.ID("SQLSerializationFailure", "SQL", true, 0)
	SQLConnectionError      = fail.ID("SQLConnectionError", "SQL", true, 0)
	SQLUnknownError         = fail.ID("SQLUnknownError", "SQL", false, 0)
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
