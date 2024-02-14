package postgres

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

var pgErr *pgconn.PgError

func isConnException(err error) bool {
	if errors.As(err, &pgErr) {
		return pgerrcode.IsConnectionException(pgErr.Code)
	}
	return false
}
