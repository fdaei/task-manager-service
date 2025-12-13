package cmd

import (
	"context"
	"testing"

	"taskservice/internal/config"
)

func TestRunServeFailsWhenDepsUnavailable(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost:0/postgres?connect_timeout=1")
	t.Setenv("REDIS_HOST", "localhost:0")
	t.Setenv("SERVICE_NAME", "taskservice-test")

	if err := runServe(nil, nil); err == nil {
		t.Fatalf("expected runServe to fail with unreachable dependencies")
	}
}

func TestSetupCacheSkipsWhenRedisDown(t *testing.T) {
	cache, closer := setupCache(context.Background(), config.Config{RedisAddr: "localhost:0"})
	if cache != nil {
		t.Fatalf("expected cache to be nil when redis is unreachable")
	}
	if closer != nil {
		t.Fatalf("expected closer to be nil when cache is nil")
	}
}
