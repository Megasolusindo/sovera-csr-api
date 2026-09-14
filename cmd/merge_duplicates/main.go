package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DuplicatePair struct {
	CanonicalID   string
	CanonicalName string
	DuplicateID   string
	DuplicateName string
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://sovera:sover4@10.10.29.177:5432/sovera?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	log.Println("Connected to PostgreSQL database. Searching for duplicate company records...")

	// 1. Query candidates where RTRIM(name, '.') matches another company without trailing dot, or duplicates with identical normalized names
	query := `
		SELECT 
			c1.id::text AS canonical_id, 
			c1.name AS canonical_name,
			c2.id::text AS duplicate_id,
			c2.name AS duplicate_name
		FROM company.companies c1
		JOIN company.companies c2 
		  ON c1.id != c2.id 
         AND LOWER(RTRIM(c1.name, '.')) = LOWER(RTRIM(c2.name, '.'))
		WHERE c1.created_at <= c2.created_at
		  AND c1.name NOT LIKE '%.'
		  AND c2.name LIKE '%.';
	`

	rows, err := pool.Query(ctx, query)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	var pairs []DuplicatePair
	for rows.Next() {
		var p DuplicatePair
		if err := rows.Scan(&p.CanonicalID, &p.CanonicalName, &p.DuplicateID, &p.DuplicateName); err == nil {
			pairs = append(pairs, p)
		}
	}
	rows.Close()

	log.Printf("Found %d duplicate company pairs to merge.", len(pairs))

	mergedCount := 0
	for _, p := range pairs {
		log.Printf("[%d/%d] Merging '%s' (%s) <- duplicate '%s' (%s)",
			mergedCount+1, len(pairs), p.CanonicalName, p.CanonicalID, p.DuplicateName, p.DuplicateID)

		tx, err := pool.Begin(ctx)
		if err != nil {
			log.Printf("Error starting transaction for %s: %v", p.DuplicateID, err)
			continue
		}

		// Transfer crawling_targets
		_, _ = tx.Exec(ctx, `UPDATE crawling_targets SET company_id = $1 WHERE company_id = $2`, p.CanonicalID, p.DuplicateID)

		// Transfer company_signals
		_, _ = tx.Exec(ctx, `UPDATE intelligence.company_signals SET company_id = $1 WHERE company_id = $2`, p.CanonicalID, p.DuplicateID)

		// Transfer company_csr_profiles if exists
		_, _ = tx.Exec(ctx, `UPDATE company_csr_profiles SET company_id = $1 WHERE company_id = $2`, p.CanonicalID, p.DuplicateID)

		// Transfer company_csr_programs if exists
		_, _ = tx.Exec(ctx, `UPDATE company_csr_programs SET company_id = $1 WHERE company_id = $2`, p.CanonicalID, p.DuplicateID)

		// Transfer company_key_persons if table exists
		_, _ = tx.Exec(ctx, `UPDATE company_key_persons SET company_id = $1 WHERE company_id = $2`, p.CanonicalID, p.DuplicateID)

		// Copy over non-null metadata from duplicate to canonical if canonical has nulls
		_, _ = tx.Exec(ctx, `
			UPDATE company.companies c1
			SET linkedin_url = COALESCE(c1.linkedin_url, c2.linkedin_url),
				linkedin_status = COALESCE(c1.linkedin_status, c2.linkedin_status),
				instagram_url = COALESCE(c1.instagram_url, c2.instagram_url),
				instagram_status = COALESCE(c1.instagram_status, c2.instagram_status),
				facebook_url = COALESCE(c1.facebook_url, c2.facebook_url),
				youtube_url = COALESCE(c1.youtube_url, c2.youtube_url),
				website = COALESCE(c1.website, c2.website),
				ticker = COALESCE(c1.ticker, c2.ticker)
			FROM company.companies c2
			WHERE c1.id = $1 AND c2.id = $2;
		`, p.CanonicalID, p.DuplicateID)

		// Delete duplicate row
		_, err = tx.Exec(ctx, `DELETE FROM company.companies WHERE id = $1`, p.DuplicateID)
		if err != nil {
			log.Printf("Error deleting duplicate %s: %v", p.DuplicateID, err)
			_ = tx.Rollback(ctx)
			continue
		}

		if err := tx.Commit(ctx); err != nil {
			log.Printf("Error committing transaction for %s: %v", p.DuplicateID, err)
		} else {
			mergedCount++
		}
	}

	log.Printf("Deduplication cleanup complete! Successfully merged and removed %d duplicate companies.", mergedCount)
}
