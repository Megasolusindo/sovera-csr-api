# OpenClaw AI Dedicated API Specification (`/api/v1/ai/*`)

Dokumen ini mendokumentasikan spesifikasi lengkap API khusus (**OpenClaw AI Agent Dedicated Namespace**) pada backend `sovera-csr-api`. Namespace `/api/v1/ai/*` dirancang khusus untuk interaksi agen AI OpenClaw, crawler worker, automasi riset CSR, pemantauan watchlist, dan mesin *partner matching*.

---

## 🔒 1. Otentikasi & Keamanan (Authentication)

Setiap request ke namespace `/api/v1/ai/*` dilindungi oleh **AIAgentAuthMiddleware**.

### Header Wajib:
```http
Authorization: Bearer <OPENCLAW_AGENT_TOKEN>
Content-Type: application/json
```

- **Default Agent Token**: `openclaw_agent_live_key_998877665544` (Dapat dikonfigurasi melalui variabel lingkungan `OPENCLAW_AGENT_TOKEN`).
- **Audit Logging**: Setiap transaksi API yang berhasil atau gagal akan secara otomatis dicatat ke tabel `ai_audit_logs` untuk akuntabilitas sistem intelijen.

---

## 📡 2. Format Respon Standar (Response Format)

### Success Response (HTTP 200 / 201)
```json
{
  "success": true,
  "message": "Deskripsi singkat hasil eksekusi",
  "data": { ... },
  "count": 1,
  "timestamp": "2026-09-13T09:30:00Z"
}
```

### Error Response (HTTP 400 / 401 / 403 / 404 / 500)
```json
{
  "success": false,
  "error": "ERROR_CODE",
  "message": "Penjelasan detail mengenai error yang terjadi"
}
```

---

## 📋 3. Ringkasan Daftar Endpoint (API Summary)

| Method | Endpoint | Fungsi / Deskripsi |
| :--- | :--- | :--- |
| **GET** | `/api/v1/ai/companies/search` | Mencari daftar perusahaan master berdasarkan kata kunci. |
| **GET** | `/api/v1/ai/companies/:id` | Mengambil detail profil perusahaan & metadata CSR lengkap. |
| **POST** | `/api/v1/ai/companies/:id/enrich` | Melakukan pengayaan profil perusahaan (website, phone, ESG topics). |
| **GET** | `/api/v1/ai/search` | Pencarian sinyal CSR & program perusahaan pada System of Record DB. |
| **POST** | `/api/v1/ai/research/findings` | Mengirim temuan riset/watchlist baru ke Review Queue. |
| **GET** | `/api/v1/ai/research/findings` | Mengambil daftar temuan riset (filter berdasarkan status). |
| **GET** | `/api/v1/ai/watchlist` | Mengambil daftar perusahaan aktif yang dipantau di Watchlist. |
| **POST** | `/api/v1/ai/watchlist` | Menambahkan / memperbarui perusahaan ke Watchlist pemantauan. |
| **POST** | `/api/v1/ai/companies/:id/monitor` | Mengatur status monitoring aktif/non-aktif suatu perusahaan. |
| **POST** | `/api/v1/ai/matching` | Menghitung peringkat matching mitra korporasi untuk proposal. |

---

## 🔍 4. Dokumentasi Detail Endpoint

### 4.1. Search Master Companies Directory
Mencari entitas perusahaan yang terdaftar pada directory master database.

- **Method**: `GET`
- **Endpoint**: `/api/v1/ai/companies/search`
- **Query Parameters**:
  - `q` (required): Kata kunci nama perusahaan (misal: `Telkom`, `Pertamina`, `Indosat`).
  - `limit` (optional, default: `10`, max: `50`): Jumlah data per halaman.
  - `offset` (optional, default: `0`): Offset pagination.

#### Example Request:
```bash
curl -X GET "http://localhost:4000/api/v1/ai/companies/search?q=Indosat" \
  -H "Authorization: Bearer openclaw_agent_live_key_998877665544"
```

#### Example Response (HTTP 200):
```json
{
  "success": true,
  "count": 2,
  "total": 2,
  "data": [
    {
      "id": "44c7b855-c1be-454f-8f2c-58ca409b85dc",
      "name": "PT Indosat Tbk",
      "legal_name": "PT Indosat Tbk",
      "slug": "pt-indosat-tbk",
      "industry_sector": "Infrastructure",
      "company_type": "SWASTA_TBK",
      "website": "https://www.isat.com",
      "ticker": "ISAT",
      "priority_tier": "TIER_1",
      "csr_category": "POTENSIAL"
    }
  ],
  "timestamp": "2026-09-13T09:30:00Z"
}
```

---

### 4.2. Get Detailed Company Profile
Mengambil detail informasi entitas perusahaan beserta profil CSR dan ESG.

- **Method**: `GET`
- **Endpoint**: `/api/v1/ai/companies/:id`
- **URL Parameters**:
  - `id`: Company UUID atau Company Slug ID (misal: `pt-indosat-tbk`).

#### Example Request:
```bash
curl -X GET "http://localhost:4000/api/v1/ai/companies/pt-indosat-tbk" \
  -H "Authorization: Bearer openclaw_agent_live_key_998877665544"
```

---

### 4.3. Enrich Company Profile
Mengirimkan data pengayaan profil perusahaan (seperti nomor telepon resmi, website yang terverifikasi, topik material ESG, atau NGO mitra).

- **Method**: `POST`
- **Endpoint**: `/api/v1/ai/companies/:id/enrich`
- **Body Payload**:
```json
{
  "website": "https://www.pertamina.com",
  "phone": "+6221135",
  "headquarters": "Jakarta Pusat, DKI Jakarta",
  "industry_sector": "Energi & Migas",
  "csr_category": "FOCUSED",
  "priority_tier": "TIER_1",
  "csr_focuses": ["Pendidikan", "Lingkungan", "UMKM"],
  "esg_material_topics": ["Decarbonization", "Biodiversity", "Community Empowerment"]
}
```

#### Example Response (HTTP 200):
```json
{
  "success": true,
  "message": "Company 'PT Pertamina Patra Niaga' profile enriched successfully",
  "data": {
    "company_id": "905c7939-8c84-4600-a0b7-49dc2736cbf1",
    "company_name": "PT Pertamina Patra Niaga",
    "fields_enriched": ["website", "phone", "headquarters", "csr_focuses"],
    "enriched_at": "2026-09-13T09:30:00Z"
  }
}
```

---

### 4.4. Search System of Record Corporate Signals & Programs
Melakukan pencarian sinyal CSR intelijen dan program CSR perusahaan pada tabel `public_corporate_signals` dan `company_csr_programs`.

- **Method**: `GET`
- **Endpoint**: `/api/v1/ai/search`
- **Query Parameters**:
  - `q` (required): Kata kunci pencarian (misal: `Air Bersih NTT`, `Stunting`, `Beasiswa`).

#### Example Request:
```bash
curl -X GET "http://localhost:4000/api/v1/ai/search?q=Air%20Bersih%20NTT" \
  -H "Authorization: Bearer openclaw_agent_live_key_998877665544"
```

#### Example Response (HTTP 200):
```json
{
  "success": true,
  "query": "Air Bersih NTT",
  "count": 1,
  "data": [
    {
      "id": "74aeeeed-565f-4b41-a539-bedb072160de",
      "company_name": "PT Pertamina Patra Niaga",
      "summary": "Pertamina Patra Niaga meluncurkan bantuan penyediaan sarana air bersih di NTT.",
      "pillar": "Lingkungan & Kesehatan",
      "target_regions": ["NTT", "Nusa Tenggara Timur"],
      "estimated_budget": 250000000,
      "source_url": "https://news.google.com/...",
      "created_at": "2026-09-13T09:00:00Z"
    }
  ]
}
```

---

### 4.5. Submit AI Research / Watchlist Finding
Mengirimkan hasil temuan riset web atau pemantauan watchlist otomatis dari OpenClaw AI ke dalam Review Queue (`ai_research_findings`). Jika temuan di-approve oleh Admin, sistem akan secara otomatis mempublikasikannya ke tabel sinyal intelijen `/signals`.

- **Method**: `POST`
- **Endpoint**: `/api/v1/ai/research/findings` (Alias: `/api/v1/ai/csr-programs`)
- **Body Payload**:
```json
{
  "company_name": "PT Pertamina Patra Niaga",
  "finding_type": "csr_program",
  "title": "Program Ekowisata Mangrove Berkelanjutan",
  "summary": "Pertamina Patra Niaga meluncurkan program rehabilitasi ekosistem mangrove di Pesisir Utara Jawa dengan total bibit 50.000 pohon.",
  "source_url": "https://www.pertaminapatraniaga.com/news/mangrove-pesisir-jawa",
  "source_name": "Pertamina Patra Niaga Official",
  "source_type": "company_website",
  "confidence_score": 0.95,
  "evidence_data": {
    "agent_name": "openclaw-research-agent",
    "execution_time": "2026-09-13T09:30:00Z"
  },
  "idempotency_key": "openclaw_res_mangrove_pertamina_50k"
}
```

#### Example Response (HTTP 201 Created):
```json
{
  "success": true,
  "message": "Research finding submitted successfully and queued for admin review",
  "data": {
    "id": "27c1dbbb-6783-4239-8fb8-7ea3c3f0e374",
    "company_name": "PT Pertamina Patra Niaga",
    "finding_type": "csr_program",
    "title": "Program Ekowisata Mangrove Berkelanjutan",
    "status": "pending_review",
    "confidence_score": 0.95
  },
  "timestamp": "2026-09-13T09:30:00Z"
}
```

---

### 4.6. Add / Update Company to OpenClaw Watchlist
Menambahkan atau memperbarui perusahaan yang akan dipantau secara otomatis oleh OpenClaw Agent. Jika entitas perusahaan belum ada di master database, sistem akan secara otomatis membuatkan (*provision*) record perusahaan baru.

- **Method**: `POST`
- **Endpoint**: `/api/v1/ai/watchlist`
- **Body Payload**:
```json
{
  "company_name": "PT XL Axiata Tbk",
  "monitoring_keywords": ["CSR", "TJSL", "Keberlanjutan", "ESG", "PT XL Axiata Tbk"],
  "check_interval_hours": 12
}
```

#### Example Response (HTTP 200):
```json
{
  "success": true,
  "message": "Company 'PT XL Axiata Tbk' added to watchlist successfully",
  "data": {
    "id": "89ab12cd-34ef-56gh-78ij-90kl12mn34op",
    "company_id": "eb983716-b90b-4f31-ba1e-2764fe6ec7fd",
    "company_name": "PT XL Axiata Tbk",
    "monitoring_keywords": ["CSR", "TJSL", "Keberlanjutan", "ESG", "PT XL Axiata Tbk"],
    "check_interval_hours": 12,
    "is_active": true,
    "created_at": "2026-09-13T09:30:00Z"
  },
  "timestamp": "2026-09-13T09:30:00Z"
}
```

---

### 4.7. Get Active Watchlist Companies
Mengambil daftar seluruh perusahaan yang saat ini aktif dipantau oleh OpenClaw Agent.

- **Method**: `GET`
- **Endpoint**: `/api/v1/ai/watchlist`

#### Example Response (HTTP 200):
```json
{
  "success": true,
  "count": 5,
  "data": [
    {
      "id": "89ab12cd-34ef-56gh-78ij-90kl12mn34op",
      "company_id": "eb983716-b90b-4f31-ba1e-2764fe6ec7fd",
      "company_name": "PT XL Axiata Tbk",
      "monitoring_keywords": ["CSR", "TJSL", "Keberlanjutan", "ESG"],
      "check_interval_hours": 12,
      "is_active": true
    }
  ],
  "timestamp": "2026-09-13T09:30:00Z"
}
```

---

### 4.8. Program Partner Matching Engine
Menghitung peringkat pencocokan (*matching score*) antara proposal lembaga dengan mitra korporasi yang paling relevan.

- **Method**: `POST`
- **Endpoint**: `/api/v1/ai/matching`
- **Body Payload**:
```json
{
  "program_title": "Program Pemberdayaan Sekolah Digital & Sanitasi",
  "csr_category": "Pendidikan",
  "target_location": "Jawa Barat",
  "target_beneficiaries": "Siswa & Sekolah Pedesaan",
  "estimated_budget": 500000000,
  "keywords": ["Pendidikan", "Beasiswa", "Digitalisasi"],
  "limit": 5
}
```

#### Example Response (HTTP 200):
```json
{
  "success": true,
  "message": "Program matching executed successfully",
  "data": {
    "program_title": "Program Pemberdayaan Sekolah Digital & Sanitasi",
    "csr_category": "Pendidikan",
    "matches_found": 3,
    "corporate_matches": [
      {
        "company_id": "44c7b855-c1be-454f-8f2c-58ca409b85dc",
        "company_name": "PT Telkom Indonesia (Persero) Tbk",
        "industry_sector": "Telekomunikasi",
        "priority_tier": "TIER_1",
        "match_score": 95.0,
        "match_grade": "HIGHLY_RECOMMENDED",
        "match_rationale": "Sangat cocok dengan fokus program digitalisasi dan CSR pendidikan.",
        "key_focus_areas": ["Pendidikan", "Digitalisasi"],
        "website": "https://www.telkom.co.id"
      }
    ],
    "evaluated_at": "2026-09-13T09:30:00Z"
  }
}
```

---

## 💻 5. Ringkasan Perintah cURL Lengkap (Testing Cheat Sheet)

```bash
# 1. Search Company
curl -s -X GET "http://localhost:4000/api/v1/ai/companies/search?q=Telkom" \
  -H "Authorization: Bearer openclaw_agent_live_key_998877665544"

# 2. Search Database Signals
curl -s -X GET "http://localhost:4000/api/v1/ai/search?q=Air%20Bersih" \
  -H "Authorization: Bearer openclaw_agent_live_key_998877665544"

# 3. Add to Watchlist
curl -s -X POST "http://localhost:4000/api/v1/ai/watchlist" \
  -H "Authorization: Bearer openclaw_agent_live_key_998877665544" \
  -H "Content-Type: application/json" \
  -d '{"company_name": "PT Telkom Indonesia", "monitoring_keywords": ["CSR", "Digitalisasi"]}'

# 4. Fetch Active Watchlist
curl -s -X GET "http://localhost:4000/api/v1/ai/watchlist" \
  -H "Authorization: Bearer openclaw_agent_live_key_998877665544"

# 5. Submit Finding to Review Queue
curl -s -X POST "http://localhost:4000/api/v1/ai/research/findings" \
  -H "Authorization: Bearer openclaw_agent_live_key_998877665544" \
  -H "Content-Type: application/json" \
  -d '{
    "company_name": "PT Pertamina Patra Niaga",
    "finding_type": "csr_program",
    "title": "Program Ekowisata Mangrove",
    "summary": "Rehabilitasi ekosistem mangrove 50.000 pohon.",
    "source_url": "https://www.pertaminapatraniaga.com",
    "source_name": "Official Web",
    "source_type": "company_website",
    "confidence_score": 0.95
  }'

# 6. Execute Partner Matching
curl -s -X POST "http://localhost:4000/api/v1/ai/matching" \
  -H "Authorization: Bearer openclaw_agent_live_key_998877665544" \
  -H "Content-Type: application/json" \
  -d '{
    "program_title": "Program Beasiswa Pendidikan",
    "csr_category": "Pendidikan",
    "target_location": "Jawa Barat",
    "estimated_budget": 300000000
  }'
```
