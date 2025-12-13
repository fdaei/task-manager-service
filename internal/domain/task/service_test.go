package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"taskservice/internal/domain/user"
	"taskservice/pkg/types"
)

func TestCreateTaskValidatesUserExists(t *testing.T) {
	repo := &fakeRepo{}
	userRepo := &fakeUserRepo{exists: true}
	svc := NewService(repo, NewValidator(userRepo), nil)

	created, err := svc.CreateTask(context.Background(), CreateTaskParams{
		UserID: types.ID(1), Title: "title", Status: "todo",
	})

	require.NoError(t, err)
	require.Equal(t, types.ID(99), created.ID)
	require.True(t, repo.createCalled)
}

func TestCreateTaskFailsWhenUserMissing(t *testing.T) {
	repo := &fakeRepo{}
	userRepo := &fakeUserRepo{exists: false}
	svc := NewService(repo, NewValidator(userRepo), nil)

	_, err := svc.CreateTask(context.Background(), CreateTaskParams{
		UserID: types.ID(2), Title: "title", Status: "todo",
	})

	require.Error(t, err)
}

func TestListTasksUsesCache(t *testing.T) {
	cache := newFakeCache()
	cached := []Task{{ID: 1, Title: "cached"}}
	cache.store["tasks:all"] = cached

	repo := &fakeRepo{}
	svc := NewService(repo, NewValidator(&fakeUserRepo{exists: true}), cache)

	result, err := svc.ListTasks(context.Background(), ListTasksParams{})

	require.NoError(t, err)
	require.Equal(t, cached, result)
	require.Zero(t, repo.listCalls)
}

func TestListTasksCachesMisses(t *testing.T) {
	cache := newFakeCache()
	repo := &fakeRepo{listResp: []Task{{ID: 2, Title: "fresh"}}}
	svc := NewService(repo, NewValidator(&fakeUserRepo{exists: true}), cache)

	result, err := svc.ListTasks(context.Background(), ListTasksParams{Limit: 10})

	require.NoError(t, err)
	require.Equal(t, repo.listResp, result)
	require.Equal(t, repo.listResp, cache.store["tasks:all:limit:10"])
}

func TestDeleteInvalidatesCache(t *testing.T) {
	cache := newFakeCache()
	repo := &fakeRepo{}
	svc := NewService(repo, NewValidator(&fakeUserRepo{exists: true}), cache)

	require.NoError(t, svc.DeleteTask(context.Background(), 1))
	require.True(t, cache.invalidated)
}

func TestUpdateTaskValidates(t *testing.T) {
	svc := NewService(&fakeRepo{}, nil, nil)
	_, err := svc.UpdateTask(context.Background(), UpdateTaskParams{})
	require.Error(t, err)
}

func TestUpdateTaskInvalidatesCache(t *testing.T) {
	cache := newFakeCache()
	repo := &fakeRepo{}
	svc := NewService(repo, NewValidator(&fakeUserRepo{exists: true}), cache)

	_, err := svc.UpdateTask(context.Background(), UpdateTaskParams{
		ID:     1,
		UserID: 1,
		Title:  "title",
		Status: "todo",
	})

	require.NoError(t, err)
	require.True(t, cache.invalidated)
	require.True(t, repo.updateCalled)
}

func TestListTasksSkipsCacheWhenOffset(t *testing.T) {
	cache := newFakeCache()
	repo := &fakeRepo{listResp: []Task{{ID: 1, Title: "offset"}}}
	svc := NewService(repo, NewValidator(&fakeUserRepo{exists: true}), cache)

	_, err := svc.ListTasks(context.Background(), ListTasksParams{Offset: 5})

	require.NoError(t, err)
	require.Empty(t, cache.store)
	require.Equal(t, 1, repo.listCalls)
}

func TestCountTasksDelegates(t *testing.T) {
	repo := &fakeRepo{countResp: 3}
	svc := NewService(repo, NewValidator(&fakeUserRepo{exists: true}), nil)

	total, err := svc.CountTasks(context.Background(), ListTasksParams{})

	require.NoError(t, err)
	require.Equal(t, 3, total)
}

func TestGetTaskDelegates(t *testing.T) {
	repo := &fakeRepo{getResp: &Task{ID: 7, Title: "found"}}
	svc := NewService(repo, nil, nil)

	task, err := svc.GetTask(context.Background(), 7)

	require.NoError(t, err)
	require.Equal(t, types.ID(7), repo.getCalledWith)
	require.Equal(t, repo.getResp, task)
}

// --- fakes ---

type fakeRepo struct {
	createCalled  bool
	listResp      []Task
	listCalls     int
	countResp     int
	createErr     error
	listErr       error
	countErr      error
	updateErr     error
	deleteErr     error
	updateCalled  bool
	getResp       *Task
	getErr        error
	getCalledWith types.ID
}

func (f *fakeRepo) Create(_ context.Context, t *Task) error {
	f.createCalled = true
	t.ID = 99
	t.CreatedAt = time.Now()
	t.UpdatedAt = t.CreatedAt
	return f.createErr
}

func (f *fakeRepo) GetByID(_ context.Context, id types.ID) (*Task, error) {
	f.getCalledWith = id
	if f.getResp != nil {
		return f.getResp, f.getErr
	}
	return nil, errors.New("not implemented")
}

func (f *fakeRepo) List(_ context.Context, _ ListTasksParams) ([]Task, error) {
	f.listCalls++
	return f.listResp, f.listErr
}

func (f *fakeRepo) Count(_ context.Context, _ ListTasksParams) (int, error) {
	return f.countResp, f.countErr
}

func (f *fakeRepo) Update(_ context.Context, _ *Task) error {
	f.updateCalled = true
	return f.updateErr
}
func (f *fakeRepo) Delete(_ context.Context, _ types.ID) error {
	return f.deleteErr
}

type fakeUserRepo struct {
	exists bool
	err    error
}

func (f *fakeUserRepo) Create(_ context.Context, _ *user.User) error {
	return errors.New("not implemented")
}

func (f *fakeUserRepo) GetByID(_ context.Context, _ types.ID) (user.User, error) {
	return user.User{}, errors.New("not implemented")
}

func (f *fakeUserRepo) Exists(_ context.Context, _ types.ID) (bool, error) {
	return f.exists, f.err
}

func (f *fakeUserRepo) ExistsByEmail(context.Context, string) (bool, error) {
	return false, nil
}

type fakeCache struct {
	store        map[string][]Task
	invalidated  bool
	getError     error
	setError     error
	invalidateEr error
}

func newFakeCache() *fakeCache {
	return &fakeCache{store: make(map[string][]Task)}
}

func (f *fakeCache) GetTasks(_ context.Context, key string) ([]Task, error) {
	if f.getError != nil {
		return nil, f.getError
	}
	if val, ok := f.store[key]; ok {
		return val, nil
	}
	return nil, nil
}

func (f *fakeCache) SetTasks(_ context.Context, key string, tasks []Task, _ time.Duration) error {
	if f.setError != nil {
		return f.setError
	}
	f.store[key] = tasks
	return nil
}

func (f *fakeCache) InvalidateAll(_ context.Context) error {
	f.invalidated = true
	if f.invalidateEr != nil {
		return f.invalidateEr
	}
	for k := range f.store {
		delete(f.store, k)
	}
	return nil
}
