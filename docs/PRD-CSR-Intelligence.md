# PRD — CSR Intelligence

**Product:** CSRmatics  
**Module:** CSR Intelligence  
**Route:** `/intelligence`  
**Primary Persona:** Corporate / CSR / TJSL / ESG Team  
**Document Status:** Draft  
**Version:** 1.0

---

## 1. Overview

CSR Intelligence adalah workspace intelligence pada CSRmatics yang membantu **Corporate** memahami ekosistem CSR dan menemukan organisasi, program, peluang, serta tren yang relevan dengan kebutuhan CSR perusahaan.

Module ini berbeda dengan **Corporate Feed (`/signals`)**.

- `/signals` berfokus pada **Company Signals**, yaitu sinyal aktivitas dan intent CSR perusahaan.
- `/intelligence` berfungsi sebagai **workspace analitik dan discovery** untuk Corporate, dengan menggabungkan berbagai sumber intelligence yang relevan.

Tujuan utamanya:

> **Help corporate discover the right organizations, programs and CSR opportunities using data-driven intelligence.**

---

## 2. Problem Statement

Tim CSR/TJSL/ESG perusahaan sering menghadapi beberapa masalah:

1. Sulit menemukan organisasi kemanusiaan/NGO yang sesuai dengan kebutuhan program.
2. Informasi mengenai aktivitas organisasi tersebar di banyak sumber.
3. Sulit mengetahui organisasi mana yang aktif pada bidang tertentu.
4. Sulit mengetahui program sosial yang sedang berjalan.
5. Sulit membandingkan kandidat organisasi secara objektif.
6. Sulit menemukan partner yang sesuai dengan wilayah, pilar CSR, beneficiary, dan SDGs perusahaan.
7. Data CSR industri dan aktivitas perusahaan lain sulit dipantau secara terstruktur.
8. Proses discovery masih banyak dilakukan secara manual melalui Google, media sosial, berita, dan networking.

CSR Intelligence menjadi layer yang mengubah data mentah menjadi informasi yang dapat digunakan untuk pengambilan keputusan.

---

## 3. Goals

### 3.1 Primary Goals

- Menjadi pusat intelligence CSR untuk Corporate.
- Membantu Corporate menemukan organisasi yang relevan.
- Membantu Corporate menemukan program yang relevan.
- Memberikan rekomendasi partner berdasarkan data.
- Memberikan insight mengenai tren CSR.
- Menyediakan contextual intelligence sebelum Corporate melakukan partnership.
- Mengurangi waktu riset partner CSR.

### 3.2 Secondary Goals

- Mendorong Corporate menggunakan fitur Organization Directory.
- Mendorong Corporate membuat dan membuka CSR Opportunity.
- Menjadi foundation untuk AI Assistant CSRmatics.
- Menjadi foundation untuk scoring dan matching engine.

---

## 4. Non-Goals

Untuk versi awal, CSR Intelligence **bukan**:

- Sistem accounting CSR.
- Sistem pengelolaan dana CSR.
- Sistem monitoring project secara penuh.
- Sistem procurement.
- Sistem legal contract management.
- Social media management.
- Platform fundraising publik.
- Pengganti CRM partnership.

Fungsi tersebut dapat menjadi module terpisah di masa depan.

---

# 5. Target User

## Primary User

### Corporate CSR/TJSL Manager

Kebutuhan:

- mencari organisasi partner;
- mencari program;
- memahami aktivitas CSR;
- menemukan peluang partnership;
- membuat shortlist partner.

### Corporate ESG/Sustainability Team

Kebutuhan:

- melihat tren CSR;
- benchmarking;
- melihat aktivitas perusahaan lain;
- menemukan organisasi untuk program sosial;
- memahami landscape CSR.

## Secondary User

- CSR Officer
- TJSL Officer
- Sustainability Officer
- Community Development Officer
- Corporate Affairs
- Foundation/CSR Manager

---

# 6. User Journey

## Main Journey

```text
Login
  ↓
Corporate Dashboard
  ↓
CSR Intelligence
  ↓
View Intelligence Overview
  ↓
Explore Organizations / Programs / Trends
  ↓
Open Organization Profile
  ↓
Evaluate Match Score
  ↓
Save Organization
  ↓
Add to Partnership Pipeline
  ↓
Contact / Proposal / Partnership
```

## Alternative Journey

```text
CSR Intelligence
  ↓
Recommended Organizations
  ↓
Select Organization
  ↓
View Organization Intelligence
  ↓
Compare with CSR Program
  ↓
Match Score
  ↓
Add to Pipeline
```

---

# 7. Information Architecture

Route:

```text
/intelligence
```

Primary sections:

```text
CSR Intelligence
├── Overview
├── Organizations
├── Programs
├── Opportunities
├── Industry Insights
└── Trends
```

Untuk MVP, semua section dapat berada dalam satu halaman dengan tab/filter.

Recommended UI:

```text
CSR Intelligence

[Overview] [Organizations] [Programs] [Opportunities] [Trends]
```

---

# 8. Dashboard / Overview

## 8.1 Header

Title:

> CSR Intelligence

Subtitle:

> Discover organizations, programs and CSR opportunities that match your company's social impact goals.

Actions:

- Search
- Filter
- Save Search
- Ask AI

---

## 8.2 KPI Cards

Contoh:

```text
Organizations
2,438

Active Programs
816

New Opportunities
37

Recommended Partners
24
```

KPI harus bersifat dynamic berdasarkan:

- tenant;
- industry;
- CSR focus;
- geographic coverage;
- saved preferences.

---

# 9. Recommended Organizations

Section:

> Recommended Organizations

Menampilkan organisasi yang memiliki relevansi tertinggi dengan corporate.

Contoh:

```text
Yayasan ABC
94% Match

Focus:
Education · Health

Regions:
West Java · Jakarta

Programs:
18

Active Programs:
5

[View Profile]
[Add to Pipeline]
```

## Matching Factors

Minimal:

- CSR pillar;
- program category;
- geographic coverage;
- beneficiary;
- SDGs;
- organization activity;
- historical partnership;
- corporate industry relevance;
- current program activity.

---

# 10. Organization Intelligence

Corporate dapat membuka detail organisasi.

Route:

```text
/organizations/{slug}
```

atau:

```text
/intelligence/organizations/{id}
```

## Information

### Profile

- Nama organisasi
- Jenis organisasi
- Website
- Tahun berdiri
- Lokasi
- Wilayah operasi
- Legal status
- Description

### Focus Areas

- Education
- Health
- Environment
- Disaster
- Poverty
- Community Development
- Economic Empowerment
- Disability
- Children
- Women
- Other

### Geographic Coverage

Contoh:

```text
National
West Java
Central Java
East Java
Kalimantan
Sulawesi
```

### Program Activity

```text
Total Programs: 42
Active Programs: 8
Completed Programs: 34
```

### CSR/Partnership History

Jika tersedia:

- corporate partners;
- program partners;
- historical collaborations.

### Verification

Badge:

```text
Verified Organization
```

Jika profil telah diklaim/diverifikasi oleh organisasi.

---

# 11. Organization Match Score

Setiap organisasi dapat mempunyai score:

```text
94% Match
```

Breakdown:

```text
CSR Focus          96
Geographic Fit     92
Beneficiary Fit    95
Program Fit        94
Activity Level     91
```

Sistem harus menjelaskan:

> Why this organization matches

Contoh:

```text
✓ Active in healthcare programs
✓ Operates in East Java
✓ Has experience with corporate partnerships
✓ Works with your target beneficiary
✓ Has active programs in 2026
```

Jangan hanya menampilkan angka tanpa alasan.

---

# 12. Programs Intelligence

Corporate dapat melihat program yang sedang atau pernah dijalankan organisasi.

Contoh:

```text
Program:
Mobile Health Clinic

Organization:
Yayasan ABC

Category:
Health

Location:
East Java

Beneficiaries:
Low-income communities

Status:
Active

[View Program]
```

Filter:

- Category
- Location
- SDG
- Beneficiary
- Status
- Organization
- Date

---

# 13. Opportunity Intelligence

Opportunity adalah program yang secara eksplisit membutuhkan partner atau membuka peluang collaboration.

Contoh:

```text
Open Partnership Opportunity

Community Health Program 2027

Organization:
Yayasan ABC

Location:
Central Java

Funding Need:
Rp500M

Focus:
Healthcare

Deadline:
30 Nov 2026

[View Opportunity]
[Add to Pipeline]
```

Opportunity berbeda dengan Program biasa.

### Program

> Program yang sedang/akan dijalankan.

### Opportunity

> Program yang secara eksplisit membuka peluang partnership.

---

# 14. Industry Intelligence

Section:

> Industry Insights

Tujuan:

membantu Corporate memahami aktivitas CSR berdasarkan industri.

Contoh:

```text
Energy
────────────────────
CSR Activities       128
Active Companies      42
Top Pillar:
Community Development

Banking
────────────────────
CSR Activities       214
Active Companies      57
Top Pillar:
Education
```

Data dapat menggunakan `intelligence.company_signals`.

---

# 15. Company Signals Integration

Existing table:

```text
intelligence.company_signals
```

tetap menjadi sumber utama Company Signal.

Contoh data:

- company_id
- company_name
- industry_sector
- source_type
- source_url
- summary
- extracted_pillar
- target_regions
- estimated_budget_signal
- trigger_event
- intent_score
- csr_relevance
- opportunity_alert
- published_date

CSR Intelligence dapat menggunakan data tersebut untuk membuat:

### Industry Activity

```text
Companies with recent CSR activity
```

### CSR Trends

```text
Most active CSR pillars
```

### Geographic Trends

```text
Regions with increasing CSR activity
```

### Competitive Intelligence

```text
Companies active in similar CSR areas
```

Catatan:

> `/signals` tetap menjadi dedicated Company Feed untuk NGO. `/intelligence` menggunakan data intelligence yang sama sebagai salah satu sumber analisis untuk Corporate.

---

# 16. CSR Trends

Contoh:

```text
CSR Trends — Last 30 Days

Education              ↑ 18%
Health                 ↑ 12%
Environment             ↑ 9%
Community Development   ↑ 7%
Disaster Relief         ↑ 4%
```

Trend dapat dihitung berdasarkan:

- jumlah signals;
- jumlah programs;
- jumlah companies;
- jumlah organizations;
- geographic activity.

Trend harus memiliki periode yang jelas.

Contoh:

> Compared with previous 30 days.

---

# 17. Geographic Intelligence

Menampilkan persebaran aktivitas CSR.

Contoh:

```text
CSR Activity by Region

West Java       238
East Java       194
Central Java    176
Jakarta         151
South Sulawesi   87
```

Future UI:

- map;
- heatmap;
- region ranking.

MVP dapat dimulai dengan tabel/list.

---

# 18. Search

Global search pada CSR Intelligence.

User dapat mencari:

```text
Yayasan kesehatan
```

```text
NGO di Jawa Barat
```

```text
program pendidikan anak
```

```text
organisasi yang aktif menangani bencana
```

Search harus mendukung:

- organization;
- program;
- opportunity;
- location;
- category;
- SDG;
- beneficiary.

Future:

Natural Language Search menggunakan AI.

---

# 19. Filter

Filter minimal:

### Organization

- Type
- Focus area
- Location
- Verification
- Activity level

### Program

- Pillar
- SDG
- Location
- Beneficiary
- Status
- Date

### Opportunity

- Pillar
- Location
- Funding range
- Deadline
- Status

### Company Intelligence

- Industry
- CSR pillar
- Region
- Intent score
- Date
- Opportunity alert

---

# 20. Save / Bookmark

Corporate dapat menyimpan:

- Organization
- Program
- Opportunity
- Company Signal

Contoh:

```text
Saved Organizations
Saved Programs
Saved Opportunities
```

Data bersifat tenant-specific.

---

# 21. Add to Partnership Pipeline

Dari CSR Intelligence user dapat melakukan:

```text
[Add to Pipeline]
```

Target dapat berupa:

- Organization;
- Opportunity;
- Program.

Contoh:

```text
Yayasan ABC
        ↓
Add to Partnership Pipeline
        ↓
Prospect
```

Tidak perlu membuat CRM baru di `/intelligence`.

---

# 22. AI Integration

AI Assistant dapat menggunakan CSR Intelligence sebagai context.

Contoh query:

> "Cari 10 organisasi terbaik untuk program kesehatan kami di Jawa Barat."

AI harus:

1. memahami kebutuhan;
2. mencari organisasi;
3. melakukan ranking;
4. memberikan match score;
5. menjelaskan alasan;
6. menyediakan link ke profile.

Contoh:

```text
Top Recommendations

1. Yayasan ABC — 94%
2. NGO XYZ — 91%
3. Yayasan DEF — 88%
```

---

# 23. Data Model

Konsep entity:

```text
Corporate Tenant
      │
      ├── CSR Preferences
      │
      ├── Saved Organizations
      ├── Saved Programs
      ├── Saved Opportunities
      └── Partnership Pipeline
```

Shared intelligence:

```text
Company
Company Signal
Organization
Organization Program
CSR Opportunity
CSR Activity
CSR Trend
```

Tenant-specific data tidak boleh tercampur dengan global intelligence.

---

# 24. Recommended Database Structure

Existing:

```text
intelligence.company_signals
```

Future:

```text
intelligence.company_signals
intelligence.organization_signals
intelligence.csr_activities
intelligence.csr_trends
```

Domain data:

```text
company.companies
organization.organizations
program.programs
opportunity.opportunities
```

Tenant-specific:

```text
tenant.saved_organizations
tenant.saved_programs
tenant.saved_opportunities
tenant.csr_preferences
```

Nama schema dapat disesuaikan dengan arsitektur existing CSRmatics.

---

# 25. API Requirements

Recommended endpoints:

```http
GET /api/v1/intelligence
```

Overview.

```http
GET /api/v1/intelligence/organizations
```

Recommended/discovered organizations.

```http
GET /api/v1/intelligence/organizations/:id
```

Organization intelligence detail.

```http
GET /api/v1/intelligence/programs
```

Program discovery.

```http
GET /api/v1/intelligence/opportunities
```

Opportunity discovery.

```http
GET /api/v1/intelligence/trends
```

CSR trends.

```http
GET /api/v1/intelligence/industries
```

Industry insights.

```http
GET /api/v1/intelligence/regions
```

Geographic insights.

```http
GET /api/v1/intelligence/search
```

Cross-entity search.

---

# 26. API Query Parameters

Example:

```http
GET /api/v1/intelligence/organizations?
    search=health&
    pillar=health&
    region=west-java&
    min_match_score=80&
    verified=true
```

Program:

```http
GET /api/v1/intelligence/programs?
    pillar=education&
    region=west-java&
    status=active
```

Opportunity:

```http
GET /api/v1/intelligence/opportunities?
    pillar=health&
    region=central-java&
    status=open
```

---

# 27. Authorization

CSR Intelligence adalah fitur Corporate.

Authorization:

```text
tenant.type = corporate
```

Organization/NGO tenant tidak boleh mengakses Corporate Intelligence workspace melalui UI.

Namun beberapa public intelligence data dapat tetap tersedia melalui:

- public pages;
- shared APIs;
- NGO-specific modules.

---

# 28. Personalization

Dashboard harus dipersonalisasi berdasarkan Corporate Profile.

Input:

```text
Industry
CSR Pillars
Operating Regions
Target Beneficiaries
SDGs
Preferred Program Types
```

Contoh:

Corporate:

```text
Industry:
Mining

CSR Focus:
Community Development
Education
Health

Regions:
Kalimantan
Sulawesi
```

Maka recommendation engine memberikan prioritas pada organisasi/program yang sesuai.

---

# 29. Scoring Model — MVP

Initial scoring:

```text
Match Score =
    Pillar Fit           25%
  + Geographic Fit       20%
  + Beneficiary Fit      15%
  + Program Fit          15%
  + Activity Level       10%
  + Partnership History  10%
  + Verification          5%
```

Bobot harus configurable di backend.

Future:

- ML ranking;
- behavioral signals;
- historical conversion;
- proposal success;
- corporate feedback.

---

# 30. UX Requirements

## Principle

CSR Intelligence harus terasa seperti:

> **decision support system**

bukan:

> news portal.

Prioritas informasi:

1. Recommendation
2. Match
3. Why it matches
4. Evidence
5. Action

Setiap card idealnya memiliki CTA.

Contoh:

```text
Yayasan ABC
94% Match

Why:
✓ Health
✓ West Java
✓ Corporate partnership experience

[View Profile]
[Add to Pipeline]
```

---

# 31. Evidence & Data Trust

Setiap intelligence yang berasal dari external source harus memiliki:

- source;
- source URL;
- published date;
- data timestamp;
- confidence bila tersedia.

Contoh:

```text
Source:
Company Sustainability Report 2026

Published:
12 Aug 2026

Source:
[View Source]
```

AI tidak boleh membuat klaim tanpa evidence.

---

# 32. MVP Scope

MVP CSR Intelligence cukup mencakup:

### P0

- Intelligence Overview
- Recommended Organizations
- Organization Search
- Organization Detail
- Program Search
- Program Detail
- Opportunity Search
- Match Score
- Save Organization
- Add to Partnership Pipeline
- Basic CSR Trends
- Company Signal aggregation

### P1

- Geographic Intelligence
- Industry Intelligence
- Advanced filtering
- Saved Search
- Alerts
- AI natural-language search

### P2

- Predictive opportunity scoring
- Advanced benchmarking
- AI-generated intelligence report
- Automated competitor analysis
- Custom intelligence dashboard

---

# 33. Success Metrics

### Product Metrics

- Time to find relevant organization.
- Organization profile views.
- Search usage.
- Match usage.
- Saved organizations.
- Add-to-pipeline rate.
- Opportunity views.
- Proposal initiation.

### Business Metrics

- Corporate activation rate.
- Corporate weekly active users.
- Conversion from Free → Paid.
- Number of corporate organizations using matching.
- Number of partnership opportunities generated.
- Partnership conversion rate.

Primary KPI yang direkomendasikan:

> **Qualified Partnership Discovery**

Definisi:

> Corporate menemukan organization/program dengan match score yang relevan dan memasukkannya ke Partnership Pipeline.

---

# 34. Relationship dengan Sidebar Corporate

Recommended sidebar:

```text
Overview
AI Chat Assistant
Organization Directory
CSR Intelligence
Programs
Partnership Pipeline
Proposal Masuk
Settings
```

Mapping:

```text
Organization Directory
        ↓
/organizations

CSR Intelligence
        ↓
/intelligence

Programs
        ↓
/programs

Partnership Pipeline
        ↓
/partnerships

Proposal Masuk
        ↓
/proposals
```

---

# 35. Relationship dengan NGO Dashboard

NGO tetap memiliki:

```text
Overview
AI Chat Assistant
Corporate Directory
Corporate Feed
Programs
Deal Pipeline
Proposal Terkirim
Settings
```

Corporate Feed:

```text
/signals
```

menggunakan:

```text
intelligence.company_signals
```

Corporate Intelligence:

```text
/intelligence
```

menggabungkan intelligence yang lebih luas.

Tidak perlu membuat `/intelligence` sebagai duplicate `/signals`.

---

# 36. Future Architecture

Target jangka panjang:

```text
                    CSRmatics Intelligence Layer
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
 Company Intelligence   Organization Intelligence   Program Intelligence
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              │
                              ▼
                       Matching Engine
                              │
               ┌──────────────┴──────────────┐
               ▼                             ▼
              NGO                        Corporate
               │                             │
       Find Companies                Find Organizations
       Find Opportunities            Find Programs
       Send Proposals                Receive Proposals
```

---

# 37. Product Principle

CSR Intelligence harus mengikuti prinsip:

> **Data → Intelligence → Recommendation → Action**

Bukan:

> Data → List → End

Contoh:

```text
Company/Organization Data
          ↓
CSR Activity
          ↓
Intelligence
          ↓
Match Score
          ↓
Recommendation
          ↓
Add to Pipeline
          ↓
Partnership
```

Inilah yang membedakan CSRmatics dari directory biasa.

---

# 38. Final Recommendation

`/intelligence` sebaiknya diposisikan sebagai **strategic intelligence workspace untuk Corporate**, bukan sekadar halaman feed.

Untuk MVP, jangan mencoba memasukkan seluruh fitur analytics sekaligus.

Prioritaskan:

```text
1. Recommended Organizations
2. Organization Intelligence
3. Program Discovery
4. Opportunity Discovery
5. Match Score
6. Save
7. Add to Partnership Pipeline
8. Basic Industry/CSR Trends
```

Sedangkan `intelligence.company_signals` yang saat ini sudah ada tetap dipertahankan sebagai foundation untuk Company Intelligence dan dapat digunakan untuk industry/trend analysis.

Target akhirnya:

> **Corporate datang ke CSRmatics bukan hanya untuk mencari NGO, tetapi untuk mengetahui siapa partner yang paling tepat, mengapa mereka cocok, dan apa yang harus dilakukan selanjutnya.**
