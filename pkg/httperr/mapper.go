package httperr

import (
	"errors"
	nethttp "net/http"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func MapServiceError(err error) int {
	switch {
	case err == nil:
		return nethttp.StatusInternalServerError
	case isValidationError(err):
		return nethttp.StatusBadRequest
	case errors.Is(err, pgx.ErrNoRows):
		return nethttp.StatusNotFound
	case err != nil && err.Error() == "user not found":
		return nethttp.StatusNotFound
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return nethttp.StatusConflict
	}

	return nethttp.StatusInternalServerError
}

func isValidationError(err error) bool {
	if _, ok := err.(validation.Errors); ok {
		return true
	}

	var validationErr validation.ErrorObject
	return errors.As(err, &validationErr)
}
