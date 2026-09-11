# Sovera Core API & Ingestion Engine - Developer Context (Claude Code)

## Project Overview
Sovera (FundIQ) is an enterprise B2B fundraising intelligence and deal-preparation engine tailored for Islamic philanthropy (LAZNAS/BMT), NGOs, and higher education endowments.

## Tech Stack & Runtime
- **Language & Runtime:** Go 1.22+
- **Web Framework:** [Go-Fiber](https://gofiber.io/) (`gofiber/fiber`) (High-throughput REST API & Webhooks)
- **Database & Driver:** PostgreSQL 16+ with `pgvector`, `uuid-ossp`, and [`pgx/v5`](https://github.com/jackc/pgx)
- **Queue/Worker:** [Asynq](https://github.com/hibiken/asynq) (`hibiken/asynq`) + Redis
- **AI & Embedding:** Google GenAI SDK for Go (`google.golang.org/genai`) & 1536-dim vector embeddings
- **Document Export:** Go PDF (`maroto` / `gofpdf`) & DOCX generator

## Strict Architecture & Multi-Tenancy Rules
1. **Kernel RLS Isolation:** Never query tenant-isolated tables (`institution_programs`, `deal_pipelines`) directly without an active transaction setting `SET LOCAL app.current_org_id = $1`. Always use the `WithTenantContext()` helper with `pgx.Tx`.
2. **Shared Data:** `public_corporate_signals` is a read-only shared dataset across all tenants.
3. **Decoupled Processing:** Ingestion endpoints (`/api/v1/webhooks/crawler`) must validate HMAC SHA-256 signatures, deduplicate payloads using `content_hash`, and return `202 Accepted` immediately while delegating LLM processing to Asynq background workers.
4. **Anti-429 Rate Limiting & Cooldown:** Enforce per-host inter-request delay (min 3s) and progressive backoff (`NOW() + INTERVAL '30m/2h/6h/24h'`) for rate-limited crawling targets.
5. **Zero-Contamination AI Policy:** Never use private tenant proposals, pipeline notes, or beneficiary data for fine-tuning or public model ingestion.

## Standard Workflow & Commands
- **Dev API Server:** `go run cmd/api/main.go` (or `air` for hot reload)
- **Dev Worker:** `go run cmd/worker/main.go`
- **Build API:** `go build -o bin/api cmd/api/main.go`
- **Build Worker:** `go build -o bin/worker cmd/worker/main.go`
- **Run Tests:** `go test ./...`

## Directory Structure
- `docs/`: Master specification documents (`PRD.md`, `ARCHITECTURE.md`, `DATABASE_SCHEMA.md`, `API_SPEC.md`, `SCRAPER_API_SPEC.md`, `CRAWLER_ORCHESTRATION_SPEC.md`, `FEED_STRATEGY_SPEC.md`, `INTENT_SCORE_SPEC.md`, `RBAC_SPEC.md`)
- `cmd/api/`: HTTP server entry point (`main.go`)
- `cmd/worker/`: Asynq background worker entry point (`main.go`)
- `internal/handler/`: Fiber HTTP route handlers (Controllers)
- `internal/middleware/`: JWT authentication and HMAC webhook validators
- `internal/repository/`: `pgx` connection pool, SQL queries, and RLS context wrappers (`WithTenantContext`)
- `internal/queue/`: Asynq task payloads and worker handlers (`dispatch`, `poll_pending`, `extraction`, `esg`, `proposal`, `health_check`)
- `internal/service/ai/`: LLM prompt templates, structured parsers, and embedding generators
- `internal/service/matcher/`: Cosine similarity vector search functions
- `internal/service/exporter/`: PDF & DOCX document generator
- `db/migrations/`: SQL DDL & RLS initialization scripts