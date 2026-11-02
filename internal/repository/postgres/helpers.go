// Package postgres implements every domain repository interface against
// Postgres via pgx. Helpers here are shared plumbing (constraint-violation
// detection, nullable-UUID handling) used across the per-entity repository
// files so each of those stays focused on its own queries.
package postgres

import (
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

const uniqueViolationCode = "23505"

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == uniqueViolationCode
	}
	return false
}

// nullableID converts an empty "exclude this ID" filter into the nil UUID so
// a "slug unique except for this record" query can use a plain != comparison
// without special-casing the create-vs-update path in SQL.
func nullableID(id string) string {
	if id == "" {
		return uuid.Nil.String()
	}
	return id
}
