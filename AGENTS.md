# task-api — Task Manager API

REST API manajemen task. Dibangun sambil belajar Go Backend Engineering; target proyek di roadmap adalah **task-manager**, kode saat ini ada di module `task-api`.

## Status
Fase saat ini: **Phase 0 — Repository & Baseline Audit** (belum selesai). Jangan menambah fitur sebelum Phase 0 lulus. Status di file ini bisa tertinggal; Notion Roadmap yang berlaku.

## Stack
- Go 1.27.3, module `task-api`, `net/http` + `ServeMux` (pattern `METHOD /path/{id}`), tanpa framework.
- PostgreSQL via `database/sql` + driver `pgx/v5/stdlib`.
- Auth: JWT HS256 (`golang-jwt/jwt/v5`), hash password bcrypt.
- Port server `:8081` (8080 dipakai aplikasi lain di mesin ini).
- Env: `DATABASE_URL` (wajib), `JWT_SECRET` (wajib di produksi, jangan di-commit).

## Struktur
```
main.go                       wiring server + route
internal/database/db.go       koneksi PostgreSQL
internal/auth/token.go        generate/validate JWT
internal/auth/middleware.go   cek Bearer token, simpan user_id di context
internal/handler/task.go      CRUD task + health
internal/handler/auth.go      register + login
internal/handler/task_test.go test handler (integrasi)
internal/model/               Task, User, validasi, JSON (de)serialization
migrations/                   SQL manual: 000001..000003
```

## Endpoint
| Method | Path | Auth |
|---|---|---|
| GET | `/health` | tidak |
| POST | `/register` | tidak |
| POST | `/login` | tidak |
| POST | `/tasks` | Bearer token |
| GET | `/tasks` | Bearer token |
| GET | `/tasks/{id}` | Bearer token |
| PUT | `/tasks/{id}` | Bearer token |
| DELETE | `/tasks/{id}` | Bearer token |

## Build dan jalankan
```bash
cd ~/projects-go/task-api

export DATABASE_URL="postgres://<user>:<password-lokal>@localhost:5432/taskdb?sslmode=disable"
export JWT_SECRET="<secret-lokal-jangan-di-commit>"

go build ./...   # compile
go run .         # server :8081
```
Go tidak hot-reload: hentikan (Ctrl+C) lalu jalankan ulang setelah mengubah kode.

## Test dan vet
```bash
go test ./...    # test integrasi: butuh PostgreSQL hidup + DATABASE_URL
go vet ./...     # wajib bersih
gofmt -l .       # harus kosong
```
Test saat ini membersihkan tabel `tasks` dan `users` pada database yang dipakai. Jangan arahkan ke database berisi data penting.

## Migrasi
Belum ada migration runner; jalankan manual dan berurutan:
```bash
psql "$DATABASE_URL" -f migrations/000001_create_tasks_table.sql
psql "$DATABASE_URL" -f migrations/000002_create_users_table.sql
psql "$DATABASE_URL" -f migrations/000003_add_user_id_to_tasks.sql
```
`000003` memuat `TRUNCATE TABLE tasks;` → destruktif. Jangan jalankan pada data yang mau disimpan.

## Roadmap
Phase 0 → Phase 9, urut, tanpa lompat. Definisi fase dan task ada di Notion **Task Manager Project Roadmap** (URL di `../.kilo/rules/00-source-of-truth.md`). Gate fase: `../.kilo/rules/20-roadmap-gate.md`.

## Mentor workflow
Agent default: `go-backend-learning-mentor`. Siklus **Learn → Practice → Debug → Review → Evaluate**: satu tugas kecil per turn, hint sebelum solusi, verifikasi wajib dengan output asli.

## Aturan aman
- Jangan commit, push, atau mengubah migrasi tanpa persetujuan pengguna.
- Jangan menyimpan secret di repo; `.env` sudah di `.gitignore`. `JWT_SECRET` dari environment variable.
- Jangan menghapus file atau mengubah konfigurasi Kilo Code / VS Code tanpa persetujuan.
- Jangan menjalankan `TRUNCATE`, `DROP`, atau `DELETE` massal tanpa persetujuan.
- Jangan mengklaim test lulus tanpa output.

## Sumber kebenaran
Notion → `AGENTS.md` ini → `.kilo/rules/*` (workspace `projects-go`) → percakapan.
Detail: `../.kilo/rules/00-source-of-truth.md`.
