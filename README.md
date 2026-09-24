# Sovera (FundIQ) - Core API & Intelligence Engine

> Enterprise B2B Fundraising Intelligence & Deal-Preparation Engine for Islamic Philanthropy, NGOs, and Higher Education Endowments.

---

## 1. System Overview

**Sovera** adalah mesin intelijen B2B yang mengotomatisasi pemantauan sinyal kemitraan korporasi (CSR, TJSL BUMN, dan Zakat Perniagaan), mencocokkannya dengan portofolio program lembaga filantropi menggunakan pencarian semantik vektor (`pgvector`), serta menghasilkan naskah proposal terpersonalisasi secara instan.

Platform ini mengusung **Enterprise Multi-Tenancy** dengan proteksi isolasi data berbasis **PostgreSQL Row-Level Security (RLS)** di level kernel database serta arsitektur **AI Autonomous v1.2** anti-prompt injection & zero data mocking.

---

## 2. Tech Stack

* **Language & Runtime:** Go 1.22+
* **Web Framework:** [Go-Fiber](https://gofiber.io/) (`gofiber/fiber`) (High-throughput REST API & Webhooks)
* **Database & Driver:** PostgreSQL 16+ dengan ekstensi [`pgvector`](https://github.com/pgvector/pgvector) & driver [`pgx/v5`](https://github.com/jackc/pgx)
* **Job Queue Broker:** [Asynq](https://github.com/hibiken/asynq) + Redis (Distributed task queue)
* **AI & Embeddings:** Claude 3.5 Sonnet / GPT-4o / Gemini 1.5 Flash (Extraction) & text-embedding-004 (via GenAI Go SDK)
* **Document Engine:** Go PDF (`maroto` / `gofpdf`) & DOCX Generator

---

## 3. Directory Structure

```text
sovera-core-api/
├── docs/
│   ├── GRAND-DESIGN-AI-AUTONOMOUS-CSRmatics-v1.2.md # Approved AI Autonomous Architecture Baseline v1.2
│   ├── PRD.md                       # Product Requirements & Universal NGO Expansion
│   ├── ARCHITECTURE.md              # System Architecture & Sequence Diagrams
│   ├── DATABASE_SCHEMA.md           # DDL, Indexes & RLS Policy Guide
│   ├── API_SPEC.md                  # Core Backend REST API & Webhook Specifications
│   ├── SCRAPER_API_SPEC.md          # Scraper Engine API & Ingestion Contract
│   ├── CRAWLER_ORCHESTRATION_SPEC.md# Crawler Dispatcher, Circuit Breaker & Operations Guide
│   ├── FEED_STRATEGY_SPEC.md        # Source Catalog, Taxonomy, Search Infrastructure & Quality Roadmap
│   ├── INTENT_SCORE_SPEC.md         # CSR Intent Scoring & Proposal Matcher Calculation Spec
│   └── RBAC_SPEC.md                 # Role-Based Access Control & Multi-Tenant Security Spec
├── cmd/
│   ├── api/
│   │   └── main.go             # Fiber REST API server entry point
│   └── worker/
│       └── main.go             # Asynq queue background worker entry point
├── internal/
│   ├── config/                 # Env variables & runtime constants
│   ├── handler/                # Fiber HTTP route handlers (Controllers)
│   ├── middleware/             # JWT Auth & Webhook HMAC validators
│   ├── repository/             # pgx connection pool & guarded RLS transactions
│   ├── queue/                  # Asynq tasks & background workers
│   │   ├── tasks.go
│   │   ├── retraction_worker.go # Asynchronous Lineage Retraction Pipeline (§9.1)
│   │   ├── extraction_worker.go
│   │   └── proposal_worker.go
│   ├── service/
│   │   ├── ai/                 # AI Autonomous v1.2 Engine & LLM Adapters
│   │   │   ├── security.go      # HMAC Signed Task Context Generator & Validator
│   │   │   ├── nonce_store.go   # Replay Nonce Store with Redis TTL Eviction (§9.2)
│   │   │   ├── cost_monitor.go  # Hard-Cap Cost Circuit Breaker ($0.50/task, $10/day)
│   │   │   ├── research_agent.go# Research Agent Sandbox (Scope GLOBAL, 5-step cap)
│   │   │   ├── demand_aggregator.go # Anonymized Demand Aggregator (§22)
│   │   │   ├── secure_fetcher.go # SSRF Shield & Egress Query Filter
│   │   │   ├── extraction_workflow.go # Flexible Idempotency Key (§18)
│   │   │   ├── press_release_tracker.go # Press Release Wire Origin Hash (§15)
│   │   │   ├── grounding_verifier.go # Temporal, Negation & Acronym Grounding (§5.1)
│   │   │   ├── verification_workflow.go # Source Tiering Matrix (§5.2)
│   │   │   └── json_repair_guard.go # Go JSON Repair & EXTRACTION_FAILED Guard (§9.3)
│   │   ├── matcher/            # Cosine similarity vector search
│   │   └── exporter/           # PDF & Word document generator
│   └── model/                  # Data structs & domain entities
├── db/
│   └── migrations/             # SQL DDL & RLS initialization scripts
├── docker-compose.yml          # Docker composition for API, Worker, Redis & PostgreSQL
├── .env.example                # Configuration template
├── go.mod
└── go.sum
```

---

## 4. Getting Started (Local & Docker Development)

### 4.1 Run via Docker Compose (Recommended)

Jalankan seluruh layanan API backend, worker, PostgreSQL, dan Redis:

```bash
docker compose up -d --build
```

### 4.2 Manual Environment Setup

Salin template `.env.example` ke `.env`:

```bash
cp .env.example .env
```

Isi konfigurasi database dan API secrets:

```env
PORT=4000
NODE_ENV=development
DATABASE_URL="postgres://postgres:ResulteW212%23@10.10.29.177:5432/sovera?sslmode=disable"
REDIS_URL=localhost:6379
WEBHOOK_SECRET_KEY=super_secret_crawler_key_123
JWT_SECRET=super_secret_jwt_key_enterprise
AI_API_KEY=your_gemini_api_key_here
OPENCLAW_AGENT_TOKEN=openclaw_agent_secret_token_2026
```

### 4.3 Run Automated AI Suite Unit Tests

Eksekusi seluruh 22+ unit test otomatis untuk memvalidasi fitur keamanan, sandbox, dan grounding AI:

```bash
go test -v ./internal/service/ai/... ./internal/queue/...
```

---

## 5. Security & AI Autonomous v1.2 Standard

1. **Kernel RLS Isolation:** Setiap operasi data privat wajib dibungkus dalam `WithTenantContext(ctx, pool, orgID, func(tx pgx.Tx) error { ... })` yang otomatis menjalankan `SELECT set_config('app.current_org_id', $1, true)` dengan tipe data terikat yang aman.
2. **HMAC Signed Task Context & Replay Protection:** Setiap penugasan AI agent dibatasi oleh signed token HMAC-SHA256 (TTL 5 menit) dan single-use `nonce` store (Redis TTL eviction `volatile-ttl`).
3. **Research Agent Sandbox (Anti-Prompt Injection):** Research Agent 100% berjalan dalam scope `GLOBAL` (tanpa secret, JWT, atau tenant context) dengan hard limit max 5 step reasoning dan 10 tool calls. Output dilabeli `UNTRUSTED_DATA`.
4. **Anonymized Watchlist Demand Aggregator (§22):** Watchlist NGO diagregasikan secara depersonalisasi tanpa membawa `tenant_id` ke luar.
5. **Hard-Cap Cost Circuit Breaker (§3.3):** Pembatasan biaya otomatis $0.50 USD / task dan $10.00 USD / hari / tenant dengan Kill Switch global.
6. **Strict Zero Data Mocking:** Seluruh sinyal CSR dan profil korporasi wajib berasal dari sumber empirical terverifikasi.

---

## 6. License

Proprietary & Confidential. All rights reserved.
