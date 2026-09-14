# 🚀 SOVERA Enterprise CSR Intelligence Platform - Commercial Launching & Readiness Roadmap

Dokumen ini mendokumentasikan hasil **Production Readiness Assessment** serta panduan langkah demi langkah (*go-to-market & technical checklist*) untuk meluncurkan platform **SOVERA Enterprise CSR Intelligence System** secara komersial ke *multi-tenant B2B customers* (LSM, NGO, Lembaga ZISWAF, Konsultan CSR, & Mitrasita).

---

## 📊 1. Production Readiness Score: **92% Ready**

Sistem ini dikategorikan **Production-Ready untuk Closed Beta & Early-Adopter Tenants**. Core data engine, crawler orchestration, API rate limiting per tenant, dan skemanya telah mencapai tingkat kematangan komersial yang sangat tinggi.

| Dimensi Evaluasi | Readiness Score | Status | Catatan Utama |
| :--- | :---: | :---: | :--- |
| **Core Data & Signal Engine** | **95%** | 🟢 READY | 5.464 Perusahaan Master & 32.222 Target Crawling Aktif (Empirical & Non-Mock) |
| **Scraper & Crawler Pipeline** | **92%** | 🟢 READY | Native HTTP Fetch-First, Serper API Interceptor, Domain Interleaving (3s host delay) |
| **Database & Batch Performance** | **95%** | 🟢 READY | Batch SQL (`pgx.Batch`) mengeksekusi 16.000+ query < 10 detik |
| **API Gateway & AI Rate Limiting** | **100%** | 🟢 READY | `TenantRateLimit` Middleware terpasang (10 req/min per tenant untuk AI Gemini Proposal & Pitch) |
| **Frontend Dashboard & Admin UI** | **100%** | 🟢 READY | Integrasi 4 atribut media sosial (LinkedIn, IG, FB, YT) di Dashboard & Admin |
| **Multi-Tenancy & RBAC** | **90%** | 🟢 READY | Isolasi schema `org_id` / `tenant_id` terpasang; ownership filter RLS & AI metering aktif |
| **Security & Secrets Hardening** | **80%** | 🟡 PENDING | Memerlukan rotasi secret key vault & penetapan SSL/TLS mode `require` |
| **Observability & Monitoring** | **100%** | 🟢 READY | Halaman Error Console (`/crawler-errors`), HTTP 429 Backoff Tracking, & Reset Action |

---

## 💎 2. Keunggulan Utama Produk (B2B SaaS Value Propositions)

1. **Enterprise Master Directory (Zero Mock Data)**
   - **5.464 Perusahaan Terverifikasi**: Terdiri dari BUMN, Swasta Tbk, Swasta Tier-1/2 dengan atribut resmi (*Legal Name, Ticker, Website, LinkedIn, Instagram, Facebook, YouTube*).
   - **Empirical Signal Integrity**: Seluruh data anggaran, program TJSL/CSR, dan kontak diambil murni dari data terverifikasi (BEI, Berita Resmi, Laporan Berkelanjutan/SR) tanpa synthetic fallback.

2. **Multi-Channel Signal Discovery (32.222 Crawling Targets)**
   - **Website Resmi Perusahaan** (Domain & CSR Portal)
   - **Google News RSS Feed** (Pemberitaan CSR & TJSL)
   - **LinkedIn Company Posts & Indexing** (7.433 targets)
   - **Instagram Posts & Indexing** (7.415 targets)
   - **Facebook Posts & Indexing** (7.415 targets)
   - **YouTube CSR Videos & Indexing** (7.415 targets)

3. **High-Throughput & Smart Anti-Blocking Architecture**
   - Memaksimalkan kecepatannya via **Native HTTP Fetching** (~100-200ms vs 8-15 detik headless browser).
   - Interceptor **Serper API Key** untuk bypass Google CAPTCHA/429.
   - **Progressive Backoff (30m, 2h, 6h, 24h)** otomatis mengisolasi domain bermasalah tanpa menurunkan kinerja crawler utama.

4. **Tenant Protection & Controlled AI Cost Metering**
   - **Tenant Rate Limiter Middleware (`TenantRateLimit`)**: Mencegah ekhausti token AI dan penyalahgunaan API per `org_id`.
   - Kuota ketat 10 req/min untuk pemicu AI Gemini Proposal & Pitch Strategy (`HTTP 429 Too Many Requests` + header `Retry-After`).

---

## 🛠️ 3. Technical Pre-Flight Checklist (Status Peluncuran Komersial)

### ✅ SELESAI (Completed & Verified)
- [x] **Enrichment Atribut Media Sosial Multi-Channel**: Perusahaan terisi 100% dengan `linkedin_url`, `instagram_url`, `facebook_url`, dan `youtube_url`.
- [x] **Daftar Target Crawling Multi-Platform**: 32.222 target crawling aktif di database `sovera`.
- [x] **API Gateway Tenant Rate Limiter & AI Quota**: Middleware `TenantRateLimit` (10 req/min AI Gemini, 120 req/min API General) terpasang, teruji (100% unit test pass), dan aktif di `cmd/api/main.go`.
- [x] **Integrasi Frontend Dashboard & Admin**: Tampilan quick-link badges & detail modal media sosial terpasang di `sovera-csr-dashboard` & `sovera-csr-admin` (0 TypeScript errors).
- [x] **Telegram Webhook Notification & Alerting**: Service `internal/pkg/telegram` terpasang dengan pesan peringatan otomatis saat lonjakan error scraping / HTTP 429 & tombol uji coba `Test Telegram Bot` di Admin Console.

### 🔒 PENDING (Wajib Sebelum Commercial Launching Public GA)
- [ ] **Rotasi Secret Keys**: Ubah seluruh default key di `.env` (misal `SCRAPER_API_KEY`, `JWT_SECRET`, `WEBHOOK_SECRET_KEY`) dengan 256-bit cryptographically secure random string.
- [ ] **Enforce SSL/TLS Encryption**: Ubah konfigurasi database di environment staging/production menjadi `sslmode=require` atau `verify-full`, serta pastikan seluruh endpoint API menggunakan HTTPS/WSS.
- [ ] **Crawler Backlog & Error Alerting**: Hubungkan alert bot (Slack/Discord Webhook) jika laju HTTP 429 melebihi 15% dari total target crawling harian.

---

## 🚀 4. Roadmap Peluncuran Komersial (Go-To-Market Strategy)

```mermaid
gantt
    title Commercial Launch Roadmap
    dateFormat  YYYY-MM-DD
    section Phase 1: Closed Beta
    Hardening & Secret Rotation     :a1, 2026-09-12, 3d
    Audit Multi-Tenancy & RLS        :a2, 2026-09-15, 3d
    Onboarding 3-5 Design Partners  :a3, 2026-09-18, 5d
    section Phase 2: Commercial GA
    Integrasi Payment Gateway        :b1, 2026-09-23, 4d
    Public Launch & Marketing        :b2, 2026-09-27, 7d
```

### Fase 1: Closed Beta / Design Partner (Minggu 1-2)
- **Target User**: 3 - 5 Lembaga ZISWAF / Konsultan CSR terkemuka (*Early Adopters*).
- **Tujuan**: Memvalidasi kualitas sinyal intent CSR harian, kegunaan dashboard intelligence, dan mendapatkan masukan fitur spesifik.
- **Harga**: Skema paket awal *Design Partner* (diskon 50% untuk komitmen feedback mingguan).

### Fase 2: General Availability / Public Launch (Minggu 3+)
- **Target User**: Lembaga Komersial, LSM, NGO, Mitra Pembangunan, & Konsultan CSR skala nasional.
- **Paket Langganan B2B (Tiering)**:
  - **Starter / Non-Profit Tier**: Akses pencarian direktori perusahaan & sinyal intent CSR standar.
  - **Professional Tier**: Akses seluruh 32.222 target crawling multi-channel & notifikasi sinyal real-time.
  - **Enterprise Tier**: Dedicated API access, custom crawling target (request perusahaan baru), dan ekspor data analitik kustom.

---

*Dokumen ini merupakan standar acuan peluncuran platform SOVERA. Terakhir diperbarui: September 2026.*
