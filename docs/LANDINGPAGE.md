# LANDINGPAGE.md — CSRmatics Landing Page & Pricing Architecture

> Specification and Design Guide for CSRmatics — CSR Intelligence & Partnership Platform

---

## 1. Overview & Positioning

**CSRmatics** adalah *Two-Sided CSR Intelligence & Partnership Platform* yang menghubungkan **Perusahaan / Unit ESG (Corporate)** dengan **Lembaga Kemanusiaan / LAZ (NGO)** untuk memotong friksi riset, verifikasi, dan penyaluran program keberlanjutan.

### Core Value Proposition

- **Positioning**: CSR Intelligence & Partnership Platform (Connecting companies with organizations to create greater social impact).
- **Brand Tagline**: `✦ Next-Gen CSR Intelligence Platform`
- **Main Headline**: `Akselerasi Kemitraan CSR berbasis AI.`
- **Sub-headline**: `Dari riset program hingga penyaluran proposal: platform terpadu bagi Korporasi dan NGO untuk merealisasikan inisiatif keberlanjutan.`

---

## 2. Diagram Navigasi & Persona Journey

Pengunjung landing page dapat mengeksplorasi platform sesuai persona mereka tanpa mengalami kebingungan konteks:

```text
                                [ LANDING PAGE (/) ]
                                         │
        ┌────────────────────────────────┴────────────────────────────────┐
        │                                                                 │
  JALUR A (Eksplisit Persona)                                     JALUR B (Menu Header)
  Problem-Solution Bridge CTA                                     Navbar: "Harga & Paket" (/pricing)
  • "Eksplor Solusi Korporasi"                                            │
  • "Eksplor Solusi NGO"                                                  ▼
        │                                                    ┌──────────────────────────┐
        ▼                                                    │   DEDICATED PRICING PAGE │
  pricing?persona=corporate                                  │     (/pricing?persona=*) │
  pricing?persona=ngo                                        └────────────┬─────────────┘
        │                                                                 │
        └────────────────────────────────┬────────────────────────────────┘
                                         ▼
                             [ LOGIN / SELECTION (/login) ]
                             • type=organization (Lembaga & NGO)
                             • type=corporate (Perusahaan & TJSL)
```

---

## 3. Landing Page Component Architecture (`/`)

Implemented at: `sovera-csr-dashboard/src/app/page.tsx`

### 3.1. Sticky Header Navigation (`<header>`)
- **Logo & Brand**: CSRmatics (`FundIQ Enterprise` badge)
- **Nav Links**:
  - `Fitur Platform` (`#features`)
  - `Corporate Directory` (`/corporates` with `3.000+` verified badge)
  - `Interactive Demo` (`#demo`)
  - `Harga & Paket` (`/pricing`)
  - `SDG & Fiqh Alignment` (`#sdgs`)
- **CTAs**:
  - `Masuk Sesi` (`/login`)
  - `Buka Dashboard` (`/dashboard`)

---

### 3.2. Hero Section (`<section>`)
- **Release Pill Badge**: `✦ Next-Gen CSR Intelligence Platform` (Pill badge emerald glow)
- **Main Headline**: `Akselerasi Kemitraan CSR berbasis AI.`
- **Sub-headline**: `Dari riset program hingga penyaluran proposal: platform terpadu bagi Korporasi dan NGO untuk merealisasikan inisiatif keberlanjutan.`
- **Live Metrics Cards Grid**:
  1. `3.000+` — Database Perusahaan (BUMN, Tbk, Bank & Swasta)
  2. `2.800+` — Website Live Verified (100% HTTP 200 OK Ping)
  3. `24/7` — AI WebScraper Crawler (Monitoring Sinyal CSR Real-Time)
  4. `1536d` — Vector Embedding (Pencocokan Pilar & Fiqh Asnaf)

---

### 3.3. Problem-Solution Bridge Section (`<section>`)
*Tepat di bawah Hero (Side-by-Side Value Proposition)*

- **Judul Section**: `Dari Berbulan-bulan Riset Menjadi Kemitraan Nyata dalam Hitungan Hari`
- **Sub-judul**: `Proses kemitraan CSR tradisional sering terhambat proposal yang tidak sesuai fokus, verifikasi legalitas manual, dan riset kontak yang melelahkan. CSRmatics menghapus friksi tersebut dengan data intelijen dan pencocokan terverifikasi.`
- **Side-by-Side Value Cards**:
  - **Card 1 (Untuk Korporasi & TJSL)**:
    - *Judul*: Tepat Sasaran Tanpa Seleksi Manual
    - *Point 1*: **Kurasi Mitra Terverifikasi** — Temukan NGO yang sudah tervalidasi legalitas dan rekam jejak lapangannya secara komprehensif.
    - *Point 2*: **Selaras Target ESG & SDGs** — Pencocokan otomatis memastikan dana CSR tersalur ke program yang sesuai pilar keberlanjutan perusahaan.
    - *CTA*: `Eksplor Solusi Korporasi` (`/pricing?persona=corporate`)
  - **Card 2 (Untuk Lembaga Sosial & NGO)**:
    - *Judul*: Tembus Korporasi yang Tepat Sasaran
    - *Point 1*: **Hentikan "Cold Proposal"** — Kirim program ke perusahaan yang memang memiliki anggaran, fokus, dan lokasi yang selaras.
    - *Point 2*: **Pantau Status Transparan** — Ketahui kapan proposal Anda dibaca dan diproses tanpa perlu follow-up manual berulang kali.
    - *CTA*: `Eksplor Solusi NGO` (`/pricing?persona=ngo`)

---

### 3.4. Bento Grid Features Section (`#features`)
- **Card 1 — Real-Time Corporate Signal Feed**: Crawler otomatis memantau rilis berita, laporan tahunan BEI, portal BUMN, dan rilis ESG 24 jam sehari.
- **Card 2 — Direktori 2.080+ Perusahaan Verified**: Database autentik perusahaan perbankan, BUMN, emiten Tbk, dan swasta dengan verifikasi domain resmi.
- **Card 3 — AI Program Matcher & Fiqh Asnaf**: Kecerdasan buatan berbasis Gemini 1.5 Flash & 1536d vector embedding yang mencocokkan program LAZ/Yayasan dengan alokasi CSR korporasi.

---

### 3.5. Interactive Product Preview Section (`#demo`)
- Simulasi live pencarian kecerdasan CSR perusahaan (e.g. `PT Bank Central Asia Tbk` / `BBCA`).
- **Interactive Tabs**:
  - `AI Match Score (94.8%)`: Menampilkan skor kecocokan vector embedding dan alokasi anggaran CSR.
  - `Domain Status (Verified 200 OK)`: Menampilkan pilar utama CSR, mitra NGO terdahulu, dan email publik CSR.
  - `Draf Icebreaker Pitch`: Menampilkan draf proposal yang di-generate otomatis oleh AI.

---

### 3.6. Compliance & Fiqh Alignment Section (`#sdgs`)
- **SDG 1 & 2**: Tanpa Kemiskinan (Asnaf Fakir & Miskin)
- **SDG 4**: Pendidikan Berkualitas (Beasiswa & Literasi Digital)
- **SDG 13**: Aksi Iklim & ESG (Konservasi Lingkungan)
- **SDG 17**: Kemitraan Tujuan (Kolaborasi LAZ & Korporasi)

---

### 3.7. Footer Navigation (`<footer>`)
- Brand: `CSRmatics FundIQ Enterprise`
- Copyright: `© 2026 CSRmatics. Enterprise B2B Philanthropy & CSR Match Engine.`
- Quick Links: `Corporate Directory` (`/corporates`) & `Buka System Dashboard` (`/dashboard`).

---

## 4. Pricing Architecture (`/pricing`)

Implemented at: `sovera-csr-dashboard/src/app/pricing/page.tsx`

### 4.1. Segmented Controller Switch
Mengendalikan persona pengguna secara real-time via URL parameter (`?persona=ngo` vs `?persona=corporate`) dan tersinkronisasi otomatis dengan `sessionStorage`:

```tsx
[ 🤝 Lembaga Sosial / NGO ]   [ 🏢 Perusahaan / TJSL ]
```

---

### 4.2. Skema Biaya Lembaga Sosial / NGO (`persona=ngo`)

| Tier | Harga Bulanan | Harga Tahunan (Hemat 17%) | Fitur Utama |
|---|---|---|---|
| **Starter** | **Rp 0** | Rp 0 / selamanya | Profil publik dasar, browse direktori perusahaan, submit 3 proposal/bulan, support email. |
| **Professional** *(Populer)* | **Rp 399.000** / bln | **Rp 3.990.000** / thn | Full CSR Intelligence Engine, AI Vector Matchmaking (1536d), Monitoring Sinyal CSR Real-Time, Unlimited Proposals, 5 seats team. |
| **Growth** | **Rp 999.000** / bln | **Rp 9.990.000** / thn | Semua fitur Pro, custom API access, prioritas pencocokan RFP, dedicated account manager, 15 seats team. |

---

### 4.3. Skema Biaya Perusahaan / Corporate & TJSL (`persona=corporate`)

| Tier | Harga Bulanan | Fitur Utama |
|---|---|---|
| **Verified Claim** | **Rp 0** | Klaim & verifikasi profil korporasi, unggah laporan ESG/CSR publik, terima proposal terverifikasi dari NGO. |
| **Corporate Starter** | **Rp 1.500.000** / bln | Terbitkan hingga 3 *CSR Opportunities (RFP)* / bulan, kurasi & AI vetting proposal masuk, dashboard analisis dampak ESG & SDGs. |
| **Enterprise Solution** | **Custom Contact** | Unlimited RFP opportunities, custom integration dengan sistem audit internal, dedicated ESG analyst & onboarding pendampingan. |

---

## 5. File Mapping & Tech Stack

| Domain | File Path | Scope / Responsibility |
|---|---|---|
| **Landing Page** | `sovera-csr-dashboard/src/app/page.tsx` | Main homepage UI, Hero, Problem-Solution Bridge, Bento Grid, Live Demo, Compliance. |
| **Pricing Engine** | `sovera-csr-dashboard/src/app/pricing/page.tsx` | Persona Switcher, NGO & Corporate pricing cards, billing cycle toggle. |
| **Corporate Directory** | `sovera-csr-dashboard/src/app/corporates/page.tsx` | Public 3.000+ corporate search & discovery directory. |
| **Authentication** | `sovera-csr-dashboard/src/app/login/page.tsx` | Two-sided login & registration flow (`NGO` vs `CORPORATE`). |
| **API Endpoints** | `sovera-csr-api/internal/handler/` | Backend REST API for claims, opportunities, proposals, and search. |

---

## 6. Last Updated

- **Date**: 2026-09-14
- **Version**: 2.0.0
- **Status**: Production Ready & Fully Implemented in `sovera-csr-dashboard` and `sovera-csr-api`.
