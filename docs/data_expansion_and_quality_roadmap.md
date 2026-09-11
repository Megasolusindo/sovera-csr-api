# Strategic Roadmap: Ekspansi & Peningkatan Kualitas Data Corporate CSR

Dokumen ini berisi panduan strategis dan cetak biru teknis (*technical blueprint*) untuk **menambah data korporasi baru (expansion)** serta **meningkatkan kualitas dan kedalaman data existing (enrichment)** secara berkelanjutan tanpa melanggar *Zero Data Mocking Directive* pada platform **Sovera CSR Intelligence**.

---

## 🏛️ 1. Strategi Penambahan Data Corporate Baru (*Corporate Expansion*)

Untuk mengekspansi data perusahaan dari **4.502 entitas terverifikasi** saat ini menuju Tier 2 & Tier 3 (25.000+ korporasi nasional & daerah):

```text
               SUMBER DATA KORPORASI RESMI
  ┌───────────────────────────────────────────────────┐
  │ 1. BEI / IDX Disclosures (Situs idx.co.id)         │
  │ 2. Kemenkumham / AHU Online Business Registry     │
  │ 3. Direktori Kementerian BUMN & Pemda TJSL        │
  └─────────────────────────┬─────────────────────────┘
                            │
                            ▼
              PIPELINE VALIDASI EMPIRIS
  ┌───────────────────────────────────────────────────┐
  │ • Check Canonical Name & Legal Entity (PT/Tbk)   │
  │ • Check Reachable Official Domain (urlverifier)   │
  │ • Auto-generate Slug & Alias Keywords             │
  └─────────────────────────┬─────────────────────────┘
                            │
                            ▼
               INSERT DATABASE (master.companies)
```

### Langkah-Langkah Konkret Eksekusi
1. **Scraper Otomatis BEI / IDX Disclosures**:
   - Memonitor emiten baru yang IPO di Bursa Efek Indonesia (`idx.co.id`) untuk mendaftarkan kode ticker & profil emiten baru secara *real-time*.
2. **Validator Website Resmi (`urlverifier`)**:
   - Setiap kandidat perusahaan baru wajib lolos verifikasi HTTP Reachability (`2xx`/`3xx` status) sebelum masuk ke database master `companies`.
3. **Ingestion Portal Pemda & BUMN**:
   - Mengambil direktori anak/cucu perusahaan BUMN & mitra TJSL Pemda yang terdaftar resmi.

---

## 🔍 2. Peningkatan Kualitas Data Existing (*Data Enrichment & Depth*)

Untuk memperdalam data 4.502 perusahaan yang sudah ada agar menghasilkan **Corporate CSR Intelligence berharga tinggi**:

### A. Penambangan Data Laporan Keberlanjutan (*Sustainability Report & PDF Extractor*)
- **Data yang Diperoleh**: Anggaran CSR tahunan riil, realisasi dana, lokasi program, jumlah penerima manfaat, pilar SDGs, dan topik material ESG.
- **Strategi Teknis**: Membangun *PDF Text & Table Extractor* untuk mengunduh Laporan Keberlanjutan (POJK 51 OJK) dari situs resmi perusahaan/OJK, lalu mengekstrak tabel keuangan CSR-nya menggunakan Gemini AI.

### B. Enrichment Newsroom Perusahaan & Press Release
- **Data yang Diperoleh**: Aktivitas CSR terbaru (*News & CSR Events*).
- **Strategi Teknis**: Mengaktifkan crawler pada jalur URL newsroom korporasi (`/news`, `/press-release`, `/csr`, `/tjsl`, `/sustainability`, `/media`) dengan strategi *Native HTTP Fetch* (kecepatan ~100ms per halaman).

### C. Discovery Sinyal Awal via LinkedIn Sensor
- **Data yang Diperoleh**: Sinyal awal kegiatan CSR & penemuan nama PIC/kontak CSR.
- **Strategi Teknis**: Memanfaatkan Google News RSS Indexing (`site:linkedin.com <Nama_Perusahaan> TJSL OR CSR OR Beasiswa`) untuk menangkap postingan LinkedIn tanpa perlu scraping login yang berisiko.

---

## 📊 3. Matriks Peningkatan Kualitas Data (*Data Quality Metrics*)

Setiap entitas perusahaan di database harus diperkaya hingga memenuhi 5 indikator kelengkapan (*Intelligence Completeness Score*):

| Indikator Kualitas Data | Target Kelengkapan | Sumber Data & Metode Verifikasi |
| :--- | :---: | :--- |
| **1. Official Domain & HTTP Health** | **100% Verified** | Pengecekan rutin status `last_http_status` domain via `url_health_check_worker`. |
| **2. CSR Department & Official Email** | **>80% Coverage** | Ekstraksi email resmi (`csr@company.com`, `tjsl@company.com`) dari halaman kontak/newsroom via regex + AI. |
| **3. CSR Budget Range & History** | **>70% Coverage** | Diambil dari *Sustainability Report* PDF & laporan tahunan RUPS. |
| **4. CSR Focus Pillars & SDGs** | **>90% Coverage** | Klasifikasi AI 3-Level (*Pendidikan, Lingkungan, UMKM, Kesehatan, Infrastruktur*). |
| **5. Vector Embeddings (1536-dim)** | **100% Generated** | Regenerasi *vector embedding* dari rangkuman sinyal untuk mempercepat pencocokan (*Match Score*) dengan proposal NGO. |

---

## 🕸️ 4. Pembentukan Grafik Hubungan (*CSR Relationship Graph*)

Data korporasi akan bernilai monetisasi maksimal jika terhubung ke ekosistem NGO & Yayasan:

```text
Company (BUMN / Tbk / Swasta)
   │
   ├── NGO Partner      ➔ Terhubung ke LAZ / Yayasan Mitra (BAZNAS, Rumah Zakat, dll)
   ├── Program Event    ➔ Terhubung ke Aktivitas CSR Lapangan
   └── Opportunity Flag ➔ Marked "opportunity_alert = true" jika membuka pengajuan proposal
```

---

## 🗓️ 5. Roadmap Eksekusi Implementasi

1. **Tahap 1 (Jangka Pendek: 1-2 Minggu)**:
   - Jalankan `url_health_check_worker` untuk memverifikasi ulang seluruh URL website 4.502 perusahaan existing.
   - Aktifkan pemindaian rutin target `NEWS_RSS` & `COMPANY_WEBSITE` untuk mengisi `company_signals` harian.

2. **Tahap 2 (Jangka Menengah: 1 Bulan)**:
   - Bangun pipeline **Sustainability Report PDF Extractor** untuk mengambil data anggaran riil dari OJK / IDX.
   - Tambahkan fitur **Automated Partnership Matchmaker** yang mencocokkan `opportunity_alert` sinyal perusahaan dengan program `institution_programs` milik NGO.
