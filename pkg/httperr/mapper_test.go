package httperr

import (
	"errors"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestMapServiceError(t *testing.T) {
	require.Equal(t, http.StatusNotFound, MapServiceError(pgx.ErrNoRows))
	require.Equal(t, http.StatusNotFound, MapServiceError(errors.New("user not found")))
	require.Equal(t, http.StatusInternalServerError, MapServiceError(errors.New("boom")))
}
