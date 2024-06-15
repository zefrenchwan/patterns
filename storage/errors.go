package storage

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

//  P0002	no_data_found
// 42501	insufficient_privilege

const (
	AUTH_CODE          = "42501"
	RESOURCE_CODE      = "P0002"
	INCONSISTENCY_CODE = "23503"
)

func FindCodeInPSQLException(sourceError error) string {
	var pgErr *pgconn.PgError
	var result string
	if errors.As(sourceError, &pgErr) {
		result = pgErr.Code
	}

	return result
}
