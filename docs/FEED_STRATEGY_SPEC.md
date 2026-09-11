# FEED_STRATEGY_SPEC.md - Master Specification for CSR Intelligence Data Feeds, Sources & Expansion Roadmap

Dokumen ini merupakan spesifikasi komprehensif mengenai strategi pengumpulan data (*data ingestion*), taksonomi kata kunci, hierarki sumber (*source ecosystem*), arsitektur crawler/search engine, serta roadmap ekspansi dan peningkatan kualitas data (*data quality roadmap*) untuk **Sovera CSR Intelligence Platform**.

---

## Daftar Isi
1. [Visi Arsitektur & Signal Engine](#1-visi-arsitektur--signal-engine)
2. [Ekosistem & Katagori Sumber Feed (Source Catalog)](#2-ekosistem--kategori-sumber-feed-source-catalog)
3. [Taksonomi Kata Kunci & Discovery Layer](#3-taksonomi-kata-kunci--discovery-layer)
4. [Strategi LinkedIn & Media Sosial](#4-strategi-linkedin--media-sosial)
5. [Infrastruktur Crawler & Alternatif Mesin Pencari](#5-infrastruktur-crawler--alternatif-mesin-pencari)
6. [Roadmap Ekspansi & Peningkatan Kualitas Data (Enrichment)](#6-roadmap-ekspansi--peningkatan-kualitas-data-enrichment)

---

## 1. Visi Arsitektur & Signal Engine

Sovera CSR Intelligence Platform dibangun sebagai **Mesin Pengumpulan dan Normalisasi Intelligence** yang memetakan hubungan antara:
$$\text{Perusahaan (Company)} \longleftrightarrow \text{Program Sosial (Activity)} \longleftrightarrow \text{Lembaga/NGO (Partnership)}$$

Tujuan utamanya adalah menjawab pertanyaan bernilai tinggi bagi pengguna SaaS:
> **"Perusahaan mana yang saat ini paling mungkin menjadi partner untuk proposal saya?"**

### Struktur Entitas Utama
```text
Company
   │
   ├── CSR Profile (Kategori Fokus, Wilayah Operasional, Anggaran)
   ├── CSR History (Histori Program Berdasarkan Sustainability/TJSL Report)
   ├── CSR Score (Tingkat Aktivitas & Kecocokan Alignment)
   │
   ├── CSR Personnel (Kontak Pengelola / Personnel CSR)
   │
   └── CSR Events & Signals
          ├── News (Berita Aktivitas Terbaru)
          ├── Program (Program Binaan / Ongoing CSR)
          ├── Partnership (Kemitraan yang Terjalin)
          ├── Funding (Dana Hibah / Donasi)
          └── Opportunity (Peluang Open Call / Proposal Partnership)
```

---

## 2. Ekosistem & Kategori Sumber Feed (Source Catalog)

Untuk membangun database intelligence yang kaya, sumber data dikelompokkan berdasarkan nilai informasi (*signal value*) dan tingkat kepercayaan (*authority*):

| Sumber Data | Nilai | Data Utama yang Diperoleh | Peran dalam Sistem |
| :--- | :---: | :--- | :--- |
| **Website/Newsroom Perusahaan** | ⭐⭐⭐⭐⭐ | Program CSR terbaru, siaran pers resmi | **Primary Source** |
| **Laporan TJSL Perusahaan** | ⭐⭐⭐⭐⭐ | Realisasi program, anggaran, lokasi, penerima manfaat | **Primary Source (Histori)** |
| **Sustainability Report (OJK/IDX)** | ⭐⭐⭐⭐⭐ | Histori ESG, komitmen keanekaragaman hayati, SDGs | **Primary Source (ESG)** |
| **ANTARA & Media Nasional** | ⭐⭐⭐⭐⭐ | Berita liputan aktivitas CSR korporasi | **News Feed** |
| **Media Regional** | ⭐⭐⭐⭐⭐ | Aktivitas CSR perusahaan di tingkat daerah/lokal | **Regional News** |
| **Portal Pemerintah / TJSL Pemda** | ⭐⭐⭐⭐⭐ | Data perusahaan aktif & mitra daerah | **Official Data** |
| **Website NGO & Yayasan** | ⭐⭐⭐⭐⭐ | Program kemitraan, proposal terbuka, open calls | **Partnership Lead** |
| **OJK & BEI/IDX** | ⭐⭐⭐⭐ | Repository Laporan Keberlanjutan & Emiten Publik | **Official Repository** |
| **CSR/TJSL Media Khusus** | ⭐⭐⭐⭐ | Berita penghargaan, isu industri, regulasi | **Industry News** |
| **Press Release Distribution** | ⭐⭐⭐⭐ | Pengumuman resmi korporasi | **Early Signal** |
| **LinkedIn (Public Signals)** | ⭐⭐⭐⭐ | Signal awal aktivitas & identifikasi personil | **Discovery Sensor** |
| **Instagram / Social Media** | ⭐⭐⭐ | Activity signals & visual footprint | **Discovery Sensor** |
| **CSR Awards & Accreditations** | ⭐⭐⭐ | Identifikasi perusahaan aktif CSR | **Verification Signal** |

---

### 2.1 Enam Kanal Utama Feed Korporat

1. **Laporan Keterbukaan & Keberlanjutan Bursa Efek Indonesia (`BEI_REPORT` / `PDF_DOCUMENT`)**:
   Dokumen resmi emiten terbuka (Tbk), meliputi *Sustainability Report*, laporan tahunan (*Annual Report*), dan rilis keterbukaan informasi publik terkait realisasi maupun komitmen alokasi anggaran TJSL/CSR perusahaan (`idx.co.id`).
2. **Portal Berita Bisnis & Finansial (`NEWS_ARTICLE` / `NEWS_RSS`)**:
   Artikel media massa daring (Bisnis.com, Kontan, Antara, Republika) yang memberitakan aksi korporasi, inisiatif bantuan bencana, program pemberdayaan masyarakat, pencapaian laba kuartalan, maupun nota kesepahaman (MoU) kemitraan baru.
3. **Peluang Hibah & Call for Proposal CSR (`CSR_OPPORTUNITY_SEARCH`)**:
   Pengumuman pendaftaran proposal hibah publik, dana padanan (*matching grant*), atau program kompetisi sosial yang dibuka oleh yayasan korporat (*corporate foundation*) atau departemen TJSL perusahaan (misal: UNDP, DAP AusEmbassy, GGP Japan).
4. **Pengayaan Profil Korporasi & ESG (`COMPANY_ENRICHMENT`)**:
   Halaman profil resmi korporasi, direktori kontak departemen TJSL/CSR, alamat kantor pusat (HQ), serta fokus pilar ESG utama perusahaan.
5. **Pencarian Domain & Kandidat Baru (`SEARCH_DISCOVERY`)**:
   Hasil pemindaian mesin pencari eksternal untuk menemukan domain, rilis pers, atau entitas korporat baru yang berpotensi menjadi calon mitra fundraising.
6. **Kanal Komunikasi & Media Sosial Resmi Korporasi (`SOCIAL_POST`)**:
   Pengumuman resmi dari akun terverifikasi perusahaan (seperti LinkedIn Company Page, siaran pers korporat, atau kanal publikasi humas) mengenai pembukaan kemitraan dan program pembinaan masyarakat.

---

### 2.2 Strategi Penggelaran MVP (100 Sumber Pertama)

Untuk tahap awal (*MVP*), crawling difokuskan pada **100 sumber berkualitas tinggi**:
* **30–40 Media Nasional/Regional** (Antara, Bisnis.com, Kontan, Republika).
* **30–50 Perusahaan Besar & Emiten BUMN/Swasta** (Pertamina, PLN, Telkom, Pelindo, PTPP, Timah).
* **10–20 NGO/Yayasan Utama** (BAZNAS, Dompet Dhuafa, Rumah Zakat, EBI Foundation).
* **10 Portal Pemerintah / Pemda**.

```text
100 Sources
     ↓
RSS / Sitemap / Targeted Web Scraper
     ↓
~200–500 Raw Articles / Day
     ↓
Deduplication & Content Hashing
     ↓
AI CSR Classification & Scoring
     ↓
~30–80 High-Value Intelligence Events / Day
```

---

## 3. Taksonomi Kata Kunci & Discovery Layer

Kata kunci (*keywords*) digunakan sebagai **Discovery Layer** untuk menemukan kandidat berita/artikel. Keputusan akhir mengenai relevansi CSR dilakukan oleh **AI Classifier**.

### 3.1 Kamus Kata Kunci Kategori CSR (12 Kluster Utama)

1. **Istilah Utama CSR & TJSL**:
   `CSR`, `Corporate Social Responsibility`, `Corporate Responsibility`, `Tanggung Jawab Sosial`, `TJSL`, `Tanggung Jawab Sosial Perusahaan`, `Program TJSL`, `Kegiatan CSR`.
2. **Sustainability & ESG**:
   `Sustainability`, `Sustainable Development`, `ESG`, `Environmental Social Governance`, `Sustainability Report`, `ESG Commitment`, `Social Impact`, `Green Initiative`.
3. **Community & Social Development**:
   `Community Development`, `Community Empowerment`, `Pemberdayaan Masyarakat`, `Pengembangan Masyarakat`, `Desa Binaan`, `Kampung Binaan`, `Kemandirian Masyarakat`.
4. **Lingkungan & Keanekaragaman Hayati**:
   `Program Lingkungan`, `Environmental Conservation`, `Pelestarian Lingkungan`, `Penghijauan`, `Reboisasi`, `Ekonomi Sirkular`, `Circular Economy`, `Waste Management`, `Net Zero`, `Dekarbonisasi`, `Biodiversity`.
5. **Ekonomi & Pemberdayaan UMKM**:
   `Pemberdayaan UMKM`, `Pengembangan UMKM`, `UMKM Binaan`, `UMKM Mitra`, `Entrepreneur Development`, `Economic Empowerment`, `Local Economic Development`.
6. **Pendidikan & Beasiswa**:
   `Program Pendidikan`, `Education Initiative`, `Beasiswa`, `Scholarship`, `Sekolah Binaan`, `Literasi Digital`, `Pendidikan Vokasi`, `Capacity Building`.
7. **Kesehatan & Sanitasi**:
   `Program Kesehatan`, `Kesehatan Masyarakat`, `Community Health`, `Stunting`, `Penanganan Stunting`, `Gizi`, `Sanitasi`, `Air Bersih`.
8. **Bantuan Kemanusiaan & Filantropi**:
   `Bantuan Perusahaan`, `Bantuan Sosial`, `Bantuan Bencana`, `Corporate Giving`, `Corporate Philanthropy`, `Donasi`, `Disaster Relief`, `Emergency Relief`.
9. **Kemitraan & Kolaborasi (Partnership)**:
   `CSR Partnership`, `Partnership CSR`, `Corporate Partnership`, `Strategic Partnership`, `Kemitraan Perusahaan`, `Kolaborasi CSR`, `Open Partnership`, `Invitation for Partnership`.
10. **Creating Shared Value (CSV)**:
    `Creating Shared Value`, `CSV`, `Shared Value Creation`, `Inclusive Business`, `Social Enterprise`, `Impact Investment`.
11. **Volunteerism & Bakti Sosial**:
    `Corporate Volunteering`, `Employee Volunteering`, `Volunteer Program`, `Relawan Perusahaan`, `Bakti Sosial`.
12. **Community Relations & Keterlibatan Pemangku Kepentingan**:
    `Community Relations`, `Community Engagement`, `Stakeholder Engagement`, `Public Engagement`, `Community Outreach`, `Hubungan Komunitas`.

---

### 3.2 3-Level Arsitektur Taksonomi

```text
Level 1: CSR Relevance
  ├── CSR / TJSL / Sustainability / ESG / Social Impact / Philanthropy

Level 2: Activity Domain
  ├── Education / Health / Environment / UMKM / Economic Empowerment / Disaster Relief / Infrastructure

Level 3: Specific Action
  ├── Scholarship / Training / Donation / Grant / Volunteer / Facility Construction / Partnership
```

---

## 4. Strategi LinkedIn & Media Sosial

LinkedIn dan media sosial merupakan sumber sinyal awal (*early signals*):

* **JANGAN melakukan scraping langsung pada LinkedIn** menggunakan headless browser/login otomatis (melanggar *LinkedIn User Agreement & Crawling Terms*).
* **Gunakan LinkedIn sebagai Discovery Sensor**:
  1. Manfaatkan mesin pencari (Google/Bing) untuk mengindeks halaman/post publik LinkedIn (`site:linkedin.com/posts "CSR" "Indonesia"`).
  2. Ketika sinyal posting ditemukan, AI mencatat perusahaan dan nama program.
  3. Sistem melakukan **Verifikasi** dengan melakukan crawling ke website/newsroom resmi perusahaan terkait.
  4. Simpan data yang telah terverifikasi ke dalam basis data utama.

```text
LinkedIn Signal (Public Index)
      ↓
System Verification (Official Newsroom Crawl)
      ↓
Structured CSR Intelligence Entry
```

---

## 5. Infrastruktur Crawler & Alternatif Mesin Pencari

1. **Google Custom Search JSON API (Resmi)**:
   * Menggunakan API resmi Google (`GOOGLE_API_KEY` & `GOOGLE_SEARCH_ENGINE_ID`).
   * Output JSON terstruktur tanpa parsing HTML dan bebas dari risiko CAPTCHA/429.
2. **Layanan SERP API Pihak Ketiga (Serper.dev)**:
   * Menggunakan **Serper API Interceptor** pada `web-scraper` (`SERPER_API_KEY`).
   * Mengubah URL pencarian menjadi panggilan REST API terenkripsi (~100ms response time).
3. **Direct Site-Scraping & Targeted Feeds**:
   * Menghindari pencarian luas di search engine dengan menembak langsung feed RSS, sitemap, dan portal resmi perusahaan target.

### Perlindungan Rate-Limit Multi-Tier
* **Tier 1 (Per-Host Domain Throttling & Round-Robin Interleaving)**: Jeda 3 detik per-host domain.
* **Tier 2 (Scraper Internal Rate Limiter)**: `WORKER_RATE_LIMIT_DELAY_MS=5000`.
* **Tier 3 (Progressive Cooldown Backoff)**: Penundaan bertahap 30m $\rightarrow$ 2h $\rightarrow$ 6h $\rightarrow$ 24h.

---

## 6. Roadmap Ekspansi & Peningkatan Kualitas Data (Enrichment)

Untuk mengekspansi data perusahaan dari entitas terverifikasi menuju 25.000+ korporasi nasional & daerah:

### A. Matriks Peningkatan Kualitas Data (Data Quality Metrics)

Setiap entitas perusahaan di database harus diperkaya hingga memenuhi 5 indikator kelengkapan (*Intelligence Completeness Score*):

| Indikator Kualitas Data | Target Kelengkapan | Sumber Data & Metode Verifikasi |
| :--- | :---: | :--- |
| **1. Official Domain & HTTP Health** | **100% Verified** | Pengecekan rutin status `last_http_status` domain via `url_health_check_worker`. |
| **2. CSR Department & Official Email** | **>80% Coverage** | Ekstraksi email resmi (`csr@company.com`, `tjsl@company.com`) dari halaman kontak/newsroom via regex + AI. |
| **3. CSR Budget Range & History** | **>70% Coverage** | Diambil dari *Sustainability Report* PDF & laporan tahunan RUPS. |
| **4. CSR Focus Pillars & SDGs** | **>90% Coverage** | Klasifikasi AI 3-Level (*Pendidikan, Lingkungan, UMKM, Kesehatan, Infrastruktur*). |
| **5. Vector Embeddings (1536-dim)** | **100% Generated** | Regenerasi *vector embedding* dari rangkuman sinyal untuk mempercepat pencocokan (*Match Score*) dengan proposal NGO. |

### B. Pembentukan Grafik Hubungan (CSR Relationship Graph)
```text
Company (BUMN / Tbk / Swasta)
   │
   ├── NGO Partner      ➔ Terhubung ke LAZ / Yayasan Mitra (BAZNAS, Rumah Zakat, dll)
   ├── Program Event    ➔ Terhubung ke Aktivitas CSR Lapangan
   └── Opportunity Flag ➔ Marked "opportunity_alert = true" jika membuka pengajuan proposal
```
