package mappers

import "fail"

// PGXMapper creates a mapper for pgx/pgconn errors
// Users can customize this with their own constraint mappings
type PGXMapper struct {
	UniqueConstraints map[string]fail.ErrorID
	CheckConstraints  map[string]fail.ErrorID
	ForeignKeys       map[string]fail.ErrorID
}

// NewPGXMapper creates a new pgx error mapper
func NewPGXMapper() *PGXMapper {
	return &PGXMapper{
		UniqueConstraints: make(map[string]fail.ErrorID),
		CheckConstraints:  make(map[string]fail.ErrorID),
		ForeignKeys:       make(map[string]fail.ErrorID),
	}
}

// MapUniqueConstraint maps a constraint name to a specific error ID
func (m *PGXMapper) MapUniqueConstraint(constraintName string, errorID fail.ErrorID) *PGXMapper {
	m.UniqueConstraints[constraintName] = errorID
	return m
}

// MapCheckConstraint maps a check constraint to a specific error ID
func (m *PGXMapper) MapCheckConstraint(constraintName string, errorID fail.ErrorID) *PGXMapper {
	m.CheckConstraints[constraintName] = errorID
	return m
}

// MapForeignKey maps a foreign key constraint to a specific error ID
func (m *PGXMapper) MapForeignKey(constraintName string, errorID fail.ErrorID) *PGXMapper {
	m.ForeignKeys[constraintName] = errorID
	return m
}

// ToGenericMapper converts this PGX mapper to a GenericMapper for registration
func (m *PGXMapper) ToGenericMapper() fail.GenericMapper {
	return fail.GenericMapper{
		Name:     "pgx",
		Priority: 90,
		Matcher: func(err error) bool {
			// This would check if it's a pgconn.PgError
			// For now, simplified - you'd import pgx here
			return false // Placeholder
		},
		Transform: func(err error) *fail.Error {
			// This would extract the PgError and map based on code and constraint
			// Placeholder implementation
			return fail.New(SQLUnknownError).With(err)
		},
	}
}
