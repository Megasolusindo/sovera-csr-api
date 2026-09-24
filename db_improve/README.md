# Migration Plan — soveradb

## Urutan Eksekusi

Jalankan **secara berurutan**. Setiap migration adalah prasyarat untuk berikutnya.

| File | Tujuan | Risiko | Estimasi Waktu |
|------|--------|--------|----------------|
| `000001` | Merge 9.821 baris legacy → `company.companies` | Medium — bulk insert, slug conflict handling | < 1 menit |
| `000002` | Repoint FK dari `companies_legacy_backup` → `company.companies` | Low — setelah 001 selesai, tidak ada orphan | Detik |
| `000003` | Drop tabel kosong duplikat (intelligence, crm, scraper, source) + `organization_programs` + `ngo_programs` | Low — semua 0 baris | Detik |
| `000004` | Drop `companies_legacy_backup`, recreate view `public.companies` lengkap | Low — setelah 001–003 | Detik |
| `000005` | Fix bug kritis: billing CASCADE→RESTRICT, users email unique per org, crawling_targets unique per org, trigger sync claim | Medium — ALTER constraint, cek konflik email dulu | < 1 menit |
| `000006` | Drop `public.sources`, missing FK indexes, GIN indexes, trigger `updated_at` | Low | < 1 menit |
| `000007` | Perluas RLS ke semua tabel multi-tenant, composite FK untuk AI chat | Medium — perlu `app.current_user_id` di Go handler | Detik |
| `000008` | Partisi `crawling_logs` (104k baris) by month | **Tinggi** — butuh maintenance window | 1–2 menit |
| `000009` | Install pgvector, convert embedding columns | Medium — perlu pgvector di OS dulu | Detik |
| `000010` | Restrukturisasi tabel program (rename, tambah kolom, buat baru) | Medium — koordinasi dengan Go developer | < 1 menit |
| `000011` | Buat partnership_milestones, milestone_disbursements, milestone_reports, impact_reports | Low — tabel baru | < 1 menit |
| `000012` | Pindah crm.ai_token_logs + crm.tenant_templates ke public, drop schema crm, buat tenant_csr_focuses + tenant_esg_material_topics, RLS organization_prospects | Medium — migrasi data aktif (1.496 baris) | < 1 menit |
| `000015` | Drop view kompatibilitas deprecated (institution_programs, ngo_programs, company_csr_programs) | Low — pastikan tidak ada query yang masih pakai view ini | Detik |

---

## Prasyarat Sebelum Migration 001

```sql
-- Cek slug conflict yang akan di-handle dengan suffix:
SELECT b.slug, count(*)
FROM public.companies_legacy_backup b
JOIN company.companies c ON c.slug = b.slug AND c.id <> b.id
GROUP BY b.slug;

-- Cek email duplikat lintas org (prasyarat migration 005):
SELECT email, count(*) FROM public.users GROUP BY email HAVING count(*) > 1;
```

---

## Prasyarat Sebelum Migration 008 (partisi)

```sql
-- Cek range tanggal aktual di crawling_logs untuk menentukan partisi yang perlu dibuat:
SELECT
  date_trunc('month', min(created_at)) AS bulan_pertama,
  date_trunc('month', max(created_at)) AS bulan_terakhir,
  count(*) AS total
FROM public.crawling_logs;
```

Sesuaikan daftar partisi di `000008` dengan hasil query ini.

---

## Prasyarat Sebelum Migration 009 (pgvector)

```bash
# Di server PostgreSQL (Debian/Ubuntu):
apt install postgresql-18-pgvector

# Verifikasi:
psql -c "SELECT * FROM pg_available_extensions WHERE name = 'vector';"
```

---

## Perubahan yang Diperlukan di `sovera-csr-api` (Go)

### Setelah Migration 007 (RLS)

Setiap database transaction yang menyentuh tabel multi-tenant harus set session variables:

```go
// Di middleware atau setiap handler yang butuh akses data tenant
func withTenantContext(ctx context.Context, db *sql.DB, orgID, userID uuid.UUID) error {
    _, err := db.ExecContext(ctx,
        "SET LOCAL app.current_org_id = $1; SET LOCAL app.current_user_id = $2",
        orgID.String(), userID.String(),
    )
    return err
}
```

**Penting:** Gunakan `SET LOCAL` (bukan `SET`) agar nilai reset otomatis setelah transaksi selesai. Jangan pakai di luar transaksi.

### Setelah Migration 005 (users unique per org)

Query login yang sebelumnya:
```sql
SELECT * FROM users WHERE email = $1
```

Harus tetap bisa resolve tenant dari email. Pola yang disarankan:
```sql
-- Login: ambil semua org yang terkait email ini
SELECT u.*, o.slug AS org_slug
FROM users u
JOIN organizations o ON o.id = u.org_id
WHERE u.email = $1 AND u.is_active = true;
```

Jika user hanya di satu org, langsung login. Jika di beberapa org, tampilkan pilihan org (multi-org login flow).

### Setelah Migration 009 (pgvector)

Ganti pemanggilan manual similarity di Go dengan function SQL:
```sql
SELECT * FROM public.match_ngo_managed_programs(
  $1::vector,  -- embedding dari model
  0.7,         -- threshold
  10,          -- jumlah hasil
  $2           -- org_id filter (NULL = semua)
);
```

### Setelah Migration 010 (restrukturisasi program)

Query yang perlu diupdate di seluruh codebase:

| Referensi lama | Ganti dengan |
|----------------|--------------|
| `institution_programs` | `ngo_managed_programs` |
| `company_csr_programs` | `company_enriched_programs` |
| `ngo_programs` | `ngo_managed_programs WHERE visibility = 'published'` |
| `organization_programs` | dihapus, tidak ada padanan |

**Kolom baru di `ngo_managed_programs`** yang harus diisi saat INSERT:
- `visibility` — wajib eksplisit, default `'private'`. Program tidak muncul di marketplace sampai di-set `'published'`.
- `budget_needed` — opsional
- `fiqh_asnaf` — opsional
- `location` — opsional
- `target_beneficiaries_unit` — wajib diisi jika `target_beneficiaries` diisi
- `created_by_user_id` — isi dari JWT token user

**View kompatibilitas** tersedia sampai migration 015 (jadwal: 3 bulan setelah migration 010 deploy):
- `public.institution_programs` → alias ke `ngo_managed_programs`
- `public.ngo_programs` → alias ke `ngo_managed_programs WHERE visibility = 'published'`
- `public.company_csr_programs` → alias ke `company_enriched_programs`

Hapus semua referensi ke view ini sebelum migration 015 dijalankan.

---

## Tabel yang TIDAK ada dalam migration ini (perlu keputusan terpisah)

- `crm.ai_token_logs` — masih ada data? Jika kosong, drop. Jika tidak, migrate ke `public` atau biarkan.
- `crm.tenant_templates` — belum dikonfirmasi jumlah baris.
- `public.organization_prospects` — sales CRM internal, tidak ada RLS. Pertimbangkan apakah perlu.
- `public.csr_focuses` / `public.esg_material_topics` — taksonomi bersama, pertimbangkan schema `ref` tersendiri.

---

---

# ADR-001: Restrukturisasi Tabel Program

## Status
Diterima — September 2026

## Konteks

Database soveradb memiliki lima tabel terkait program yang tumpang tindih
fungsi dan tidak jelas kepemilikannya:

| Tabel | Baris | Masalah |
|-------|-------|---------|
| `institution_programs` | 377 | Nama ambigu — "institution" bisa NGO atau korporasi |
| `ngo_programs` | 0 | Duplikasi fungsi dengan `institution_programs` |
| `organization_programs` | 0 | Tidak jelas untuk siapa; scraping NGO tidak diperlukan |
| `company_csr_programs` | 76 | Tidak ada padanan untuk input manual korporasi |
| `intelligence.company_csr_programs` | 0 | Duplikat kosong dari versi public |

Selain itu, tidak ada tabel untuk program korporasi yang diinput manual
oleh admin tenant bertipe `CORPORATE`.

## Keputusan

### Konvensi naming

```
{pemilik}_{layer}_programs

pemilik : ngo / company
layer   : managed (input user) / enriched (scraping/AI)
```

### Tabel yang dihapus

| Tabel | Alasan |
|-------|--------|
| `ngo_programs` | 0 baris. Fungsinya (publish ke marketplace) cukup dengan kolom `visibility` di `ngo_managed_programs` |
| `organization_programs` | 0 baris. Scraping program NGO tidak diperlukan — NGO yang terdaftar input sendiri |
| `intelligence.company_csr_programs` | 0 baris, duplikat dari `public.company_csr_programs` |

### Tabel yang diubah nama + dimodifikasi

| Lama | Baru | Perubahan kolom |
|------|------|-----------------|
| `institution_programs` | `ngo_managed_programs` | Tambah: `visibility`, `budget_needed`, `fiqh_asnaf`, `location`, `target_beneficiaries_unit`, `created_by_user_id` |
| `company_csr_programs` | `company_enriched_programs` | Tidak ada perubahan kolom |

### Tabel baru

| Tabel | Fungsi |
|-------|--------|
| `company_managed_programs` | Program korporasi yang diinput manual oleh admin tenant `CORPORATE`. Bisa link ke `company_enriched_programs` jika berasal dari data scraping yang diklaim. |

### Keputusan produk yang mendasari

1. **NGO terdaftar + admin** → CRUD program via `ngo_managed_programs`
2. **NGO dari scraping** → belum punya program di platform (tidak perlu tabel scraping program NGO)
3. **Korporasi terdaftar + admin** → CRUD program via `company_managed_programs`
4. **Korporasi terdaftar, belum ada admin** → program dari scraping/AI di `company_enriched_programs`
5. **Setiap korporasi wajib terdaftar di `company.companies`** sebagai directory, sebelum bisa jadi tenant `organizations`

### Visibility program NGO

Published/unpublished adalah **toggle status**, bukan tabel terpisah.
`ngo_programs` (tabel terpisah untuk marketplace) tidak diperlukan.

```
ngo_managed_programs.visibility = 'private'    → draft internal
ngo_managed_programs.visibility = 'published'  → visible di marketplace
```

## Alur lengkap

```
KORPORASI                                    NGO
─────────────────────────────                ──────────────────────────────
company_enriched_programs                    ngo_managed_programs
(scraping BEI/berita/AI,                     (input admin NGO)
 read-only di dashboard corp)                visibility: private → published
        │                                            │
        │ admin corp klaim & enrich                  │
        ▼                                            │
company_managed_programs                             │
(CRUD oleh admin corp)                               │
        │                                            │
        │ buka tender spesifik                       │
        ▼                                            │
   csr_opportunities  ◄──────────────────────────────┘
   (tender dengan deadline & budget)      NGO browse & apply
        │
        ▼
     proposals
(NGO apply ke opportunity)
        │
        ▼
  partnership_milestones   ← belum dibuat, migration berikutnya
        │
        ▼
   impact_reports          ← belum dibuat, migration berikutnya
```

## Konsekuensi

### Positif
- Naming konsisten dan self-explanatory
- Tidak ada tabel kosong yang membingungkan
- Ada jalur jelas dari scraping → klaim → managed untuk kedua sisi

### Negatif / Risiko
- Semua query Go yang menyebut `institution_programs`, `ngo_programs`, `company_csr_programs` harus diupdate
- View kompatibilitas mengurangi risiko tapi menambah objek sementara di database
- `company_managed_programs` perlu dirancang kolom-kolomnya sebelum migration 010 dijalankan

## View kompatibilitas (sementara, hapus di migration 015)

```sql
CREATE VIEW public.institution_programs AS
  SELECT * FROM public.ngo_managed_programs;

CREATE VIEW public.ngo_programs AS
  SELECT * FROM public.ngo_managed_programs
  WHERE visibility = 'published';

CREATE VIEW public.company_csr_programs AS
  SELECT * FROM public.company_enriched_programs;
```

---

## Catatan Infrastruktur: Postgres Role untuk Internal Sovera

Migration 012 mengaktifkan RLS pada `organization_prospects` dengan policy
yang hanya mengizinkan role `sovera_internal`. Perlu disiapkan di database:

```sql
-- Jalankan sekali sebagai superuser, sebelum migration 012
CREATE ROLE sovera_internal LOGIN PASSWORD 'ganti_dengan_password_kuat';
GRANT USAGE ON SCHEMA public TO sovera_internal;
GRANT SELECT, INSERT, UPDATE, DELETE ON public.organization_prospects TO sovera_internal;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO sovera_internal;
```

Di Go backend, gunakan dua connection pool terpisah:

```go
// Pool untuk API publik (tenant user)
dbPublic, _ := sql.Open("pgx", os.Getenv("DATABASE_URL"))

// Pool untuk internal admin dashboard (Sovera team)
dbInternal, _ := sql.Open("pgx", os.Getenv("DATABASE_INTERNAL_URL"))
// DATABASE_INTERNAL_URL memakai credentials role sovera_internal
```

Variabel environment yang perlu ditambahkan:
- `DATABASE_URL` — role sovera_app (existing)
- `DATABASE_INTERNAL_URL` — role sovera_internal (baru)
