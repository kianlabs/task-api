# API Contract v1 — task-api

Kontrak ini adalah acuan tunggal untuk backend dan frontend. Base URL: `http://localhost:8081`.

## Konvensi

- Semua request/response `application/json`, kecuali `DELETE` yang mengembalikan `204 No Content`.
- Autentikasi: header `Authorization: Bearer <jwt>`.
- Error selalu `{"error":"pesan"}`.
- Tanggal task (`deadline`) memakai format `YYYY-MM-DD`. Timestamp memakai RFC3339.
- List response memakai envelope paginasi:
  ```json
  {
    "data": [],
    "pagination": { "page": 1, "limit": 20, "total": 0, "total_pages": 0 }
  }
  ```
- Query param list: `page` (default 1, min 1), `limit` (default 20, max 100).

## Auth

### POST /register
Body: `{"email":"a@b.com","password":"min6char"}`
- 201: `{"id":1,"email":"a@b.com","created_at":"2026-11-02T10:00:00Z"}`
- 400 validasi, 409 email sudah terdaftar

### POST /login
Body: `{"email":"a@b.com","password":"min6char"}`
- 200: `{"token":"<jwt>","user":{"id":1,"email":"a@b.com","created_at":"..."}}`
- 401 kredensial salah

## Profil

### GET /me
- 200: `{"id":1,"email":"a@b.com","created_at":"..."}`

### PATCH /me
Body: `{"email":"baru@b.com"}` (semua field opsional)
- 200: user object, 409 email dipakai

### PATCH /me/password
Body: `{"current_password":"...","new_password":"min6char"}`
- 204, 400 salah, 401 current password salah

## Task

Model Task:
```json
{
  "id": 1,
  "user_id": 1,
  "project_id": null,
  "assignee_id": null,
  "title": "Belajar Go",
  "description": "",
  "status": "todo",
  "priority": "medium",
  "deadline": "2026-11-30",
  "created_at": "2026-11-02T10:00:00Z",
  "updated_at": "2026-11-02T10:00:00Z"
}
```

- `status` ∈ `todo` | `in_progress` | `done`
- `priority` ∈ `low` | `medium` | `high` | `urgent`

### POST /tasks
Body: `{"title":"...","description":"...","status":"todo","priority":"medium","deadline":"2026-11-30","project_id":null,"assignee_id":null}`
- 201 task, 400 validasi, 403 bukan member project

### GET /tasks
Query: `page`, `limit`, `status`, `priority`, `project_id`, `q` (search title/description), `sort` (`id`|`deadline`|`priority`|`created_at`), `order` (`asc`|`desc`)
- 200 envelope. Hanya task milik user (owner) atau yang di-assign ke user.

### GET /tasks/{id}
- 200 task, 404 jika bukan hak akses

### PUT /tasks/{id}
Body sama dengan create (semua field, validasi penuh)
- 200 task, 400, 404

### DELETE /tasks/{id}
- 204, 404

## Project

Model Project:
```json
{ "id": 1, "name": "Proyek", "description": "", "owner_id": 1, "created_at": "...", "updated_at": "..." }
```

### POST /projects
Body: `{"name":"...","description":"..."}`
- 201 project; pembuat otomatis jadi `owner` di `project_members`

### GET /projects
Query: `page`, `limit`
- 200 envelope project yang user ikuti

### GET /projects/{id}
- 200, 404 jika bukan member

### PUT /projects/{id}
Body: `{"name":"...","description":"..."}` — hanya `owner`/`admin`
- 200, 403, 404

### DELETE /projects/{id}
- 204 — hanya `owner`; cascade ke task & member

## Members

Model Member:
```json
{ "project_id": 1, "user_id": 2, "email": "b@c.com", "role": "member", "created_at": "..." }
```
Role: `owner` | `admin` | `member`.

### GET /projects/{id}/members
- 200 `{"data":[...]}` — hanya member project

### POST /projects/{id}/members
Body: `{"user_id":2,"role":"member"}` — hanya `owner`/`admin`
- 201 member, 403, 404 user tidak ada, 409 sudah member

### PATCH /projects/{id}/members/{userID}
Body: `{"role":"admin"}` — hanya `owner`; role `owner` tidak boleh diubah
- 200, 403

### DELETE /projects/{id}/members/{userID}
Request body: `{"user_id": 3}` — hanya `owner`/`admin`; `owner` tidak bisa dihapus
- 204, 403

## Project Tasks

### GET /projects/{id}/tasks
Query sama dengan `GET /tasks`
- 200 envelope task milik project tersebut (semua member boleh lihat)

### POST /projects/{id}/tasks
Body: task tanpa `project_id` (diisi dari path)
- 201, 403

## Dashboard

### GET /dashboard
- 200:
```json
{
  "total_tasks": 12,
  "by_status": { "todo": 5, "in_progress": 4, "done": 3 },
  "by_priority": { "low": 2, "medium": 6, "high": 3, "urgent": 1 },
  "overdue": 2,
  "due_today": 1,
  "projects": 3
}
```

## Health

### GET /health
- 200: `{"status":"ok"}`
