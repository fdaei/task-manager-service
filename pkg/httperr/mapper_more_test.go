package httperr

import (
	"errors"
	"net/http"
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestMapServiceError_ValidationAndPgErrors(t *testing.T) {
	valErrs := validation.Errors{
		"name": validation.ErrRequired,
	}
	require.Equal(t, http.StatusBadRequest, MapServiceError(valErrs))

	require.Equal(t, http.StatusBadRequest, MapServiceError(validation.ErrRequired))

	pgErr := &pgconn.PgError{Code: "23505"}
	require.Equal(t, http.StatusConflict, MapServiceError(pgErr))

	require.Equal(t, http.StatusInternalServerError, MapServiceError(nil))
	require.Equal(t, http.StatusInternalServerError, MapServiceError(errors.New("other")))
}
