package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	HTTPPort    int
	DatabaseURL string
	RedisAddr   string
	ServiceName string
	EnablePprof bool
}

func Load() (Config, error) {
	cfg := Config{
		HTTPPort:    intFromEnv("HTTP_PORT", 8080),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisAddr:   defaultString("REDIS_HOST", "localhost:6379"),
		ServiceName: defaultString("SERVICE_NAME", "taskservice"),
		EnablePprof: boolFromEnv("PPROF_ENABLED", false),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func intFromEnv(key string, def int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}

	return parsed
}

func defaultString(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

func boolFromEnv(key string, def bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return def
	}

	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return def
	}

	return parsed
}
