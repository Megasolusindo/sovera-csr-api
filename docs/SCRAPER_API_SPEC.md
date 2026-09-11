# API Contract Specification (WebScraper Service v2)

Dokumen ini berisi spesifikasi lengkap API Contract untuk **WebScraper Service (v1 & v2)**, termasuk penambahan dukungan pengayaan profil perusahaan (*Company Enrichment*) dan pencarian peluang CSR aktif (*CSR Opportunity Feed*). Semua request yang dilindungi wajib menyertakan header otentikasi.

---

## 🔑 Authentication & Standards

### Header Otentikasi API
```http
Authorization: Bearer <API_AUTH_KEY>
Content-Type: application/json
```

### Header Otentikasi Webhook Callback (SHA-256 HMAC)
Crawler service **wajib** menyertakan header tanda tangan digital pada setiap pengiriman callback ke backend Sovera:
```http
Content-Type: application/json
X-Hub-Signature-256: sha256=<hmac_hex_digest>
```

### Standard Error Response (HTTP 4xx / 5xx)
```json
{
  "error": "invalid_request",
  "message": "penjelasan detail kesalahan"
}
```

---

## 🛠️ Endpoints API

### 1. Universal Ingestion Task (`POST /api/v1/scrape-tasks`)

Membuat pekerjaan pemindaian dokumen PDF (*Sustainability Report*), portal berita, rilis bursa (IDX/RSS), pencarian peluang CSR aktif, atau pengayaan profil entitas korporasi.

* **Method & Path:** `POST /api/v1/scrape-tasks`
* **Content-Type:** `application/json`

#### **Request Body**
```json
{
  "task_id": "job_csr_idx_991823",
  "target_id": "a0ac2304-d0d4-4d14-a5db-ea22966c71c6",
  "client_origin": "sovera_b2b_engine",
  "source_type": "COMPANY_ENRICHMENT",
  "target_url": "https://telkom.co.id/csr",
  "callback_url": "https://api.sovera.id/api/v1/webhooks/crawler",
  "config": {
    "render_js": false,
    "bypass_anti_bot": true,
    "max_pages": 50
  }
}
```

| Parameter | Tipe | Wajib | Keterangan |
| :--- | :--- | :---: | :--- |
| `task_id` | `string` | Ya | ID Unik pekerjaan dari sistem pemanggil |
| `target_id` | `string` | Tidak | UUID Target dari `crawling_targets` (untuk pelacakan Circuit Breaker) |
| `client_origin` | `string` | Ya | Identifikasi asal sistem pemanggil (`sovera_b2b_engine`) |
| `source_type` | `string` | Ya | Pilihan: `PDF_DOCUMENT`, `NEWS_ARTICLE`, `NEWS_RSS`, `BUMN_PORTAL`, `GRANTS_PORTAL`, **`CSR_OPPORTUNITY_SEARCH`**, **`COMPANY_ENRICHMENT`** |
| `target_url` | `string` | Ya | URL target dokumen PDF / Artikel / Portal Web / Halaman Kontak CSR Perusahaan |
| `callback_url` | `string` | Ya | URL Webhook tunggal (*single callback*) penerima hasil callback di backend |
| `config.render_js` | `boolean` | Tidak | Batasi render Headless JS Browser (Default: `false`) |
| `config.max_pages` | `integer` | Tidak | Batas maks halaman PDF (Default: `100`) |

#### **Success Response (`HTTP 202 Accepted`)**
```json
{
  "task_id": "job_csr_idx_991823",
  "status": "ACCEPTED"
}
```

---

### 2. Single Webhook Callback Delivery Specification (`POST` to `callback_url`)

Crawler service wajib mengirimkan hasil pemindaian ke **`callback_url` yang tunggal** dengan header otentikasi `X-Hub-Signature-256`. Atribut `source_type` menentukan kategori data yang diproses backend.

#### **A. Payload Callback: Laporan PDF & Berita CSR (`source_type: "PDF_DOCUMENT"` / `"NEWS_ARTICLE"`)**
```json
{
  "task_id": "job_csr_idx_991823",
  "target_id": "a0ac2304-d0d4-4d14-a5db-ea22966c71c6",
  "status": "COMPLETED",
  "http_status_code": 200,
  "error_message": "",
  "source_type": "PDF_DOCUMENT",
  "source_url": "https://example-corpo.com/reports/sustainability-report-2025.pdf",
  "author_or_account": "PT Maju Sejahtera Tbk",
  "published_date": "2026-04-12T00:00:00Z",
  "raw_text": "PT Maju Sejahtera Tbk mengalokasikan dana TJSL sebesar Rp 25 Miliar...",
  "markdown_content": "# Laporan Keberlanjutan 2025\n\nRealisasi pilar pendidikan...",
  "execution_time_ms": 1565
}
```

#### **B. Payload Callback: Peluang Hibah CSR Aktif (`source_type: "CSR_OPPORTUNITY_SEARCH"`)**
```json
{
  "task_id": "job_opp_idx_881204",
  "target_id": "b1bd3405-e1e5-5e25-b6fc-fb33077d82d7",
  "status": "COMPLETED",
  "http_status_code": 200,
  "error_message": "",
  "source_type": "CSR_OPPORTUNITY_SEARCH",
  "source_url": "https://yayasan.djarumfoundation.org/call-for-proposals-2026",
  "author_or_account": "Djarum Foundation",
  "published_date": "2026-09-01T00:00:00Z",
  "raw_text": "Djarum Foundation membuka pendaftaran proposal hibah program konservasi air...",
  "markdown_content": "# Panggilan Proposal Hibah Konservasi Air 2026\n\nSyarat dan ketentuan...",
  "execution_time_ms": 1820
}
```

#### **C. Payload Callback: Pengayaan Profil Perusahaan (`source_type: "COMPANY_ENRICHMENT"`)**
```json
{
  "task_id": "job_enrich_telkom_3391",
  "target_id": "c2ce4516-f2f6-6f36-c7ad-gc44188e93e8",
  "status": "COMPLETED",
  "http_status_code": 200,
  "error_message": "",
  "source_type": "COMPANY_ENRICHMENT",
  "source_url": "https://telkom.co.id/tjsl-contact",
  "author_or_account": "PT Telkom Indonesia (Persero) Tbk",
  "published_date": "2026-09-03T00:00:00Z",
  "raw_text": "Kontak Departemen TJSL Telkom: tjsl@telkom.co.id. Kantor Pusat: Jl. Japati No. 1 Bandung...",
  "markdown_content": "## Profil TJSL & ESG Telkom Indonesia\n\n- Email Public CSR: tjsl@telkom.co.id\n- Alamat HQ: Bandung\n- Fokus: Digitalisasi Pendidikan & EBT",
  "execution_time_ms": 2100
}
```

#### **D. Payload Callback Gagal / Dead Link (`status: "FAILED"`)**
```json
{
  "task_id": "job_csr_idx_991823",
  "target_id": "a0ac2304-d0d4-4d14-a5db-ea22966c71c6",
  "status": "FAILED",
  "http_status_code": 404,
  "error_message": "HTTP 404 Not Found - Target URL does not exist or has been removed",
  "source_type": "PDF_DOCUMENT",
  "source_url": "https://example-corpo.com/reports/sustainability-report-2025.pdf",
  "execution_time_ms": 450
}
```

---

### 3. Task Status Polling API (`GET /api/v1/tasks/{task_id}`)

Mengambil status dan detail pekerjaan pemindaian berbasis `task_id`.

* **Method & Path:** `GET /api/v1/tasks/{task_id}`

#### **Success Response (`HTTP 200 OK`)**
```json
{
  "task_id": "job_csr_idx_991823",
  "target_id": "a0ac2304-d0d4-4d14-a5db-ea22966c71c6",
  "client_origin": "sovera_b2b_engine",
  "source_type": "COMPANY_ENRICHMENT",
  "target_url": "https://telkom.co.id/csr",
  "callback_url": "https://api.sovera.id/api/v1/webhooks/crawler",
  "status": "COMPLETED",
  "http_status_code": 200,
  "execution_time_ms": 1565,
  "content_hash": "ff67a9d764d6a2367a187734e697f6a53217db9a21c101d410a113ca871a299d",
  "created_at": "2026-09-03T16:20:00Z",
  "updated_at": "2026-09-03T16:20:01Z"
}
```

---

### 4. Health & Readiness Check

Mengecek kesehatan server WebScraper API.

* **Method & Path:** `GET /health` atau `GET /ready`

#### **Success Response (`HTTP 200 OK`)**
```json
{
  "status": "ok"
}
```

Berikut adalah penambahan endpoint baru yang belum ada di dokumen spesifikasi (`Discovery`, `Crawling Job`, `Batch Change Inspection`, dan `Document Head Check`) dengan format standar yang selaras:

---

### 5. Search & Discovery Task (`POST /api/v1/discovery`)

Digunakan oleh Company Intelligence untuk menemukan domain, sub-halaman, atau kandidat perusahaan baru melalui mesin pencari atau direktori eksternal secara asinkron.

* **Method & Path:** `POST /api/v1/discovery`
* **Content-Type:** `application/json`

#### **Request Body**

```json
{
  "task_id": "disc_job_881920",
  "client_origin": "sovera_b2b_engine",
  "query": "perusahaan CSR Jawa Barat TJSL",
  "engine": "google",
  "limit": 20,
  "callback_url": "https://api.sovera.id/api/v1/webhooks/crawler"
}

```

| Parameter | Tipe | Wajib | Keterangan |
| --- | --- | --- | --- |
| `task_id` | `string` | Ya | ID unik tugas dari sistem pemanggil |
| `client_origin` | `string` | Ya | Identifikasi caller (`sovera_b2b_engine`) |
| `query` | `string` | Ya | Kata kunci pencarian web mentah |
| `engine` | `string` | Tidak | Pilihan: `google`, `bing`, `duckduckgo` (Default: `google`) |
| `limit` | `integer` | Tidak | Batas maksimal kandidat URL yang diambil (Default: `10`, Maks: `50`) |
| `callback_url` | `string` | Ya | URL Webhook penerima hasil discovery mentah |

#### **Success Response (`HTTP 202 Accepted`)**

```json
{
  "task_id": "disc_job_881920",
  "status": "ACCEPTED"
}

```

#### **Payload Callback Discovery (`source_type: "SEARCH_DISCOVERY"`)**

Dikirimkan ke `callback_url` setelah scraping mesin pencari selesai:

```json
{
  "task_id": "disc_job_881920",
  "status": "COMPLETED",
  "http_status_code": 200,
  "error_message": "",
  "source_type": "SEARCH_DISCOVERY",
  "query": "perusahaan CSR Jawa Barat TJSL",
  "discovered_items": [
    {
      "title": "Program TJSL & Keberlanjutan - PT ABC Tbk",
      "url": "https://abc.co.id/tjsl",
      "snippet": "Inisiatif pemberdayaan masyarakat dan lingkungan Jawa Barat oleh PT ABC...",
      "rank": 1
    },
    {
      "title": "Laporan CSR - PT Migas Perkasa",
      "url": "https://migasperkasa.com/csr",
      "snippet": "Program keberlanjutan tahun 2025/2026 difokuskan pada konservasi air...",
      "rank": 2
    }
  ],
  "execution_time_ms": 2340
}

```

---

### 6. Recursive Crawl Job (`POST /api/v1/crawl-jobs`)

Digunakan untuk menjelajahi tautan internal (*link traversal*) pada domain perusahaan target secara terkontrol tanpa melibatkan analisis konten.

* **Method & Path:** `POST /api/v1/crawl-jobs`
* **Content-Type:** `application/json`

#### **Request Body**

```json
{
  "task_id": "crawl_site_abc_001",
  "target_id": "a0ac2304-d0d4-4d14-a5db-ea22966c71c6",
  "client_origin": "sovera_b2b_engine",
  "base_url": "https://abc.co.id",
  "url_patterns": ["*/csr*", "*/sustainability*", "*/tjsl*", "*/esg*"],
  "max_depth": 2,
  "max_pages": 25,
  "callback_url": "https://api.sovera.id/api/v1/webhooks/crawler"
}

```

| Parameter | Tipe | Wajib | Keterangan |
| --- | --- | --- | --- |
| `task_id` | `string` | Ya | ID unik tugas |
| `target_id` | `string` | Tidak | UUID Target untuk tracking circuit breaker |
| `base_url` | `string` | Ya | Domain / root URL perusahaan yang akan dijelajahi |
| `url_patterns` | `array[string]` | Tidak | Filter whitelist regex/glob URL yang wajib diikuti |
| `max_depth` | `integer` | Tidak | Batas kedalaman penelusuran tautan (Default: `2`, Maks: `4`) |
| `max_pages` | `integer` | Tidak | Batas kuota total URL yang diekstrak (Default: `20`, Maks: `100`) |
| `callback_url` | `string` | Ya | URL callback Webhook hasil penelusuran |

#### **Success Response (`HTTP 202 Accepted`)**

```json
{
  "task_id": "crawl_site_abc_001",
  "status": "ACCEPTED"
}

```

---

### 7. Batch Change Inspection (`POST /api/v1/inspect-batch`)

Endpoint sinkron ringan (*lightweight*) untuk mendukung Cron Company Intelligence. Mengecek apakah konten halaman web telah berubah dengan membandingkan hash tanpa mengunduh keseluruhan body respon ke database pemanggil.

* **Method & Path:** `POST /api/v1/inspect-batch`
* **Content-Type:** `application/json`

#### **Request Body**

```json
{
  "items": [
    {
      "reference_id": "comp_target_01",
      "target_url": "https://telkom.co.id/csr",
      "known_hash": "ff67a9d764d6a2367a187734e697f6a53217db9a21c101d410a113ca871a299d"
    },
    {
      "reference_id": "comp_target_02",
      "target_url": "https://djarumfoundation.org/updates",
      "known_hash": "a1b2c3d4e5f6g7h812345678abcdef0123456789abcdef0123456789abcdef01"
    }
  ]
}

```

| Parameter | Tipe | Wajib | Keterangan |
| --- | --- | --- | --- |
| `items` | `array[object]` | Ya | Maksimal 50 item per batch request |
| `items[].reference_id` | `string` | Ya | ID referensi entitas dari Company Intelligence |
| `items[].target_url` | `string` | Ya | URL target yang diverifikasi |
| `items[].known_hash` | `string` | Ya | Hash teks bersih terakhir yang tersimpan di CI |

#### **Success Response (`HTTP 200 OK`)**

```json
{
  "checked_at": "2026-09-04T22:30:00Z",
  "total_checked": 2,
  "results": [
    {
      "reference_id": "comp_target_01",
      "target_url": "https://telkom.co.id/csr",
      "http_status_code": 200,
      "is_modified": false,
      "current_hash": "ff67a9d764d6a2367a187734e697f6a53217db9a21c101d410a113ca871a299d"
    },
    {
      "reference_id": "comp_target_02",
      "target_url": "https://djarumfoundation.org/updates",
      "http_status_code": 200,
      "is_modified": true,
      "current_hash": "9c8b7a6f5e4d3c2b1a0f9e8d7c6b5a4f3e2d1c0b9a8f7e6d5c4b3a2f1e0d9c8b"
    }
  ]
}

```

---

### 8. Document Metadata & Header Check (`POST /api/v1/documents/inspect`)

Pemeriksaan berkas statis (PDF/laporan) berbasis HTTP `HEAD` untuk memvalidasi perubahan ukuran berkas atau ETag sebelum melakukan download penuh.

* **Method & Path:** `POST /api/v1/documents/inspect`
* **Content-Type:** `application/json`

#### **Request Body**

```json
{
  "file_url": "https://telkom.co.id/sustainability-report-2025.pdf",
  "known_etag": "\"5d-5c697b0\"",
  "known_content_length": 14205810
}

```

| Parameter | Tipe | Wajib | Keterangan |
| --- | --- | --- | --- |
| `file_url` | `string` | Ya | Direct link file dokumen PDF |
| `known_etag` | `string` | Tidak | ETag berkas yang tercatat sebelumnya |
| `known_content_length` | `integer` | Tidak | Ukuran berkas (bytes) yang tercatat sebelumnya |

#### **Success Response (`HTTP 200 OK`)**

```json
{
  "file_url": "https://telkom.co.id/sustainability-report-2025.pdf",
  "http_status_code": 200,
  "is_modified": false,
  "etag": "\"5d-5c697b0\"",
  "content_length": 14205810,
  "content_type": "application/pdf",
  "last_modified": "2026-04-10T08:12:00Z"
}

```