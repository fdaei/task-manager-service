package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	os.Unsetenv("DATABASE_URL")
	_, err := Load()
	require.Error(t, err)
}

func TestLoadUsesDefaults(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://example")
	defer os.Unsetenv("DATABASE_URL")
	os.Unsetenv("HTTP_PORT")
	os.Unsetenv("REDIS_HOST")
	os.Unsetenv("SERVICE_NAME")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, 8080, cfg.HTTPPort)
	require.Equal(t, "postgres://example", cfg.DatabaseURL)
	require.Equal(t, "localhost:6379", cfg.RedisAddr)
	require.Equal(t, "taskservice", cfg.ServiceName)
}

func TestIntFromEnvFallsBackOnInvalid(t *testing.T) {
	os.Setenv("SOME_INT", "not-a-number")
	defer os.Unsetenv("SOME_INT")

	require.Equal(t, 42, intFromEnv("SOME_INT", 42))

	os.Setenv("SOME_INT", "123")
	require.Equal(t, 123, intFromEnv("SOME_INT", 42))
}

func TestDefaultStringUsesEnvWhenPresent(t *testing.T) {
	os.Setenv("MY_KEY", "value")
	defer os.Unsetenv("MY_KEY")

	require.Equal(t, "value", defaultString("MY_KEY", "fallback"))
}
