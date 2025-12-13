package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"taskservice/internal/domain/user"
	"taskservice/pkg/types"
)

type UserRepository struct {
	pool pgxPool
}

func NewUserRepository(pool pgxPool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	now := time.Now().UTC()

	const query = `
		INSERT INTO users (name, email, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	var id int64
	if err := r.pool.QueryRow(ctx, query, u.Name, u.Email, now, now).
		Scan(&id, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return err
	}

	u.ID = types.ID(id)
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id types.ID) (user.User, error) {
	const query = `SELECT id, name, email, created_at, updated_at FROM users WHERE id = $1`

	var u user.User
	var userID int64
	if err := r.pool.QueryRow(ctx, query, uint64(id)).
		Scan(&userID, &u.Name, &u.Email, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return user.User{}, err
	}

	u.ID = types.ID(userID)
	return u, nil
}

func (r *UserRepository) Exists(ctx context.Context, id types.ID) (bool, error) {
	const query = `SELECT 1 FROM users WHERE id = $1`

	return rowExists(ctx, r.pool, query, uint64(id))
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	const query = `SELECT 1 FROM users WHERE email = $1`
	return rowExists(ctx, r.pool, query, email)
}

func rowExists(ctx context.Context, pool pgxPool, query string, arg any) (bool, error) {
	var exists int
	if err := pool.QueryRow(ctx, query, arg).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return exists == 1, nil
}
