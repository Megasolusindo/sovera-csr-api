Rancangan Role-Based Access Control (RBAC) di platform Sovera memadukan **isolasi horizontal multi-tenancy** via PostgreSQL Row-Level Security (RLS) dengan **otorisasi vertikal berbasis peran** di tingkat aplikasi.

---

### 1. Hierarki & Matriks Peran Pengguna Organisasi

Di level tenant (organisasi lembaga kemanusiaan/NGO/Zakat), terdapat 3 peran utama:

| Hak Akses / Modul | `ORG_ADMIN` (Pimpinan / IT) | `DIRECTOR` (Head of Partnership)

 | `FUNDRAISER` (Account Executive)

 |
| --- | --- | --- | --- |
| **Intel Feed & Matching** (`/signals`) | Lihat sinyal pasar & rekomendasi program

 | Lihat sinyal & filter potensi strategis

 | Eksplorasi sinyal & klaim prospek

 |
| **Master Program** (`/programs`) | Buat, Edit, Hapus portofolio program

 | Buat & Edit program lembaga

 | Hanya Lihat portofolio program

 |
| **Deal Pipeline** (`/pipeline`) | Akses & kelola seluruh deals organisasi

 | Pantau seluruh deals tim & nilai agregat

 | Kelola deals milik sendiri (*PIC*)

 |
| **Proposal Studio** (`/pipeline/:id`) | Akses editor & ekspor naskah

 | Akses editor, review, & ekspor berkas

 | Generate AI & edit proposal pribadi

 |
| **Kelola Tim & Tenant** (`/settings`) | Undang staf, ubah role, atur profil

 | Hanya lihat daftar anggota tim

 | Tidak memiliki akses |

---

### 2. Skema Database & Enum Role

Implementasi RBAC disimpan pada tabel `users` dengan foreign key ke `organizations`:

```sql
-- 1. Definisi Enum Role Organisasi
CREATE TYPE user_org_role AS ENUM (
    'ORG_ADMIN',
    'DIRECTOR',
    'FUNDRAISER'
);

-- 2. Modifikasi / Pembuatan Tabel Users
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    full_name VARCHAR(150) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role user_org_role NOT NULL DEFAULT 'FUNDRAISER',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexing untuk query login dan tenant lookups
CREATE INDEX idx_users_org_id ON users(org_id);

```

---

### 3. Integrasi RBAC dengan JWT & PostgreSQL RLS

Otorisasi dieksekusi dalam dua lapis:

1. **Lapis 1 (Kernel Database - RLS):** Memastikan pengguna hanya bisa mengakses baris data milik `org_id` organisasinya sendiri, apa pun rolenya.


2. **Lapis 2 (Application Middleware):** Membatasi mutasi data berdasarkan klaim `role` yang tertera di JWT.



#### Payload Token JWT

Saat pengguna login, backend meng-encode klaim esensial berikut:

```json
{
  "sub": "b2c9d618-912b-4e67-8e6d-742bc511aa12",
  "org_id": "99f182ea-2009-4ce4-89c0-998811223344",
  "role": "FUNDRAISER",
  "exp": 1725350400
}

```

#### Middleware Otorisasi (Go-Fiber / TypeScript)

Handler rute privat memeriksa izin sebelum memproses request:

```go
// Contoh Middleware Guard di Go-Fiber
func RequireRole(allowedRoles ...string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        userRole := c.Locals("user_role").(string)
        
        for _, role := range allowedRoles {
            if role == userRole {
                return c.Next()
            }
        }
        
        return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
            "success": false,
            "error":   "INSUFFICIENT_PERMISSIONS",
            "message": "Anda tidak memiliki wewenang untuk tindakan ini.",
        })
    }
}

```

---

### 4. Batasan Akses Spesifik di Modul Pipeline (Fundraiser vs Director)

* **Ownership Isolation pada Deals:**
Setiap baris di tabel `deal_pipelines` memiliki kolom `pic_user_id`.


* Jika pengguna adalah `FUNDRAISER`: Query otomatis menyaring `WHERE org_id = :orgId AND (pic_user_id = :userId OR pic_user_id IS NULL)`.
* Jika pengguna adalah `DIRECTOR` atau `ORG_ADMIN`: Query membaca seluruh deals lembaga tanpa batasan `pic_user_id`.





Dengan rancangan ini, kerahasiaan pipeline antar-fundraiser tetap terjaga jika diperlukan, pimpinan mendapatkan visibilitas analitik menyeluruh, dan integritas data tenant terkunci aman di level database.