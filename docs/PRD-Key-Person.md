Menyimpan profil *key person* dan memantau aktivitas media sosial mereka untuk isu CSR membutuhkan pemisahan yang jelas antara **arsitektur penyimpanan profil (Person Master)** dan **pipeline *event monitoring* (Social Feed Ingestion)**.

Mengawasi semua eksekutif di seluruh media sosial secara massal sangat mahal dan rawan terblokir API limit/bot detection. Oleh karena itu, gunakan strategi penargetan peran (*role filtering*) dan pemantauan berbasis kata kunci.

---

### 1. Skema Database Person & Social Monitor

Tambahkan dua tabel relasional yang terhubung ke `company_master`:

```sql
-- Menyimpan data profil eksekutif & PIC CSR
CREATE TABLE company_key_persons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES company_master(id) ON DELETE CASCADE,
    full_name VARCHAR(255) NOT NULL,
    normalized_name VARCHAR(255) NOT NULL,
    current_title VARCHAR(255),
    role_category VARCHAR(50), -- 'csr_lead', 'corp_sec', 'c_level', 'foundation_head'
    linkedin_url VARCHAR(255),
    twitter_handle VARCHAR(100),
    instagram_handle VARCHAR(100),
    is_decision_maker BOOLEAN DEFAULT FALSE,
    is_monitored BOOLEAN DEFAULT FALSE, -- Flag apakah medsosnya aktif di-scrape
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Menyimpan sinyal/postingan media sosial yang terdeteksi
CREATE TABLE key_person_social_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    person_id UUID REFERENCES company_key_persons(id) ON DELETE CASCADE,
    company_id UUID REFERENCES company_master(id) ON DELETE CASCADE,
    platform VARCHAR(30) NOT NULL, -- 'linkedin', 'twitter', 'instagram'
    post_url VARCHAR(500) UNIQUE,
    post_text TEXT NOT NULL,
    posted_at TIMESTAMP WITH TIME ZONE,
    matched_keywords TEXT[], -- ['bencana', 'beasiswa', 'stunting', 'sdg']
    sentiment VARCHAR(20), -- 'positive', 'call_for_proposal', 'event_recap'
    ai_summary TEXT, -- Ringkasan 1-2 kalimat hasil ekstraksi LLM
    is_actionable BOOLEAN DEFAULT FALSE, -- Flag sinyal matang untuk draf outreach
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_social_signals_company ON key_person_social_signals(company_id);
CREATE INDEX idx_key_persons_role ON company_key_persons(role_category, is_monitored);

```

---

### 2. Strategi Discovery & Kurasi Key Person

Jangan simpan sembarang staf. Tim fundraising hanya butuh pihak yang memegang bujet atau mengurus kemitraan:

* **Filter Jabatan Target (*Role Whitelist*):**
* *Tingkat 1 (Direct Buyers):* "Head of CSR", "Manager TJSL", "Sustainability Manager", "Corporate Secretary", "Director of Corporate Affairs", "Ketua Yayasan [Nama Perusahaan]".
* *Tingkat 2 (High-Level Sponsors):* "CEO", "President Director", "Managing Director", "VP Human Resources" (biasanya terlibat dalam program *employee volunteering*).


* **Sumber Data Key Person:**
* **Laporan Tahunan / Sustainability Report (IDX/Web):** Bagian profil direksi, komisaris, dan tim komite CSR/tata kelola. Ini data paling valid dan resmi.
* **LinkedIn Discovery:** Gunakan search/enrichment API pihak ketiga (seperti RapidAPI scraper atau Proxycurl) yang memfilter parameter: `company_domain` + `title: ("CSR" OR "Sustainability" OR "Corporate Secretary")`.



---

### 3. Pipeline Pemantauan Media Sosial (Social Monitoring Engine)

Media sosial memiliki proteksi anti-scraping yang ketat. Eksekusi monitoring dibagi berdasarkan karakteristik platform:

**Kanal Prioritas:**

* **LinkedIn (Prioritas Utama):** Tempat eksekutif paling sering membagikan *milestone* ESG, serah terima bantuan, atau program yayasan.
* *Metode:* Jangan scrape akun pengguna perorangan setiap jam secara langsung karena risiko akun scraper diblokir. Gunakan worker terjadwal (misal: tiap 3–7 hari) atau manfaatkan layanan scraping feed LinkedIn eksternal yang mengekstrak aktivitas publik terbaru.


* **X / Twitter & Instagram:**
* *Metode:* Lebih mudah dipantau via web search query berulang secara harian, contoh: `site:[twitter.com/](https://twitter.com/)[handle] "CSR" OR "bantuan" OR "kemitraan"`.



**Filter Keyword & Intent Matching:**
Teks postingan yang terjaring masuk ke pipeline evaluasi:

1. **RegEx / Keyword Filter Awal:** Saring apakah postingan mengandung kata kunci spesifik kemanusiaan (`bencana`, `stunting`, `beasiswa`, `air bersih`, `dhuafa`, `tjsl`, `esg`, `volunteering`).
2. **LLM Extraction Layer (Menggunakan Model Ringan):**
Postingan yang lolos filter diekstraksi menggunakan prompt terarah:
* *Apakah ini pengumuman alokasi dana baru atau sekadar acara seremonial masa lalu?*
* *Pilar SDG apa yang disasar?*
* *Output JSON:* `{ is_actionable: true/false, program_type: "...", proposed_outreach_angle: "..." }`.



---

### 4. Contoh Hasil Feed untuk Dashboard Lembaga

Ketika sistem mendeteksi postingan dari Head of CSR:

> **[LinkedIn Alert] PT Bank Mandiri Tbk**
> **Aktor:** Rahmat Santoso (VP TJSL / CSR)
> **Sinyal Terdeteksi:** *"Baru saja meresmikan pilot project sanitasi air bersih di NTT bersama tim sustainability..."*
> **Analisis Sistem:** Fokus aktif pada SDG 6 (Air Bersih) & wilayah Indonesia Timur.
> **Rekomendasi Aksi Lembaga:** Tombol *Generate Outreach Proposal* bertema penyediaan sarana air bersih di wilayah NTT untuk diajukan ke divisi TJSL bersangkutan.