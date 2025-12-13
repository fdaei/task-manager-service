package user

import (
	"context"

	"taskservice/pkg/types"
)

type Repository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id types.ID) (User, error)
	Exists(ctx context.Context, id types.ID) (bool, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}
