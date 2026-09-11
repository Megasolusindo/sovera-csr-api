Selain LinkedIn, **cukup banyak sumber** yang bisa Anda gunakan. Bahkan untuk produk Anda, saya akan menempatkan LinkedIn sebagai **secondary signal**, bukan sumber utama.

Saya akan membangun source ecosystem seperti ini:

| Sumber                          | Nilai | Data yang bisa diperoleh                    |
| ------------------------------- | ----- | ------------------------------------------- |
| **Website/newsroom perusahaan** | ⭐⭐⭐⭐⭐ | Program CSR terbaru                         |
| **Laporan TJSL perusahaan**     | ⭐⭐⭐⭐⭐ | Program, anggaran, lokasi, penerima manfaat |
| **Sustainability Report**       | ⭐⭐⭐⭐⭐ | Histori + ESG + target                      |
| **ANTARA & media nasional**     | ⭐⭐⭐⭐⭐ | Berita aktivitas CSR                        |
| **Media regional**              | ⭐⭐⭐⭐⭐ | CSR perusahaan di daerah                    |
| **Portal pemerintah/TJSL**      | ⭐⭐⭐⭐⭐ | Program & perusahaan mitra                  |
| **Website NGO/yayasan**         | ⭐⭐⭐⭐⭐ | Partnership & program                       |
| **OJK**                         | ⭐⭐⭐⭐  | Sustainability report                       |
| **BEI/IDX**                     | ⭐⭐⭐⭐  | Laporan perusahaan publik                   |
| **CSR/TJSL media khusus**       | ⭐⭐⭐⭐  | Berita & award                              |
| **Press release**               | ⭐⭐⭐⭐  | Aktivitas perusahaan                        |
| **LinkedIn**                    | ⭐⭐⭐⭐  | Early signal                                |
| **Instagram/Facebook/YouTube**  | ⭐⭐⭐   | Early signal                                |
| **CSR awards**                  | ⭐⭐⭐   | Identifikasi perusahaan aktif               |

### 1. Website perusahaan — **sumber nomor 1**

Ini yang paling saya prioritaskan.

Buat crawler yang memonitor:

```text
/news
/newsroom
/media
/press-release
/csr
/tjsl
/sustainability
/esg
/community
/social-impact
```

Contohnya Telkom secara langsung menerbitkan aktivitas TJSL di newsroom-nya. ([Telkom Indonesia][1])

Pelindo bahkan memiliki halaman khusus TJSL yang menjelaskan program pendidikan, ekonomi, dan kesejahteraan sosial. ([Pelindo][2])

---

### 2. Sustainability Report — **sangat bernilai**

Ini menurut saya salah satu **aset data terbesar** untuk produk Anda.

OJK memiliki repository **Laporan Keberlanjutan** yang dapat dicari berdasarkan institusi/sektor. ([Keuangan Berkelanjutan][3])

Dari laporan tersebut Anda bisa mendapatkan:

* program CSR/TJSL
* target
* realisasi
* anggaran
* lokasi
* penerima manfaat
* SDGs
* ESG
* community development
* environmental programs
* histori beberapa tahun

Jadi:

> **News = apa yang sedang terjadi sekarang.**
> **Sustainability Report = pemahaman mendalam tentang perusahaan.**

Keduanya harus digabung.

---

### 3. Laporan TJSL perusahaan

Ini bahkan lebih spesifik daripada sustainability report.

Misalnya PTPP menyediakan laporan TJSL tahun 2018–2025. ([PT PP (Persero) Tbk][4])

PT Timah juga menyediakan **TJSL Report** dan *Partnership & Community Development Report* beberapa tahun ke belakang. ([Timah][5])

Ini bisa menjadi sumber untuk membangun:

> **CSR History**

Misalnya:

```text
PT TIMAH

2019
├── UMKM
├── Pendidikan
└── Infrastruktur

2020
├── UMKM
├── Kesehatan
└── Lingkungan

...

2025
├── UMKM
├── Pendidikan
└── Lingkungan
```

Ini jauh lebih valuable daripada sekadar daftar berita.

---

### 4. Portal pemerintah / TJSL daerah

Ini juga sangat menarik.

Pemerintah daerah sering mengetahui:

* perusahaan yang aktif;
* program yang berjalan;
* lokasi program;
* penerima manfaat;
* jenis bantuan;
* perusahaan yang menjadi mitra.

Selain pemerintah daerah, **OJK dan kementerian** juga relevan untuk sustainability/ESG. OJK, misalnya, memiliki ketentuan dan panduan sustainability reporting untuk emiten, perusahaan publik, dan sektor jasa keuangan. ([OJK][6])

---

### 5. Website NGO/Yayasan

Ini saya beri prioritas sangat tinggi karena berhubungan langsung dengan fitur:

> **"Perusahaan yang membuka partnership."**

Contoh, sebuah yayasan dapat mempunyai halaman:

> "Kemitraan Perusahaan"

dan mencantumkan program yang siap didukung perusahaan. Contoh aktualnya EBI Foundation menampilkan program aktif, target penerima manfaat, anggaran, SDGs, dan mekanisme kerja sama. ([EBI Foundation][7])

Jadi Anda bisa mendeteksi:

```text
Company → NGO
Company → Program
NGO → Program
Program → Location
Program → CSR Category
```

Ini bisa menjadi **relationship graph**.

---

### 6. CSR/TJSL-specific media

Ada situs yang memang fokus pada CSR/TJSL.

Misalnya Masyarakat Pemantau CSR/TJSL mempunyai kanal berita khusus dan bahkan menyatakan melakukan monitoring perusahaan wajib CSR/TJSL. ([Masyarakat Pemantau CSR/TJSL][8])

Sumber seperti ini bagus untuk menemukan:

* perusahaan aktif;
* aktivitas CSR;
* isu CSR;
* penghargaan;
* monitoring;
* perkembangan regulasi.

---

### 7. BAZNAS / lembaga filantropi

Ini menarik karena mereka sering menjadi **intermediary antara perusahaan dan program sosial**.

Contohnya BAZNAS secara eksplisit menawarkan kemitraan CSR/TJSL perusahaan untuk program lingkungan, pendidikan, UMKM, kesehatan, ketahanan pangan, dan kemanusiaan. ([Baznas][9])

Jadi database Anda sebaiknya juga memonitor:

* BAZNAS
* Dompet Dhuafa
* Rumah Zakat
* ACT/organisasi sejenis yang masih aktif
* yayasan besar
* lembaga filantropi
* NGO nasional
* NGO regional

---

# Yang saya sarankan untuk MVP

Jangan langsung crawler 1.000 website.

Mulai dari:

### Tier 1 — 50–100 sumber

**30–40 media**

*

**30–50 perusahaan besar**

*

**10–20 NGO/yayasan**

*

**10 pemerintah/portal**

Lalu:

```text
100 Sources
     ↓
RSS / Sitemap / Search / Crawler
     ↓
~200–500 raw articles/day
     ↓
Deduplication
     ↓
AI CSR Classification
     ↓
~30–80 high-value events/day
```

### Dan database Anda bukan cuma `news`

Saya akan membuat minimal 4 entity:

```text
Company
   │
   ├── CSR Profile
   ├── CSR History
   ├── CSR Score
   │
   └── CSR Events
          │
          ├── News
          ├── Program
          ├── Partnership
          ├── Funding
          └── Opportunity
```

Lalu sumbernya:

```text
News              → Media
CSR Event         → Company website
CSR History       → Sustainability Report
TJSL              → TJSL Report
Partnership       → NGO / Company
Opportunity       → NGO / Company / Government
Early Signal      → LinkedIn / Social Media
```

**Ini menurut saya jauh lebih kuat.** Anda tidak sedang membangun "website berita CSR", tetapi **mesin pengumpulan dan normalisasi intelligence tentang hubungan perusahaan ↔ program sosial ↔ NGO**.

Dan yang paling bernilai untuk pelanggan Anda akhirnya bukan beritanya, melainkan:

> **"Perusahaan mana yang saat ini paling mungkin menjadi partner untuk proposal saya?"**

Itulah data yang bisa Anda monetisasi.

[1]: https://www.telkom.co.id/sites/berita/id_ID/news/61-tahun-telkom-ajak-karyawan-terlibat-aktif-dalam-program-sosial-dan-lingkungan-3839?utm_source=chatgpt.com "HUT ke-61, Telkom Ajak Karyawan Sukseskan Program TJSL | Telkom"
[2]: https://www.pelindo.id/page/tjsl?utm_source=chatgpt.com "TJSL - Pelindo"
[3]: https://keuanganberkelanjutan.ojk.go.id/keuanganberkelanjutan/FileList/Index/11/laporan-keberlanjutan?utm_source=chatgpt.com "Laporan Keberlanjutan Sustainable Finance OJK"
[4]: https://www.ptpp.co.id/id/keberlanjutan/laporan-tjsl?utm_source=chatgpt.com "PTPP - Laporan TJSL"
[5]: https://timah.com/blog/laporan/tjsl-report.html?utm_source=chatgpt.com "TJSL Report | PT TIMAH TBK"
[6]: https://ojk.go.id/id/Publikasi/Roadmap-dan-Pedoman/Sektor-Jasa-Keuangan/Keuangan-Berkelanjutan/Pages/Petunjuk-Teknis-Bagi-Perusahaan-Pembiayaan-dan-Perusahaan-Pembiayaan-Syariah-POJK-51-2017.aspx?utm_source=chatgpt.com "Petunjuk Teknis Bagi Perusahaan Pembiayaan dan Perusahaan Pembiayaan Syariah Terkait Implementasi POJK 51/POJK.03/2017"
[7]: https://ebifoundation.id/csr?utm_source=chatgpt.com "Kemitraan Perusahaan | EBI Foundation"
[8]: https://pemantau-csr.org/berita.php?utm_source=chatgpt.com "Berita & Opini - Masyarakat Pemantau CSR/TJSL"
[9]: https://baznas.go.id/csr/tjsl?utm_source=chatgpt.com "Csr/tjsl - BAZNAS"
