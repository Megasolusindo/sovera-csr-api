package main

import (
	"context"
	"log"
	"time"

	"sovera-core-api/internal/config"
	"sovera-core-api/internal/repository"
)

func main() {
	cfg := config.LoadConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbPool, err := repository.InitDBPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	grants := []string{
		"GRANT ALL ON ALL TABLES IN SCHEMA public TO sovera;",
		"GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO sovera;",
		"GRANT ALL ON ALL TABLES IN SCHEMA company TO sovera;",
		"GRANT ALL ON ALL SEQUENCES IN SCHEMA company TO sovera;",
		"ALTER TABLE organizations OWNER TO sovera;",
		"ALTER TABLE company_claims OWNER TO sovera;",
		"ALTER TABLE csr_opportunities OWNER TO sovera;",
		"ALTER TABLE ngo_programs OWNER TO sovera;",
		"ALTER TABLE proposals OWNER TO sovera;",
		"ALTER TABLE user_invitations OWNER TO sovera;",
	}

	for _, stmt := range grants {
		_, err := dbPool.Exec(ctx, stmt)
		if err != nil {
			log.Printf("Grant statement notice: %v", err)
		}
	}

	log.Println("Permissions updated successfully!")
}
