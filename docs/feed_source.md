Untuk **CSR Intelligence Platform** sebaiknya diperluas karena perusahaan dan media sering memakai istilah yang berbeda untuk aktivitas yang sebenarnya berkaitan.


### 1. Istilah utama CSR

* CSR
* Corporate Social Responsibility
* Corporate Responsibility
* Tanggung Jawab Sosial
* Tanggung Jawab Sosial Perusahaan
* TJSL
* Tanggung Jawab Sosial dan Lingkungan
* TJSL Perusahaan
* Program TJSL
* Program CSR
* Kegiatan CSR

### 2. Sustainability / ESG

* Sustainability
* Sustainable Development
* Sustainability Program
* Sustainability Initiative
* ESG
* Environmental, Social, Governance
* ESG Program
* ESG Initiative
* ESG Commitment
* Sustainable Business
* Sustainability Report
* ESG Report
* Sustainability Strategy
* Sustainable Investment
* Green Initiative
* Social Impact

**Catatan:** ESG lebih luas daripada CSR. Jadi jangan menganggap semua berita ESG sebagai CSR. Lebih baik AI Anda menentukan relevansinya.

### 3. Community & social development

* Community Development
* Community Empowerment
* Pemberdayaan Masyarakat
* Pengembangan Masyarakat
* Community Investment
* Social Investment
* Social Development
* Social Impact
* Social Program
* Program Sosial
* Pembangunan Sosial
* Pengembangan Ekonomi Masyarakat
* Peningkatan Kesejahteraan Masyarakat
* Kemandirian Masyarakat
* Desa Binaan
* Kampung Binaan
* Masyarakat Binaan

### 4. Lingkungan

* Program Lingkungan
* Environmental Program
* Environmental Initiative
* Environmental Conservation
* Konservasi Lingkungan
* Pelestarian Lingkungan
* Penghijauan
* Reboisasi
* Penanaman Pohon
* Pengelolaan Sampah
* Circular Economy
* Ekonomi Sirkular
* Waste Management
* Climate Action
* Climate Program
* Net Zero
* Dekarbonisasi
* Biodiversity
* Konservasi Keanekaragaman Hayati

### 5. Ekonomi / UMKM

Ini menurut saya **sangat penting** untuk database Anda.

* Pemberdayaan UMKM
* Pengembangan UMKM
* UMKM Binaan
* UMKM Mitra
* Entrepreneur Development
* Economic Empowerment
* Economic Development
* Kewirausahaan
* Pengembangan Usaha
* Pembinaan Usaha
* Local Economic Development
* Livelihood Program
* Peningkatan Pendapatan Masyarakat

### 6. Pendidikan

* Program Pendidikan
* Education Program
* Education Initiative
* Beasiswa
* Scholarship
* Bantuan Pendidikan
* Sekolah Binaan
* Sekolah Mitra
* Literasi
* Digital Literacy
* Pendidikan Vokasi
* Pelatihan
* Capacity Building
* Pengembangan SDM

### 7. Kesehatan

* Program Kesehatan
* Health Program
* Health Initiative
* Bantuan Kesehatan
* Pelayanan Kesehatan
* Kesehatan Masyarakat
* Community Health
* Medical Assistance
* Stunting
* Penanganan Stunting
* Gizi
* Sanitasi
* Air Bersih

### 8. Bantuan / philanthropy

Ini juga penting karena banyak perusahaan tidak menyebut aktivitasnya sebagai CSR.

* Bantuan Perusahaan
* Bantuan Sosial
* Bantuan Kemanusiaan
* Bantuan Bencana
* Corporate Giving
* Corporate Philanthropy
* Philanthropic Program
* Donation
* Donasi
* Charity
* Charitable Giving
* Humanitarian Assistance
* Disaster Relief
* Emergency Relief
* Community Giving

### 9. Partnership / collaboration

**Ini sangat penting untuk fitur nomor 4 Anda: perusahaan yang membuka partnership.**

* CSR Partnership
* Partnership CSR
* Corporate Partnership
* Strategic Partnership
* Community Partnership
* NGO Partnership
* Nonprofit Partnership
* Kemitraan Perusahaan
* Kemitraan CSR
* Kemitraan TJSL
* Kolaborasi CSR
* Kolaborasi Sosial
* Collaboration
* Open Partnership
* Partnership Opportunity
* Call for Partnership
* Open Collaboration
* Invitation for Partnership

### 10. Creating Shared Value

Tambahkan:

* Creating Shared Value
* CSV
* Shared Value
* Shared Value Creation
* Inclusive Business
* Inclusive Growth
* Social Enterprise
* Impact Business
* Impact Investment

### 11. Volunteerism

Ini belum ada di daftar Anda dan **menurut saya perlu ditambahkan**.

* Corporate Volunteering
* Employee Volunteering
* Employee Volunteer
* Volunteer Program
* Volunteerism
* Employee Engagement
* Relawan Perusahaan
* Relawan Karyawan
* Kegiatan Relawan
* Bakti Sosial

### 12. Program masyarakat / community relation

* Community Relations
* Community Engagement
* Stakeholder Engagement
* Public Engagement
* Social Engagement
* Community Outreach
* Hubungan Masyarakat
* Hubungan Komunitas
* Keterlibatan Masyarakat
* Dialog Masyarakat

---

# Yang paling penting: jangan jadikan semua keyword sama

Untuk sistem Anda, saya menyarankan **3 level taxonomy**.

### Level 1 — CSR relevance

```text
CSR
TJSL
Corporate Responsibility
Sustainability
ESG
Social Impact
Community Development
Corporate Philanthropy
```

### Level 2 — Activity

```text
Education
Health
Environment
UMKM
Economic Empowerment
Humanitarian
Disaster Relief
Community Development
Infrastructure
Digital Inclusion
```

### Level 3 — Action

```text
Donation
Partnership
Scholarship
Training
Volunteer
Infrastructure Development
Empowerment
Grant
Funding
Community Program
```

Jadi ketika crawler menemukan:

> **"PT ABC memberikan beasiswa kepada 500 siswa di Jawa Barat."**

AI tidak hanya menyimpan keyword `CSR`.

Ia mengubahnya menjadi:

```text
Company
  PT ABC

CSR Relevance
  HIGH

Activity
  Education

Action
  Scholarship

Location
  Jawa Barat

Beneficiary
  Students

Status
  ACTIVE

Event Type
  CSR Program
```

---

## Saya juga akan menambahkan keyword yang sangat penting untuk fitur "perusahaan membuka partnership"

Cari pola seperti:

```text
membuka kerja sama
membuka kemitraan
terbuka untuk kerja sama
mengajak kolaborasi
mengundang mitra
kesempatan kemitraan
peluang kerja sama
program kemitraan
mitra CSR
mitra TJSL
CSR partnership
partnership opportunity
open for partnership
call for partnership
call for proposal
open call
grant opportunity
funding opportunity
```

**`Call for Proposal`, `Grant`, dan `Funding Opportunity` sangat penting.**

Karena dari perspektif user SaaS Anda, berita:

> "Perusahaan X menjalankan CSR"

nilainya bagus.

Tetapi berita:

> **"Perusahaan X membuka kesempatan partnership untuk program pendidikan"**

nilainya **jauh lebih tinggi**.

Itu bisa langsung menjadi:

> 🔔 **OPPORTUNITY ALERT**

---

## Taxonomy awal yang saya rekomendasikan

Untuk MVP, saya tidak akan memasukkan ratusan keyword secara hard-code. Buat **keyword dictionary ±150–250 istilah**, kemudian gunakan keyword tersebut hanya sebagai **discovery layer**.

Pipeline:

```text
150–250 keywords
        ↓
Search / RSS / Crawler
        ↓
Raw articles
        ↓
Deduplication
        ↓
AI Classification
        ↓
CSR Relevance Score
        ↓
Entity Extraction
        ↓
CSR Event
        ↓
Company Profile
        ↓
Scoring
        ↓
Matching
        ↓
Alert
```

Dengan pendekatan ini, **keyword bukan "otak" sistem**. Keyword hanya digunakan untuk menemukan kandidat berita. **AI yang menentukan apakah berita tersebut benar-benar merupakan CSR intelligence.**

Dan saya akan membuat **CSR taxonomy sebagai data yang bisa berubah**, bukan enum yang ditanam permanen di source code. Ini akan sangat membantu ketika nanti Anda menemukan istilah baru dari berita Indonesia.
