# Product Requirement Document (PRD)

**Product Name:** Sovera (FundIQ)  
**Category:** Enterprise B2B Fundraising Intelligence & Deal-Preparation Engine  
**Target Market:** Islamic Philanthropy Institutions (LAZNAS/BMT), NGOs, Foundations & Higher Education Endowments  
**Version:** 1.0 (Production / Enterprise Baseline)  
**Author:** Solo Engineering Lead  
**Document Status:** Approved for Implementation  

---

## 1. Executive Summary & Problem Statement

### 1.1 Context & Problem
Lembaga filantropi dan zakat di Indonesia menargetkan penghimpunan dana miliaran rupiah per tahun dari sektor korporasi (Zakat Perniagaan, Program Kemitraan CSR, dan TJSL BUMN). Namun, operasional *fundraising* institusional saat ini menghadapi kendala kritis:
* **Manual Research Burden:** Tim *fundraiser* menghabiskan 60–70% waktunya membaca ratusan halaman dokumen publik (Laporan Keberlanjutan/POJK 51, laporan keuangan BEI, rilis berita) secara manual untuk mencari peluang pendanaan.
* **Program Mismatch:** Terjadi ketidaksesuaian (*mismatch*) antara penawaran program lembaga dengan pilar prioritas ESG/SDGs korporasi target.
* **Slow Deal Turnaround:** Penyusunan proposal kemitraan institusional rata-rata memakan waktu 3–5 hari kerja per korporasi, sehingga sering kehilangan momentum aksi korporasi atau tanggap darurat.

### 1.2 Solution Statement
**Sovera** adalah platform B2B SaaS *intelligence & deal-preparation engine* terpadu yang memantau sinyal pasar korporasi publik secara otomatis, mencocokkannya dengan program lembaga menggunakan *semantic vector matching*, dan menghasilkan draf proposal siap-presentasi berbasis AI dalam hitungan detik dengan jaminan isolasi data multi-tenant tingkat *enterprise* (*Kernel Database Row-Level Security*).

---

## 2. User Personas & Roles

| Role | Target User | Primary Responsibilities & Pain Points | Primary Jobs to be Done (JTBD) |
| :--- | :--- | :--- | :--- |
| **Executive / Director** | Direktur Kemitraan, Kadiv Fundraising B2B | Memantau target pencapaian penghimpunan dana miliaran rupiah, efektivitas tim, dan alokasi pipeline. | Melihat ringkasan *deal pipeline*, konversi kemitraan, dan metrik perolehan dana secara *real-time*. |
| **Fundraiser / AE** | Corporate Fundraiser, Partnership Specialist | Melakukan prospeksi harian, riset korporasi, audiensi dengan pimpinan CSR, dan menyusun proposal. | Mendapatkan *feed* peluang terkurasi, analisis keselarasan program, *ice-breaker*, dan draf proposal instan. |
| **System Admin** | IT/Operations Lead Lembaga | Mengelola akses pengguna internal lembaga dan sinkronisasi data program unggulan. | Mengelola data program lembaga, klasifikasi asnaf, dan integrasi API CRM/ERP. |

---

## 3. System Architecture & High-Level Flow

Sistem mengadopsi pola **3-Tier Decoupled Architecture**:

```
┌─────────────────────────────────────────────────────────────┐
│                 Crawler Service (VPS / Docker)              │
│   - Crawling: Keterbukaan BEI, Berita Bisnis, PDF CSR, Medsos │
└──────────────────────────────┬──────────────────────────────┘
                               │ HTTP Webhook POST (Raw Text/PDF)
                               ▼
┌─────────────────────────────────────────────────────────────┐
│             Dedicated Backend API Service (Go-Fiber)        │
│   - Webhook Ingestion & Signature Verification              │
│   - Async Processing Queue (Asynq / Redis)                  │
│   - LLM Orchestration (Entity Extractor & Text Generator)   │
│   - Embedding Engine & Semantic Matcher                     │
│   - Multi-Tenant RLS Database Connection Manager (pgx/v5)   │
│   - REST API / OpenAPI 3.0 Endpoints                        │
└──────────────────────────────┬──────────────────────────────┘
                               │
            ┌──────────────────┴──────────────────┐
            ▼                                     ▼
┌──────────────────────────────┐    ┌──────────────────────────────┐
│    PostgreSQL + pgvector     │    │       Next.js Frontend       │
│  - Shared Public Signals     │    │  - App Router / React Server │
│  - Isolated Tenant Tables    │    │  - Enterprise Dashboard      │
│  - Enforced RLS Context      │    │  - Proposal Rich Editor      │
└──────────────────────────────┘    └──────────────────────────────┘
```

---

## 4. Database Schema & Multi-Tenancy (Row-Level Security)

Arsitektur database mengisolasi data institusi privat menggunakan PostgreSQL Row-Level Security (RLS) di level kernel database.

### 4.1 SQL DDL & RLS Policies

```sql
-- Aktivasi ekstensi wajib
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "vector";

-- 1. TABEL ORGANISASI (TENANT)
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    subscription_tier VARCHAR(50) DEFAULT 'PRO',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 2. TABEL PENGGUNA
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) UNIQUE NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'FUNDRAISER',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 3. TABEL PROGRAM LEMBAGA (ISOLATED - RLS)
CREATE TABLE institution_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    asnaf_category VARCHAR(100),
    esg_pillar VARCHAR(100),
    target_beneficiaries VARCHAR(255),
    program_embedding vector(1536),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 4. TABEL DEAL PIPELINE (ISOLATED - RLS)
CREATE TABLE deal_pipelines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    company_name VARCHAR(255) NOT NULL,
    deal_stage VARCHAR(50) DEFAULT 'DISCOVERED', -- 'DISCOVERED','RESEARCH','PITCHED','NEGOTIATION','CLOSED_WON','CLOSED_LOST'
    estimated_value NUMERIC(15, 2),
    target_program_id UUID REFERENCES institution_programs(id) ON DELETE SET NULL,
    generated_icebreaker TEXT,
    generated_proposal TEXT,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 5. TABEL SINYAL INTELIJEN PUBLIK (SHARED READ-ONLY)
CREATE TABLE public_corporate_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_name VARCHAR(255) NOT NULL,
    industry_sector VARCHAR(100),
    source_type VARCHAR(50), -- 'BEI_REPORT','NEWS','CSR_PDF','SOCIAL'
    source_url TEXT,
    summary TEXT,
    extracted_pillar VARCHAR(100),
    target_regions TEXT[],
    estimated_budget_signal NUMERIC(15, 2),
    trigger_event VARCHAR(255),
    intent_score INT,
    signal_embedding vector(1536),
    published_date DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- AKTIVASI & ENFORCEMENT ROW-LEVEL SECURITY
ALTER TABLE institution_programs ENABLE ROW LEVEL SECURITY;
ALTER TABLE institution_programs FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_programs ON institution_programs
    FOR ALL
    USING (org_id = NULLIF(current_setting('app.current_org_id', true), '')::UUID)
    WITH CHECK (org_id = NULLIF(current_setting('app.current_org_id', true), '')::UUID);

ALTER TABLE deal_pipelines ENABLE ROW LEVEL SECURITY;
ALTER TABLE deal_pipelines FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_deals ON deal_pipelines
    FOR ALL
    USING (org_id = NULLIF(current_setting('app.current_org_id', true), '')::UUID)
    WITH CHECK (org_id = NULLIF(current_setting('app.current_org_id', true), '')::UUID);

-- INDEXING UNTUK QUERY CEPAT
CREATE INDEX idx_programs_embedding ON institution_programs USING hnsw (program_embedding vector_cosine_ops);
CREATE INDEX idx_signals_embedding ON public_corporate_signals USING hnsw (signal_embedding vector_cosine_ops);
CREATE INDEX idx_deals_org_id ON deal_pipelines(org_id);
```

---

## 5. Functional Requirements (Module Breakdown)

### Module 1: Ingestion & Intelligence Processing (Backend Worker)
* **FR-1.1:** Backend menyediakan endpoint `POST /api/v1/webhooks/crawler` yang menerima payload dari *crawler service*.
* **FR-1.2:** Backend memverifikasi *HMAC SHA256 Signature* pada header untuk menjamin keaslian data *crawler*.
* **FR-1.3:** Queue worker memicu LLM Parsing untuk mengekstrak entitas JSON:
  * `company_name`, `industry_sector`, `csr_pillar_focus`, `target_regions`, `estimated_budget_signal`, `trigger_event`, `intent_score` (1–100).
* **FR-1.4:** Sistem menghasilkan *vector embedding* (1536 dimensi) dari teks ringkasan profil prospek dan menyimpannya ke `public_corporate_signals`.
* **FR-1.5:** Backend secara periodik (cron job / scheduled task) memicu *scraper service* via HTTP POST ke URL yang dikonfigurasi pada `SCRAPER_SERVICE_URL` dengan mengirimkan parameter `WEBHOOK_URL` sebagai URL tujuan balik penerima payload hasil scraping. Variabel lingkungan wajib yang diperlukan: `SCRAPER_SERVICE_URL` dan `WEBHOOK_URL`.

### Module 2: ESG-to-Asnaf Semantic Matching Engine
* **FR-2.1:** Sistem menyediakan antarmuka CRUD bagi admin lembaga untuk mendaftarkan program unggulan beserta deskripsi lengkapnya.
* **FR-2.2:** Setiap program yang disimpan otomatis diubah menjadi *vector embedding* ke kolom `institution_programs.program_embedding`.
* **FR-2.3:** Saat user membuka detail sinyal korporasi publik, sistem menjalankan query *cosine similarity* terisolasi:
  ```sql
  SELECT 
      p.id, 
      p.title, 
      p.asnaf_category, 
      p.esg_pillar,
      (1 - (p.program_embedding <=> s.signal_embedding)) AS similarity_score
  FROM institution_programs p, public_corporate_signals s
  WHERE s.id = :signal_id
  ORDER BY similarity_score DESC
  LIMIT 3;
  ```
* **FR-2.4:** Sistem menampilkan rekomendasi 3 program teratas dengan persentase keselarasan (*Match Score*).

### Module 3: Deal-Preparation & Proposal Generator
* **FR-3.1:** Sistem menyediakan tombol satu-klik *"Generate Pitch Strategy"* pada detail prospek.
* **FR-3.2:** LLM menyusun 3 elemen output siap-pakai:
  1. **Executive Ice-Breaker:** 2–3 paragraf pembuka komunikasi via email/WhatsApp kepada PIC CSR.
  2. **Pitch Deck Outline:** Struktur 5 slide presentasi solusi program kemitraan.
  3. **Full Narrative Proposal:** Draf proposal lengkap yang mengawinkan profil CSR korporasi target dengan program lembaga.
* **FR-3.3:** Pengguna dapat mengedit draf langsung pada antarmuka *rich text editor* dan mengekspor dokumen ke format `.docx` dan `.pdf`.

### Module 4: B2B Deal Pipeline Tracking
* **FR-4.1:** Pengguna dapat memindahkan status prospek dari *Feed* ke *Pipeline* lembaga (tahap: `DISCOVERED` $\rightarrow$ `RESEARCH` $\rightarrow$ `PITCHED` $\rightarrow$ `NEGOTIATION` $\rightarrow$ `CLOSED_WON` / `CLOSED_LOST`).
* **FR-4.2:** Pipeline bersifat eksklusif untuk organisasi pengguna (*RLS Enforced*), mencegah lembaga lain melihat catatan internal atau status negosiasi.

---

## 6. Non-Functional Requirements (NFR)

* **Security & Isolation:** 
  * Wajib mengeksekusi `SET LOCAL app.current_org_id = :org_id` pada setiap sesi transaksi database privat.
  * Zero AI Data Contamination: Data internal proposal dan catatan negosiasi tidak boleh dipakai untuk pelatihan (*training*) model publik.
* **Performance:**
  * Waktu muat halaman dashboard (*First Contentful Paint*) $< 1.2 \text{ detik}$.
  * Latensi API endpoint data feed $< 200 \text{ ms}$.
  * Generasi proposal via LLM dijalankan melalui *streaming* atau *background worker* dengan notifikasi UI *real-time*.
* **Reliability & Availability:**
  * Target Uptime sistem backend $\ge 99.9\%$.
  * Mekanisme *retry* otomatis pada antrean LLM jika terjadi *rate limit* atau *upstream timeout*.

---

## 7. API Contract Specifications (Sample Endpoints)

### 7.1 Webhook Receiver (Internal / Crawler)
* **Endpoint:** `POST /api/v1/webhooks/crawler`
* **Headers:** `X-Hub-Signature-256: <hash_signature>`
* **Payload:**
```json
{
  "task_id": "job_idx_20260831_001",
  "source_type": "BEI_REPORT",
  "source_url": "https://idx.co.id/reports/emiten_csr_2025.pdf",
  "raw_text": "PT Telko Nusantara Tbk mengalokasikan dana TJSL sebesar Rp 45 Miliar dengan fokus digitalisasi pendidikan desa tertinggal...",
  "published_date": "2026-08-30"
}
```
* **Response:** `202 Accepted` `{ "status": "QUEUED", "job_id": "queue_9921" }`

### 7.2 Generate Proposal Endpoint (Frontend to Backend)
* **Endpoint:** `POST /api/v1/deals/:dealId/generate-proposal`
* **Headers:** `Authorization: Bearer <JWT_TOKEN>`
* **Payload:**
```json
{
  "program_id": "8c843e9a-5b12-4c6e-8d99-923e3e012345",
  "tone": "FORMAL_STRATEGIC",
  "target_funding": 250000000
}
```
* **Response:** `200 OK`
```json
{
  "deal_id": "a91b2c3d-...",
  "icebreaker": "Yth. Pimpinan TJSL PT Telko Nusantara...",
  "proposal_markdown": "# Proposal Kemitraan Program Akselerasi Literasi Digital...",
  "generated_at": "2026-08-31T17:00:00Z"
}
```

---

## 8. Implementation Roadmap

```
├── Phase 1: Database Setup & Secure Infrastructure (Week 1–2)
│   ├── Setup PostgreSQL + pgvector
│   ├── Implement RLS Policies & Tenant Context Wrapper
│   └── Setup Dedicated Backend API (Go-Fiber) & Asynq Redis Queue
│
├── Phase 2: Ingestion Pipeline & AI Engine (Week 3–4)
│   ├── Connect Ingestion Webhook to Crawler Service
│   ├── Build LLM Entity Extractor & Embedding Workers
│   └── Build Program Semantic Matcher & Proposal Generator Engine
│
├── Phase 3: Next.js Enterprise Dashboard (Week 5–6)
│   ├── Build Corporate Signal Feed & Filters
│   ├── Build Institution Program Manager
│   ├── Build Proposal Editor & Docx Export Engine
│   └── Implement Kanban Deal Pipeline
│
└── Phase 4: Security Audit & Pilot Deployment (Week 7)
    ├── Multi-tenant Data Leakage Penetration Testing
    ├── Load Testing (Queue Processing)
    └── Pilot Rollout with Target Partner Institution
```

---

## 9. Universal Humanitarian & NGO Expansion (PRD v2.0)

### 9.1 Overview & Business Objectives
Sovera dirancang untuk menyerap anggaran CSR, TJSL BUMN, hibah korporasi, dan Zakat Perusahaan bagi seluruh ekosistem lembaga nirlaba (NGO, Yayasan Kebencanaan, Lingkungan Hidup, Pendidikan, Kesehatan, Pemberdayaan Masyarakat, serta Lembaga Zakat/Wakaf).

1. **Perluasan Total Addressable Market (TAM):** Membuka akses platform bagi ribuan NGO nasional dan internasional yang beroperasi di Indonesia.
2. **Standardisasi Taksonomi Universal:** Mengadopsi pilar SDG dan Klaster Kemanusiaan Standar Nasional/Internasional sebagai jembatan semantik utama antara program lembaga dan fokus korporasi.
3. **Konfigurasi Dinamis Berbasis Tenant (`org_type`):** Antarmuka, taksonomi form, dan prompt AI secara otomatis beradaptasi dengan identitas masing-masing lembaga (`HUMANITARIAN_NGO`, `DISASTER_RELIEF`, `ENVIRONMENT_CONSERVATION`, `HEALTH_EDUCATION`, `ZAKAT_WAQF_INSTITUTION`, `UNIVERSITY_ENDOWMENT`).

### 9.2 Universal Sector & Intervention Framework
Taksonomi program mengadopsi 3 layer:
1. **Primary Cluster:** Disaster & Emergency, Education & Literacy, Health & WASH, Economic Empowerment, Climate & Environment, Social Protection, Fikih Zakat/Wakaf.
2. **SDG Mapping:** SDG 1 hingga SDG 17.
3. **ESG Alignment:** Environmental (E), Social (S), Governance (G).

```sql
-- Penyesuaian Skema Database Multi-Sector
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS org_type VARCHAR(50) DEFAULT 'HUMANITARIAN_NGO';
ALTER TABLE institution_programs
  ADD COLUMN IF NOT EXISTS primary_cluster VARCHAR(100) NOT NULL DEFAULT 'COMMUNITY_DEVELOPMENT',
  ADD COLUMN IF NOT EXISTS target_sdgs TEXT[] DEFAULT '{}',
  ADD COLUMN IF NOT EXISTS esg_pillar VARCHAR(20) DEFAULT 'SOCIAL',
  ALTER COLUMN asnaf_category DROP NOT NULL;
```