package redis

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"taskservice/internal/domain/task"
	"taskservice/pkg/types"
)

func TestTaskCacheRoundTrip(t *testing.T) {
	cache := &TaskCache{
		client: newStubClient(),
		ttl:    time.Minute,
		prefix: "tasks:",
	}

	ctx := context.Background()
	sample := []task.Task{{ID: 1, UserID: types.ID(1), Title: "demo", Status: "todo"}}

	require.NoError(t, cache.SetTasks(ctx, "tasks:all", sample, time.Minute))

	got, err := cache.GetTasks(ctx, "tasks:all")
	require.NoError(t, err)
	require.Equal(t, sample, got)

	require.NoError(t, cache.InvalidateAll(ctx))
	got, err = cache.GetTasks(ctx, "tasks:all")
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestNewTaskCacheConfiguresPrefixAndTTL(t *testing.T) {
	cache := NewTaskCache("localhost:6379", time.Minute)
	require.Equal(t, "tasks:", cache.prefix)
	require.Equal(t, time.Minute, cache.ttl)
	require.NoError(t, cache.Close())
}

func TestWarmUpAndClose(t *testing.T) {
	client := newStubClient()
	cache := &TaskCache{
		client: client,
		ttl:    time.Minute,
		prefix: "tasks:",
	}

	require.NoError(t, cache.WarmUp(context.Background()))
	require.True(t, client.pinged)

	require.NoError(t, cache.Close())
	require.True(t, client.closed)
}

type stubClient struct {
	store  map[string]string
	pinged bool
	closed bool
}

func newStubClient() *stubClient {
	return &stubClient{store: make(map[string]string)}
}

func (s *stubClient) Get(ctx context.Context, key string) *redis.StringCmd {
	if val, ok := s.store[key]; ok {
		return redis.NewStringResult(val, nil)
	}
	return redis.NewStringResult("", redis.Nil)
}

func (s *stubClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	switch v := value.(type) {
	case string:
		s.store[key] = v
	case []byte:
		s.store[key] = string(v)
	default:
		s.store[key] = ""
	}
	return redis.NewStatusResult("OK", nil)
}

func (s *stubClient) Scan(ctx context.Context, cursor uint64, match string, count int64) *redis.ScanCmd {
	keys := make([]string, 0, len(s.store))
	for k := range s.store {
		keys = append(keys, k)
	}
	return redis.NewScanCmdResult(keys, 0, nil)
}

func (s *stubClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	var removed int64
	for _, k := range keys {
		if _, ok := s.store[k]; ok {
			delete(s.store, k)
			removed++
		}
	}
	return redis.NewIntResult(removed, nil)
}

func (s *stubClient) Ping(ctx context.Context) *redis.StatusCmd {
	s.pinged = true
	return redis.NewStatusResult("PONG", nil)
}

func (s *stubClient) Close() error {
	s.closed = true
	return nil
}
