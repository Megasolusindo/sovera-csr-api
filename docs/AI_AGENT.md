# 🤖 Status Implementasi 15 AI Agents (v1.2 Production Audit)

Dokumen ini mendokumentasikan pemetaan status implementasi riil **15 AI Agents** dari arsitektur AI-Operated CSRmatics v1.2 ke dalam codebase backend (`sovera-csr-api`), Admin Platform (`sovera-csr-admin`), dan User Dashboard (`sovera-csr-dashboard`).

---

## 📊 Summary Ringkasan Status

* 🟢 **100% Terimplementasi (Operational)**: **12 / 15 Agents** (80.0%)
* 🟡 **Terpakai di Dev Workflow / Metrik Analytics**: **2 / 15 Agents** (13.3%)
* ⚪ **Roadmap Jangka Panjang (Post-Launch)**: **1 / 15 Agents** (6.7%)

---

## 📋 Tabel Status Audit 15 AI Agents

| No | Nama Agent | Status | Lokasi File Implementasi | Catatan & Guardrail Utama |
| :--- | :--- | :---: | :--- | :--- |
| **1** | **Research Agent** | 🟢 **100%** | [`internal/service/ai/research_agent.go`](file:///Users/mluludk/Works/sovera-csr-api/internal/service/ai/research_agent.go)<br>[`internal/service/ai/secure_fetcher.go`](file:///Users/mluludk/Works/sovera-csr-api/internal/service/ai/secure_fetcher.go) | Scope `GLOBAL`, hard-cap 5 reasoning steps & 10 tool calls, `UNTRUSTED_DATA` tag, SSRF Shield & Egress filter. |
| **2** | **Company Intelligence Agent** | 🟢 **100%** | `internal/service/companyenricher/`<br>[`internal/queue/company_contact_discovery_worker.go`](file:///Users/mluludk/Works/sovera-csr-api/internal/queue/company_contact_discovery_worker.go)<br>[`internal/service/ai/research_agent.go`](file:///Users/mluludk/Works/sovera-csr-api/internal/service/ai/research_agent.go) | Pengayaan profil 360° korporasi (BEI/IDX, Website, Phone/Email, Serper News, Medsos). Enforces **Strict No Synthetic URL Reconstruction Directive** & Auto-Purge invalid URLs. |
| **3** | **Opportunity Discovery Agent** | 🟢 **100%** | [`internal/service/ai/extraction_workflow.go`](file:///Users/mluludk/Works/sovera-csr-api/internal/service/ai/extraction_workflow.go)<br>[`internal/service/ai/verification_workflow.go`](file:///Users/mluludk/Works/sovera-csr-api/internal/service/ai/verification_workflow.go) | Penemuan hibah CSR (`DISCOVERED` $\rightarrow$ `VERIFIED` $\rightarrow$ `PUBLISHED`) dengan Idempotency Key (§18) & Wire Origin Hash (§15). |
| **4** | **Watchlist Research Agent** | 🟢 **100%** | [`internal/service/ai/demand_aggregator.go`](file:///Users/mluludk/Works/sovera-csr-api/internal/service/ai/demand_aggregator.go) | Agregasi anonim kata kunci watchlist NGO (§22) tanpa membocorkan `tenant_id` ke luar. |
| **5** | **Support Agent** | 🟢 **100%** | [`internal/service/ai/support_agent.go`](file:///Users/mluludk/Works/sovera-csr-api/internal/service/ai/support_agent.go)<br>[`sovera-csr-dashboard/src/app/(dashboard)/chat/page.tsx`](file:///Users/mluludk/Works/sovera-csr-dashboard/src/app/\(dashboard\)/chat/page.tsx) | Scope `TENANT`, read-only tools (`search_companies`, `get_csr_signals`, `match_opportunity`, `send_session_alert`). |
| **6** | **Matching Agent** | 🟢 **100%** | `internal/service/matcher/`<br>[`internal/handler/ai_tools_handler.go`](file:///Users/mluludk/Works/sovera-csr-api/internal/handler/ai_tools_handler.go) | Pencocokan semantik vektor `pgvector` + evaluasi syarat proposal NGO. |
| **7** | **Competitor/Market Intel Agent** | 🟢 **100%** | [`internal/repository/intelligence_repo.go`](file:///Users/mluludk/Works/sovera-csr-api/internal/repository/intelligence_repo.go)<br>[`sovera-csr-admin/src/app/(admin)/esg-intelligence/page.tsx`](file:///Users/mluludk/Works/sovera-csr-admin/src/app/\(admin\)/esg-intelligence/page.tsx) | Agregasi tren sektor CSR dan pergeseran pilar ESG berdasarkan data terverifikasi. |
| **8** | **Data Quality Agent** | 🟢 **100%** | `internal/service/entityresolver/`<br>`cmd/merge_duplicates/`<br>`db_improve/000018_cleanup_legacy_duplicate_companies.up.sql` | Resolusi entitas ganda (*duplicate companies, aliases, subsidiaries*) ke master canonical. Auto-reroute relasi & purge orphaned legacy duplicates (`[LEGACY <uuid>]`). |
| **9** | **Data Investigation Agent** | 🟢 **100%** | [`sovera-csr-admin/src/app/(admin)/ai-lineage/page.tsx`](file:///Users/mluludk/Works/sovera-csr-admin/src/app/\(admin\)/ai-lineage/page.tsx)<br>[`internal/queue/retraction_worker.go`](file:///Users/mluludk/Works/sovera-csr-api/internal/queue/retraction_worker.go) | Penyelidikan *provenance* klaim bukti (Wire Origin Hash, Temporal Score, Negation Check) dan retraksi data bertingkat. |
| **10** | **Customer Support / Troubleshooting** | 🟢 **100%** | [`internal/service/ai/support_agent.go`](file:///Users/mluludk/Works/sovera-csr-api/internal/service/ai/support_agent.go) | AI Assistant yang menjelaskan alasan penarikan sinyal (`RETRACTED`) atau verifikasi secara transparan. |
| **11** | **Dev Agent** | 🟡 **Dev Workflow** | IDE Pair-Programming Assistant (Antigravity) | Digunakan pengembang solo untuk pembuatan endpoint, refactoring, linter, & kompilasi Docker. |
| **12** | **Content Agent** | 🟢 **100%** | `sovera-csr-dashboard/src/app/(dashboard)/database-perusahaan/`<br>`sovera-csr-dashboard/src/app/(dashboard)/blog/` | Generasi otomatis halaman Programmatic SEO & direktori terverifikasi. |
| **13** | **Growth Agent** | ⚪ **Roadmap** | Post-Launch Roadmap | Analisis eksperimen konversi lalu lintas website secara otomatis di masa depan. |
| **14** | **Product Analyst Agent** | 🟡 **Metrik** | [`sovera-csr-admin/src/app/(admin)/analytics/page.tsx`](file:///Users/mluludk/Works/sovera-csr-admin/src/app/\(admin\)/analytics/page.tsx) | Pelacakan aktivitas pengguna dan fitur di Admin Platform. |
| **15** | **Security Agent** | 🟢 **100%** | [`internal/service/ai/cost_monitor.go`](file:///Users/mluludk/Works/sovera-csr-api/internal/service/ai/cost_monitor.go)<br>[`sovera-csr-admin/src/app/(admin)/ai-metering/page.tsx`](file:///Users/mluludk/Works/sovera-csr-admin/src/app/\(admin\)/ai-metering/page.tsx)<br>[`sovera-csr-admin/src/app/(admin)/ai-policies/page.tsx`](file:///Users/mluludk/Works/sovera-csr-admin/src/app/\(admin\)/ai-policies/page.tsx) | Rate Limiter per tenant, Hard-Cap Circuit Breaker ($0.50/task & $10/day), Replay Nonce Guard, & Global Kill Switch. |

---

Bisa sangat banyak. Tetapi untuk **CSRmatics**, saya akan membedakan dengan tegas antara pekerjaan yang memang cocok menjadi **AI Agent** dan pekerjaan yang lebih tepat menjadi **workflow otomatis + LLM**.

Berdasarkan desain v1.2 Anda, agent seharusnya dipakai terutama untuk pekerjaan yang **eksploratif, membutuhkan keputusan beberapa langkah, dan tidak bisa ditentukan dengan rule sederhana**. 

## 1. Research Agent — kandidat utama

Ini salah satu pekerjaan yang paling cocok.

Contoh:

> "Cari perusahaan di Indonesia yang dalam 30 hari terakhir mengumumkan program CSR bidang pendidikan."

Agent:

```text
Tujuan riset
    ↓
Cari kandidat sumber
    ↓
Buka sumber
    ↓
Evaluasi relevansi
    ↓
Cari sumber tambahan
    ↓
Kumpulkan evidence
    ↓
Submit hasil penelitian
```

Bisa otomatis:

* mencari berita CSR
* mencari press release perusahaan
* mencari laporan sustainability
* mencari program baru
* mencari perubahan aktivitas perusahaan
* mencari peluang partnership
* mencari perusahaan yang aktif di wilayah tertentu
* mencari informasi yang belum ada di database
* melakukan research lanjutan ketika evidence pertama belum cukup

**Ini benar-benar agentic**, karena agent menentukan langkah berikutnya berdasarkan hasil sebelumnya.

---

# 2. Company Intelligence Agent

Agent dapat diberi tugas:

> "Investigasi perusahaan X."

Kemudian melakukan:

```text
Company X
   │
   ├── Website
   ├── News
   ├── Sustainability Report
   ├── CSR Program
   ├── Recent Activity
   ├── Partnership
   └── Funding
          ↓
     Intelligence Report
```

Output misalnya:

```text
Company:
PT XYZ

Recent CSR:
- Pendidikan
- Kesehatan

Active Region:
- Jawa Barat
- Jawa Tengah

Recent Program:
Program pendidikan ...

Potential NGO Opportunity:
High relevance

Evidence:
...
```

Namun **hasilnya tetap masuk ke verification workflow**, bukan langsung dianggap fakta. Desain v1.2 memang menempatkan verification sebagai workflow terpisah. 

---

# 3. Opportunity Discovery Agent

Ini sangat menarik untuk CSRmatics.

Agent mencari:

> "Perusahaan mana yang kemungkinan sedang membuka peluang kerja sama dengan NGO?"

Agent dapat mengeksplorasi:

```text
News
  +
Company Website
  +
CSR Page
  +
Press Release
  +
Program Announcement
  +
Partnership Announcement
        ↓
Opportunity Candidate
```

Kemudian menghasilkan:

```text
Opportunity Candidate

Company: ABC
Program: Education
Region: Bekasi
Evidence: ...
Why relevant: ...
Potential NGO profile: ...
```

Tetapi statusnya:

```text
DISCOVERED
     ↓
VERIFIED
     ↓
PUBLISHED
```

bukan langsung `PUBLISHED`.

---

# 4. Research Agent untuk Watchlist

Ini justru salah satu fitur yang bisa membuat CSRmatics berbeda.

Misalnya NGO memasukkan:

```text
Watchlist:

Company:
- Astra
- Unilever
- Pertamina

Topic:
- Pendidikan
- Kesehatan

Region:
- Jawa Barat
```

Sistem **tidak mengirim watchlist tersebut ke web sebagai identitas NGO**.

Sebagaimana desain v1.2, watchlist diubah menjadi **aggregated demand signal**, lalu digunakan untuk memprioritaskan global research. 

Contohnya:

```text
100 NGO mencari:
"CSR pendidikan Jawa Barat"

                ↓

Demand Aggregator

                ↓

Priority:
CSR Education
West Java
Corporate Partnership

                ↓

Research Agent
```

Ini memungkinkan AI melakukan research secara global tanpa membocorkan:

> "NGO A sedang mencari perusahaan X."

---

# 5. Support Agent

Ini juga true agent.

User NGO bertanya:

> "Perusahaan mana yang sedang aktif di bidang pendidikan?"

Agent:

```text
User Question
      ↓
Understand Intent
      ↓
Search CSRmatics
      ↓
Filter
      ↓
Compare
      ↓
Generate Answer
```

Contoh:

> "Carikan perusahaan yang punya program pendidikan di Jawa Barat."

Agent dapat menggunakan:

```text
search_companies()
get_csr_signals()
search_opportunities()
match_opportunity()
```

Kemudian menjawab berdasarkan data CSRmatics.

Yang penting: Support Agent **tidak boleh melakukan external side-effect secara bebas**; desain v1.2 membatasi tooling-nya ke read-only internal dan session alert. 

---

# 6. Matching Agent

Misalnya NGO memiliki proposal:

```text
Program:
Pendidikan anak

Lokasi:
Bekasi

Budget:
Rp500 juta

Target:
500 anak
```

Agent bisa melakukan eksplorasi:

```text
Proposal NGO
      ↓
Understand requirements
      ↓
Search companies
      ↓
Search CSR programs
      ↓
Search opportunities
      ↓
Compare requirements
      ↓
Generate candidate matches
```

Output:

```text
Candidate #1
Company: XYZ
Reason:
- Education focus
- Active in West Java
- Similar program
- Partnership opportunity detected

Evidence:
...
```

Tetapi **scoring/matching final sebaiknya workflow deterministic**, bukan agent bebas. Jadi:

```text
Agent → menemukan kandidat

Workflow → menghitung score
```

---

# 7. Competitor / Market Intelligence Agent

Agent bisa melakukan research seperti:

> "Apa yang dilakukan perusahaan-perusahaan besar dalam CSR pendidikan?"

Kemudian:

```text
Company A
Company B
Company C
Company D
       ↓
Research
       ↓
Classification
       ↓
Trend Detection
       ↓
Market Intelligence
```

Hasilnya dapat menjadi:

```text
CSR Trend:

Education
████████████

Health
████████

Environment
██████
```

Tetapi angka/trend harus berasal dari data terukur, bukan opini agent.

---

# 8. Data Quality Agent

Agent bisa melakukan investigasi terhadap database.

Contoh:

> "Periksa company records yang mencurigakan."

Agent mencari:

```text
Duplicate companies
       ↓
Different aliases
       ↓
Different domains
       ↓
Potential subsidiaries
       ↓
Corporate relationship
```

Contoh:

```text
PT ABC Indonesia
ABC Indonesia
ABC Corp Indonesia
ABC Group
```

Agent mengusulkan:

```text
Possible same entity: 87%
```

Kemudian deterministic entity-resolution workflow + human review menangani keputusan akhirnya.

Ini penting karena v1.2 memang memiliki **canonical company, aliases, dan corporate graph**. 

---

# 9. Data Investigation Agent

Agent dapat diberi tugas:

> "Mengapa jumlah CSR signals perusahaan X tiba-tiba turun?"

Agent melakukan:

```text
Database
   ↓
Recent signals
   ↓
Source history
   ↓
Verification state
   ↓
Retraction
   ↓
Pipeline failures
   ↓
Investigation report
```

Ini lebih cocok daripada dashboard biasa karena agent melakukan **investigasi multi-step**.

---

# 10. Customer Support / Troubleshooting Agent

Misalnya customer mengatakan:

> "Kenapa opportunity yang kemarin saya lihat sekarang hilang?"

Agent dapat:

```text
User
 ↓
Find opportunity
 ↓
Check current status
 ↓
Check source
 ↓
Check verification
 ↓
Check retraction
 ↓
Explain
```

Misalnya:

> Opportunity tersebut ditandai `RETRACTED` setelah sumber aslinya melakukan koreksi.

Lineage retraction memang sudah menjadi bagian desain CSRmatics. 

---

# 11. Dev Agent

Ini bukan untuk customer, tetapi untuk **Anda sebagai solo developer**.

Agent dapat mengerjakan:

```text
Issue
 ↓
Understand codebase
 ↓
Locate code
 ↓
Implement
 ↓
Run test
 ↓
Fix failure
 ↓
Create PR
```

Contoh:

> "Implement endpoint `/api/v1/ai/tools/search_companies`."

Agent:

```text
Read architecture
      ↓
Read existing repository
      ↓
Implement handler
      ↓
Implement service
      ↓
Implement repository
      ↓
Test
      ↓
Lint
      ↓
PR
```

Tetapi merge/deploy production tetap Tier 3 dan membutuhkan approval menurut desain v1.2. 

---

# 12. Content Agent

Untuk tahap berikutnya.

Misalnya:

> "Buat artikel tentang tren CSR pendidikan Indonesia berdasarkan intelligence CSRmatics."

Agent:

```text
Verified Signals
      ↓
Research
      ↓
Outline
      ↓
Draft
      ↓
Fact Check
      ↓
Source Check
      ↓
SEO Check
      ↓
Human Approval
      ↓
Publish
```

Ini cocok untuk otomatisasi content operation.

---

# 13. Growth Agent

Jangka lebih panjang:

```text
Traffic
Search Console
Analytics
User Behavior
      ↓
Growth Agent
      ↓
Find opportunities
      ↓
Suggest experiments
      ↓
Create tasks
      ↓
Measure results
```

Misalnya:

> "Halaman `/intelligence` banyak dikunjungi tetapi conversion rendah."

Agent melakukan investigasi dan mengusulkan eksperimen.

Untuk tahap awal, **jangan diberi akses langsung mengubah website production**.

---

# 14. Product Analyst Agent

Agent dapat menganalisis:

```text
User activity
Feature usage
Search queries
Watchlists
Opportunities viewed
Matches
Churn
```

Lalu mencari pola:

> "Banyak NGO mencari perusahaan berdasarkan lokasi tetapi filter wilayah jarang digunakan."

atau:

> "Opportunity dengan kategori pendidikan mempunyai engagement lebih tinggi."

Ini bisa menjadi sumber otomatis untuk product roadmap.

---

# 15. Security Agent

Jangka panjang, agent dapat membantu:

```text
Logs
   ↓
Anomaly Detection
   ↓
Investigate
   ↓
Correlate events
   ↓
Risk assessment
   ↓
Alert
```

Contoh:

> "Ada 1 tenant yang menghasilkan 20× lebih banyak API request dari baseline."

Agent melakukan investigation, tetapi **tidak langsung suspend akun**. Suspend termasuk high-risk action.

---

# Yang paling penting: jangan semuanya dibuat Agent

Untuk CSRmatics saya akan membaginya seperti ini:

| Pekerjaan                | Bentuk                        |
| ------------------------ | ----------------------------- |
| Web Research             | 🤖 **Agent**                  |
| Open-ended investigation | 🤖 **Agent**                  |
| Customer Support         | 🤖 **Agent**                  |
| Coding / Development     | 🤖 **Agent**                  |
| Extraction               | ⚙️ Workflow + LLM             |
| Entity Resolution        | ⚙️ Workflow + LLM             |
| Verification             | ⚙️ **Deterministic Workflow** |
| Signal Generation        | ⚙️ Workflow                   |
| Deduplication            | ⚙️ Workflow                   |
| Matching                 | ⚙️ Workflow + LLM             |
| Notification             | ⚙️ Workflow                   |
| Retraction               | ⚙️ **Deterministic Workflow** |
| Billing                  | ⚙️ Workflow + Approval        |
| Database CRUD            | ⚙️ API                        |
| Authentication           | ⚙️ API                        |
| RLS                      | ⚙️ Database                   |
| Cost control             | ⚙️ Control Plane              |

Ini sebenarnya salah satu keputusan arsitektur terpenting.

**Agent jangan menjadi pengganti backend.**

Agent adalah **problem solver**, sedangkan workflow adalah **process executor**.

---

## Kalau targetnya "CSRmatics hampir autopilot"

Saya akan membangun **7 AI worker utama**:

```text
                  CSRmatics
                      │
       ┌──────────────┼──────────────┐
       │              │              │
       ▼              ▼              ▼
  Research        Support          Dev
    Agent          Agent           Agent
       │
       ▼
┌─────────────────────────────────────┐
│       DETERMINISTIC AI WORKFLOWS    │
│                                     │
│ Extraction                          │
│ Entity Resolution                   │
│ Verification                        │
│ Intelligence                        │
│ Opportunity Discovery              │
│ Matching                            │
│ Notification                        │
└─────────────────────────────────────┘
       │
       ▼
   CSRmatics Data
       │
       ├── Companies
       ├── Signals
       ├── Opportunities
       ├── Matches
       └── Evidence
```

Dengan demikian **AI agent melakukan pekerjaan yang manusia biasanya lakukan**, sementara workflow mengerjakan pekerjaan mesin yang harus konsisten.

Dan menurut saya, untuk visi CSRmatics Anda, **Research Agent adalah pusat otomatisasi paling bernilai**, karena dari sana bisa menghidupkan hampir seluruh pipeline:

**Research → Evidence → Intelligence → Opportunity → Matching → Alert → Customer value.**
