package cmd

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"taskservice/internal/infrastructure/postgres"
)

func maybeRunMigrations(ctx context.Context, pool *pgxpool.Pool, skip bool) error {
	if skip {
		log.Print("skipping migrations (flag: --skip-migrations)")
		return nil
	}

	if err := postgres.RunMigrations(ctx, pool); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}
