# CSRmatics — Comprehensive System Architecture & Integration Design Document
> **Version:** 2.0  
> **Target Platform:** CSRmatics (Two-Sided CSR Intelligence & Partnership Platform)  
> **Author:** Antigravity Engineering  
> **Synthesis of:** `ANALISYS_CHATGPT.md`, `ANALISYS_2_SIDED_CHATGPT.md`, `LANDINGPAGE_CHATGPT.md`, `ANALISYS_GEMINI.md`

---

## 1. Executive Summary & Platform Positioning

**CSRmatics** bertransformasi dari sekadar direktori perusahaan CSR statis menjadi **Two-Sided CSR Intelligence & Partnership Platform**. 

Platform ini menjembatani dua pemangku kepentingan utama:
1. **Lembaga Kemanusiaan / NGO / Yayasan / LAZ (`organization`)**: Membutuhkan kecerdasan data alokasi CSR korporasi, pencocokan program berbasis AI & Fiqh Asnaf, serta pengelolaan prospek partnership secara terstruktur.
2. **Perusahaan / TJSL / CSR / ESG Unit (`corporate`)**: Membutuhkan penyaluran dana CSR yang tepat sasaran, pencarian NGO terverifikasi, publikasi *CSR Opportunity*, dan pengelolaan proposal masuk secara transparan.

```text
                               ┌─────────────────────────────────────────┐
                               │           CSRmatics Platform            │
                               │   (Single Multi-Tenant System Architecture)│
                               └────────────────────┬────────────────────┘
                                                    │
                      ┌─────────────────────────────┴─────────────────────────────┐
                      ▼                                                           ▼
        ┌───────────────────────────┐                               ┌───────────────────────────┐
        │    Lembaga Kemanusiaan    │                               │     Perusahaan CSR / TJSL │
        │       (Organization)      │                               │        (Corporate)        │
        └─────────────┬─────────────┘                               └─────────────┬─────────────┘
                      │                                                           │
        - CSR Intelligence & Research                               - Claim Profile & Verification
        - AI Program Matching (1536d)                               - Publish CSR Opportunity
        - Fiqh Asnaf & SDG Alignment                                - Receive & Review Proposals
        - Submit & Track Proposals                                  - Discover Verified NGOs
```

---

## 2. Arsitektur Multi-Tenant & Model Peran (RBAC)

Platform dibangun di atas **satu basis kode (Single-Platform)** dengan pengisolasian data bertingkat (*Row-Level Isolation*) menggunakan 3 peran utama (`tenant.type`):

```text
                       ┌──────────────────────────────────────┐
                       │          Database Schema             │
                       └──────────────────┬───────────────────┘
                                          │
            ┌─────────────────────────────┼─────────────────────────────┐
            ▼                             ▼                             ▼
  ┌──────────────────┐          ┌──────────────────┐          ┌──────────────────┐
  │ Shared Data      │          │ Org Tenant Data  │          │ Corporate Data   │
  │ (Public Domain)  │          │ (Isolated)       │          │ (Isolated)       │
  ├──────────────────┤          ├──────────────────┤          ├──────────────────┤
  │ - Companies      │          │ - Programs       │          │ - Private Drafts │
  │ - CSR History    │          │ - Proposals      │          │ - Opportunities  │
  │ - CSR News       │          │ - Saved Targets  │          │ - Review Notes   │
  │ - SDGs / Asnaf   │          │ - CRM Status     │          │ - Claim Status   │
  └──────────────────┘          └──────────────────┘          └──────────────────┘
```

---

## 3. Fitur Utama & Alur Kerja (Workflows)

### 3.1. Company & CSR Intelligence (Core Engine)
- **Database Perusahaan Verified**: Menyimpan data 3.000+ perusahaan (BUMN, Tbk, Swasta, Perbankan) lengkap dengan status verifikasi domain live HTTP.
- **CSR History & Signal Tracker**: Agregasi riwayat program CSR tahunan, berita terkini, dan pilar prioritas (Pendidikan, Kesehatan, Lingkungan, Pemberdayaan UMKM).

### 3.2. Two-Way AI Matching Engine (1536d Vector + Gemini 1.5)
Mencocokkan program NGO dengan prioritas CSR perusahaan secara otomatis dengan skor persentase (misal **94% Match**):
- **Kriteria Matched**: Sektor industri ↔ Kategori program, Lokasi operasi ↔ Target wilayah, Keselarasan 17 SDGs ↔ 8 Fiqh Asnaf Zakat.

### 3.3. Sistem Verifikasi Klaim Perusahaan (Company Claim - Tiered Verification)
Memecahkan masalah *cold-start (chicken-and-egg)* secara otomatis & aman:

```text
User klik "Claim Company" di /companies/pt-abc
                   │
                   ├──> [Tier 1: Otomatis via Work Email Domain]
                   │       └ Email match @pertamina.com -> Magic Link -> Claim Approved (Instant)
                   │
                   └──> [Tier 2: Manual via Upload Dokumen / Surat Tugas]
                           └ Upload ID Card/SK -> Status PENDING -> Webhook Bot Telegram Admin -> 1-Click Approve
```

**State Machine Klaim Perusahaan:**
$$\text{UNCLAIMED} \xrightarrow{\text{Submit Claim}} \text{CLAIM\_PENDING} \xrightarrow{\text{Domain / Admin Approve}} \text{VERIFIED\_CLAIMED} \xrightarrow{\text{Dispute}} \text{SUSPENDED}$$

### 3.4. Lightweight Proposal & Partnership Pipeline
Menghindari kerumitan CRM berat dengan alur status sederhana:
$$\text{Prospect} \longrightarrow \text{Contacted} \longrightarrow \text{Proposal Submitted} \longrightarrow \text{Under Review} \longrightarrow \text{Approved / Rejected}$$

---

## 4. Desain Skema Database (PostgreSQL DDL)

```sql
-- 1. Master Tenants & Multi-Role User Structure
CREATE TYPE tenant_type AS ENUM ('ORGANIZATION', 'CORPORATE', 'ADMIN');

CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    type tenant_type NOT NULL DEFAULT 'ORGANIZATION',
    is_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Master Perusahaan Publik (Public Intelligence Data)
CREATE TABLE companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    ticker VARCHAR(20),
    industry VARCHAR(100),
    website VARCHAR(255),
    corporate_domain VARCHAR(100),
    headquarters_city VARCHAR(100),
    is_claimed BOOLEAN DEFAULT FALSE,
    claimed_by_tenant_id UUID REFERENCES tenants(id),
    csr_rating_score INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Klaim Profil Perusahaan (Verification System)
CREATE TYPE claim_status AS ENUM ('PENDING', 'APPROVED', 'REJECTED', 'REVOKED');
CREATE TYPE claim_method AS ENUM ('CORPORATE_EMAIL', 'DOCUMENT_UPLOAD');

CREATE TABLE company_claims (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    requested_by_user_id UUID NOT NULL,
    work_email VARCHAR(255),
    document_proof_url TEXT,
    method claim_method NOT NULL,
    status claim_status DEFAULT 'PENDING',
    verification_notes TEXT,
    verified_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (company_id, tenant_id, status)
);

-- 4. Program CSR Perusahaan & Opportunity Terbuka
CREATE TABLE csr_opportunities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id),
    tenant_id UUID REFERENCES tenants(id),
    title VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL, -- Education, Health, Environment, Economic
    target_location VARCHAR(100),
    budget_amount NUMERIC(15, 2),
    open_until TIMESTAMPTZ,
    status VARCHAR(50) DEFAULT 'OPEN', -- OPEN, CLOSED, IN_REVIEW
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 5. Program NGO & Proposal Submission Workflow
CREATE TABLE ngo_programs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    title VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL,
    location VARCHAR(100),
    target_beneficiaries INT,
    budget_needed NUMERIC(15, 2),
    sdg_goals INT[],
    fiqh_asnaf VARCHAR(50),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE proposals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    opportunity_id UUID REFERENCES csr_opportunities(id),
    ngo_program_id UUID NOT NULL REFERENCES ngo_programs(id),
    org_tenant_id UUID NOT NULL REFERENCES tenants(id),
    corp_tenant_id UUID REFERENCES tenants(id),
    company_id UUID NOT NULL REFERENCES companies(id),
    proposal_file_url TEXT,
    status VARCHAR(50) DEFAULT 'SUBMITTED', -- SUBMITTED, UNDER_REVIEW, MEETING, APPROVED, REJECTED
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

## 5. Strategi API & Endpoint Routing

| HTTP Method | Endpoint Path | Akses Peran | Deskripsi |
| :--- | :--- | :--- | :--- |
| **GET** | `/api/v1/public/companies` | Public / SEO | Pencarian direktori perusahaan & profil CSR publik |
| **GET** | `/api/v1/public/companies/:slug` | Public / SEO | Detail profil perusahaan + Tombol *Claim Profile* |
| **POST** | `/api/v1/corporate/claims` | Authenticated | Pengajuan klaim profil perusahaan (Domain/Upload) |
| **POST** | `/api/v1/admin/claims/:id/verify` | Admin / Telegram Bot | Verifikasi 1-click klaim perusahaan via Telegram Webhook |
| **GET** | `/api/v1/org/matches` | Org Tenant | AI Vector matching program NGO ↔ CSR Korporasi (1536d) |
| **POST** | `/api/v1/corporate/opportunities` | Corporate Tenant | Publikasi program *CSR Opportunity* terbuka |
| **POST** | `/api/v1/proposals` | Org Tenant | Pengajuan proposal program ke perusahaan / opportunity |
| **PATCH** | `/api/v1/proposals/:id/status` | Corporate / Org | Perubahan status pipeline proposal (`SUBMITTED` $\rightarrow$ `APPROVED`) |

---

## 6. UI/UX & Routing Landing Page

```text
https://csrmatics.id/
│
├── / (Homepage)
│     ├── Hero Section (Persona Switcher: [For Organizations] vs [For Corporate])
│     ├── Fitur Platform (Bento Grid)
│     ├── Interactive Product Demo
│     ├── Seksi Paket & Harga (Pricing Switcher: Bulanan vs Tahunan)
│     └── SDG & Fiqh Alignment
│
├── /organizations (Landing Page Khusus NGO / LAZ)
├── /corporate (Landing Page Khusus CSR Unit)
│
├── /companies/[slug] (SEO Public Profile + Claim Button)
├── /opportunities/[slug] (SEO Public CSR Opportunity)
│
├── /login (Single Login Route -> Auto Redirect ke /dashboard)
└── /register (Step 1: Choose Role -> Step 2: Org/Corp Profile)
```

---

## 7. Roadmap Pelaksanaan Solo Developer

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│ TAHAP 1: Data Intelligence & Public SEO Magnet                              │
│ - Scraping & database 3.000+ perusahaan verified                            │
│ - Halaman publik /companies/[slug] + Fitur "Claim Profile"                  │
├─────────────────────────────────────────────────────────────────────────────┤
│ TAHAP 2: Tenant NGO (Early Adopters & Monetisasi Pertama)                   │
│ - Pendaftaran NGO & Profil Program                                          │
│ - AI Vector Matching (1536d) & Pencarian Intelligence                      │
│ - Ekspor Proposal PDF & Draf Icebreaker                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│ TAHAP 3: Tenant Corporate (Marketplace Liquidity)                           │
│ - Domain matching otomatis (@company.com) & Approval Bot Telegram           │
│ - Dashboard Corporate untuk Post Opportunity & Review Proposal              │
├─────────────────────────────────────────────────────────────────────────────┤
│ TAHAP 4: Network Effect & Monetisasi Lanjutan                               │
│ - Payment Gateway Faspay/Midtrans (Auto Activation)                         │
│ - Notifikasi Real-time WhatsApp/Telegram Alert                              │
│ - Reporting CSR/ESG Analytics                                               │
└─────────────────────────────────────────────────────────────────────────────┘
```
