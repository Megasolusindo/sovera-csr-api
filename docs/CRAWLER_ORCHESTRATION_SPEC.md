# Crawler Orchestration & Scheduler Specification (CRAWLER_ORCHESTRATION_SPEC.md)

**Product:** Sovera (FundIQ) Core API & Ingestion Engine  
**Component:** Backend Orchestrator & External Scraper Dispatcher  
**Target Runtime:** Go 1.22+ (`sovera-core-api`), Redis (Asynq Scheduler), WebScraper Service  
**Document Version:** 2.0  

---

## 1. Overview & Architectural Philosophy

Dokumen ini mendefinisikan arsitektur dan spesifikasi teknis untuk **Orkestrasi & Penjadwalan Pemindaian Data** (*Crawler Scheduling & Dispatching Pattern*).

### Prinsip Utama
1. **Decoupled Stateless Scraper:** *WebScraper Service* berperan sebagai *Execution Engine* murni yang bersifat *stateless* (mengeksekusi tugas crawling/parsing PDF/Web berbasis HTTP request). Scraper **tidak** mengelola state database target atau jadwal cron internal.
2. **Backend Engine as Orchestrator:** *Sovera Core Backend API* bertindak sebagai *Orchestrator & Scheduler*. Backend menyimpan registri situs target, mengelola jadwal eksekusi (Asynq Cron Scheduler), mentrigger scraper, dan menerima *callback webhook* hasil ekstraksi.
3. **Idempotent Ingestion & Deduplication:** Setiap hasil crawling diverifikasi menggunakan tanda tangan HMAC SHA-256 (atau fallback token) dan di-deduplikasi berbasis hash konten (`content_hash`) untuk menghemat penggunaan kuota AI LLM.
4. **Fault Tolerance & Fallback Polling:** Selain pengiriman webhook langsung, backend menjalankan poller periodik (`task:poll_pending_tasks`) untuk memantau status pekerjaan via `GET /api/v1/tasks/{task_id}` jika callback webhook terlepas.

---

## 2. Architecture & Data Flow Diagram

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                    Sovera Core API (Cron Orchestrator)                  │
│                                                                         │
│   ┌──────────────────────────┐         ┌─────────────────────────────┐  │
│   │ PostgreSQL Database      │         │ Asynq Cron Scheduler        │  │
│   │ - `crawling_targets`     │────────►│ - Dispatcher (1 Hour Cron)  │  │
│   │ - `crawling_logs`        │         │ - Fallback Poller (15 Min)  │  │
│   └──────────────────────────┘         └──────────────┬──────────────┘  │
└───────────────────────────────────────────────────────┼─────────────────┘
                                                        │ HTTP Requests (v2 Endpoints)
                                                        │ POST /scrape-tasks | /discovery
                                                        │ GET /tasks/{task_id}
                                                        ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                     External Scraper Service Engine                     │
│                                                                         │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │ WebScraper Service (`SCRAPER_SERVICE_URL`)                      │   │
│   │ - Headless Browser / Anti-Bot / PDF Extractor / Search Engine   │   │
│   └────────────────────────────────┬────────────────────────────────┘   │
└────────────────────────────────────┼────────────────────────────────────┘
                                     │ HTTP Webhook Delivery
                                     │ POST `WEBHOOK_URL` + HMAC SHA256 / Secret
                                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    Sovera Core API (Webhook Receiver)                   │
│                                                                         │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │ Webhook Ingestion Controller (`POST /api/v1/webhooks/crawler`)  │   │
│   │ 1. Multi-Layer Auth (HMAC SHA-256 / Secret / Bearer)            │   │
│   │ 2. Compute Content Hash (`content_hash = SHA256(raw_text)`)     │   │
│   │ 3. Check Duplicate -> Enqueue `task:llm_extraction` to Redis    │   │
│   └────────────────────────────────┬────────────────────────────────┘   │
│                                    │ Async Background Processing        │
│                                    ▼                                    │
│   ┌─────────────────────────────────────────────────────────────────┐   │
│   │ Gemini LLM Extraction & Vector Embedding Engine (1536-dim)      │   │
│   │ Entity Resolver & ESG Extractor -> `public_corporate_signals`  │   │
│   └─────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Database Schema Specification

### 3.1 Table `crawling_targets` (Registri Target Pemindaian)
Tabel ini menyimpan daftar URL situs target, laporan PDF, RSS feed, atau feed hibah yang perlu dipantau secara periodik.

```sql
CREATE TABLE crawling_targets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES company_master(id) ON DELETE SET NULL,
    source_name VARCHAR(255) NOT NULL,            -- Misal: "IDX Sustainability Report Telkom"
    source_type VARCHAR(50) NOT NULL,             -- 'PDF_DOCUMENT', 'NEWS_ARTICLE', 'CSR_OPPORTUNITY_SEARCH', 'COMPANY_ENRICHMENT', 'SEARCH_DISCOVERY'
    target_url TEXT NOT NULL,                     -- URL target dokumen/halaman
    check_interval_hours INT DEFAULT 24,          -- Interval pemeriksaan dalam jam
    last_scraped_at TIMESTAMPTZ,                  -- Waktu pemindaian terakhir
    next_run_at TIMESTAMPTZ DEFAULT NOW(),        -- Jadwal eksekusi berikutnya
    is_active BOOLEAN DEFAULT TRUE,               -- Status keaktifan target
    consecutive_failures INT DEFAULT 0,
    last_http_status INT,
    last_error_message TEXT,
    health_status VARCHAR(50) DEFAULT 'HEALTHY',  -- 'HEALTHY', 'DEGRADED', 'DISABLED_DEAD_LINK'
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index untuk efisiensi query scheduler
CREATE INDEX idx_crawling_targets_active_next_run 
ON crawling_targets (is_active, next_run_at) 
WHERE is_active = TRUE;
```

### 3.2 Table `crawling_logs` (Histori Eksekusi Dispatched Tasks)
Tabel ini mencatat histori pengiriman tugas dari Backend Orchestrator ke Scraper Service beserta hasilnya.

```sql
CREATE TABLE crawling_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_id UUID REFERENCES crawling_targets(id) ON DELETE SET NULL,
    task_id VARCHAR(100) NOT NULL UNIQUE,         -- Unique ID tugas yang dikirim ke scraper
    status VARCHAR(50) NOT NULL,                  -- 'DISPATCHED', 'COMPLETED', 'FAILED'
    http_status_code INT,
    error_message TEXT,
    execution_time_ms INT,
    content_hash VARCHAR(64),                     -- SHA-256 hash hasil ekstraksi
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

## 4. Orchestration & Dispatching Lifecycle

### Step 1: Periodic Cron Scheduler Tick
1. Asynq Scheduler menjalankan job `task:dispatch_crawling` setiap 1 jam.
2. Worker mengambil daftar target aktif dari database yang sudah jatuh tempo (`next_run_at <= NOW()`).

### Step 2: HTTP Task Dispatching to WebScraper Service
Untuk setiap target yang siap dieksekusi, Backend menyusun payload HTTP POST ke `SCRAPER_SERVICE_URL` (`POST /api/v1/scrape-tasks`), `/api/v1/discovery`, atau `/api/v1/crawl-jobs`.

### Step 3: Scraper Execution & Webhook Delivery
1. Scraper Service mengeksekusi ekstraksi dan mengembalikan `202 Accepted`.
2. Setelah selesai, Scraper mengirimkan *callback webhook* ke `WEBHOOK_URL` (`POST /api/v1/webhooks/crawler`) dengan header `X-Hub-Signature-256`.

### Step 4: Webhook Ingestion & AI Processing
1. Endpoint `/api/v1/webhooks/crawler` memverifikasi HMAC signature.
2. Backend menghitung `content_hash` dan memasukkan `task:llm_extraction` ke antrean Asynq Redis.
3. Gemini LLM mengekstraksi entitas JSON, menjalankan ESG Profile Extractor (jika relevan), dan `text-embedding-004` menghasilkan vektor 1536-dimensi untuk disimpan ke `public_corporate_signals`.

### Step 5: Fallback Task Status Polling (`task:poll_pending_tasks`)
1. Setiap 15 menit, worker memicu `task:poll_pending_tasks`.
2. Worker mengambil log berstatus `DISPATCHED` yang menggantung > 10 menit dari `crawling_logs`.
3. Worker memanggil `GET /api/v1/tasks/{task_id}` pada Scraper Service untuk menyinkronkan status dan memperbarui Circuit Breaker jika terjadi failure.

---

## 5. Environment Variables Configuration

| Variable Name | Example Value | Description |
| :--- | :--- | :--- |
| `SCRAPER_SERVICE_URL` | `https://api-scraper.megasolusindo.com/api/v1/scrape-tasks` | Endpoint WebScraper Service untuk penerimaan tugas scraping |
| `WEBHOOK_URL` | `http://localhost:4000/api/v1/webhooks/crawler` | URL Webhook Callback Backend penerima payload hasil scraping |
| `WEBHOOK_SECRET_KEY` | `super_secret_crawler_key_123` | Secret key pre-shared untuk enkripsi HMAC SHA-256 signature |

---

## 6. Error Handling & Resilience Strategy

1. **Circuit Breaker via `target_id`:**
   Jika Scraper melaporkan error HTTP 404/410 atau error beruntung >= 5 kali, target ditandai `DISABLED_DEAD_LINK` dan `is_active = FALSE`.
2. **Fallback Polling:**
   Jika callback webhook terputus karena downtime/restart CI, poller periodik `GET /api/v1/tasks/{task_id}` akan menyinkronkan status tugas.
3. **Duplicate Content Protection:**
   Nilai `content_hash` diverifikasi untuk mencegah pemrosesan ulang LLM yang tidak perlu.

