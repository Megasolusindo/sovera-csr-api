# CSRmatics AI-Native Autonomous SaaS
## Grand Design & Implementation Specification — v1.2

**Tanggal:** 21 September 2026  
**Status:** Approved Architecture Baseline / Production-Grade Plan  
**Assumption:** CSRmatics berjalan dengan Go Fiber + PostgreSQL, memiliki API, dashboard web, admin platform, dan integrasi AI control plane.

---

# 1. Executive Summary & Operating Model

CSRmatics dirancang sebagai **AI-operated SaaS** — sebuah platform SaaS yang dioperasikan secara otonom oleh AI untuk tugas-tugas rutin dan berulang, sementara kendali infrastruktur, keamanan, hukum/keuangan, dan keputusan strategis tetap di tangan pemilik (*owner*).

```text
               OWNER
                 │
  ┌──────────────┼──────────────┐
  ▼              ▼              ▼
Infrastructure  Governance  Strategic Decisions
  │              │              │
  └──────────────┼──────────────┘
                 ▼
          AI OPERATIONS
                 │
   ┌─────────────┼─────────────┐
   ▼             ▼             ▼
 Data      Intelligence    Customer Ops
```

### Prinsip Utama Arsitektur (v1.2):
```text
Deterministic Workflow (DAG)
         +
LLM Step Terkontrol (Schema-bound)
         +
True Agent (Hanya untuk eksplorasi terbuka: Research, Support, Dev)
```

---

# 2. Arsitektur Anti-Prompt Injection & Privasi Watchlist

## 2.1 Scope Isolation & Research Agent Sandbox
Pengambilan data web dari sumber luar selalu berisiko *Prompt Injection*. Untuk mencegah kebocoran data (*exfiltration*), **Research Agent dijalankan dalam lingkungan terisolasi penuh**:

1. **100% GLOBAL Scope**: Research Agent **tidak memiliki akses** ke token JWT tenant, secret database, atau data privat pengguna.
2. **Hard Cap per Run**: Maksimal 5 langkah reasoning dan 10 call tool per eksekusi.
3. **Strict Egress Sanitization**: Tool fetcher memvalidasi dan membersihkan URL dari *query string* yang mencurigakan untuk mencegah eksfiltrasi data lewat parameter URL.

```text
Web Page (Untrusted)
       │
       ▼
Research Agent (GLOBAL Scope Only - No Secrets/No Tenant Context)
       │
       ▼
Fetcher Tool (SSRF Protection + Query String Filter)
       │
       ▼
Snapshot Storage (Untrusted Payload)
```

## 2.2 Anonimisasi & Agregasi Demand Watchlist (§22)
Watchlist NGO (target perusahaan, isu prioritas, daerah) adalah informasi strategis yang **tidak boleh bocor** ke pihak luar atau mesin pencari.

**Mekanisme Anonymized Demand Aggregation**:
```text
Tenant Watchlists (NGO A, NGO B, NGO C)
       │
       ▼ (Cron / System Aggregator)
Data Depersonalisasi & Agregasi Topik/Sektor
       │
       ▼
Global Priority Research Queue (No Tenant Identity)
       │
       ▼
Research Agent (Melakukan crawling secara anonim)
```
*Task riset yang berjalan ke web luar tidak pernah membawa `tenant_id` atau identitas NGO tertentu.*

## 2.3 Untrusted Data at Read Time & Support Agent Sandbox
Teks bebas hasil crawling (ringkasan, quote, deskripsi) yang tersimpan di database tetap berstatus `UNTRUSTED_DATA` saat dibaca oleh Support Agent via `search_intelligence()`.

1. **Read-Only Sandbox Tooling**: Support Agent hanya memiliki akses ke tool pembacaan internal dan notifikasi balik ke *session user* yang bertanya.
2. **No External Side-Effects**: Support Agent dilarang memicu email massal, webhook luar, atau pengubahan data secara mandiri.
3. **XSS Sanitization**: Seluruh teks bebas yang ditampilkan di dashboard web di-sanitasi menggunakan **DOMPurify** sebelum DOM rendering.

---

# 3. Risk Tiers, Permission Matrix, Approval Manager & Circuit Breaker

## 3.1 Risk Tiers & Approval Triggers
Setiap aksi sistem dikelompokkan ke dalam tingkatan risiko untuk menentukan tingkat otonomi:

| Risk Tier | Kategori Aksi | Contoh Aksi | Tingkat Otonomi |
|---|---|---|---|
| **Tier 1: Low** | Read-Only & Internal Classification | Query data publik, klasifikasi sinyal, pencocokan internal | Fully Autonomous |
| **Tier 2: Medium** | Content Drafting & Internal Alert | Pembuatan draf laporan, notifikasi dashboard internal | Sampling / Human Review |
| **Tier 3: High** | External Action & Critical Operations | Billing, refund, suspend akun, deploy prod, email massal, komitmen legal | **Mandatory Approval Required** |

```text
Agent / Workflow Request
         │
         ▼
Risk Tier Classifier
         │
 ┌───────┴───────┐
 │ High Risk?    │
 └───────┬───────┘
         ├──────────── YES ───────────► Approval Manager (Human Decision)
         │
         NO
         ▼
Execute via Policy
```

## 3.2 Matrix Izin Agent (Permission Matrix)

| Agent / Workflow | Scope | Read Access | Write Access | External Action | Approval Required |
|---|---|---|---|---|---|
| **Research Agent** | GLOBAL | Web / Snapshot | `source_snapshots`, `claims` | Controlled Fetch | Tidak (Terisolasi) |
| **Extraction WF** | GLOBAL | `source_snapshots` | `claims`, `extracted_json` | Tidak Ada | Tidak |
| **Entity Resolution WF**| GLOBAL | `companies`, `aliases` | `company_aliases` | Tidak Ada | Ya (Jika Ambigu) |
| **Verification WF** | GLOBAL | `claims`, `evidence` | `verified_signals` | Tidak Ada | Tidak |
| **Support Agent** | TENANT | `intelligence`, `tenant_profile`| `chat_history` | Session Alert Only | Tidak |
| **Dev Agent** | GLOBAL | Codebase / Issue | Git Branch / PR | Run Tests / CI | **Ya (Merge & Deploy)**|

## 3.3 Hard-Cap Cost Circuit Breaker
Sebagai perlindungan dari *runaway agents* (loop tanpa henti atau lonjakan biaya LLM):

```text
Hard Cap Limits:
- Max Cost / Task: $0.50 USD
- Max Daily Budget / Tenant: $10.00 USD
- Max Monthly System Budget: Configurable Threshold

Budget Monitoring Service (Per Second)
         │
         ▼ (Limit Exceeded?)
Automatic Circuit Breaker Triggered
         │
         ▼
Kill Switch Activated (Stop Task & Alert Admin)
```

## 3.4 Design Failure Handling
* **Retry with Exponential Backoff + Jitter**: Formula retry $T_{wait} = 2^{attempt} \times 1000\text{ms} + \text{jitter}(0-500\text{ms})$. Max retry 3 kali.
* **Model Fallback Chain**: Primary LLM (misal: Claude 3.5 / GPT-4o) $\rightarrow$ Secondary LLM (GPT-4o-mini / Llama 3.3) $\rightarrow$ Failure.
* **Dead Letter Queue (DLQ)**: Task yang gagal setelah max retry masuk ke `agent.tasks_dlq` untuk dianalisis di dashboard admin.

## 3.5 Abstraksi Provider LLM
Dukungan *multi-provider* (OpenAI, Anthropic, DeepSeek, Local Ollama) dibungkus dalam *internal Go LLM Adapter Interface* untuk mencegah *vendor lock-in*.

---

# 4. Penyelarasan Skema Database, Session & RLS Hardening

## 4.1 Penamaan Session Variable & Isolation Scope
Menyelaraskan variabel session PostgreSQL dengan konvensi DB yang sudah ada (`crawling_targets`):
* `app.current_org_id`
* `app.current_tenant_id`

## 4.2 Taksonomi Skema Database (Alined)
Sesuai dengan refactoring aktif di backend Go:

```text
core         → Users, Organizations, Tenancy, Subscriptions
company      → Canonical Companies, Aliases, Corporate Graph
intelligence → CSR Programs, CSR Signals, Opportunities, Matches
source       → Sources, Source Snapshots, Evidence, Claims
scraper      → Fetch Jobs, Fetch Results, Crawling Targets
agent        → Tasks, Runs, Tool Calls, Approvals, Policies, Cost Logs
```

## 4.3 RLS Hardening Rules
1. **Non-Owner Application Role**: Role koneksi DB aplikasi **bukan owner tabel** dan menerapkan `FORCE ROW LEVEL SECURITY`.
2. **Role DB Terpisah untuk Global Worker**: Worker `GLOBAL` menggunakan role DB terpisah yang **tidak memiliki GRANT** ke tabel-tabel ber-schema `core` atau `tenant`.
3. **Transaction-Bound `SET LOCAL`**: Set variabel session RLS **wajib** berada dalam blok transaksi:
   ```sql
   BEGIN;
   SELECT set_config('app.current_org_id', 'org_123', true); -- local to tx
   -- Execute queries...
   COMMIT;
   ```
4. **Fail-Closed Default**: Penggunaan RLS policy menggunakan `current_setting('app.current_org_id', true)` yang bernilai `NULL` (fail-closed) jika variabel tidak diset.

## 4.4 Granular API Tools Surface (`/api/v1/ai/tools/:name`)
Endpoint monolithic diganti menjadi endpoint granular agar Kill Switch & Otorisasi dapat dimatikan per-tool secara spesifik:

```text
POST /api/v1/ai/tools/search_companies
POST /api/v1/ai/tools/get_csr_signals
POST /api/v1/ai/tools/match_opportunity
POST /api/v1/ai/tools/send_session_alert
```

## 4.5 Spesifikasi Signed Task Context
Task context bertanda tangan dibuat saat penugasan task:
* **Algorithm**: HMAC-SHA256
* **TTL**: Short-lived (5 menit)
* **Payload**: `task_id`, `run_id`, `org_id`, `scope_type`, `allowed_tools[]`, `nonce`
* **Proteksi Replay**: `nonce` sekali pakai dicatat di Redis cache.

---

# 5. Penguatan Grounding, Rule Verifikasi & Syndication Tracking

## 5.1 Claim-Quote Alignment Verification
Satu kutipan di snapshot tidak otomatis membuktikan klaim benar. Verification Workflow menjalankan 3 pengujian deterministik:

1. **Temporal Alignment**: Tahun/tanggal acara dalam kutipan harus cocok dengan masa berlaku klaim.
2. **Intent & Negation Check**: Memastikan tidak ada kata negasi ("batal", "menunda") atau perbedaan intent ("berencana" vs "telah merealisasikan").
3. **Entity Match**: Nama perusahaan dalam kutipan harus ter-resolusi ke `company_id` yang sama.

## 5.2 Matrix Rule Verifikasi per Tipe Sinyal

| Tipe Sinyal | Syarat Sumber Minimum | Tindakan Jika Tidak Memenuhi |
|---|---|---|
| `FUNDING_SIGNAL` | 1x Tier A (Official Website/Report) ATAU 2x Tier B (Media Utama) | Status `PARTIALLY_VERIFIED` (Tidak masuk alert publik) |
| `PROGRAM_LAUNCH` | 1x Tier A ATAU 1x Tier B (dengan kutipan langsung & lokasi) | Status `UNVERIFIED` |
| `POLICY_CHANGE` | Wajib 1x Tier A (Kementerian / Official Press) | Rejected |

## 5.3 Syndication & Press Release Origin Tracking (§15)
Untuk mencegah 100 artikel sindikasi berita kloningan dihitung sebagai 100 sumber independen:
* Sistem mengekstrak **Press Release Hash / Wire Source Origin**.
* 100 artikel kloningan dari 1 siaran pers yang sama dihitung sebagai **1 Sumber Utama**.

---

# 6. Graph Lineage Retraction & Correction State Machine

## 6.1 Lineage Retraction Graph Traversal
Jika suatu klaim/sumber terbukti salah atau ditarik:

```text
[Claim Retracted]
       │
       ▼
[Signal Retracted / Corrected]
       │
       ▼
[Opportunity Invalidated]
       │
       ▼
[Match Score Recalculated]
       │
       ▼
[Alert Correction Notification Sent to Affected Tenants]
```

## 6.2 Integrasi State Machine Sinyal

```text
    Claim Verified (Tier A/B)
               │
               ▼
[UNVERIFIED] ──► [PARTIALLY_VERIFIED] ──► [VERIFIED]
     │                                         │
     │ (Source Error/Takedown)                 │ (Retraction Event)
     ▼                                         ▼
[REJECTED] ◄───────────────────────────── [RETRACTED]
```

## 6.3 Flexible Idempotency Key (§18)
Agar proses *reprocessing* dengan versi prompt/pipeline baru tidak terblokir oleh `UNIQUE` constraint:

$$\text{idempotency\_key} = \text{sha256}(\text{source\_id} + \text{task\_type} + \text{pipeline\_version} + \text{prompt\_version})$$

---

# 7. Rigor Statistik Golden Set & Realisme Jadwal 5–6 Minggu

## 7.1 Golden Set & Statistical Confidence
* 100 artikel pertama diposisikan sebagai **Smoke & Regression Set**.
* Setiap laporan evaluasi mencantumkan **95% Confidence Interval** (misal $n=100, p=95\% \implies \pm 4.3\%$).
* **Human Review Loop (§13)**: Setiap sampel hasil review manusia secara otomatis di-label dan di-umpankan balik (*feed back*) ke Golden Set agar berkembang dari 100 $\rightarrow$ 500 $\rightarrow$ 1.000+ data uji.

## 7.2 Realistis Timeline 5–6 Minggu (Thin Vertical Slice M0–M2)

30 Hari (5–6 Minggu) pertama difokuskan untuk membuktikan **1 Vertical Slice Pipeline**:

```text
Minggu 1: Control Plane Core, Schema Agent, Signed Task Context, Kill Switch, Golden Set Baseline.
Minggu 2: Research Agent (GLOBAL Scope), Secure Fetcher, SSRF & Injection Shield, Snapshot Storage.
Minggu 3: Canonical Entity Resolution, Alias Lookup, Event Dedup, Claim Extraction.
Minggu 4: Verification Workflow, Quote Grounding, Source Tiering Matrix, Signal Generation.
Minggu 5: Human Review Console, Retraction Lineage, Evaluation & Precision Tuning.
Minggu 6: Buffer, Security Audit, Final Production Rollout (Slice M0–M2).
```

---

# 8. Checklist Definition of Done — AI Feature

Suatu fitur AI dinyatakan **Production-Ready** apabila mematuhi:

- [ ] Scope terisolasi (GLOBAL / TENANT dengan Signed Context).
- [ ] Hard Cap Cost & Step Limit aktif.
- [ ] Permasalahan prompt injection & SSRF teruji secara otomatis.
- [ ] RLS Policy menggunakan `SET LOCAL` dan fail-closed.
- [ ] Lineage Retraction & State Machine terintegrasi.
- [ ] Evaluation Dataset & Margin of Error dilaporkan.
- [ ] Kill Switch manual dan otomatis (Circuit Breaker) teruji.

---

# 9. Aspek yang Perlu Diperhatikan Saat Implementasi (Implementation Watchlist)

## 9.1 Latensi & Concurrency Traversal Retraksi (Asynchronous Retraction Pipeline)
Jika grafik dependensi (`claim` → `signal` → `opportunity` → `match` → `alert`) telah mencapai puluhan ribu entri, penelusuran dependensi secara sinkron (*synchronous graph traversal*) dapat memicu *lock contention* dan *blocking* pada PostgreSQL.

* **Ketentuan Implementasi**:
  1. Proses pembatalan kaskade (*cascading retraction*) **wajib dijalankan secara asinkron** melalui antrean background task (`agent.tasks` / worker background).
  2. Gunakan pembagian *batch processing* (misal: 100 entri per batch) agar tidak menahan transaksi database yang lama.

## 9.2 Overhead & Eviction TTL Replay Cache (Redis Nonce Optimization)
Penanganan *replay nonce* untuk *Signed Context* berumur 5 menit membutuhkan pengelolaan memori Redis yang efisien saat *throughput* task tinggi.

* **Ketentuan Implementasi**:
  1. Gunakan nama kunci ter-prefix dengan penentuan TTL tepat (misal: `nonce:task:<task_id>:<nonce>`).
  2. Konfigurasikan *Redis Eviction Policy* ke `volatile-ttl` atau `allkeys-lru` agar Redis secara otomatis membersihkan kunci *short-lived* tanpa membebankan penggunaan memori utama.

## 9.3 Fallback LLM JSON Schema Compatibility & Deserialization Safety
Saat terjadi *fallback* model dari LLM Utama (Claude 3.5 / GPT-4o) ke LLM Sekunder (GPT-4o-mini / Llama 3.3 / Ollama), terdapat risiko model sekunder menghasilkan JSON yang tidak memenuhi skema.

* **Ketentuan Implementasi**:
  1. Pastikan model sekunder dikonfigurasi menggunakan mode *Strict JSON Schema / Structured Output*.
  2. Pada lapisan backend Go, wajib menyertakan **Go JSON Repair & Validation Guardrail** (`go-playground/validator`).
  3. Jika ekstraksi gagal diparse, hentikan eksekusi dengan status `EXTRACTION_FAILED` secara eksplisit dan catat error ke log — **dilarang keras** mengembalikan data dummy atau fallback kosong secara diam-diam.

