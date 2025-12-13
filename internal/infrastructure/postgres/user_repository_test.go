package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/require"

	"taskservice/internal/domain/user"
	"taskservice/pkg/types"
)

func TestUserRepository(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewUserRepository(mock)
	ctx := context.Background()
	now := time.Now()

	mock.ExpectQuery("INSERT INTO users").
		WithArgs("name", "email@example.com", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(int64(5), now, now))

	u := user.User{Name: "name", Email: "email@example.com"}
	require.NoError(t, repo.Create(ctx, &u))

	mock.ExpectQuery("SELECT id, name, email").
		WithArgs(uint64(5)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "email", "created_at", "updated_at"}).
			AddRow(int64(5), "name", "email@example.com", now, now))
	found, err := repo.GetByID(ctx, types.ID(5))
	require.NoError(t, err)
	require.Equal(t, uint64(5), uint64(found.ID))

	mock.ExpectQuery("SELECT 1 FROM users").WithArgs(uint64(5)).
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(1))
	exists, err := repo.Exists(ctx, types.ID(5))
	require.NoError(t, err)
	require.True(t, exists)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExistsByEmail(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewUserRepository(mock)
	ctx := context.Background()

	mock.ExpectQuery("SELECT 1 FROM users WHERE email").
		WithArgs("demo@example.com").
		WillReturnRows(pgxmock.NewRows([]string{"exists"}).AddRow(1))

	mock.ExpectQuery("SELECT 1 FROM users WHERE email").
		WithArgs("missing@example.com").
		WillReturnError(pgx.ErrNoRows)

	found, err := repo.ExistsByEmail(ctx, "demo@example.com")
	require.NoError(t, err)
	require.True(t, found)

	found, err = repo.ExistsByEmail(ctx, "missing@example.com")
	require.NoError(t, err)
	require.False(t, found)

	require.NoError(t, mock.ExpectationsWereMet())
}
