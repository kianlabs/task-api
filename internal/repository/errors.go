package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// Error sentinel agar lapisan service tidak perlu mengetahui detail driver.
var (
	ErrDuplicate = errors.New("duplicate record")
	ErrReference = errors.New("referenced record does not exist")
)

func mapPgError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return ErrDuplicate
		case "23503": // foreign_key_violation
			return ErrReference
		}
	}
	return err
}
