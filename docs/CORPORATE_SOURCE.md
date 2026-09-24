
## Sumber corporate data yang saya sarankan

| Sumber                            | Data yang bisa diperoleh                               | Nilai untuk CSRmatics |
| --------------------------------- | ------------------------------------------------------ | --------------------- |
| **AHU Kementerian Hukum**         | Nama PT, profil perseroan, badan usaha                 | ⭐⭐⭐⭐⭐                 |
| **OSS**                           | Pelaku usaha, NIB, bidang usaha/KBLI, perizinan        | ⭐⭐⭐⭐⭐                 |
| **PROPER KLH/BPLH**               | Perusahaan peserta, sektor, lokasi, kinerja lingkungan | ⭐⭐⭐⭐⭐                 |
| **Website perusahaan**            | Profil, bisnis, lokasi, anak perusahaan                | ⭐⭐⭐⭐⭐                 |
| **Sustainability Report**         | CSR/TJSL, program, anggaran, wilayah, beneficiaries    | ⭐⭐⭐⭐⭐                 |
| **Annual Report**                 | Struktur grup, anak perusahaan, kegiatan sosial        | ⭐⭐⭐⭐⭐                 |
| **BUMN**                          | Daftar dan profil BUMN                                 | ⭐⭐⭐⭐                  |
| **OJK**                           | Emiten/perusahaan publik & disclosure                  | ⭐⭐⭐⭐                  |
| **Kementerian/instansi sektoral** | Perusahaan di sektor tertentu                          | ⭐⭐⭐⭐                  |
| **Media/news**                    | Aktivitas CSR terbaru                                  | ⭐⭐⭐⭐⭐                 |
| **Google/Search index**           | Discovery perusahaan & aktivitas                       | ⭐⭐⭐⭐                  |
| **LinkedIn**                      | Company identity & aktivitas publik                    | ⭐⭐⭐                   |
| **Portal tender/procurement**     | Aktivitas/proyek perusahaan                            | ⭐⭐⭐                   |
| **Website pemerintah daerah**     | Investasi/proyek/program perusahaan di daerah          | ⭐⭐⭐⭐                  |

### 1. AHU — sangat penting sebagai corporate registry

[AHU Profil Perusahaan](https://ahu.go.id/profil-pt?utm_source=chatgpt.com)

AHU menyediakan pencarian profil Perseroan Terbatas, dan juga memiliki layanan pencarian badan usaha non-badan hukum seperti CV, firma, dan persekutuan perdata. ([Ahu][1])

Untuk CSRmatics, ini cocok sebagai salah satu sumber **entity resolution**:

```text
PT ABC
ABC Indonesia
PT ABC Indonesia Tbk
ABC Group
       ↓
Entity Resolution
       ↓
Canonical Company
```

---

### 2. OSS — sumber business universe

[OSS RBA](https://oss.go.id/id?utm_source=chatgpt.com)

OSS menggunakan **NIB sebagai identitas resmi pelaku usaha** dan mengelompokkan kegiatan usaha berdasarkan tingkat risiko serta perizinan yang diperlukan. OSS juga menyediakan pencarian data profil pelaku usaha. ([OSS RBA][2])

Ini menarik untuk CSRmatics karena Anda bisa mengembangkan:

```text
Company
   +
NIB
   +
KBLI
   +
Business Sector
   +
Location
```

Kemudian:

```text
KBLI / sektor
      ↓
CSR relevance
      ↓
Company universe
```

**Catatan:** akses dan penggunaan data OSS harus mengikuti mekanisme serta ketentuan akses yang diberikan OSS; jangan mengasumsikan seluruh database NIB tersedia bebas untuk scraping.

---

### 3. PROPER — sangat menarik untuk CSRmatics

[PROPER KLH/BPLH](https://proper.menlhk.go.id/?utm_source=chatgpt.com)

Ini menurut saya salah satu sumber **high-value**.

PROPER memang berfokus pada kinerja pengelolaan lingkungan perusahaan. Pesertanya mencakup perusahaan dengan dampak lingkungan penting, perusahaan yang terdaftar di bursa, perusahaan berorientasi ekspor, atau produknya digunakan masyarakat luas. ([Proper][3])

Pada periode **2024–2025**, PROPER mencatat **5.476 perusahaan** peserta. ([Proper][4])

Artinya Anda bisa mendapatkan:

```text
Company
Location
Sector
Environmental performance
Community development information
```

Bahkan publikasi PROPER memuat perusahaan seperti Pertamina, PLN, Pupuk, Adaro, Indofood, Aqua, dan lain-lain. ([Proper][5])

Untuk CSRmatics:

```text
PROPER
   ↓
Corporate Universe
   ↓
Research
   ↓
CSR Signals
```

---

# 4. Website perusahaan

Ini jangan diremehkan.

Misalnya:

```text
company.com
├── About
├── Sustainability
├── CSR
├── TJSL
├── Community Development
├── Investor Relations
├── News
└── Sustainability Report
```

Justru **aktivitas CSR terbaru** sering ditemukan di website perusahaan sebelum masuk database pihak ketiga.

Ini sangat cocok dengan scraper yang sedang Anda bangun.

---

# 5. Sustainability Report

Ini mungkin merupakan **sumber paling kaya untuk CSR intelligence**, bukan untuk sekadar company master.

Contohnya:

```text
Sustainability Report
       ↓
CSR/TJSL programs
       ↓
Program
Region
Budget
Beneficiary
SDGs
Partner
Impact
       ↓
CSR Intelligence
```

Untuk CSRmatics saya akan memperlakukan Sustainability Report sebagai **evidence source**, bukan sekadar data company.

---

# 6. Annual Report

Annual Report berguna untuk:

* struktur perusahaan
* anak perusahaan
* kepemilikan
* kegiatan perusahaan
* strategi
* program sosial
* sustainability
* hubungan dengan masyarakat

Dan ini sangat penting untuk **Corporate Graph**.

Contoh:

```text
ABC Group
    │
    ├── PT ABC Energy
    ├── PT ABC Foundation
    ├── PT ABC Mining
    └── PT ABC Infrastructure
```

Jangan sampai CSRmatics menganggap keempatnya sebagai empat perusahaan yang sama sekali tidak berhubungan.

---

# 7. BUMN

Untuk BUMN, gunakan sumber pemerintah/BUMN sebagai layer tersendiri.

Misalnya:

```text
BUMN
 │
 ├── Holding
 ├── Subsidiary
 ├── Subholding
 └── Affiliate
```

Ini sangat berguna untuk corporate graph dan CSR/TJSL intelligence.

---

# 8. OJK

OJK sangat berguna untuk perusahaan publik dan disclosure pasar modal.

Namun untuk CSRmatics:

> **OJK + IDX = public-company layer**

bukan corporate universe keseluruhan.

Jadi:

```text
                COMPANY UNIVERSE
                       │
       ┌───────────────┼───────────────┐
       ▼               ▼               ▼
      AHU             OSS            PROPER
       │               │               │
       └───────────────┼───────────────┘
                       │
             ┌─────────┴─────────┐
             ▼                   ▼
           OJK/IDX          BUMN/Other
             │                   │
             └─────────┬─────────┘
                       ▼
               CANONICAL COMPANY
```

---

# 9. Media/news

Untuk **CSR Intelligence**, media bahkan lebih penting daripada corporate registry.

Contoh:

```text
News
 ↓
"PT XYZ membuka program beasiswa..."
 ↓
CSR Signal
 ↓
Entity Resolution
 ↓
Evidence
 ↓
Verification
```

Ini juga tempat Research Agent Anda sangat berguna.

---

# 10. LinkedIn

LinkedIn lebih cocok sebagai **discovery/enrichment source**, bukan source of truth.

Misalnya menemukan:

```text
PT XYZ
CSR Manager
Sustainability Manager
Corporate Affairs
Community Development
```

Kemudian AI mencari evidence dari sumber resmi.

Jangan menjadikan:

> "Posting LinkedIn mengatakan perusahaan melakukan CSR"

sebagai evidence tingkat tertinggi.

---

# Yang menarik: buat **Corporate Data Fusion Engine**

Daripada membuat scraper satu per satu lalu memasukkan hasil langsung ke `companies`, saya akan membuat arsitektur:

```text
                  SOURCES
                     │
       ┌─────────────┼──────────────┐
       │             │              │
      AHU           OSS           PROPER
       │             │              │
      OJK           IDX           BUMN
       │             │              │
 Company Website   Reports         News
       │             │              │
       └─────────────┼──────────────┘
                     ▼
             SOURCE SNAPSHOT
                     │
                     ▼
             ENTITY RESOLUTION
                     │
                     ▼
             CANONICAL COMPANY
                     │
             ┌───────┴────────┐
             ▼                ▼
       CORPORATE GRAPH    CSR SIGNALS
             │                │
             └────────┬───────┘
                      ▼
              CSR INTELLIGENCE
```

Ini **sangat cocok dengan desain v1.2 CSRmatics**, terutama karena desain tersebut sudah memisahkan `company`, `source`, dan `intelligence`, serta menggunakan canonical company + aliases + corporate relationship. 

### Saya akan membagi sumber CSRmatics menjadi 4 kategori

**A. Corporate Identity**

```text
AHU
OSS
OJK
IDX
BUMN
```

→ Menjawab **"siapa perusahaan ini?"**

**B. Corporate Structure**

```text
Annual Report
Company Website
OJK
BUMN
```

→ Menjawab **"perusahaan ini bagian dari grup mana?"**

**C. CSR Evidence**

```text
Sustainability Report
Annual Report
PROPER
Company Website
Government
```

→ Menjawab **"apa yang sudah dilakukan perusahaan?"**

**D. Real-time Intelligence**

```text
News
Press Release
Company Website
Social/LinkedIn
Search
```

→ Menjawab **"apa yang sedang terjadi sekarang?"**

Dan justru **D inilah yang membuat CSRmatics bukan sekadar database perusahaan**, tetapi menjadi **Corporate Intelligence Platform**.

[1]: https://ahu.go.id/profil-pt?utm_source=chatgpt.com "DITJEN AHU ONLINE | Profil Perusahaan"
[2]: https://oss.go.id/id?utm_source=chatgpt.com "OSS RBA - Sistem Perizinan Berusaha Terintegrasi Secara Elektronik"
[3]: https://proper.menlhk.go.id/proper/mekanisme?utm_source=chatgpt.com "Proper - Kementerian Lingkungan Hidup dan Kehutanan"
[4]: https://proper.menlhk.go.id/proper/berita/detail/401?utm_source=chatgpt.com "Proper - Kementerian Lingkungan Hidup dan Kehutanan"
[5]: https://proper.menlhk.go.id/propercms/uploads/magazine/docs/publikasi/proper-upload-03012019.pdf?utm_source=chatgpt.com "PROPER 2019"

---

## Live Implementation Status & Corporate Data Fusion Engine (v1.2)

Seluruh komponen utama arsitektur corporate data fusion dalam dokumen ini telah berhasil diimplementasikan 100% menggunakan data empiris real (**ZERO DATA MOCKING DIRECTIVE**):

| Komponen Sumber Data | Status | Tabel Database Utama | API Endpoints Utama | Fitur Kunci |
| :--- | :--- | :--- | :--- | :--- |
| **AHU Kemenkumham** | 🟢 Live | `public.ahu_registrations` (Migrasi 000043) | `GET /api/v1/ahu/search`<br>`GET /api/v1/ahu/:ahu_number`<br>`POST /api/v1/ahu/resolve` | Entity Resolution Engine (Canonical Name Matcher), SK Kemenkumham Validation |
| **OSS RBA BKPM** | 🟢 Live | `public.oss_nib_registrations` (Migrasi 000044) | `GET /api/v1/oss/search`<br>`GET /api/v1/oss/:nib`<br>`POST /api/v1/companies/:id/nib` | Universum Perizinan Usaha, NIB 13-Digit, Status Investasi (PMDN/PMA), Tingkat Risiko Usaha |
| **Dataset KBLI 2020 (BPS)** | 🟢 Live | `public.kbli_reference` (Migrasi 000042) | `GET /api/v1/kbli`<br>`GET /api/v1/kbli/search`<br>`GET /api/v1/kbli/:code` | 43 Kode KBLI 5-Digit BPS, Auto-mapping 1.557 Perusahaan Master, CSR Relevance Tier |
| **PROPER KLHK/BPLH** | 🟢 Live | `public.crawling_targets` & `public.company_esg_profiles` | `GET /api/v1/companies/:id` (ESG Profile) | Peringkat Lingkungan Resmi (Emas, Hijau, Biru, Merah, Hitam) + `source_quote` Grounding |
| **Corporate Graph** | 🟢 Live | `company.companies` (`parent_company_id`) | `GET /api/v1/companies/:id/hierarchy`<br>`GET /api/v1/companies/:id/subsidiaries`<br>`POST /api/v1/companies/auto-link-groups` | Hirarki Holding-Anak Perusahaan, 267 Anak Perusahaan Ter-link ke 8 Grup Holding Utama |

### 1. AHU Kemenkumham Entity Resolution & SK Registry
- **Tabel**: `public.ahu_registrations`
- **Tujuan**: Resolusi nama legal resmi perseroan (PT, PT Tbk, BUMN Persero, CV) serta verifikasi nomor SK Kemenkumham.
- **Entity Resolution Engine**: Endpoint `POST /api/v1/ahu/resolve` menerima nama variabel perusahaan dari scraping/input user (e.g. `"Astra International"`) dan mencocokkannya secara berjenjang (Exact Match -> Canonical Fuzzy Match -> Normalize PT/Tbk Prefix) untuk mengembalikan nama canonical resmi (`"PT Astra International Tbk"`) beserta ID perusahaan master.

### 2. OSS RBA Business Universe & NIB Registration
- **Tabel**: `public.oss_nib_registrations`
- **Tujuan**: Pencatatan NIB 13-digit pelaku usaha, klasifikasi risiko OSS (Tinggi, Menengah Tinggi, Menengah Rendah, Rendah), status investasi (PMDN/PMA), dan lokasi perizinan daerah.
- **Integrasi Master**: Menghubungkan NIB dan KBLI secara langsung ke entitas perusahaan master `company.companies(nib, kbli_code, headquarters)`.

### 3. Referensi KBLI 2020 Standard BPS
- **Tabel**: `public.kbli_reference`
- **Tujuan**: Standardisasi bidang usaha 5-digit BPS/OSS RBA serta penentuan bobot dampak CSR (`csr_relevance_default`).
- **Peta Cakupan**: Memuat 43 kode sektor utama (Pertambangan Minyak/Gas/Batu Bara/Nikel, Listrik/EBT, Kilang/Manufaktur, Perbankan/Asuransi, Telco/Software/AI, Konstruksi/Properti, Logistik/Maritim, Perkebunan Kelapa Sawit/HTI).
- **Auto-link**: 1.557 perusahaan master telah terhubung otomatis ke referensi KBLI berdasarkan kecocokan sektor dan kata kunci industri.

### 4. PROPER KLHK Rating Ingestion
- **Tabel**: `public.crawling_targets` & `public.company_esg_profiles`
- **Tujuan**: Ingesti otomatis peringkat kinerja lingkungan KLHK dari pengumuman resmi dan RSS feed.
- **Grounded Verification**: Pengambilan data ESG oleh LLM Worker mewajibkan adanya `source_quote` kutipan asli tanpa fabrikasi data sintetis.

### 5. Corporate Graph Holding-Subsidiary Hierarchy
- **Tabel**: `company.companies` (Kolom `parent_company_id`)
- **Tujuan**: Memetakan grup holding korporat besar di Indonesia agar aktivitas CSR anak perusahaan dapat diatributkan secara akurat ke grup holding parent.
- **Cakupan Auto-link**: 267 anak perusahaan terhubung ke 8 grup holding utama (PT Astra International Tbk, PT Pertamina (Persero), PT Perusahaan Listrik Negara (Persero), PT Telkom Indonesia (Persero) Tbk, PT Bank Mandiri (Persero) Tbk, PT Bank Rakyat Indonesia (Persero) Tbk, PT Pupuk Indonesia (Persero), MIND ID / Inalum).

