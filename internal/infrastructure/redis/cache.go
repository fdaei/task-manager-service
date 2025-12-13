package redis

import (
	"context"
	"encoding/json"
	"time"

	redis "github.com/redis/go-redis/v9"

	"taskservice/internal/domain/task"
)

type TaskCache struct {
	client redisClient
	ttl    time.Duration
	prefix string
}

type redisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Ping(ctx context.Context) *redis.StatusCmd
	Close() error
}

func NewTaskCache(addr string, ttl time.Duration) *TaskCache {
	return &TaskCache{
		client: redis.NewClient(&redis.Options{Addr: addr}),
		ttl:    ttl,
		prefix: "tasks:",
	}
}

func (c *TaskCache) GetTasks(ctx context.Context, key string) ([]task.Task, error) {
	data, err := c.client.Get(ctx, c.prefix+key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var tasks []task.Task
	if err := json.Unmarshal([]byte(data), &tasks); err != nil {
		return nil, err
	}

	return tasks, nil
}

func (c *TaskCache) SetTasks(ctx context.Context, key string, tasks []task.Task, ttl time.Duration) error {
	payload, err := json.Marshal(tasks)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, c.prefix+key, payload, ttl).Err()
}

func (c *TaskCache) InvalidateAll(ctx context.Context) error {
	keys, _, err := c.client.Scan(ctx, 0, c.prefix+"*", 0).Result()
	if err != nil {
		return err
	}
	if len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

func (c *TaskCache) WarmUp(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *TaskCache) Close() error {
	return c.client.Close()
}
