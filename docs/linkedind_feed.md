**LinkedIn sangat berguna untuk CSR Intelligence**, bahkan menurut saya termasuk sumber yang bernilai tinggi. Tetapi ada perbedaan penting antara **menggunakan LinkedIn sebagai sumber informasi** dan **melakukan scraping LinkedIn**.

Saya menemukan contoh nyata bahwa perusahaan Indonesia memang mempublikasikan aktivitas CSR/TJSL di LinkedIn. Misalnya Pertamina Energy Terminal memposting program TJSL dan CSR award, WIKA mempublikasikan program TJSL, dan DAHANA mempublikasikan aktivitas CSR/TJSL. ([LinkedIn][1])

### Apa yang bisa Anda cari?

Untuk produk Anda, LinkedIn bisa menjadi sumber untuk menemukan:

* CSR
* TJSL
* Sustainability
* ESG
* Community Development
* Community Empowerment
* Social Impact
* Corporate Philanthropy
* Social Investment
* Creating Shared Value / CSV
* CSR Partnership
* NGO Partnership
* Program pendidikan
* Program kesehatan
* Program lingkungan
* Pemberdayaan UMKM
* Bantuan bencana
* Volunteer program
* CSR award

Contohnya sebuah posting bisa mengandung:

> "PT X menjalankan program pemberdayaan UMKM di Jawa Barat."

AI Anda kemudian mengubahnya menjadi:

```text
Company: PT X
Activity: Community Empowerment
Category: UMKM
Location: Jawa Barat
Status: Active
Source: LinkedIn
```

---

## Yang bahkan lebih menarik: LinkedIn bisa menemukan "siapa orangnya"

Ini berbeda dengan media.

Misalnya Anda menemukan:

> **Satya Nugraha — Division Head of Corporate Social Responsibility**

Posting tersebut dapat memberikan indikasi bahwa orang tersebut memang berada di fungsi CSR/TJSL. ([LinkedIn][2])

Jadi database Anda nantinya bisa memiliki:

```text
Company
   │
   ├── CSR Programs
   ├── CSR History
   ├── CSR Score
   │
   └── CSR Personnel
          ├── CSR Manager
          ├── CSR Head
          ├── Sustainability Manager
          └── Community Development
```

Ini **sangat bernilai untuk pelanggan Anda**.

---

# Tetapi jangan scrape LinkedIn

Ini bagian yang penting untuk proyek Anda.

LinkedIn secara eksplisit melarang penggunaan crawler, bot, script, atau proses otomatis untuk scraping/copying layanan mereka tanpa izin. User Agreement mereka juga melarang scraping profil dan data lainnya. ([LinkedIn][3])

LinkedIn memang memiliki ketentuan khusus untuk crawling yang mengharuskan izin eksplisit, dan penggunaan data hasil crawling juga memiliki pembatasan yang ketat. ([LinkedIn][4])

Jadi saya **tidak menyarankan arsitektur**:

```text
Playwright
   ↓
login LinkedIn
   ↓
search CSR
   ↓
scrape posts
   ↓
database Anda
```

Untuk SaaS komersial, ini terlalu berisiko.

---

# Tapi ada cara yang lebih aman

Gunakan LinkedIn sebagai **discovery signal**, bukan database scraping.

Misalnya:

```text
Google/Bing
     ↓
Index LinkedIn public pages/posts
     ↓
Temukan URL/post yang relevan
     ↓
Identifikasi Company
     ↓
Cari sumber resmi perusahaan
     ↓
Ambil data dari website/newsroom
     ↓
CSR Intelligence DB
```

Contohnya search:

```text
site:linkedin.com/posts "CSR" "Indonesia"
site:linkedin.com/posts "TJSL"
site:linkedin.com/posts "CSR partnership"
site:linkedin.com/posts "program CSR"
site:linkedin.com/posts "community development"
site:linkedin.com/posts "sustainability" "Indonesia"
```

Search engine memang masih dapat menemukan posting publik LinkedIn. Saya menemukan banyak contoh posting perusahaan seperti Pertamina Energy Terminal, WIKA, DAHANA, dan Jasa Marga melalui indeks web. ([LinkedIn][1])

**Tetapi:** jangan kemudian melakukan scraping otomatis terhadap halaman LinkedIn tersebut untuk membangun database Anda tanpa izin. Gunakan hasil pencarian sebagai **lead/discovery**, lalu verifikasi dari sumber yang boleh Anda crawl.

---

## Untuk CSR Intelligence Anda, saya akan memberi LinkedIn posisi seperti ini

| Sumber                      | Nilai | Peran                  |
| --------------------------- | ----: | ---------------------- |
| Company newsroom            | ⭐⭐⭐⭐⭐ | **Primary source**     |
| Website CSR/TJSL perusahaan | ⭐⭐⭐⭐⭐ | **Primary source**     |
| ANTARA/media                | ⭐⭐⭐⭐⭐ | News                   |
| Government/TJSL portal      | ⭐⭐⭐⭐⭐ | Official data          |
| NGO/Yayasan                 | ⭐⭐⭐⭐⭐ | Partnership            |
| **LinkedIn**                |  ⭐⭐⭐⭐ | **Discovery + signal** |
| Social media perusahaan     |  ⭐⭐⭐⭐ | Discovery              |
| Instagram                   |   ⭐⭐⭐ | Discovery              |

Jadi **LinkedIn jangan menjadi database utama Anda**.

Justru saya akan membuat konsep:

> **Signal → Verify → Intelligence**

Contohnya:

**LinkedIn signal**

> PT ABC baru saja memposting program pemberdayaan UMKM.

↓

**Verification**

Cari berita/newsroom resmi PT ABC.

↓

**Intelligence**

```text
PT ABC
CSR Status: ACTIVE
Focus: UMKM
Location: Jawa Barat
Program: Community Empowerment
Last Activity: 10 Sep 2026
CSR Score: 87
```

Kemudian kalau ada user yang punya proposal UMKM Jawa Barat:

> 🔔 **PT ABC cocok dengan proposal Anda — Match Score 91%**

**Menurut saya ini jauh lebih powerful daripada sekadar "scrape berita CSR".** LinkedIn menjadi salah satu sensor yang membantu Anda menemukan perubahan aktivitas perusahaan lebih cepat, sementara sumber resmi menjadi dasar data yang disimpan di database.

[1]: https://www.linkedin.com/posts/pertamina-energy-terminal_pertaminaenergyterminal-csrbrandequityawards2026-activity-7464902602762248192-twW8?utm_source=chatgpt.com "#pertaminaenergyterminal #csrbrandequityawards2026 #tjslpet #energikebaikan | Pertamina Energy Terminal"
[2]: https://www.linkedin.com/posts/satyanugraha1302_pelatihan-perencanaan-csr-berbasis-risiko-activity-7392465832255152128-HQUL?utm_source=chatgpt.com "Pelatihan Perencanaan CSR Berbasis Risiko Grup Mining Industry Indonesia Dalam rangka memperkuat kapasitas pengelola program TJSL di lingkungan Grup MIND ID, kami menyelenggarakan Workshop &… | Satya Nugraha"
[3]: https://ac.linkedin.com/legal/user-agreement?utm_source=chatgpt.com "User Agreement | LinkedIn"
[4]: https://www.linkedin.com/legal/crawling-terms?utm_source=chatgpt.com "LinkedIn Crawling Terms and Conditions"
