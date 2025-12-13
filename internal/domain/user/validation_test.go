package user

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"taskservice/pkg/types"
)

func TestValidateCreateRejectsDuplicateEmail(t *testing.T) {
	repo := &validationRepo{existsByEmail: true}
	validator := NewValidator(repo)

	err := validator.ValidateCreate(context.Background(), CreateUserParams{
		Name:  "demo",
		Email: "demo@example.com",
	})

	require.Error(t, err)
}

func TestValidateCreatePropagatesRepoError(t *testing.T) {
	repo := &validationRepo{err: errors.New("db down")}
	validator := NewValidator(repo)

	err := validator.ValidateCreate(context.Background(), CreateUserParams{
		Name:  "demo",
		Email: "demo@example.com",
	})

	require.EqualError(t, err, "db down")
}

type validationRepo struct {
	existsByEmail bool
	err           error
}

func (v *validationRepo) Create(context.Context, *User) error {
	return nil
}

func (v *validationRepo) GetByID(context.Context, types.ID) (User, error) {
	return User{}, nil
}

func (v *validationRepo) Exists(context.Context, types.ID) (bool, error) {
	return false, nil
}

func (v *validationRepo) ExistsByEmail(context.Context, string) (bool, error) {
	return v.existsByEmail, v.err
}
