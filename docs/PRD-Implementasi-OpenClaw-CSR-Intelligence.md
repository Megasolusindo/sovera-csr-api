# PRD — Implementasi OpenClaw untuk CSR Intelligence

**Versi:** 1.1  
**Status:** Production Implemented & Verified  
**Tanggal:** 2026-09-13  
**Produk:** CSR Intelligence  
**Komponen:** OpenClaw AI Agent Integration

---

## 1. Ringkasan

Dokumen ini mendefinisikan Product Requirements Document (PRD) untuk mengintegrasikan **OpenClaw** sebagai AI Agent/AI Operator ke dalam platform **CSR Intelligence**.

CSR Intelligence saat ini memiliki:
- Backend API
- Dashboard Web
- Platform Admin
- PostgreSQL sebagai database utama

OpenClaw ditempatkan sebagai **agent layer** yang berinteraksi dengan CSR Intelligence melalui **API**, bukan melalui koneksi langsung ke PostgreSQL.

### Prinsip arsitektur

> **OpenClaw → AI/Agent API → CSR Intelligence Backend → PostgreSQL**

OpenClaw bertugas melakukan pekerjaan agentic seperti research, enrichment, monitoring, klasifikasi, dan pembuatan draft data. CSR Intelligence tetap menjadi **system of record**.

### 1.1 Ringkasan Implementasi & Update Fitur Production (v1.1 Update)

1. **Spesifikasi API Khusus AI (`/api/v1/ai/*`)**:
   - Seluruh kontrak API dedicated untuk OpenClaw Agent didokumentasikan resmi pada [AI_API_SPEC.md](file:///Users/mluludk/Works/sovera-csr-api/AI_API_SPEC.md).
   - Penambahan endpoint `GET /api/v1/ai/search` (Search System of Record DB) dan `POST /api/v1/ai/watchlist` (Penambahan Watchlist via API/Telegram).
2. **Promosi Temuan Review Queue ke Feed `/signals`**:
   - Ketika Admin menekan *Approve* pada temuan riset/watchlist, sistem secara otomatis memasukkan dan mempublikasikan data tersebut ke tabel `intelligence.company_signals` agar tampil secara *real-time* pada halaman Dashboard `/signals` (Corporate Feed).
3. **Pemisahan Metrik Database Real-Time**:
   - Sistem membedakan metrik entitas non-profit/penerima bantuan (`organizations`: 105 lembaga) dan perusahaan pendana (`companies`: 5.464 perusahaan).
4. **Mekanisme Telegram Chat: Database First ➔ Live Crawling Fallback**:
   - Pencarian perusahaan/pendana memeriksa Database System of Record terlebih dahulu. Jika belum ada di database, bot memberikan laporan awal dan secara otomatis meluncurkan *live crawling background task*.
5. **Interactive Watchlist Addition**:
   - Pengguna dapat menambahkan perusahaan ke Watchlist secara langsung melalui Telegram Chat (`/watchlist add <Nama>` atau *"Tambahkan Pertamina ke watchlist"*).
6. **Strict Domain Guardrails untuk Gemini AI Chat**:
   - Pembatasan ranah percakapan Gemini AI agar berfokus pada CSR, TJSL BUMN, ESG, dan metrik platform OpenClaw, serta menolak pertanyaan di luar topik secara sopan.

---

# 2. Tujuan Produk

## 2.1 Tujuan Utama

Membangun AI Agent yang dapat membantu mengotomatisasi proses pengumpulan, analisis, enrichment, dan monitoring informasi CSR perusahaan.

### Target hasil

1. Mengurangi pekerjaan manual research.
2. Mempercepat penambahan perusahaan dan program CSR.
3. Meningkatkan freshness database.
4. Menyimpan sumber informasi secara terstruktur.
5. Memberikan rekomendasi dan intelligence yang dapat ditinjau admin.
6. Menyediakan automation berkala.
7. Memastikan seluruh perubahan data dapat diaudit.

---

# 3. Non-Goals

OpenClaw **tidak** bertanggung jawab untuk:

- Menjadi database utama.
- Mengakses PostgreSQL production secara langsung.
- Mengelola user aplikasi.
- Mengelola billing/subscription.
- Mengubah konfigurasi sistem secara bebas.
- Menghapus data production secara otomatis.
- Mengambil keputusan final tanpa mekanisme review untuk operasi berisiko tinggi.

---

# 4. Aktor Sistem

| Aktor | Deskripsi |
|---|---|
| Admin | Memvalidasi dan mengelola hasil AI |
| CSR Researcher | Menggunakan intelligence dan hasil research |
| Customer | Menggunakan data/intelligence melalui aplikasi |
| OpenClaw Agent | Melakukan research dan automation |
| CSR API | Interface resmi OpenClaw ke sistem |
| External Sources | Website, news, LinkedIn, laporan perusahaan, dll. |

---

# 5. Arsitektur Target

```text
                         ┌─────────────────────┐
                         │ Telegram / Chat      │
                         │ / Internal Trigger   │
                         └──────────┬──────────┘
                                    │
                                    ▼
                         ┌─────────────────────┐
                         │      OpenClaw       │
                         │                     │
                         │  AI Agent / Skills  │
                         │  Scheduler          │
                         │  Web Research       │
                         └──────────┬──────────┘
                                    │
                              HTTPS / API
                                    │
                                    ▼
                         ┌─────────────────────┐
                         │   CSR Intelligence  │
                         │      AI API         │
                         │                     │
                         │ Auth / RBAC         │
                         │ Validation           │
                         │ Business Logic       │
                         │ Audit Log            │
                         └──────────┬──────────┘
                                    │
                                    ▼
                         ┌─────────────────────┐
                         │     PostgreSQL      │
                         │  System of Record   │
                         └─────────────────────┘

        External Sources
        ┌────────┬────────┬────────┬─────────┐
        ▼        ▼        ▼        ▼         ▼
      News   Company   LinkedIn  Reports   Websites
```

---

# 6. Prinsip Integrasi

## 6.1 API First

OpenClaw **WAJIB** menggunakan API untuk berkomunikasi dengan CSR Intelligence.

OpenClaw tidak boleh:

- Membuka koneksi PostgreSQL production.
- Menjalankan SQL terhadap production database.
- Mengakses database credential.
- Mengubah schema database.

## 6.2 Least Privilege

Credential OpenClaw hanya mendapatkan permission yang dibutuhkan.

Contoh:

```text
company:read
company:create
company:update
csr_program:read
csr_program:create
csr_program:update
source:create
research:create
```

Permission berisiko tinggi seperti:

```text
user:delete
company:delete
billing:write
system_config:write
```

harus ditolak.

---

# 7. Modul OpenClaw

Implementasi dibagi menjadi beberapa skill/agent.

## 7.1 Research Agent

Bertugas menemukan informasi baru mengenai perusahaan dan aktivitas CSR.

Input:

```text
Nama perusahaan
Keyword
Industri
Lokasi
Kategori CSR
```

Output:

```text
Company
CSR Program
Source
Evidence
Published Date
Confidence Score
```

---

## 7.2 Company Enrichment Agent

Memperkaya data perusahaan yang sudah ada.

Contoh:

- Website
- Industri
- Lokasi
- Contact information publik
- CSR focus
- ESG focus
- Sustainability initiatives
- Program history

---

## 7.3 CSR Program Discovery Agent

Mencari program CSR baru.

Contoh:

> "Cari program CSR baru dari perusahaan sektor FMCG selama 30 hari terakhir."

Agent:

1. Search web.
2. Identifikasi sumber.
3. Ekstrak program.
4. Deduplicate.
5. Mengirim hasil ke API.
6. Menandai hasil sebagai `pending_review`.

---

## 7.4 Partnership Monitoring Agent

Mencari informasi:

- Open partnership
- Call for proposal
- CSR collaboration
- Community partnership
- Grant
- Sponsorship
- Social program collaboration

---

## 7.5 Company Monitoring Agent

Memantau perusahaan tertentu.

Contoh:

```text
PT ABC
Monitoring:
- CSR
- TJSL
- Sustainability
- ESG
- Community Development
```

Jika ada perubahan:

```text
New Finding
     │
     ▼
OpenClaw
     │
     ▼
CSR API
     │
     ▼
Notification
```

---

## 7.6 Matching Agent

Mencocokkan program lembaga dengan perusahaan.

Input:

```text
Proposal:
- Pendidikan
- Jawa Barat
- Anak
- Pemberdayaan
```

Output:

```text
Potential Companies:
1. Company A — Score 91
2. Company B — Score 84
3. Company C — Score 79
```

Matching tidak langsung mengubah data master.

---

## 7.7 Reporting Agent

Membuat laporan intelligence.

Contoh:

> "Buat laporan tren CSR sektor energi Indonesia selama Q3 2026."

Output dapat berupa:

- Executive summary
- Company activity
- Program categories
- Geographic distribution
- Emerging trends
- Sources
- Recommendations

---

# 8. AI Agent API

Disarankan membuat namespace API khusus:

```text
/api/v1/ai/
```

Contoh endpoint:

```http
GET  /api/v1/ai/companies/search
GET  /api/v1/ai/companies/{id}
POST /api/v1/ai/companies
PATCH /api/v1/ai/companies/{id}

GET  /api/v1/ai/csr-programs/search
POST /api/v1/ai/csr-programs

POST /api/v1/ai/research/findings
POST /api/v1/ai/research/jobs

POST /api/v1/ai/companies/{id}/enrich
POST /api/v1/ai/companies/{id}/monitor

POST /api/v1/ai/matching
POST /api/v1/ai/reports
```

Endpoint final harus mengikuti API architecture existing CSR Intelligence.

---

# 9. Research Finding

Semua informasi yang ditemukan AI harus memiliki evidence.

Minimum schema:

```json
{
  "company_id": "uuid",
  "finding_type": "csr_program",
  "title": "Program Pemberdayaan Masyarakat",
  "summary": "....",
  "source_url": "https://example.com/article",
  "source_name": "Example News",
  "published_at": "2026-09-10",
  "discovered_at": "2026-09-11T10:00:00Z",
  "confidence_score": 0.91,
  "status": "pending_review"
}
```

## Aturan

AI tidak boleh membuat fakta tanpa source.

Jika source tidak tersedia:

```text
confidence_score = 0
status = rejected
```

atau hasil tidak dikirim sebagai factual finding.

---

# 10. Data Lifecycle

```text
External Source
      │
      ▼
OpenClaw Research
      │
      ▼
Extract
      │
      ▼
Normalize
      │
      ▼
Deduplicate
      │
      ▼
AI API
      │
      ▼
Pending Review
      │
      ├──── Approved ────► Production Data
      │
      └──── Rejected
```

Untuk data berisiko rendah dan sangat terstruktur, sistem dapat menyediakan opsi **auto-approve** setelah validasi dan confidence threshold terpenuhi.

---

# 11. Confidence Score

Gunakan skala:

```text
0.00 - 0.49 = Low
0.50 - 0.74 = Medium
0.75 - 0.89 = High
0.90 - 1.00 = Very High
```

Confidence bukan pengganti evidence.

Confidence score harus mempertimbangkan:

- Authority source
- Recency
- Source agreement
- Completeness
- Extraction certainty
- Entity matching certainty

---

# 12. Deduplication

OpenClaw harus melakukan deduplication sebelum membuat record baru.

Contoh:

```text
Artikel A
Artikel B
Press Release C
```

Jika ketiganya membahas program yang sama:

```text
CSR Program
     │
     ├── Source A
     ├── Source B
     └── Source C
```

Bukan membuat tiga CSR Program.

---

# 13. Source Management

Setiap finding wajib menyimpan:

```text
source_url
source_name
source_type
published_at
discovered_at
content_hash / fingerprint
```

Jenis source:

```text
company_website
company_press_release
news
government
social_media
linkedin
report
annual_report
sustainability_report
other
```

---

# 14. Automation

OpenClaw dapat menjalankan scheduled jobs.

### Daily Research

```text
06:00
   │
   ▼
Research Agent
   │
   ▼
New CSR Findings
   │
   ▼
AI API
   │
   ▼
Pending Review
```

### Company Monitoring

```text
Setiap 24 jam
      │
      ▼
Company Watchlist
      │
      ▼
Search New Information
      │
      ▼
Compare Previous Data
      │
      ▼
Create Finding
```

### Weekly Intelligence

```text
Every Monday
      │
      ▼
Aggregate Findings
      │
      ▼
Analyze Trends
      │
      ▼
Generate Report
      │
      ▼
Notify Admin
```

---

# 15. Google Drive / Google Sheets Integration

OpenClaw dapat menggunakan Google Drive/Google Sheets sebagai external workspace.

Namun Google Drive/Sheets **bukan system of record utama**.

Contoh:

```text
Google Sheet
      │
      ▼
OpenClaw
      │
      ▼
Validation
      │
      ▼
CSR API
      │
      ▼
PostgreSQL
```

Use case:

- Import company list.
- Export research results.
- Review queue.
- Generate report.
- Collaboration dengan researcher.

Untuk production, gunakan OAuth dengan scope minimum yang diperlukan.

---

# 16. Authentication

OpenClaw ke AI API harus menggunakan credential khusus.

Disarankan:

```text
Authorization: Bearer <AI_AGENT_TOKEN>
```

Token harus:

- khusus OpenClaw
- dapat di-revoke
- memiliki expiry/rotation
- memiliki permission terbatas
- tidak digunakan oleh dashboard/admin

Jika sistem mendukung OAuth2 service account/client credentials, lebih disarankan untuk environment production.

---

# 17. Audit Log

Semua action OpenClaw harus tercatat.

Minimum:

```text
id
agent_id
action
endpoint
request_id
target_type
target_id
timestamp
result
status
```

Contoh:

```text
Agent: research-agent
Action: create_finding
Company: PT ABC
Finding: CSR Program XYZ
Status: success
Request ID: req_123
```

Audit log tidak boleh dapat dihapus oleh OpenClaw.

---

# 18. Rate Limiting

API AI harus memiliki rate limit khusus.

Contoh awal:

```text
Research:
60 requests/minute

Write:
30 requests/minute

Enrichment:
20 requests/minute
```

Angka final disesuaikan berdasarkan workload.

Tambahkan:

- timeout
- retry
- exponential backoff
- idempotency key

---

# 19. Idempotency

Untuk operasi create:

```http
Idempotency-Key: research-{hash}
```

Jika OpenClaw mengirim request yang sama dua kali, backend tidak boleh membuat duplicate record.

---

# 20. Error Handling

Kategori:

```text
400 Validation Error
401 Authentication Error
403 Permission Error
404 Not Found
409 Duplicate
429 Rate Limited
500 Internal Error
503 Temporary Unavailable
```

OpenClaw harus membedakan:

```text
retryable
non-retryable
```

Contoh:

```text
429 → retry
503 → retry
400 → jangan retry
403 → jangan retry
409 → resolve/check existing
```

---

# 21. Security Architecture

## Wajib

- HTTPS
- API authentication
- Least privilege
- Secret management
- Audit logging
- Rate limiting
- Request validation
- Input sanitization
- Network restriction

## Tidak diperbolehkan

```text
OpenClaw
   │
   ▼
Production PostgreSQL
```

atau:

```text
OpenClaw
   │
   ▼
Docker Socket Host
```

kecuali ada kebutuhan operasional yang jelas dan telah diamankan.

Untuk kemampuan shell/browser, gunakan sandbox terisolasi.

---

# 22. Deployment

OpenClaw dijalankan sebagai container tersendiri.

Contoh:

```text
VPS
│
├── Nginx
│
├── CSR API
│
├── Dashboard
│
├── Admin
│
└── OpenClaw
    └── Docker Container
```

OpenClaw tidak perlu ditempatkan dalam container yang sama dengan API.

---

# 23. Network Policy

Ideal:

```text
Internet
    │
    ▼
Nginx
    │
    ├── Dashboard
    ├── Admin
    └── API
             ▲
             │ HTTPS
             │
         OpenClaw
```

OpenClaw Gateway tidak boleh diekspos secara bebas ke public internet.

Jika diperlukan remote access:

- VPN
- private network
- authenticated tunnel
- reverse proxy dengan authentication

---

# 24. Observability

Monitor:

### OpenClaw

- Agent execution
- Job success/failure
- Tool execution
- Token/model usage
- Latency
- Error rate

### API

- Request count
- Response latency
- 4xx/5xx
- 429
- Agent-specific traffic

### Research

- Findings/day
- Approval rate
- Rejection rate
- Duplicate rate
- Source success rate
- Confidence distribution

---

# 25. Admin Dashboard

Platform Admin sebaiknya memiliki halaman:

## AI Agents

```text
Agent
Status
Last Run
Success Rate
Last Error
```

## Research Jobs

```text
Job
Agent
Started
Finished
Findings
Approved
Rejected
Status
```

## AI Findings

```text
Company
Finding
Source
Confidence
Status
Discovered At
```

Admin dapat:

- approve
- reject
- edit
- merge duplicate
- view source
- view evidence
- view agent execution

---

# 26. MVP Scope

Implementasi tahap pertama:

### Phase 1 — Foundation

- Deploy OpenClaw Docker
- Configure Gateway
- Configure model provider
- Secure secrets
- Create AI API authentication
- Logging
- Basic monitoring

### Phase 2 — Research

Implement:

- Company search
- CSR program discovery
- Source extraction
- Finding submission
- Deduplication
- Confidence score
- Pending review

### Phase 3 — Monitoring

Implement:

- Company watchlist
- Scheduled monitoring
- New finding detection
- Admin notification

### Phase 4 — Enrichment

Implement:

- Company enrichment
- CSR profile enrichment
- Source aggregation

### Phase 5 — Intelligence

Implement:

- Matching
- Trend analysis
- Report generation

---

# 27. MVP Acceptance Criteria

MVP dianggap berhasil jika:

### OpenClaw

- [ ] Berjalan stabil dalam Docker.
- [ ] Gateway tidak exposed tanpa authentication.
- [ ] Credential tersimpan aman.
- [ ] Agent dapat menjalankan research task.

### API

- [ ] OpenClaw dapat authenticate.
- [ ] RBAC berjalan.
- [ ] Rate limit berjalan.
- [ ] Audit log tersedia.
- [ ] Idempotency berjalan.

### Research

- [ ] Agent dapat mencari perusahaan.
- [ ] Agent dapat menemukan CSR program.
- [ ] Setiap finding memiliki source.
- [ ] Duplicate dapat dicegah.
- [ ] Finding masuk ke review queue.
- [ ] Admin dapat approve/reject.

### Monitoring

- [ ] Scheduled job berjalan.
- [ ] Perubahan perusahaan dapat dideteksi.
- [ ] Admin menerima notifikasi.

---

# 28. KPI

Target awal:

| KPI | Target MVP |
|---|---:|
| Research automation | > 70% |
| Finding dengan source | 100% |
| Duplicate finding | < 5% |
| API success rate | > 99% |
| High-confidence findings | > 70% |
| Manual research time reduction | > 50% |
| Audit coverage | 100% |

Target harus dievaluasi setelah mendapatkan data penggunaan nyata.

---

# 29. Roadmap

```text
Phase 1
Infrastructure
     │
     ▼
Phase 2
Research Agent
     │
     ▼
Phase 3
Monitoring Agent
     │
     ▼
Phase 4
Enrichment Agent
     │
     ▼
Phase 5
Matching Agent
     │
     ▼
Phase 6
Intelligence / Reporting
     │
     ▼
Phase 7
Multi-Agent System
```

---

# 30. Future Multi-Agent Architecture

Pada tahap lanjut:

```text
                         OpenClaw
                            │
                ┌───────────┼────────────┐
                ▼           ▼            ▼
           Research       Monitor     Enrichment
             Agent         Agent        Agent
                │           │            │
                └───────────┼────────────┘
                            ▼
                       AI API Layer
                            │
                            ▼
                       PostgreSQL
```

Kemudian:

```text
Research Agent
     │
     ▼
Enrichment Agent
     │
     ▼
Matching Agent
     │
     ▼
Reporting Agent
```

Setiap agent memiliki permission dan responsibility yang berbeda.

---

# 31. Keputusan Arsitektur Utama

| Keputusan | Pilihan |
|---|---|
| AI Agent | OpenClaw |
| Deployment | Docker |
| Integration | REST API |
| Database access | Tidak langsung |
| Database | PostgreSQL |
| System of Record | CSR Intelligence |
| Authentication | Dedicated Agent Credential |
| Authorization | RBAC / Scope |
| Research result | Pending Review |
| Source evidence | Wajib |
| Audit | Wajib |
| Scheduling | OpenClaw / backend scheduler |
| Google Drive | External integration |
| Google Sheets | External integration |
| Production DB access | Dilarang |
| Docker socket | Dilarang secara default |

---

# 32. Rekomendasi Implementasi

Urutan implementasi yang direkomendasikan:

```text
1. Finalisasi AI API
        ↓
2. Authentication + RBAC
        ↓
3. Audit Log
        ↓
4. Deploy OpenClaw Docker
        ↓
5. Research Agent
        ↓
6. Finding + Source Model
        ↓
7. Review Workflow
        ↓
8. Company Monitoring
        ↓
9. Enrichment
        ↓
10. Matching
        ↓
11. Reporting
        ↓
12. Multi-Agent
```

**Jangan mulai dari agent terlebih dahulu.** Pastikan API contract, permission, audit trail, dan data lifecycle sudah siap.

---

# 33. Definition of Done

Implementasi OpenClaw dianggap production-ready apabila:

- OpenClaw berjalan isolated dalam Docker.
- Tidak ada direct database credential di OpenClaw.
- Semua operasi data melalui API.
- AI API memiliki authentication dan RBAC.
- Semua AI action memiliki audit trail.
- Research finding selalu menyimpan source/evidence.
- Duplicate protection aktif.
- Review workflow tersedia.
- Scheduled jobs memiliki retry dan failure handling.
- Monitoring dan alerting tersedia.
- Secrets dapat dirotasi.
- OpenClaw dapat dinonaktifkan/revoke tanpa mempengaruhi aplikasi utama.
- Kegagalan OpenClaw tidak menyebabkan core CSR Intelligence berhenti.

---

# 34. Prinsip Akhir

Arsitektur yang digunakan harus mempertahankan batas:

> **OpenClaw adalah AI Operator, bukan database dan bukan core application.**

CSR Intelligence tetap menjadi:

> **System of Record + Business Logic + Authorization + Audit**

Sedangkan OpenClaw menjadi:

> **Research + Automation + Reasoning + Tool Orchestration**

Dengan pemisahan ini, OpenClaw dapat dikembangkan menjadi AI layer yang kuat tanpa membuat core application bergantung langsung pada agent.
