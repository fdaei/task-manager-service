package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/require"

	"taskservice/internal/domain/task"
	"taskservice/pkg/types"
)

func TestTaskRepositoryCRUD(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewTaskRepository(mock)
	ctx := context.Background()
	now := time.Now()

	mock.ExpectQuery("INSERT INTO tasks").
		WithArgs(uint64(1), "title", "desc", "todo", pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnRows(pgxmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(int64(10), now, now))

	tk := task.Task{UserID: 1, Title: "title", Description: "desc", Status: "todo"}
	require.NoError(t, repo.Create(ctx, &tk))

	mock.ExpectQuery("SELECT id, user_id, title").
		WithArgs(uint64(10)).
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "title", "description", "status", "created_at", "updated_at"}).
			AddRow(int64(10), int64(1), "title", "desc", "todo", now, now))
	found, err := repo.GetByID(ctx, types.ID(10))
	require.NoError(t, err)
	require.Equal(t, uint64(10), uint64(found.ID))

	mock.ExpectQuery("FROM tasks").
		WithArgs(int64(1), "todo", 50, 0).
		WillReturnRows(pgxmock.NewRows([]string{"id", "user_id", "title", "description", "status", "created_at", "updated_at"}).
			AddRow(int64(10), int64(1), "title", "desc", "todo", now, now))
	list, err := repo.List(ctx, task.ListTasksParams{UserID: ptrID(1), Status: ptrString("todo")})
	require.NoError(t, err)
	require.Len(t, list, 1)

	mock.ExpectQuery("UPDATE tasks").
		WithArgs(uint64(1), "title2", "desc2", "doing", pgxmock.AnyArg(), uint64(10)).
		WillReturnRows(pgxmock.NewRows([]string{"updated_at"}).AddRow(now.Add(time.Minute)))
	require.NoError(t, repo.Update(ctx, &task.Task{
		ID:          10,
		UserID:      1,
		Title:       "title2",
		Description: "desc2",
		Status:      "doing",
	}))

	mock.ExpectExec("DELETE FROM tasks").WithArgs(uint64(10)).WillReturnResult(pgxmock.NewResult("DELETE", 1))
	require.NoError(t, repo.Delete(ctx, types.ID(10)))

	mock.ExpectQuery("SELECT COUNT\\(\\*\\)").
		WithArgs(nil, nil).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))
	count, err := repo.Count(ctx, task.ListTasksParams{})
	require.NoError(t, err)
	require.Equal(t, 1, count)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTaskRepositoryDeleteNotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	repo := NewTaskRepository(mock)
	ctx := context.Background()

	mock.ExpectExec("DELETE FROM tasks").WithArgs(uint64(99)).WillReturnResult(pgxmock.NewResult("DELETE", 0))

	err = repo.Delete(ctx, types.ID(99))
	require.ErrorIs(t, err, pgx.ErrNoRows)
	require.NoError(t, mock.ExpectationsWereMet())
}

func ptrID(id uint64) *types.ID {
	val := types.ID(id)
	return &val
}

func ptrString(v string) *string {
	return &v
}
