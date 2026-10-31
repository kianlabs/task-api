# Task Manager API

REST API manajemen tugas (Task Management) yang dibangun menggunakan **Go standard library** (`net/http`) tanpa framework web, dengan PostgreSQL dan JWT authentication.

## Fitur & Stack

- **Go Standard Library**: Routing via `net/http` (`ServeMux`) dan `encoding/json`.
- **Database**: PostgreSQL dengan driver `jackc/pgx/v5`.
- **Authentication**: JWT HS256 (`golang-jwt/jwt/v5`) dan hash password dengan `bcrypt`.
- **Multi-Tenant Ownership**: Setiap user hanya dapat mengakses dan mengelola task miliknya sendiri.

## Endpoint

| Method | Path          | Auth         | Keterangan                       |
| ------ | ------------- | ------------ | -------------------------------- |
| GET    | `/health`     | Publik       | Cek status server                |
| POST   | `/register`   | Publik       | Registrasi user baru             |
| POST   | `/login`      | Publik       | Login & mendapatkan JWT token    |
| POST   | `/tasks`      | Bearer Token | Membuat task baru                |
| GET    | `/tasks`      | Bearer Token | Mengambil daftar task milik user |
| GET    | `/tasks/{id}` | Bearer Token | Mengambil detail task milik user |
| PUT    | `/tasks/{id}` | Bearer Token | Memperbarui task milik user      |
| DELETE | `/tasks/{id}` | Bearer Token | Menghapus task milik user        |

## Konfigurasi Environment

Aplikasi membutuhkan environment variable berikut:

```bash
export DATABASE_URL="postgres://taskapi:devpassword@localhost:5432/taskdb?sslmode=disable"
export JWT_SECRET="ganti-dengan-secret-aman"
```

### Untuk menjalankan integration test:

```bash
export TEST_DATABASE_URL="postgres://taskapi:devpassword@127.0.0.1:5432/taskdb_test?sslmode=disable"
```

### Migrasi Database

### Jalankan script SQL berikut secara berurutan pada database PostgreSQL:

```bash
psql "$DATABASE_URL" -f migrations/000001_create_tasks_table.sql
psql "$DATABASE_URL" -f migrations/000002_create_users_table.sql
psql "$DATABASE_URL" -f migrations/000003_add_user_id_to_tasks.sql
```

### Cara Menjalankan

### Jalankan server:

```bash
go run .
```

Server akan aktif di port :8081.

### Menjalankan test:

# Unit test saja (aman tanpa database)

```bash
go test -v ./...
```

# Integration test (menggunakan database test terpisah)

```bash
TEST_DATABASE_URL="postgres://taskapi:devpassword@127.0.0.1:5432/taskdb_test?sslmode=disable" go test -v ./...
```
