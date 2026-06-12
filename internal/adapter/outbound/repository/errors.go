package repository

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var ErrDuplicate = errors.New("duplicate record")

func IsDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrDuplicate) || errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
