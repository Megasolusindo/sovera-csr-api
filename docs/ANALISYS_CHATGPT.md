Kalau **CSRmatics** adalah SaaS dengan **tenant = lembaga kemanusiaan/NGO/yayasan**, saya akan memposisikannya bukan sekadar sebagai “direktori perusahaan CSR”, tetapi sebagai **CSR Intelligence & Partnership Management Platform**.

Intinya: lembaga kemanusiaan menggunakan CSRmatics untuk **menemukan perusahaan yang potensial, memahami aktivitas CSR mereka, menemukan peluang partnership, lalu mengelola proses pendekatan sampai menjadi kerja sama**.

### 1. Modul utama yang saya sarankan

| Modul                    | Fitur utama                                           | Prioritas |
| ------------------------ | ----------------------------------------------------- | --------: |
| **Company Intelligence** | Database perusahaan, profil, industri, lokasi, kontak |     🔴 P0 |
| **CSR Intelligence**     | Riwayat CSR, program, bidang CSR, lokasi program      |     🔴 P0 |
| **Opportunity**          | Perusahaan yang sedang membuka partnership/program    |     🔴 P0 |
| **Company Matching**     | Matching perusahaan ↔ program lembaga                 |     🔴 P0 |
| **Proposal Management**  | Proposal, status pengajuan, follow-up                 |     🔴 P0 |
| **Monitoring**           | Pantau perusahaan tertentu & aktivitas terbaru        |     🔴 P0 |
| **Alert**                | Notifikasi perusahaan/program baru yang relevan       |     🔴 P0 |
| **CRM Partnership**      | Pipeline hubungan dengan perusahaan                   |     🟠 P1 |
| **Organization Profile** | Profil lembaga & program yang ditawarkan              |     🔴 P0 |
| **Analytics**            | Statistik peluang, perusahaan, partnership            |     🟠 P1 |
| **Team & Collaboration** | User, role, assignment                                |     🟠 P1 |
| **Report**               | Export Excel/PDF, laporan aktivitas                   |     🟡 P2 |

---

# 2. Company Intelligence

Ini sebaiknya menjadi **core database CSRmatics**.

Setiap perusahaan memiliki halaman:

**Company Profile**

* Nama perusahaan
* Logo
* Website
* Industri
* Grup perusahaan
* Anak perusahaan
* Tahun berdiri
* Kantor pusat
* Wilayah operasi
* Status perusahaan
* Bursa/privat
* Kontak publik
* Social media
* LinkedIn
* PIC CSR/ESG jika tersedia
* Email/kontak publik
* Sumber data

Kemudian:

### CSR Profile

* Fokus CSR
* Fokus ESG
* Community development
* Pendidikan
* Kesehatan
* Lingkungan
* Pemberdayaan ekonomi
* Kemanusiaan
* Disaster response
* Filantropi
* dll.

Yang menarik adalah **CSR History**.

Contoh:

> PT ABC
> **2026**
>
> * Program pendidikan di Jawa Barat
> * Bantuan bencana di Sumatera
> * Program pemberdayaan UMKM

> **2025**
>
> * Program kesehatan
> * Beasiswa
> * Program lingkungan

Ini akan menjadi **aset data yang sangat valuable**.

---

# 3. CSR Intelligence

Jangan hanya menyimpan perusahaan.

Simpan **aktivitas CSR** sebagai entitas terpisah.

Misalnya:

```text
Company
   │
   ├── CSR Program
   │      ├── Education
   │      ├── Health
   │      ├── Environment
   │      └── Community Development
   │
   ├── CSR History
   │
   └── CSR Opportunity
```

Setiap program:

* Nama program
* Perusahaan
* Tahun
* Tanggal
* Lokasi
* Kategori
* Target penerima manfaat
* Nilai/dana jika tersedia
* Partner pelaksana
* Status
* Sumber informasi
* URL sumber
* Evidence
* Related news

Ini yang nantinya memungkinkan fitur seperti:

> **“Perusahaan mana yang aktif melakukan program kesehatan di Jawa Barat?”**

---

# 4. Opportunity Intelligence ⭐

Menurut saya ini salah satu fitur **paling penting untuk willingness-to-pay**.

Misalnya CSRmatics menemukan:

> 🟢 **PT XYZ membuka peluang partnership**
> Fokus: Pendidikan
> Wilayah: Jawa Tengah
> Target: Anak sekolah
> Deadline: 30 September 2026

Tenant bisa:

**Save Opportunity**

atau:

**Match with My Program**

---

# 5. Organization Profile

Tenant bukan hanya user.

Tenant harus memiliki **profil lembaga**.

Contoh:

### Yayasan ABC

**Bidang**

* Pendidikan
* Kesehatan
* Kemanusiaan

**Wilayah operasi**

* Jawa Barat
* Banten
* Jakarta

**Program**

* Beasiswa
* Klinik gratis
* Pemberdayaan UMKM

**Target beneficiary**

* Anak
* Lansia
* Penyandang disabilitas
* Masyarakat miskin

**SDGs**

* SDG 1
* SDG 3
* SDG 4
* SDG 8

**Legalitas**

* Akta
* SK Kemenkumham
* NPWP
* dll.

Dengan data ini, CSRmatics bisa melakukan **matching otomatis**.

---

# 6. Company ↔ Program Matching ⭐⭐⭐

Ini bisa menjadi fitur pembeda CSRmatics.

Misalnya tenant memasukkan:

> Program: Beasiswa Anak Tidak Mampu
> Lokasi: Jawa Barat
> Budget: Rp500 juta
> Beneficiary: 500 siswa
> SDG: 4
> Category: Education

CSRmatics menghasilkan:

### Recommended Companies

| Company |   Match |
| ------- | ------: |
| PT ABC  | **94%** |
| PT XYZ  | **89%** |
| PT DEF  | **82%** |
| PT GHI  | **76%** |

Dan jelaskan **kenapa**:

> **94% Match**
>
> ✓ Perusahaan aktif di bidang pendidikan
> ✓ Pernah melakukan program beasiswa
> ✓ Memiliki operasi di Jawa Barat
> ✓ Memiliki partnership dengan NGO
> ✓ Program serupa dilakukan 2025

Ini jauh lebih bernilai daripada sekadar database perusahaan.

---

# 7. Monitoring

Tenant bisa melakukan:

> **Monitor PT ABC**

Kemudian CSRmatics memantau:

* CSR baru
* Press release
* Berita CSR
* Program baru
* Partnership
* Sustainability report
* ESG report
* Recruitment CSR/ESG
* Opportunity
* perubahan profil

Dashboard:

```text
PT ABC
────────────────────────

CSR Activity
██████████████████  High

Last Activity
2 days ago

New Programs
3

Partnership Opportunities
1

CSR Focus
Education
Environment
Community Development
```

---

# 8. Alert System ⭐

Contoh:

> 🔔 **New CSR Opportunity**

> PT XYZ baru saja mengumumkan program pemberdayaan masyarakat di Jawa Barat.

atau:

> 🔔 **Potential Match**

> Program Anda "Pemberdayaan UMKM Desa" memiliki **91% match** dengan PT ABC.

Channel:

* In-app
* Email
* WhatsApp — bisa menjadi fitur premium
* Telegram

---

# 9. Partnership CRM

Setelah menemukan perusahaan, jangan berhenti di intelligence.

Tenant perlu mengelola proses:

```text
Prospect
   ↓
Research
   ↓
Contacted
   ↓
Meeting
   ↓
Proposal Sent
   ↓
Negotiation
   ↓
Approved
   ↓
Partnership
   ↓
Completed
```

Mirip CRM tetapi khusus **CSR partnership**.

Setiap company:

* Contact
* PIC
* Interaction
* Meeting
* Proposal
* Follow-up
* Notes
* Documents
* Status

---

# 10. Proposal Management

Misalnya tenant mempunyai:

**Program: Sekolah Untuk Semua**

Mereka dapat menyimpan:

* Proposal
* Executive summary
* Budget
* Beneficiary
* Location
* Impact
* SDGs
* Photos
* Supporting documents

Kemudian:

**Submit to Company**

atau minimal:

**Track Proposal**

Status:

```text
Draft
↓
Submitted
↓
Under Review
↓
Meeting
↓
Revision
↓
Approved
↓
Rejected
```

---

# 11. Company Scoring

Ini fitur yang sebelumnya Anda pikirkan dan menurut saya **bagus sekali**.

Buat:

### CSR Potential Score

**PT ABC — 87/100**

Komponen:

```text
CSR Activity       92
Partnership        85
Budget Potential   80
NGO Engagement     95
Geographic Fit     88
Recent Activity    90
```

Tapi jangan sekadar satu angka.

Berikan:

> **Why this company matters**

misalnya:

> “PT ABC sangat potensial karena dalam 12 bulan terakhir aktif menjalankan 8 program sosial dan 3 di antaranya melibatkan NGO eksternal.”

---

# 12. Search yang sangat powerful

Search jangan hanya:

> Cari perusahaan

Tetapi gunakan **natural language search**.

Contoh:

> "Cari perusahaan yang aktif CSR pendidikan di Jawa Barat"

> "Perusahaan yang pernah bekerja sama dengan NGO"

> "Perusahaan yang aktif memberikan bantuan bencana"

> "Perusahaan tambang yang memiliki program pemberdayaan masyarakat"

> "Perusahaan yang melakukan CSR kesehatan di sekitar Kalimantan"

Ini bisa menjadi **AI layer CSRmatics**.

---

# 13. AI Assistant

Saya justru melihat ini sebagai fitur yang sangat potensial.

Tenant bisa bertanya:

> **“Perusahaan mana yang paling cocok untuk program saya?”**

AI menjawab:

1. PT ABC — 94%
2. PT XYZ — 89%
3. PT DEF — 85%

dan menjelaskan alasannya berdasarkan data CSRmatics.

Pertanyaan lain:

> “Apa saja program CSR PT ABC dalam 2 tahun terakhir?”

> “Siapa partner NGO PT XYZ?”

> “Apakah PT ABC pernah mendanai program kesehatan?”

> “Buatkan shortlist 20 perusahaan untuk program ini.”

---

# 14. Dashboard tenant

Dashboard jangan terlalu kompleks.

Saya akan membuat:

```text
Good morning, Yayasan ABC

──────────────────────────────────

Potential Companies       1,248
New Opportunities            27
New CSR Activities            84
Active Partnerships            8

──────────────────────────────────

🎯 Recommended Companies

PT ABC                 94%
PT XYZ                 91%
PT DEF                 88%

──────────────────────────────────

🔥 New Opportunities

Education — PT XYZ
Health — PT ABC
Disaster — PT DEF

──────────────────────────────────

📌 Your Pipeline

Prospect        42
Contacted       18
Proposal         9
Negotiation      4
Partnership      8
```

---

# 15. Fitur multi-tenant

Karena tenant Anda adalah lembaga kemanusiaan, bagian ini penting.

Struktur:

```text
CSRmatics
│
├── Tenant A - Yayasan ABC
│     ├── Users
│     ├── Programs
│     ├── Prospects
│     ├── Proposals
│     └── Partnerships
│
├── Tenant B - NGO XYZ
│     ├── Users
│     ├── Programs
│     ├── Prospects
│     └── Partnerships
│
└── Tenant C - Lembaga DEF
```

Data yang **global**:

```text
Companies
CSR Programs
CSR News
CSR Opportunities
Industry
Locations
SDGs
```

Data yang **tenant-specific**:

```text
Organization
Users
Programs
Saved Companies
Saved Opportunities
Proposals
CRM
Notes
Contacts
Pipeline
Reports
```

Ini penting untuk desain PostgreSQL Anda.

---

# 16. Fitur yang menurut saya jangan dibangun dulu

Karena Anda solo developer, saya **tidak menyarankan langsung membangun semuanya**.

### MVP CSRmatics

Saya akan fokus hanya pada:

```text
1. Company Database
        ↓
2. CSR Intelligence
        ↓
3. CSR Opportunity
        ↓
4. Organization Profile
        ↓
5. Program Profile
        ↓
6. Company ↔ Program Matching
        ↓
7. Monitoring
        ↓
8. Alert
```

Kemudian tahap kedua:

```text
9. Partnership CRM
10. Proposal Management
11. Analytics
12. Team Collaboration
```

Tahap ketiga:

```text
13. AI Assistant
14. AI Research
15. Natural Language Search
16. AI Proposal Recommendation
```

---

## Positioning yang saya rekomendasikan

Dengan fitur-fitur tersebut, saya **tidak akan memasarkan CSRmatics sebagai "direktori perusahaan CSR"**.

Direktori terdengar seperti database biasa.

Lebih kuat:

> **CSR Intelligence Platform for Humanitarian Organizations**

atau untuk pasar Indonesia:

> **Platform CSR Intelligence untuk Lembaga Kemanusiaan**

Dan value proposition-nya:

> **Temukan perusahaan yang tepat, pahami aktivitas CSR mereka, dan temukan peluang partnership untuk program Anda.**

Ini juga membuat model bisnis lebih masuk akal: tenant tidak membayar sekadar untuk **melihat database perusahaan**, tetapi membayar untuk **mendapatkan peluang partnership yang relevan**.
