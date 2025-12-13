package task

import (
	"context"
	"testing"
	"time"

	"taskservice/pkg/types"
)

func BenchmarkListTasksNoCache(b *testing.B) {
	svc := NewService(newBenchmarkRepository(200), nil, nil)
	params := ListTasksParams{}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.ListTasks(ctx, params)
	}
}

func BenchmarkListTasksCached(b *testing.B) {
	cache := newBenchmarkCache()
	svc := NewService(newBenchmarkRepository(200), nil, cache)
	params := ListTasksParams{}
	ctx := context.Background()

	_, _ = svc.ListTasks(ctx, params)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.ListTasks(ctx, params)
	}
}

type benchmarkRepository struct {
	tasks []Task
}

func newBenchmarkRepository(size int) *benchmarkRepository {
	tasks := make([]Task, size)
	for i := 0; i < size; i++ {
		tasks[i] = Task{
			ID:     types.ID(i + 1),
			UserID: 1,
			Title:  "task",
			Status: "todo",
		}
	}

	return &benchmarkRepository{tasks: tasks}
}

func (b *benchmarkRepository) Create(_ context.Context, _ *Task) error {
	return nil
}

func (b *benchmarkRepository) GetByID(_ context.Context, id types.ID) (*Task, error) {
	for i := range b.tasks {
		if b.tasks[i].ID == id {
			return &b.tasks[i], nil
		}
	}
	return nil, nil
}

func (b *benchmarkRepository) List(_ context.Context, _ ListTasksParams) ([]Task, error) {
	return b.tasks, nil
}

func (b *benchmarkRepository) Count(_ context.Context, _ ListTasksParams) (int, error) {
	return len(b.tasks), nil
}

func (b *benchmarkRepository) Update(_ context.Context, _ *Task) error {
	return nil
}

func (b *benchmarkRepository) Delete(_ context.Context, _ types.ID) error {
	return nil
}

type benchmarkCache struct {
	store map[string][]Task
}

func newBenchmarkCache() *benchmarkCache {
	return &benchmarkCache{store: make(map[string][]Task)}
}

func (c *benchmarkCache) GetTasks(_ context.Context, key string) ([]Task, error) {
	if tasks, ok := c.store[key]; ok {
		return tasks, nil
	}
	return nil, nil
}

func (c *benchmarkCache) SetTasks(_ context.Context, key string, tasks []Task, _ time.Duration) error {
	c.store[key] = tasks
	return nil
}

func (c *benchmarkCache) InvalidateAll(_ context.Context) error {
	c.store = make(map[string][]Task)
	return nil
}
