package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"taskservice/internal/domain/task"
	"taskservice/pkg/types"
)

type TaskRepository struct {
	pool pgxPool
}

func NewTaskRepository(pool pgxPool) *TaskRepository {
	return &TaskRepository{pool: pool}
}

func (r *TaskRepository) Create(ctx context.Context, t *task.Task) error {
	now := time.Now().UTC()

	const query = `
		INSERT INTO tasks (user_id, title, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	var id int64
	if err := r.pool.QueryRow(ctx, query, uint64(t.UserID), t.Title, t.Description, t.Status, now, now).
		Scan(&id, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return err
	}

	t.ID = types.ID(id)
	return nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id types.ID) (*task.Task, error) {
	const query = `SELECT id, user_id, title, description, status, created_at, updated_at FROM tasks WHERE id = $1`

	var t task.Task
	var taskID, userID int64
	if err := r.pool.QueryRow(ctx, query, uint64(id)).
		Scan(&taskID, &userID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}

	t.ID = types.ID(taskID)
	t.UserID = types.ID(userID)
	return &t, nil
}

func (r *TaskRepository) List(ctx context.Context, params task.ListTasksParams) ([]task.Task, error) {
	query := `
		SELECT id, user_id, title, description, status, created_at, updated_at
		FROM tasks
		WHERE ($1::BIGINT IS NULL OR user_id = $1)
		  AND ($2::TEXT IS NULL OR status = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	userID := any(nil)
	if params.UserID != nil {
		userID = int64(*params.UserID)
	}

	status := any(nil)
	if params.Status != nil {
		status = *params.Status
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := params.Offset
	if offset < 0 {
		offset = 0
	}

	rows, err := r.pool.Query(ctx, query, userID, status, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []task.Task
	for rows.Next() {
		var t task.Task
		var taskID, userID int64
		if err := rows.Scan(&taskID, &userID, &t.Title, &t.Description, &t.Status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.ID = types.ID(taskID)
		t.UserID = types.ID(userID)
		tasks = append(tasks, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (r *TaskRepository) Update(ctx context.Context, t *task.Task) error {
	now := time.Now().UTC()

	const query = `
		UPDATE tasks
		SET user_id = $1, title = $2, description = $3, status = $4, updated_at = $5
		WHERE id = $6
		RETURNING updated_at
	`

	if err := r.pool.QueryRow(ctx, query, uint64(t.UserID), t.Title, t.Description, t.Status, now, uint64(t.ID)).
		Scan(&t.UpdatedAt); err != nil {
		return err
	}

	return nil
}

func (r *TaskRepository) Delete(ctx context.Context, id types.ID) error {
	const query = `DELETE FROM tasks WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, uint64(id))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *TaskRepository) Count(ctx context.Context, params task.ListTasksParams) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM tasks
		WHERE ($1::BIGINT IS NULL OR user_id = $1)
		  AND ($2::TEXT IS NULL OR status = $2)
	`

	userID := any(nil)
	if params.UserID != nil {
		userID = int64(*params.UserID)
	}

	status := any(nil)
	if params.Status != nil {
		status = *params.Status
	}

	var count int
	if err := r.pool.QueryRow(ctx, query, userID, status).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}
