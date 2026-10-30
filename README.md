# Task Manager API

REST API sederhana untuk manajemen tugas (Task Management) yang dibangun menggunakan **Go standard library** (`net/http`) tanpa framework eksternal. Proyek ini merupakan bagian dari roadmap pembelajaran Go Backend Engineering (Phase 2).

## Fitur Utama

- **Zero External Dependencies**: Menggunakan murni package bawaan Go (`net/http`, `encoding/json`, `sync`).
- **Clean Architecture**: Pemisahan tanggung jawab kode ke dalam package `internal/model` dan `internal/handler`.
- **Thread-Safe Concurrency**: Proteksi _race condition_ pada penyimpanan in-memory menggunakan `sync.Mutex` dan `defer`.
- **CRUD Lengkap**:
  - `GET /health` : Health check status service
  - `POST /tasks` : Membuat task baru dengan validasi input
  - `GET /tasks` : Mengambil seluruh daftar task
  - `GET /tasks/{id}` : Mengambil detail task berdasarkan ID
  - `PUT /tasks/{id}` : Memperbarui task berdasarkan ID
  - `DELETE /tasks/{id}` : Menghapus task berdasarkan ID

## Prasyarat

- Go 1.22 atau versi lebih baru.
- PostgreSQL (atau Docker).

## Konfigurasi Database

Aplikasi menggunakan environment variable `DATABASE_URL` untuk koneksi ke database.
Format:

```bash
export DATABASE_URL="postgres://taskapi:devpassword@localhost:5432/taskdb?sslmode=disable"
```

Jika tidak disetel, aplikasi akan memakai nilai default di atas.

## Migrasi Database

Jalankan script SQL berikut pada database PostgreSQL sebelum menjalankan server:

```bash
psql "$DATABASE_URL" -f migrations/000001_create_tasks_table.sql
```

## Cara Menjalankan

1. Clone repositori:
   ```bash
   git clone https://github.com/kianlabs/task-api.git
   cd task-api
   ```
2. Jalankan server:
   ```bash
   go run .
   ```
   Server akan aktif di port `:8081`.

## Contoh Penggunaan (cURL)

1. Membuat Task (POST)
   ```bash
   curl -X POST http://localhost:8081/tasks \
       -H "Content-Type: application/json" \
       -d '{"title":"Belajar Unit Test","status":"pending","deadline":"2026-10-30"}'
   ```
   Response (201 Created):
   ```json
   {
     "id": 1,
     "title": "Belajar Unit Test",
     "status": "pending",
     "deadline": "2026-10-30"
   }
   ```
2. Mengambil Semua Task (GET)
   ```bash
   curl http://localhost:8081/tasks
   ```
