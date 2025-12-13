package task

import (
	"context"
	"time"
)

type Cache interface {
	GetTasks(ctx context.Context, key string) ([]Task, error)
	SetTasks(ctx context.Context, key string, tasks []Task, ttl time.Duration) error
	InvalidateAll(ctx context.Context) error
}
