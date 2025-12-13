package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"taskservice/pkg/types"
)

func TestCreateUserValidatesInput(t *testing.T) {
	repo := &fakeUserRepo{}
	svc := NewService(repo)

	_, err := svc.CreateUser(context.Background(), CreateUserParams{})
	require.Error(t, err)
}

func TestCreateUserSuccess(t *testing.T) {
	repo := &fakeUserRepo{}
	svc := NewService(repo)

	u, err := svc.CreateUser(context.Background(), CreateUserParams{
		Name:  "demo",
		Email: "demo@example.com",
	})

	require.NoError(t, err)
	require.Equal(t, types.ID(1), u.ID)
	require.Equal(t, "demo", repo.created.Name)
}

func TestGetUserDelegates(t *testing.T) {
	repo := &fakeUserRepo{
		getResp: User{ID: 3, Name: "user", Email: "user@example.com"},
	}
	svc := NewService(repo)

	u, err := svc.GetUser(context.Background(), 3)

	require.NoError(t, err)
	require.Equal(t, types.ID(3), u.ID)
	require.Equal(t, "user@example.com", u.Email)
}

type fakeUserRepo struct {
	created User
	getResp User
	getErr  error
}

func (f *fakeUserRepo) Create(_ context.Context, user *User) error {
	user.ID = 1
	f.created = *user
	return nil
}

func (f *fakeUserRepo) GetByID(context.Context, types.ID) (User, error) {
	return f.getResp, f.getErr
}

func (f *fakeUserRepo) Exists(context.Context, types.ID) (bool, error) {
	return true, nil
}

func (f *fakeUserRepo) ExistsByEmail(context.Context, string) (bool, error) {
	return false, nil
}
