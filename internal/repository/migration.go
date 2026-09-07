package repository

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunMigrations executes all *.up.sql migration files in sorted order from db/migrations directory into PostgreSQL.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool, migrationsDir string) error {
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("failed to list migration files: %w", err)
	}

	if len(files) == 0 {
		log.Printf("No migration files found in %s", migrationsDir)
		return nil
	}

	sort.Strings(files)

	for _, file := range files {
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		filename := filepath.Base(file)
		log.Printf("Executing migration: %s", filename)
		_, err = pool.Exec(ctx, string(sqlBytes))
		if err != nil {
			log.Printf("Notice: Migration %s execution result: %v", filename, err)
		}
	}

	return nil
}
