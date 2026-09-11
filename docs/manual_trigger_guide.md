# Panduan Pemicuan Manual (Manual Triggering Guide)
## System: Sovera Corporate CSR Intelligence & Scraper Pipeline

Dokumen ini berisi panduan lengkap untuk melakukan pemicuan manual (*manual trigger*) terhadap berbagai komponen worker, scraping batch, enrichment entitas korporasi, dan verifikasi kesehatan sistem pada platform **Sovera CSR Intelligence**.

---

## 1. Pemicuan Worker Enrichment Korporasi (Company Website & Metadata Enrichment)

Worker ini memindai perusahaan di tabel `company.companies` yang belum memiliki alamat website/telepon resmi, melakukan *direct matching* emiten Tbk (IDX), dan melempar *Discovery Task* (`POST /api/v1/discovery`) ke WebScraper.

### Cara 1: Menggunakan CLI Tool (Rekomendasi Utama)
Jalankan perintah berikut di direktori `sovera-csr-api`:

```bash
cd /Users/mluludk/Works/sovera-csr-api
go run cmd/tools/enrich_companies/main.go
```

### Cara 2: Menggunakan Asynq Redis Queue (Background Worker)
Kirimkan task `task:enrich_missing_websites` ke antrean `company_enrichment` menggunakan Asynq CLI atau skrip Go/Python.

---

## 2. Pemicuan Scraping Jobs Batch (Scrape Dispatcher)

Mememicu pengiriman tugas scraping untuk seluruh target aktif di `public.crawling_targets` yang sudah memasuki jadwal running (`next_run_at <= NOW()`).

### Cara 1: Melalui API Endpoint Admin
Kirim request `POST` ke backend API (Port 4000):

```bash
curl -X POST http://localhost:4000/api/v1/scraping-jobs/trigger \
  -H "Authorization: Bearer super_secret_jwt_key_enterprise" \
  -H "Content-Type: application/json"
```

### Cara 2: Pemicuan Target Spesifik via WebScraper API (Direct Test)
Gunakan curl atau Python untuk menguji 1 target tertentu ke `web-scraper` (Port 8088):

```bash
curl -X POST http://localhost:8088/api/v1/scrape-tasks \
  -H "Authorization: Bearer change-me" \
  -H "Content-Type: application/json" \
  -d '{
    "task_id": "test_manual_trigger_001",
    "target_id": "d70472e2-c740-4262-94fd-0d3e243c371d",
    "client_origin": "sovera_core_api",
    "source_type": "NEWS_RSS",
    "target_url": "https://news.google.com/rss/search?q=CSR+Pertamina",
    "callback_url": "http://host.docker.internal:4000/api/v1/webhooks/crawler?secret=super_secret_crawler_key_123"
  }'
```

---

## 3. Pemicuan Worker Pemeriksa Kesehatan Link (URL Health Check)

Pemeriksaan kesehatan tautan (*URL Health Check*) mengecek ketersediaan HTTP (HEAD/GET 2xx/3xx) pada target URL untuk mengisolasi link mati (*Circuit Breaker*).

### Melalui CLI Tool / Worker Go
```bash
cd /Users/mluludk/Works/sovera-csr-api
go run cmd/tools/health_check_targets/main.go
```

---

## 4. Pemicuan Ekstraksi AI Feed Intelligence (Gemini LLM Extraction)

Untuk memproses ulang artikel berita yang telah selesai discrape (`COMPLETED`) agar diekstrak entitas korporasi, anggaran CSR, dan pilar kegiatannya oleh Gemini AI:

```bash
cd /Users/mluludk/Works/sovera-csr-api
go run cmd/tools/reprocess_signals/main.go
```

---

## 5. Pemeriksaan Kesehatan Service (Health Check & Logs Verification)

### Memeriksa Status Seluruh Kontainer Docker
```bash
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
```

### Memeriksa Log Webhook Receiver Backend
```bash
docker logs --tail 50 sovera_core_api
```

### Memeriksa Log Scraper Engine & Worker
```bash
docker logs --tail 50 web-scraper-api-1
docker logs --tail 50 web-scraper-webhook-worker-1
```

### Memeriksa Jumlah Sinyal Korporasi Hasil Pemicuan di Database
```bash
python3 -c "
import psycopg2
conn = psycopg2.connect('postgres://sovera:sover4@10.10.29.177:5432/sovera?sslmode=disable')
cur = conn.cursor()
cur.execute('SELECT company_name, summary, intent_score, created_at FROM intelligence.company_signals ORDER BY created_at DESC LIMIT 5;')
for r in cur.fetchall():
    print(r)
"
```

---

## 6. Syarat Konfigurasi `.env` Penting

Pastikan file `/Users/mluludk/Works/sovera-csr-api/.env` memiliki URL antarkontainer yang tepat:

```env
SCRAPER_SERVICE_URL=http://host.docker.internal:8088/api/v1/scrape-tasks
WEBHOOK_URL=http://host.docker.internal:4000/api/v1/webhooks/crawler?secret=super_secret_crawler_key_123
```
