package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"sovera-core-api/internal/config"
	"sovera-core-api/internal/repository"
)

func main() {
	cfg := config.LoadConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbPool, err := repository.InitDBPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	var relKind string
	err = dbPool.QueryRow(ctx, "SELECT relkind::text FROM pg_class WHERE relname = 'companies'").Scan(&relKind)
	if err != nil {
		log.Printf("Failed to get companies relkind: %v", err)
	} else {
		// r = ordinary table, i = index, S = sequence, t = TOAST table, v = view, m = materialized view, c = composite type, f = foreign table, p = partitioned table
		fmt.Printf("Relation 'companies' relkind: %s\n", relKind)
	}
}
