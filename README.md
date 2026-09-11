# Sovera (FundIQ) - Core API & Intelligence Engine

> Enterprise B2B Fundraising Intelligence & Deal-Preparation Engine for Islamic Philanthropy, NGOs, and Higher Education Endowments.

---

## 1. System Overview

**Sovera** adalah mesin intelijen B2B yang mengotomatisasi pemantauan sinyal kemitraan korporasi (CSR, TJSL BUMN, dan Zakat Perniagaan), mencocokkannya dengan portofolio program lembaga filantropi menggunakan pencarian semantik vektor (`pgvector`), serta menghasilkan naskah proposal terpersonalisasi secara instan.

Platform ini mengusung **Enterprise Multi-Tenancy** dengan proteksi isolasi data berbasis **PostgreSQL Row-Level Security (RLS)** di level kernel database.

---

## 2. Tech Stack

* **Language & Runtime:** Go 1.22+
* **Web Framework:** [Go-Fiber](https://gofiber.io/) (`gofiber/fiber`) (High-throughput REST API & Webhooks)
* **Database & Driver:** PostgreSQL 16+ dengan ekstensi [`pgvector`](https://github.com/pgvector/pgvector) & driver [`pgx/v5`](https://github.com/jackc/pgx)
* **Job Queue Broker:** [Asynq](https://github.com/hibiken/asynq) + Redis (Distributed task queue)
* **AI & Embeddings:** Gemini 1.5 Flash (Extraction) & text-embedding-004 / 1536-dim vector models (via Google GenAI Go SDK)
* **Document Engine:** Go PDF (`maroto` / `gofpdf`) & DOCX Generator

---

## 3. Directory Structure

```text
sovera-core-api/
├── docs/
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
│   ├── queue/                  # Asynq tasks & worker handlers
│   │   ├── tasks.go
│   │   ├── ingestion_worker.go
│   │   ├── extraction_worker.go
│   │   └── proposal_worker.go
│   ├── service/
│   │   ├── ai/                 # LLM Structured Output & Embeddings (GenAI Go SDK)
│   │   ├── matcher/            # Cosine similarity vector search
│   │   └── exporter/           # PDF & Word document generator
│   └── model/                  # Data structs & domain entities
├── db/
│   └── migrations/             # SQL DDL & RLS initialization scripts
├── docker-compose.yml          # Local PostgreSQL (pgvector) + Redis
├── .env.example                # Configuration template
├── go.mod
└── go.sum

```

---

## 4. Getting Started (Local Development)

### 4.1 Prerequisites

* Go v1.22 or higher
* Docker & Docker Compose
* Gemini API Key

### 4.2 Start Local Infrastructure

Jalankan PostgreSQL dengan ekstensi `pgvector` dan Redis broker menggunakan Docker Compose:

```bash
docker compose up -d
```

### 4.3 Environment Setup

Salin template konfigurasi `.env.example` ke `.env`:

```bash
cp .env.example .env
```

Sesuaikan nilai variabel:

```env
PORT=4000
NODE_ENV=development
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/sovera_db
REDIS_URL=localhost:6379
WEBHOOK_SECRET_KEY=super_secret_crawler_key_123
JWT_SECRET=super_secret_jwt_key_enterprise
AI_API_KEY=your_gemini_api_key_here
```

### 4.4 Run Database Migrations

Eksekusi migrasi DDL dan Row-Level Security ke PostgreSQL:

```bash
# Menggunakan golang-migrate CLI atau runner migrasi bawaan
migrate -path db/migrations -database "$DATABASE_URL" up
```

### 4.5 Start Development Server & Worker

Jalankan server API Fiber:

```bash
go run cmd/api/main.go
```

Jalankan worker antrean Asynq di jendela terminal terpisah:

```bash
go run cmd/worker/main.go
```

Server API akan aktif di `http://localhost:4000/api/v1`.

---

## 5. Security & Isolation Standard

1. **Kernel RLS Isolation:** Setiap operasi data privat wajib dibungkus dalam fungsi `WithTenantContext(ctx, pool, orgID, func(tx pgx.Tx) error { ... })` yang otomatis menjalankan `SET LOCAL app.current_org_id = $1`.
2. **Webhook Integrity:** Seluruh payload masuk dari crawler service divalidasi tanda tangannya via header `X-Hub-Signature-256` dengan HMAC-SHA256.
3. **AI Privacy Policy:** Data privat proposal, catatan negosiasi, dan identitas donatur tidak pernah dikirimkan untuk pelatihan model publik.

---

## 6. License

Proprietary & Confidential. All rights reserved.
