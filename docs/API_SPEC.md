# OpenAPI / REST API Specification (API_SPEC.md)

**Product:** Sovera (FundIQ) Core API  
**Base URL:** `https://api.sovera.id/api/v1` (Production) / `http://localhost:4000/api/v1` (Local)  
**Authentication:** Bearer Token (`JWT`) / Multi-Layer Webhook Verification (`X-Hub-Signature-256`, `X-Webhook-Secret`, Bearer, Query Param)  

---

## 1. Authentication & Security Headers

### 1.1 Ingestion Webhook (Crawler to Backend)
Backend mendukung verifikasi otentikasi webhook bertingkat (*multi-layer verification*):
1. **HMAC SHA-256 Signature:** `X-Hub-Signature-256: sha256=<hmac_hex_digest>` (`HMAC_SHA256(raw_body, WEBHOOK_SECRET_KEY)`)
2. **Direct Secret Header:** `X-Webhook-Secret: <WEBHOOK_SECRET_KEY>`
3. **Authorization Bearer:** `Authorization: Bearer <WEBHOOK_SECRET_KEY>`
4. **Query Parameter Token:** `?secret=<WEBHOOK_SECRET_KEY>` atau `?token=<WEBHOOK_SECRET_KEY>`

### 1.2 Frontend / Client Requests
* **Header:** `Authorization: Bearer <jwt_token>`
* **JWT Payload Claims:**
  ```json
  {
    "sub": "usr_991823a",
    "org_id": "org_77123aa",
    "email": "fundraiser@laznas.org",
    "role": "FUNDRAISER",
    "exp": 1788192000
  }
  ```

---

## 2. Ingestion Webhook API

### `POST /webhooks/crawler`

Menerima hasil pemindaian mentah dari WebScraper Service (`PDF_DOCUMENT`, `NEWS_ARTICLE`, `CSR_OPPORTUNITY_SEARCH`, `COMPANY_ENRICHMENT`, `SEARCH_DISCOVERY`) dan memasukkannya ke antrean Asynq Redis.

#### Headers

* `Content-Type: application/json`
* `X-Hub-Signature-256: sha256=d5b9...` (atau `X-Webhook-Secret` / Bearer token)

#### Request Body (Contoh Laporan PDF / News)

```json
{
  "task_id": "job_idx_20260831_001",
  "target_id": "a0ac2304-d0d4-4d14-a5db-ea22966c71c6",
  "status": "COMPLETED",
  "http_status_code": 200,
  "error_message": "",
  "source_type": "PDF_DOCUMENT",
  "source_url": "https://idx.co.id/reports/emiten_csr_2025.pdf",
  "author_or_account": "PT Maju Bersama Tbk",
  "published_date": "2026-08-30",
  "raw_text": "PT Maju Bersama Tbk mengalokasikan dana TJSL sebesar Rp 25 Miliar untuk pilar pendidikan digital dan beasiswa 3T...",
  "markdown_content": "# Laporan TJSL 2025\n\nRealisasi anggaran pilar pendidikan...",
  "execution_time_ms": 1450
}
```

#### Responses

* `202 Accepted` (Payload sukses diterima & di-queue ke Redis Asynq Worker)
```json
{
  "success": true,
  "message": "Payload queued for background ingestion & LLM extraction",
  "job_id": "job_ingest_88291",
  "content_hash": "ff67a9d764d6a2367a187734e697f6a53217db9a21c101d410a113ca871a299d"
}
```

* `200 OK` (Untuk callback berstatus `FAILED`, sistem mencatat error & memperbarui Circuit Breaker)
```json
{
  "success": true,
  "message": "Scraper failure recorded and target health updated",
  "status": "FAILED"
}
```

* `401 Unauthorized`
```json
{
  "success": false,
  "error": "UNAUTHORIZED_WEBHOOK",
  "message": "Webhook authentication failed"
}
```



---

## 3. Corporate Intelligence Feeds API

### `GET /signals`

Mengambil daftar sinyal prospek CSR publik terkurasi dengan filter intent score dan sektor.

#### Query Parameters

* `limit` (optional, default: `20`)
* `offset` (optional, default: `0`)
* `min_intent` (optional, default: `70`)
* `industry` (optional, string)

#### Response `200 OK`

```json
{
  "data": [
    {
      "id": "sig_550e8400-e29b-41d4-a716-446655440000",
      "company_name": "PT Maju Bersama Tbk",
      "industry_sector": "Telecommunication",
      "source_type": "BEI_REPORT",
      "source_url": "https://idx.co.id/reports/emiten_csr_2025.pdf",
      "summary": "Perusahaan menganggarkan TJSL Rp 25 Miliar untuk digitalisasi pendidikan 3T.",
      "extracted_pillar": "Pendidikan",
      "target_regions": ["Jawa Barat", "Nusa Tenggara Timur"],
      "estimated_budget_signal": 25000000000,
      "trigger_event": "Keterbukaan Laba Q2 & Ekspansi CSR",
      "intent_score": 92,
      "published_date": "2026-08-30"
    }
  ],
  "pagination": {
    "total": 142,
    "limit": 20,
    "offset": 0
  }
}

```

---

### `GET /signals/:id/match-programs`

Menjalankan *cosine similarity match* antara embedding sinyal korporasi publik dengan program internal milik tenant (*RLS Enforced*).

#### Response `200 OK`

```json
{
  "signal_id": "sig_550e8400-e29b-41d4-a716-446655440000",
  "company_name": "PT Maju Bersama Tbk",
  "top_matched_programs": [
    {
      "program_id": "prog_11a22b33-...",
      "title": "Program Beasiswa Generasi Digital 3T",
      "asnaf_category": "Fisabilillah / Ibnu Sabil",
      "esg_pillar": "Pendidikan (SDG 4)",
      "similarity_score": 0.8924
    },
    {
      "program_id": "prog_44c55d66-...",
      "title": "Pemberdayaan SMK Vokasi Syariah",
      "asnaf_category": "Fakir / Miskin",
      "esg_pillar": "Ekonomi & Pekerjaan Layak (SDG 8)",
      "similarity_score": 0.7412
    }
  ]
}

```

---

## 4. Institution Programs API (RLS Enforced)

### `GET /programs`

Mengambil daftar portofolio program internal lembaga yang sedang login.

### `POST /programs`

Mendaftarkan program lembaga baru. Backend akan secara otomatis menghasilkan *vector embedding* (1536 dim) dan menyimpannya ke database.

#### Request Body

```json
{
  "title": "Akselerasi Madrasah Berdaya Digital",
  "description": "Program renovasi laboratorium komputer dan pelatihan literasi digital untuk 50 madrasah di kawasan 3T.",
  "asnaf_category": "Fisabilillah",
  "esg_pillar": "Pendidikan (SDG 4)",
  "target_beneficiaries": "5.000 Siswa Madrasah"
}

```

#### Response `201 Created`

```json
{
  "id": "prog_991823-...",
  "title": "Akselerasi Madrasah Berdaya Digital",
  "embedding_generated": true,
  "created_at": "2026-08-31T17:20:00Z"
}

```

---

## 5. Deal Pipeline & Proposal Studio API (RLS Enforced)

### `POST /deals`

Memasukkan prospek korporasi ke pipeline negosiasi lembaga.

#### Request Body

```json
{
  "signal_id": "sig_550e8400-e29b-41d4-a716-446655440000",
  "company_name": "PT Maju Bersama Tbk",
  "target_program_id": "prog_11a22b33-...",
  "estimated_value": 500000000
}

```

#### Response `201 Created`

```json
{
  "deal_id": "deal_7781a-...",
  "deal_stage": "DISCOVERED",
  "company_name": "PT Maju Bersama Tbk"
}

```

---

### `POST /deals/:id/generate-pitch`

Memicu LLM untuk menyusun strategi pendekatan instan (*Executive Ice-Breaker*, *Pitch Deck Outline*, dan *Full Proposal Draft*).

#### Request Body

```json
{
  "tone": "FORMAL_STRATEGIC",
  "custom_notes": "Tekankan keselarasan dengan ekspansi jaringan internet perusahaan di NTT."
}

```

#### Response `200 OK`

```json
{
  "deal_id": "deal_7781a-...",
  "icebreaker": "Yth. Pimpinan TJSL PT Maju Bersama Tbk,\n\nMenyikapi inisiatif luar biasa korporasi dalam penguatan infrastruktur digital di 3T...",
  "pitch_deck_outline": [
    "Slide 1: Urgensi Kesenjangan Digital Pendidikan 3T",
    "Slide 2: Solusi Program Kemitraan Inklusif",
    "Slide 3: Rencana Implementasi & Akuntabilitas Dampak"
  ],
  "proposal_markdown": "# PROPOSAL KEMITRAAN STRATEGIS\n\n## Ringkasan Eksekutif..."
}

```

---

### `POST /deals/:id/export`

Mengekspor draf proposal ke format berkas resmi.

#### Query Parameters

* `format`: `docx` atau `pdf`

#### Response

* `Content-Type: application/vnd.openxmlformats-officedocument.wordprocessingml.document` atau `application/pdf`
* Berkas biner terunduh dengan nama `Proposal_Kemitraan_[Nama_Perusahaan].docx`.

