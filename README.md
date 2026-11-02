# Task Manager API

REST API manajemen task dan project, dibangun dengan **Go standard library** (`net/http`) tanpa framework web, PostgreSQL, dan JWT. Proyek portfolio untuk menunjukkan alur backend yang lengkap: autentikasi, otorisasi berlapis, transaksi, paginasi/filter, observability, worker background, dan deployment yang reprodusibel.

## Masalah yang diselesaikan

Tim kecil butuh satu tempat untuk mengelola pekerjaan lintas project: siapa mengerjakan apa, kapan tenggatnya, dan siapa boleh mengubahnya. Aplikasi ini menyediakan API multi-tenant dengan batas akses yang jelas:

- Setiap user hanya melihat task miliknya, task yang di-assign kepadanya, atau task dari project tempat ia menjadi anggota.
- Project punya hierarki peran `owner > admin > member`.
- Perubahan data sensitif (hapus project, ubah role) dibatasi ke peran tertentu.

## Fitur

- **Auth**: register/login, bcrypt, JWT HS256, rate limit endpoint sensitif.
- **Profil**: lihat profil, ubah email, ganti password (verifikasi password lama).
- **Task**: CRUD, status (`todo`/`in_progress`/`done`), priority (`low`→`urgent`), deadline, deskripsi, assignment, kaitan ke project.
- **List cerdas**: paginasi, filter status/priority/project, pencarian judul+deskripsi, sorting dengan whitelist kolom.
- **Project**: CRUD, owner otomatis, penghapusan berantai (task + member ikut terhapus).
- **Membership & RBAC**: tambah/keluarkan member, ubah role, proteksi agar owner tidak bisa dihapus/diturunkan.
- **Dashboard**: agregasi total task, per status, per priority, overdue, jatuh tempo hari ini, jumlah project.
- **Worker background**: pemindaian task jatuh tempo secara berkala, berhenti bersih saat shutdown.
- **Observability**: log JSON terstruktur (`log/slog`), request logging, panic recovery, health check.

## Stack

| Lapisan | Pilihan |
|---|---|
| Bahasa | Go 1.27 |
| HTTP | `net/http` + `ServeMux` (pattern `METHOD /path/{id}`) |
| Database | PostgreSQL 18 via `database/sql` + `jackc/pgx/v5/stdlib` |
| Auth | `golang-jwt/jwt/v5` (HS256), `golang.org/x/crypto/bcrypt` |
| Test | `testing`, `httptest`, integration test ke PostgreSQL |
| Deployment | Docker multi-stage, Docker Compose, GitHub Actions |

Tanpa framework web, tanpa ORM, tanpa Redis — setiap dependency harus punya alasan.

## Arsitektur

```
main.go                     wiring dependency, routing, lifecycle
internal/config             pembacaan + validasi environment
internal/database           koneksi PostgreSQL (pool tuning)
internal/auth               generate/validasi JWT + middleware
internal/model              entity, validasi domain, filter, paginasi
internal/repository         SQL berparameter, transaksi, agregasi
internal/service            aturan bisnis + otorisasi
internal/handler            HTTP: decode, panggil service, tulis response
internal/response           helper JSON + envelope paginasi
internal/middleware         request log, panic recovery, rate limit
internal/worker             background reminder scanner
migrations/                 SQL manual berurutan (000001–000005)
openapi.yaml                spesifikasi OpenAPI 3.0.3
```

Alur request:

```
HTTP → middleware (recover → log) → auth middleware → handler → service → repository → PostgreSQL
```

Aturan penting: SQL hanya ada di `repository`, otorisasi hanya di `service`, dan handler tidak pernah menyentuh database langsung.

## ERD

```mermaid
erDiagram
    users ||--o{ tasks : "owns"
    users ||--o{ tasks : "assigned"
    users ||--o{ project_members : "joins"
    projects ||--o{ project_members : "has"
    projects ||--o{ tasks : "contains"

    users {
        bigserial id PK
        text email UK
        text password_hash
        timestamptz created_at
    }
    projects {
        bigserial id PK
        text name
        text description
        bigint owner_id FK
        timestamptz created_at
        timestamptz updated_at
    }
    project_members {
        bigserial id PK
        bigint project_id FK
        bigint user_id FK
        text role "owner|admin|member"
        timestamptz created_at
    }
    tasks {
        bigserial id PK
        bigint user_id FK
        bigint project_id FK
        bigint assignee_id FK
        text title
        text description
        text status "todo|in_progress|done"
        text priority "low|medium|high|urgent"
        date deadline
        timestamptz created_at
        timestamptz updated_at
    }
```

## Menjalankan

### 1. Prasyarat

- Go 1.27+
- PostgreSQL 18 (lokal atau Docker)
- `psql` untuk menjalankan migrasi

### 2. Environment

```bash
export DATABASE_URL="postgres://taskapi:devpassword@127.0.0.1:5432/taskdb?sslmode=disable"
export JWT_SECRET="minimal-32-karakter-jangan-di-commit"
```

Variabel opsional:

| Variabel | Default | Keterangan |
|---|---|---|
| `PORT` | `8081` | Port HTTP |
| `REMINDER_ENABLED` | `true` | Aktifkan worker reminder |
| `REMINDER_INTERVAL` | `5m` | Interval pemindaian worker |
| `ALLOWED_ORIGINS` | _(kosong)_ | Daftar origin CORS dipisah koma, mis. `http://localhost:5173` |

### 3. Migrasi (berurutan)

```bash
for f in migrations/*.sql; do psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$f"; done
```

> `000003` memuat `TRUNCATE TABLE tasks;` sehingga destruktif. Jangan jalankan pada database berisi data yang ingin dipertahankan. Migrasi `000005` idempoten dan tidak menghapus data.

### 4. Jalankan

```bash
go run .          # server di :8081
go build -o task-api . && ./task-api
```

### 5. Docker Compose

```bash
cp .env.example .env    # isi JWT_SECRET
docker compose up --build
```

## Testing

```bash
# Unit test (tanpa database)
go test ./...

# Integration test (butuh PostgreSQL terpisah)
TEST_DATABASE_URL="postgres://taskapi:devpassword@127.0.0.1:5432/taskdb_test?sslmode=disable" go test ./...

# Race detector
TEST_DATABASE_URL="..." go test -race ./...

# Kualitas kode
go vet ./...
gofmt -l .
```

Integration test dijaga agar hanya berjalan pada database berakhiran `_test` di host loopback, sehingga tidak mungkin menghapus data database development.

Hasil verifikasi terakhir (2 Nov 2026):

```
ok  task-api/internal/auth
ok  task-api/internal/config
ok  task-api/internal/handler     (integration, PostgreSQL nyata)
ok  task-api/internal/middleware
ok  task-api/internal/model
ok  task-api/internal/worker
```

## API

Spesifikasi lengkap: [`openapi.yaml`](./openapi.yaml) dan ringkasan kontrak di [`API_CONTRACT.md`](./API_CONTRACT.md).

| Method | Path | Akses |
|---|---|---|
| GET | `/health` | publik |
| POST | `/register`, `/login` | publik (rate limited) |
| GET/PATCH | `/me` | bearer |
| PATCH | `/me/password` | bearer |
| GET | `/dashboard` | bearer |
| GET/POST | `/tasks` | bearer |
| GET/PUT/DELETE | `/tasks/{id}` | bearer |
| GET/POST | `/projects` | bearer |
| GET/PUT/DELETE | `/projects/{id}` | bearer (update: owner/admin; delete: owner) |
| GET/POST | `/projects/{id}/members` | bearer (POST: owner/admin) |
| PATCH/DELETE | `/projects/{id}/members/{userID}` | bearer (owner) |
| GET/POST | `/projects/{id}/tasks` | bearer (member) |

Contoh:

```bash
TOKEN=$(curl -s -X POST localhost:8081/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"me@example.com","password":"secret123"}' | jq -r .token)

curl -s "localhost:8081/tasks?status=todo&sort=priority&order=desc&limit=10" \
  -H "Authorization: Bearer $TOKEN"
```

Response list selalu berbentuk:

```json
{
  "data": [],
  "pagination": { "page": 1, "limit": 20, "total": 0, "total_pages": 0 }
}
```

## Keputusan desain & trade-off

| Keputusan | Alasan | Konsekuensi |
|---|---|---|
| `net/http` tanpa framework | Routing modern (`METHOD /path/{id}`) sudah cukup; dependency lebih sedikit | Middleware ditulis sendiri |
| Otorisasi di service, bukan handler | Satu tempat untuk semua aturan akses, mudah diuji lewat HTTP | Service bergantung pada beberapa repository |
| 404 (bukan 403) untuk resource yang tidak boleh dilihat | Tidak membocorkan keberadaan resource | Debugging butuh log |
| Query dinamis dengan placeholder berurutan | Aman dari SQL injection tanpa ORM | Sorting dibatasi whitelist kolom |
| Envelope paginasi | Frontend butuh total untuk navigasi | Response list bukan array polos |
| Worker in-process, tanpa Redis | Satu instance, beban kecil, belum ada kebutuhan koordinasi | Tidak cocok untuk multi-instance |
| Migrasi SQL manual | Transparan untuk pembelajaran; tidak ada magic | Tidak ada auto-rollback |

## Security checklist

Ditinjau manual saat Phase 9:

- [x] Seluruh query memakai placeholder (`$1`, `$2`); tidak ada string SQL yang dirakit dari input.
- [x] Nama kolom sorting dipetakan dari whitelist, bukan dari query string.
- [x] `JWT_SECRET` wajib ≥32 karakter, tidak ada fallback hardcoded, server gagal start jika kosong.
- [x] Token kedaluwarsa 24 jam; algoritma signing divalidasi (HMAC-only).
- [x] Password di-hash bcrypt dan tidak pernah dikembalikan (`json:"-"`); panjang dibatasi 72 byte sesuai batas bcrypt.
- [x] Login memakai perbandingan bcrypt dummy untuk email tidak dikenal agar timing seragam.
- [x] Rate limit pada `/register` dan `/login` (token bucket per IP).
- [x] Body request dibatasi 1 MiB.
- [x] Otorisasi diperiksa di service, bukan hanya di handler; resource tanpa akses mengembalikan 404.
- [x] CORS hanya mengizinkan origin yang dikonfigurasi eksplisit (bukan `*`).
- [x] Log tidak memuat password, token, atau body request.
- [ ] HTTPS termination belum ditangani aplikasi — diserahkan ke reverse proxy saat deployment.

## Roadmap

Phase 0–5 selesai di backend, Phase 7 (worker) dan Phase 8 (Docker/CI) disiapkan, Phase 9 (dokumentasi) mencakup README ini. **Phase 6 (frontend React) dikerjakan terpisah oleh pemilik proyek.**

## Lisensi

Proyek pembelajaran pribadi.
