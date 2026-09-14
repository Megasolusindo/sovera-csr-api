Secara konseptual, rancangan **two-sided platform** ini sangat solid dan memvalidasi model bisnis yang jauh lebih bernilai dibanding direktori statis. Namun, jika Anda mengeksekusinya sendirian sebagai **solo developer**, scope saat ini memiliki beberapa celah kritis yang perlu disederhanakan agar tidak terjebak dalam *execution trap*.

Berikut evaluasi objektif dan rekomendasi penyesuaian untuk desain tersebut:

---

### Titik Kuat Desain (The Wins)

* **Solusi Cold-Start Melalui Claim Profile:** Memulai dengan data publik lalu menyediakan fitur `Claim Profile` bagi korporasi adalah strategi cerdas untuk memecahkan masalah *chicken-and-egg*. Korporasi tidak datang ke platform kosong, melainkan ke halaman profil perusahaannya yang sudah ada.


* **Positioning Berbasis Intelligence:** Menjual *CSR Intelligence & Matching* jauh lebih mudah dimonetisasi daripada sekadar *database kontak*.


* **Struktur Arsitektur Tunggal:** Keputusan memakai **satu platform multi-tenant** (bukan dua aplikasi terpisah) dengan pembagian role `organization`, `corporate`, dan `admin` sangat tepat untuk efisiensi basis kode.



---

### Titik Rawan & Risiko Eksekusi

| Area Risiko | Celah pada Desain Saat Ini | Rekomendasi Mitigasi |
| --- | --- | --- |
| **Beban Fitur MVP** | Dokumen mencakup CRM, Proposal Workflow, Matching Engine, Alert, Scraping, hingga Monitoring secara bersamaan. Terlalu masif untuk solo engineer.

 | **Pangkas CRM internal.** Cukup sediakan discovery data, status proposal sederhana (Submitted/Under Review/Approved), dan direct contact PIC.

 |
| **Beban Verifikasi Legalitas** | Korporasi dan NGO di Indonesia sangat sensitif soal akuntabilitas dana dan legalitas (SK Kemenkumham, NPWP, izin pengumpulan dana).

 | Tambahkan status verifikasi bertahap. Jangan biarkan sembarang akun NGO submit proposal ke korporasi sebelum lolos kurasi dokumen dasar. |
| **Matching Engine Complexity** | Menghitung bobot persentase kecocokan (misal 94%) membutuhkan data terstruktur yang konsisten di kedua belah pihak.

 | Di tahap awal, gunakan filter berbasis tag/metadata deterministik (Kategori SDG, Provinsi/Wilayah, Rentang Budget) sebelum membuat scoring algoritmik yang rumit. |

---

### Strategi Eksekusi Solo Developer

**Tahap 1: Public Intelligence Engine (Inbound Magnet)**

* Bangun pipeline data agregasi profil CSR perusahaan dan riwayat program mereka.


* Sajikan sebagai halaman publik terindeks SEO (`/companies/pt-abc`) dengan tombol **Claim this Profile**.



**Tahap 2: Sisi NGO (Early Adopters)**

* Buka pendaftaran untuk NGO guna membuat profil organisasi, melengkapi dokumen legalitas, dan mencari perusahaan yang cocok.


* NGO membayar untuk fitur *Intelligence, Monitoring,* dan filter riset mendalam.



**Tahap 3: Sisi Corporate (Marketplace Liquidity)**

* Setelah korporasi mengklaim profilnya, berikan akses gratis untuk mempublikasikan program CSR/TJSL terbuka dan menerima proposal masuk.


* Korporasi hanya dikenakan biaya ketika membutuhkan analitik portofolio lanjutan atau pencarian NGO terverifikasi secara masif.

Fitur **Verifikasi Klaim Perusahaan (Company Claim)** adalah titik paling krusial dalam model ini. Jika prosesnya terlalu longgar, ada risiko akun palsu mengambil alih profil korporasi resmi (reputational damage). Jika terlalu birokratis, korporasi akan enggan melanjutkan proses *onboarding*.

Sebagai solo developer, Anda butuh mekanisme yang **aman, semi-otomatis, dan minim beban operasional manual**.

---

### 1. Metode Verifikasi Bertingkat (Tiered Verification)

Jangan meminta semua dokumen legalitas di awal. Gunakan 2 jalur verifikasi:

```text
User klik "Claim Company"
          │
          ├──> [Jalur Cepat / Otomatis] Email Domain Perusahaan (@telkom.co.id, @pertamina.com)
          │         └── Otomatis Approved / Tier 1 (Akses Terbatas: Edit Profil & Review Opportunity)
          │
          └──> [Jalur Manual / Fallback] Email Publik (gmail/yahoo) / Agensi / Konsultan CSR
                    └── Wajib upload Bukti Penugasan / ID Card 
                    └── Status: Pending Review (Admin Verification)

```

#### Jalur A: Domain-Matching (Otomatis / Tier 1)

* Jika profil PT ABC memiliki domain publik `abc.co.id`, dan user mendaftar menggunakan email `budi@abc.co.id`:
* Kirim *magic link* / token verifikasi ke email tersebut.
* Begitu diklik, profil langsung **Terverifikasi Sementara (Claimed - Domain Verified)**.
* Fitur instan: Perbaiki kontak CSR, edit info deskripsi, pantau proposal masuk.



#### Jalur B: Manual Assignment / Document Review (Tier 2 / Fallback)

Banyak tim CSR/TJSL atau yayasan korporat memakai email berbeda (misal `yayasan@holding.com` atau PIC menggunakan email umum).

* Syarat upload:
1. **Kartu Tanda Pengenal / ID Card Karyawan**
2. **Surat Keterangan Kerja / Surat Tugas PIC CSR** (template surat formal dari CSRmatics bisa disediakan).


* Status masuk antrean: `PENDING_VERIFICATION` di dashboard internal admin Anda.

---

### 2. State Machine & Status Klaim

Untuk menjaga integritas data publik, gunakan *state machine* yang ketat:

```text
[UNCLAIMED]
     │
     │  User submit claim
     ▼
[CLAIM_PENDING]  ──(Admin Tolak / Token Expired)──> [UNCLAIMED]
     │
     │  Email domain match OR Admin Approve
     ▼
[VERIFIED_CLAIMED]
     │
     │  Dispute terjadi (Perusahaan komplain akun palsu)
     ▼
[SUSPENDED / DISPUTED]

```

* **Data Isolation:** Sebelum klaim berstatus `VERIFIED_CLAIMED`, user **tidak boleh** menimpa data publik. Perubahan profil yang diajukan masuk ke tabel draft/staging.

---

### 3. Desain Skema Database (PostgreSQL)

Pisahkan entitas data perusahaan publik (`companies`) dari entitas pengguna yang mengklaimnya (`company_claims` dan `tenants`):

```sql
-- 1. Master Company (Data publik hasil crawler/intelligence)
CREATE TABLE companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    website VARCHAR(255),
    corporate_domain VARCHAR(100), -- contoh: 'pertamina.com'
    is_claimed BOOLEAN DEFAULT FALSE,
    claimed_by_tenant_id UUID,     -- Relasi ke tenant corporate setelah lolos verifikasi
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Log & Permohonan Klaim
CREATE TYPE claim_status AS ENUM ('PENDING', 'APPROVED', 'REJECTED', 'REVOKED');
CREATE TYPE claim_method AS ENUM ('CORPORATE_EMAIL', 'DOCUMENT_UPLOAD');

CREATE TABLE company_claims (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id), -- User/Akun Corporate pemohon
    requested_by_user_id UUID NOT NULL REFERENCES users(id),
    
    status claim_status DEFAULT 'PENDING',
    method claim_method NOT NULL,
    
    -- Metadata verifikasi
    work_email VARCHAR(255),
    document_proof_url TEXT, -- Path file S3/Storage jika upload ID / SK
    verification_notes TEXT, -- Catatan admin saat reject/approve
    
    verified_at TIMESTAMPTZ,
    verified_by_user_id UUID REFERENCES users(id), -- Admin CSRmatics
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Mencegah multiple pending claims pada company yang sama oleh user yang sama
    UNIQUE (company_id, tenant_id, status)
);

```

---

### 4. Alur UI/UX Klaim (Step-by-Step)

1. **Halaman Profil Publik (`/companies/pt-abc`):**
* Tampilkan banner kecil: *“Apakah Anda perwakilan resmi PT ABC? [Klaim Profil Ini]”*.


2. **Form Modal:**
* Konfirmasi nama lengkap & posisi (e.g. CSR Manager / Head of TJSL).
* Input Email Kerja. Jika domain cocok dengan `corporate_domain`, muncul pesan: *“Kami akan mengirimkan kode konfirmasi ke email kantor Anda.”*
* Jika menggunakan domain umum (@gmail.com dll), form otomatis meminta dokumen ID Card / Surat Tugas.


3. **Pemberitahuan Pasca Submit:**
* Jalur Domain: *“Cek email Anda dalam 10 menit untuk mengaktifkan akses.”*
* Jalur Dokumen: *“Tim kami akan memverifikasi dokumen Anda dalam 1x24 jam kerja.”*



---

### 5. Rekomendasi Fitur Anti-Fraud & Operasional Solo Dev

* **Block Public Email Providers:** Tolak otomatis domain gratisan (gmail, yahoo, outlook, icloud) dari verifikasi instan. Wajibkan jalur manual.
* **Webhook Notifikasi Admin:** Saat ada klaim jalur dokumen (`method = DOCUMENT_UPLOAD`), kirim notifikasi langsung ke bot **Telegram** pribadi Anda dengan tombol `[Approve]` atau `[Reject]` langsung via webhook. Anda tidak perlu repot membuka dashboard web setiap saat.
* **Dispute Mechanism:** Sediakan link *“Laporkan profil ini”* di halaman profil publik jika suatu saat terjadi klaim oleh pihak yang tidak sah.