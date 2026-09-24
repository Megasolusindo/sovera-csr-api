# System Architecture Document (ARCHITECTURE.md)

**Product:** Sovera (FundIQ) Core API & Ingestion Engine  
**Document Version:** 1.1  
**Target Runtime:** Go 1.22+ (Go-Fiber / `pgx/v5` / Asynq) + PostgreSQL (`pgvector`) + Redis  

---

## 1. Architectural Overview & Design Principles

Sovera mengadopsi arsitektur **Decoupled 3-Tier Enterprise Pattern** yang memisahkan beban *crawling* data, orkestrasi asinkron AI, dan antarmuka pengguna dasbor.

```text
┌─────────────────────────────────────────────────────────────┐
│       1. Sovera Core Backend (Cron Scheduler Orchestrator)  │
│   - Periodic Tick: Target DB Registry (crawling_targets)    │
└──────────────────────────────┬──────────────────────────────┘
                               │ HTTP POST (Dispatch Task to SCRAPER_SERVICE_URL)
                               ▼
┌─────────────────────────────────────────────────────────────┐
│            2. External WebScraper Execution Service         │
│   - Render PDF / Anti-Bot Bypass / Raw Text Extraction      │
└──────────────────────────────┬──────────────────────────────┘
                               │ HTTP Webhook POST (Payload + HMAC SHA256)
                               ▼
┌─────────────────────────────────────────────────────────────┐
│       3. Sovera Core Backend API & Worker (Go-Fiber)        │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ Webhook Ingestion Controller (HMAC Verification)      │  │
│  └───────────────────────────┬───────────────────────────┘  │
│                              │ Enqueues Task                │
│                              ▼                              │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ Redis Task Queue Broker (Asynq)                       │  │
│  │  - Task: `task:raw_ingestion`                         │  │
│  │  - Task: `task:llm_extraction`                        │  │
│  │  - Task: `task:proposal_generation`                   │  │
│  └───────────────────────────┬───────────────────────────┘  │
│                              │ Processes via Worker         │
│                              ▼                              │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ AI Pipeline & Embedding Engine (GenAI Go SDK)         │  │
│  │  - Gemini Flash: Structured Extraction & Summary      │  │
│  │  - Embedding Engine: Vectorization (1536 dim)         │  │
│  │  - Semantic Matcher: Cosine Distance vs Programs      │  │
│  └───────────────────────────┬───────────────────────────┘  │
│                              │ Enforces RLS via pgx/v5      │
│                              ▼                              │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ Database Layer: PostgreSQL + pgvector                 │  │
│  │  - Kernel RLS: `SET LOCAL app.current_org_id = $1`    │  │
│  └───────────────────────────────────────────────────────┘  │
└──────────────────────────────▲──────────────────────────────┘
                               │ REST / SSE API (JWT Auth)
                               │
┌──────────────────────────────┴──────────────────────────────┐
│              3. Next.js Frontend (Web Dashboard)            │
│   - Institutional Corporate Intelligence Feed               │
│   - Semantic Matching Visualizer                            │
│   - AI Proposal Studio & Export (Docx / PDF)                │
└─────────────────────────────────────────────────────────────┘
```

### Core Design Principles
1. **Decoupled Pipeline:** Webhook hanya menerima payload dan segera mengembalikan respons `202 Accepted`. Seluruh pemrosesan teks, parsing PDF, dan panggilan LLM berjalan asinkron di antrean *background worker* Asynq.
2. **Zero-Trust Multi-Tenancy:** Akses tabel privat lembaga dilindungi langsung di level PostgreSQL Kernel Row-Level Security (RLS). Tidak ada dependensi isolasi di lapisan aplikasi (*application-level filtering*).
3. **Optimized AI Token Economy:** Ekstraksi dan pengkategorian skala besar menggunakan model *high-throughput/low-cost* (Gemini Flash), sedangkan penyusunan naskah proposal strategis menggunakan model *high-reasoning* (Gemini Pro / Claude Sonnet).

---

## 2. Ingestion & AI Pipeline Lifecycle

Setiap data yang masuk melalui webhook diproses melalui 4 tahapan berurutan:

```text
[Webhook Received] ──► [HMAC Validated] ──► [Save Raw Payload]
                                                   │
                                                   ▼
[Vectorize Signal] ◄── [Structured JSON] ◄── [LLM Extraction]
         │
         ▼
[Insert to `public_corporate_signals`]
         │
         ▼
[Trigger Event: Signal Ready] ──► [Match Against Tenant Programs (On-Demand / Webhook)]
```

### Pipeline Details

#### A. Webhook Ingestion & Deduplication
* Backend memverifikasi header `X-Hub-Signature-256`.
* Payload di-hash menggunakan algoritma SHA-256 (`content_hash = SHA256(raw_text)`).
* Jika `source_url` atau `content_hash` sudah ada di database dalam 30 hari terakhir, pemrosesan dihentikan (*idempotent duplicate drop*).

#### B. LLM Entity Extraction Worker
* Worker mengambil teks mentah/markdown dan memanggil LLM menggunakan fitur *Structured Outputs (JSON Schema)* via Google GenAI Go SDK.
* **Target Output:**
  ```json
  {
    "company_name": "string",
    "industry_sector": "string",
    "csr_pillar_focus": "string",
    "target_regions": ["string"],
    "estimated_budget_signal": "number | null",
    "trigger_event": "string",
    "intent_score": "number (1-100)",
    "summary": "string"
  }
  ```

#### C. Vector Embedding Worker
* Teks gabungan (`company_name` + `industry_sector` + `csr_pillar_focus` + `summary`) dikirim ke embedding model (1536 dimensi).
* Vektor disimpan ke kolom `public_corporate_signals.signal_embedding`.

---

## 3. Queue & Background Job Architecture (Asynq + Redis)

Untuk menjamin skalabilitas dan ketahanan terhadap *upstream rate limit*, arsitektur antrean dibagi menjadi 3 tipe task utama menggunakan **Asynq**:

| Task Type | Concurrency | Retry Strategy | Backoff | Deskripsi |
| :--- | :--- | :--- | :--- | :--- |
| `task:raw_ingestion` | 10 | 3 attempts | Exponential (2s, 4s, 8s) | Menerima payload mentah, validasi skema, deduplikasi. |
| `task:llm_extraction` | 5 | 5 attempts | Exponential (5s, 15s, 45s) | Ekstraksi entitas JSON dan pembuatan vector embedding. |
| `task:proposal_generation` | 3 | 2 attempts | Fixed (3s) | Generasi draf proposal personal saat dipicu oleh pengguna UI. |

### Dead Letter Queue (DLQ) Strategy
Task yang gagal setelah percobaan maksimal dialihkan ke antrean *retried/archived* beserta log eror lengkap untuk dianalisis tanpa memblokir antrean utama.

---

## 4. Multi-Tenant Database & RLS Enforcement Pattern

Untuk mencegah kebocoran data antar-lembaga saat menggunakan *connection pooling* `pgxpool`, backend menerapkan pola transaksi berpagar (*Guarded Tenant Transaction*).

```text
pgxpool.Acquire(ctx)
   │
   ▼
tx, err := pool.Begin(ctx)
   │
   ▼
tx.Exec(ctx, "SET LOCAL app.current_org_id = $1", orgID)
   │
   ├─► Query Program Lembaga (RLS Aktif)
   ├─► Query Pipeline & Catatan (RLS Aktif)
   ├─► Query Data Sinyal Publik (Shared Read-Only)
   │
   ▼
tx.Commit(ctx) / tx.Rollback(ctx);
   │
   ▼
Connection returned to pool in neutral state
```

---

## 5. Security & Isolation Matrix

| Layer | Komponen Keamanan | Implementasi |
| :--- | :--- | :--- |
| **Transport** | Ingestion Webhook | HMAC SHA-256 Signature verification via pre-shared secret. |
| **Auth** | Client API & Dashboard | JWT (JSON Web Token) dengan klaim `org_id`, `user_id`, dan `role`. |
| **Data Kernel** | PostgreSQL Database | Row-Level Security (`FORCE ROW LEVEL SECURITY`) pada tabel privat. |
| **AI Privacy** | Model API Calls | Enforced Zero-Data Retention policy (tidak ada data privat yang dijadikan materi *training*). |

---

## 6. Directory Structure (`sovera-core-api`)

```text
sovera-core-api/
├── docs/
│   ├── PRD.md
│   ├── ARCHITECTURE.md
│   ├── CRAWLER_ORCHESTRATION_SPEC.md
│   ├── DATABASE_SCHEMA.md
│   └── API_SPEC.md
├── cmd/
│   ├── api/
│   │   └── main.go         # Fiber HTTP Server Entry point
│   └── worker/
│       └── main.go         # Asynq Worker Process Entry point
├── internal/
│   ├── config/             # Environment variables & constants
│   ├── handler/            # Fiber HTTP Controllers
│   ├── middleware/         # Auth JWT & HMAC Webhook middleware
│   ├── repository/         # pgx database queries & RLS context wrappers
│   ├── queue/              # Asynq task definitions & worker handlers
│   │   ├── tasks.go
│   │   ├── ingestion_worker.go
│   │   ├── extraction_worker.go
│   │   └── proposal_worker.go
│   ├── service/
│   │   ├── ai/             # GenAI Go SDK, prompts & structured output
│   │   ├── matcher/        # Cosine similarity matching engine
│   │   └── exporter/       # PDF / Word proposal generator
│   └── model/              # Domain models & payload structs
├── db/
│   └── migrations/         # SQL DDL & RLS policies
├── docker-compose.yml       # Local PostgreSQL + pgvector + Redis setup
├── .env.example
├── go.mod
└── go.sum
```

---

## 6. Sequence Diagram & Call Flows

```mermaid
sequenceDiagram
    autonumber
    actor Cron as Asynq Scheduler / Cron
    participant DB as PostgreSQL Database
    participant Disp as Crawler Dispatcher Worker
    participant DAS as WebScraper Service (DAS)
    participant WH as Webhook Handler (Core API)
    participant Queue as Redis (Asynq Queue)
    participant AI as Extraction Worker (Gemini LLM)

    rect rgb(240, 248, 255)
    note over Cron, DAS: TAHAP 1: Task Dispatching (Periodic Scheduling)
    Cron->>Disp: Tick Trigger (task:dispatch_crawling)
    Disp->>DB: GetDueTargets(limit=50)
    DB-->>Disp: Return list of CrawlingTarget due for scrape
    loop Setiap Target (Round-Robin Interleaved & 3s Delay)
        Disp->>DB: CreateLog(task_id, status='DISPATCHED')
        Disp->>DAS: POST /api/v1/scrape-tasks (Bearer Token + Target Config)
        DAS-->>Disp: HTTP 202 Accepted {"task_id": "...", "status": "ACCEPTED"}
        Disp->>DB: UpdateTargetNextRun(target_id, check_interval_hours)
    end
    end

    rect rgb(245, 245, 220)
    note over DAS, WH: TAHAP 2: Pemrosesan Scraper & Delivery Webhook Callback
    DAS->>WH: POST /api/v1/webhooks/crawler (Header: X-Hub-Signature-256)
    WH->>WH: Verify HMAC SHA-256 Signature
    alt Callback Status FAILED / HTTP 429
        WH->>DB: UpdateLogStatus(task_id, status='FAILED')
        WH->>DB: RecordFailure(target_id) [Progressive Backoff: 30m/2h/6h/24h]
        WH-->>DAS: HTTP 200 OK
    else Callback Status COMPLETED
        WH->>WH: SelectBestContent & GenerateContentHash (SHA-256)
        WH->>DB: UpdateLogStatus(task_id, status='COMPLETED')
        WH->>DB: RecordSuccess(target_id)
        WH->>Queue: Enqueue task:llm_extraction
        WH-->>DAS: HTTP 202 Accepted
    end
    end

    rect rgb(240, 255, 240)
    note over Queue, AI: TAHAP 3: Asynchronous LLM Signal & ESG Processing
    Queue->>AI: Dequeue task:llm_extraction
    AI->>AI: Gemini LLM Signal Extraction & Entity Resolution
    AI->>AI: GenerateEmbedding (text-embedding-004, 1536-dim)
    AI->>DB: SaveSignal (Persist corporate signal + vector embedding)
    end

---

## 7. Background Workers & Automated Schedulers

Platform backend menggunakan Redis Asynq Broker (`cmd/worker/main.go`) untuk mengeksekusi worker asinkron dan scheduler otomatis berkala:

| Nama Worker | Task Queue Name | Cron Schedule | Fungsi & Alur Kerja Utama |
| :--- | :--- | :---: | :--- |
| **`CompanyEnrichmentWorker`** | `task:enrich_missing_websites` | `0 */6 * * *` (6 jam) | Menyeleksi perusahaan emiten Tbk yang websitenya kosong/placeholder (`https://-`), mencocokkan ticker dengan profil resmi IDX API (`GetCompanyProfiles`), memverifikasi domain via HTTP HEAD check, dan mengupdate `company.companies.website`. |
| **`CompanyContactDiscoveryWorker`** | `task:company_contact_discovery` | `0 */6 * * *` (6 jam) | Menyeleksi perusahaan dengan `phone` atau `email` kosong. Menjalankan 3-tier empirical pipeline (Direct Website HTML Scraping $\rightarrow$ Serper Google Search $\rightarrow$ Normalisasi `phoneverifier` & RFC 5322 regex). |
| **`CompanyLinkedInDiscoveryWorker`** | `task:company_linkedin_discovery_batch` | `0 */12 * * *` (12 jam) | Penemuan profil LinkedIn resmi korporasi melalui ekstraksi HTML website resmi dan Serper Google Search API. |
| **`CompanyInstagramDiscoveryWorker`** | `task:company_instagram_discovery_batch` | `0 */12 * * *` (12 jam) | Penemuan profil Instagram resmi korporasi secara empiris dari website & Google Search API. |
| **`URLHealthCheckWorker`** | `task:url_health_check` | `*/5 * * * *` (5 menit) | Memeriksa ketersediaan (health check) seluruh URL website dan medsos di database, mengupdate flag status `VALID` / `INVALID`. |
| **`CompanyFacebook/Instagram/Youtube/LinkedIn Workers`** | Social Media Validation Queue | Continuous / Batch | Memvalidasi URL media sosial. **Aturan Mutlak:** Jika URL media sosial dinyatakan `INVALID` (404/broken), worker secara otomatis menghapus URL dari DB (`SET url = NULL, status = NULL`). |

---

## 8. Data Integrity & Empirical Enrichment Directives

### A. Strict No Synthetic URL Reconstruction Directive
- **Dilarang Keras Menebak Handle / URL Sintetis:** Under no circumstances should any synthetic, reconstructed, or slugified URLs be generated or inserted into `crawling_targets` or `company.companies` (misal: `https://www.facebook.com/{company-slug}`, `https://www.youtube.com/@{company-slug}`).
- **Empirical & Verified Sources Only:** Seluruh URL media sosial, website, telepon, dan email wajib diperoleh dari sumber empiris aktif (ekstraksi HTML website resmi yang terverifikasi HTTP 2xx/3xx, atau hasil pencarian Google/Serper API).
- **Auto-Purge on Failure:** Jika URL media sosial yang sudah tersimpan dinyatakan `INVALID` saat diuji oleh health worker, sistem secara otomatis mengosongkan URL tersebut (`url = NULL`) agar link rusak tidak pernah tampil di UI Dashboard.

### B. Standardized Data Deduplication & Legacy Cleanup
- **Prinsip Single Source of Truth:** Seluruh data korporasi terkonsolidasi di skema `company.companies`.
- **Legacy Duplicate Resolution:** Jika terdapat record duplikat lama ber-suffix `[LEGACY <uuid>]`, sistem wajib:
  1. Mereroute seluruh referensi tabel anak (`company_enriched_programs`, `intelligence.company_signals`, `crawling_targets`) ke ID record perusahaan utama (canonical record).
  2. Menghapus (*delete*) record duplikat `[LEGACY]` yang telah kosong/orphaned secara aman melalui transaksi database.

```