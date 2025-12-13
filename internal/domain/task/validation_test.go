package task

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"taskservice/internal/domain/user"
	"taskservice/pkg/types"
)

func TestValidateCreateRejectsBadStatus(t *testing.T) {
	params := CreateTaskParams{UserID: 1, Title: "t", Status: "invalid"}
	require.Error(t, params.Validate())
}

func TestValidateUpdateRequiresID(t *testing.T) {
	params := UpdateTaskParams{UserID: 1, Title: "t", Status: "todo"}
	require.Error(t, params.Validate())
}

func TestValidateUpdateSuccess(t *testing.T) {
	params := UpdateTaskParams{ID: 1, UserID: 1, Title: "t", Status: "todo"}
	require.NoError(t, params.Validate())
}

func TestTaskValidateFailsWithMissingFields(t *testing.T) {
	task := Task{}
	require.Error(t, task.Validate())
}

func TestValidatorChecksUserExistence(t *testing.T) {
	v := NewValidator(&validationUserRepo{exists: false})
	err := v.ValidateCreate(context.Background(), CreateTaskParams{
		UserID: 1, Title: "t", Status: "todo",
	})
	require.Error(t, err)
}

func TestValidatorSkipsWhenRepoNil(t *testing.T) {
	v := NewValidator(nil)
	err := v.ValidateCreate(context.Background(), CreateTaskParams{
		UserID: 1, Title: "t", Status: "todo",
	})
	require.NoError(t, err)
}

type validationUserRepo struct {
	exists bool
	err    error
}

func (f *validationUserRepo) Create(context.Context, *user.User) error {
	return nil
}

func (f *validationUserRepo) GetByID(context.Context, types.ID) (user.User, error) {
	return user.User{}, nil
}

func (f *validationUserRepo) Exists(context.Context, types.ID) (bool, error) {
	return f.exists, f.err
}

func (f *validationUserRepo) ExistsByEmail(context.Context, string) (bool, error) {
	return false, nil
}
