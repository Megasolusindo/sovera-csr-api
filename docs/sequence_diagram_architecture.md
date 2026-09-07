# Sequence Diagram Arsitektur Komunikasi & Orkestrasi Sovera (v2)

Dokumen ini menjelaskan diagram urutan (*sequence diagram*) keseluruhan alur komunikasi antara **Sovera Core API (Brain & Orchestrator)**, **WebScraper Service (Execution Muscle)**, **Redis (Asynq Worker Queue)**, dan **Gemini LLM Engine**.

---

## 📐 Overall Sequence Diagram (End-to-End Flow)

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

    %% -----------------------------------------------------------------
    %% TAHAP 1: TASK DISPATCHING
    %% -----------------------------------------------------------------
    rect rgb(240, 248, 255)
    note over Cron, DAS: TAHAP 1: Task Dispatching (Periodic Scheduling)
    Cron->>Disp: Tick Trigger (task:dispatch_crawling)
    Disp->>DB: GetDueTargets(limit=20)
    DB-->>Disp: Return list of CrawlingTarget due for scrape
    loop Setiap Target Aktif
        Disp->>DB: CreateLog(task_id, status='DISPATCHED')
        Disp->>DAS: POST /api/v1/scrape-tasks (Bearer Token + Target Config)
        DAS-->>Disp: HTTP 202 Accepted {"task_id": "...", "status": "ACCEPTED"}
        Disp->>DB: UpdateTargetNextRun(target_id, check_interval_hours)
    end
    end

    %% -----------------------------------------------------------------
    %% TAHAP 2: ASYNCHRONOUS SCRAPING & WEBHOOK CALLBACK
    %% -----------------------------------------------------------------
    rect rgb(245, 245, 220)
    note over DAS, WH: TAHAP 2: Pemrosesan Scraper & Delivery Webhook Callback
    note over DAS: Scraper rotasi proxy,<br/>render JS, bypass bot,<br/>ekstraksi HTML/PDF/Discovery
    DAS->>WH: POST /api/v1/webhooks/crawler (Header: X-Hub-Signature-256)
    
    rect rgb(230, 230, 250)
    note over WH: Multi-layer Auth & Verification
    WH->>WH: Verify HMAC SHA-256 Signature
    alt Callback Status FAILED / HTTP >= 400
        WH->>DB: UpdateLogStatus(task_id, status='FAILED')
        WH->>DB: RecordFailure(target_id) [Circuit Breaker Check]
        WH-->>DAS: HTTP 200 OK (Konfirmasi penerimaan error)
    else Callback Status COMPLETED
        WH->>WH: SelectBestContent & GenerateContentHash (SHA-256)
        WH->>DB: UpdateLogStatus(task_id, status='COMPLETED', content_hash)
        WH->>DB: RecordSuccess(target_id) [Reset consecutive failures]
        WH->>Queue: Enqueue task:llm_extraction (Payload JSON)
        WH-->>DAS: HTTP 202 Accepted (Payload queued for ingestion)
    end
    end
    end

    %% -----------------------------------------------------------------
    %% TAHAP 3: AI LLM PROCESSING & VECTOR EMBEDDING
    %% -----------------------------------------------------------------
    rect rgb(240, 255, 240)
    note over Queue, AI: TAHAP 3: Asynchronous LLM Signal & ESG Processing
    Queue->>AI: Dequeue task:llm_extraction
    AI->>AI: Gemini LLM Signal Extraction (ExtractCorporateSignal)
    AI->>AI: Entity Resolution (ResolveCompany & Match Master)
    
    opt If SourceType in (PDF_DOCUMENT, COMPANY_ENRICHMENT, CSR_OPPORTUNITY_SEARCH)
        AI->>AI: ProcessESGExtraction (ESG Profile Extractor)
        AI->>DB: Save/Update ESG Profile
    end

    AI->>AI: GenerateEmbedding (text-embedding-004, 1536-dim)
    AI->>DB: SaveSignal (Persist corporate signal + vector embedding)
    end

    %% -----------------------------------------------------------------
    %% TAHAP 4: FALLBACK TASK POLLING MECHANISM
    %% -----------------------------------------------------------------
    rect rgb(255, 240, 245)
    note over Cron, DAS: TAHAP 4: Fallback Task Polling (Recovery Mechanism)
    Cron->>Disp: Tick Trigger (task:poll_pending_tasks - every 15 mins)
    Disp->>DB: GetPendingLogs(pendingMinutes=10)
    DB-->>Disp: Return list of stuck CrawlingLog ('DISPATCHED' > 10m)
    loop Setiap Pending Log
        Disp->>DAS: GET /api/v1/tasks/{task_id}
        DAS-->>Disp: Return TaskStatusResponse (COMPLETED / FAILED / PENDING)
        alt Status == COMPLETED
            Disp->>DB: UpdateLogStatus(COMPLETED) & RecordSuccess(target_id)
        else Status == FAILED
            Disp->>DB: UpdateLogStatus(FAILED) & RecordFailure(target_id)
        end
    end
    end
```

---

## 📌 Penjelasan Masing-Masing Tahap

### 1. **Tahap 1: Task Dispatching (Periodic Scheduling)**
* **Komponen:** Asynq Cron Scheduler, `CrawlerRepository`, `CrawlerDispatcherHandler`, WebScraper Service (DAS).
* **Alur:**
  1. Cron scheduler memicu tugas `task:dispatch_crawling` secara periodik.
  2. Dispatcher mengambil daftar situs target yang jatuh tempo dari tabel `crawling_targets` di PostgreSQL.
  3. Dispatcher mencatat log berstatus `DISPATCHED` ke tabel `crawling_logs`.
  4. Dispatcher mengirimkan request HTTP `POST /api/v1/scrape-tasks` (atau `/discovery`, `/crawl-jobs`) ke Scraper Service.
  5. Scraper Service merespons dengan `HTTP 202 Accepted` dan `task_id` tanpa menahan koneksi (*non-blocking*).

### 2. **Tahap 2: Pemrosesan Scraper & Delivery Webhook Callback**
* **Komponen:** WebScraper Service, Middleware HMAC, `WebhookHandler`, PostgreSQL, Redis Asynq.
* **Alur:**
  1. Scraper mengeksekusi ekstraksi web/PDF/discovery di background secara asinkron.
  2. Setelah selesai, Scraper menembak callback `POST /api/v1/webhooks/crawler` dengan header tanda tangan `X-Hub-Signature-256`.
  3. Middleware memverifikasi keaslian HMAC SHA-256.
  4. Jika pemindaian **Gagal**: Backend mencatat log `FAILED` dan memperbarui `consecutive_failures` pada target (Circuit Breaker).
  5. Jika pemindaian **Sukses**: Backend menghitung `content_hash`, memperbarui log `COMPLETED`, me-reset health target, dan memasukkan job ke antrean Redis `task:llm_extraction`.

### 3. **Tahap 3: Asynchronous LLM Processing & Vector Embedding**
* **Komponen:** Redis Queue, `ExtractionWorker`, Gemini Service, Entity Resolver, ESG Extractor, PostgreSQL.
* **Alur:**
  1. Worker mengambil job `task:llm_extraction` dari Redis.
  2. Gemini LLM mengekstraksi entitas bisnis, skor minat CSR (*intent score*), dan ringkasan sinyal korporasi.
  3. Entity Resolver mencocokkan nama perusahaan ke tabel master entitas (*Canonical Company Master*).
  4. Jika tipe data berupa laporan keberlanjutan, pengayaan profil (`COMPANY_ENRICHMENT`), atau hibah (`CSR_OPPORTUNITY_SEARCH`), ESG Extractor dijalankan.
  5. Model Gemini `text-embedding-004` menghasilkan vektor 1536 dimensi yang disimpan bersama sinyal ke PostgreSQL (`public_corporate_signals`).

### 4. **Tahap 4: Fallback Task Polling (Ketahanan & Recovery)**
* **Komponen:** Asynq Cron, `CrawlerRepository`, `Dispatcher`, WebScraper API.
* **Alur:**
  1. Setiap 15 menit, worker memicu `task:poll_pending_tasks`.
  2. Worker mencari log pemindaian yang menggantung di status `DISPATCHED` lebih dari 10 menit (kemungkinan callback terlepas saat CI restart/deployment).
  3. Worker menembak endpoint `GET /api/v1/tasks/{task_id}` pada Scraper Service.
  4. Jika task di Scraper sudah selesai atau gagal, backend menyinkronkan status log & health target secara otomatis.
