# Project Walkthrough — Task Manager API

Dokumen ini untuk menjelaskan proyek secara lisan (interview / demo). Isinya alasan desain, bukan tutorial.

## 1. Problem statement (30 detik)

"Task Manager API mengelola task dan project untuk banyak user sekaligus. Tantangan utamanya bukan CRUD, tapi **batas akses**: satu task bisa dimiliki user, di-assign ke user lain, dan berada dalam project dengan peran berbeda. Semua endpoint harus konsisten menegakkan aturan itu."

## 2. Arsitektur (1 menit)

```
HTTP → middleware (recover, log, CORS) → auth → handler → service → repository → PostgreSQL
```

- **handler**: hanya decode request, panggil service, tulis response. Tidak ada SQL dan tidak ada aturan bisnis.
- **service**: satu-satunya tempat aturan akses dan validasi domain. Mengembalikan error terklasifikasi (`not_found`, `forbidden`, `conflict`, `validation`).
- **repository**: satu-satunya tempat SQL. Semua query berparameter; nama kolom sorting dipetakan dari whitelist.
- **model**: entity + validasi + filter/paginasi; tidak tahu HTTP maupun database.

Alasan: kalau aturan akses tersebar di handler, cepat atau lambat satu endpoint akan lupa memeriksanya.

## 3. Alur satu request (contoh: GET /tasks/{id})

1. `Recover` membungkus panic jadi 500 JSON; `RequestLogger` mencatat durasi dan status.
2. `auth.Middleware` memvalidasi Bearer token, menaruh `user_id` di context.
3. `TaskHandler.GetTaskByID` mem-parse path, memanggil service.
4. `TaskService.GetTask` mengambil task, lalu memeriksa: pemilik? assignee? anggota project?
5. Jika tidak berhak → **404**, bukan 403, agar keberadaan resource tidak bocor.
6. `response.JSON` menulis header lalu body.

## 4. Keputusan yang sering ditanya

**Kenapa 404 untuk resource yang tidak boleh diakses?**
Karena 403 mengonfirmasi resource itu ada. Untuk data multi-tenant, kebocoran keberadaan saja sudah informasi.

**Kenapa otorisasi di service, bukan middleware?**
Aturan bergantung pada data (siapa pemilik task, role di project), jadi butuh akses database. Middleware hanya tahu identitas.

**Kenapa tidak pakai ORM?**
Query-nya sedikit dan spesifik: join membership, agregasi `FILTER`, sorting kondisional. SQL eksplisit lebih mudah dioptimalkan dan diaudit. Konsekuensinya, semua query harus ditulis tangan dan diuji.

**Kenapa paginasi memakai envelope, bukan array polos?**
Frontend butuh `total` untuk navigasi. Envelope juga memberi ruang menambah metadata tanpa breaking change.

**Kenapa worker tidak memakai Redis?**
Belum ada kebutuhan terukur: satu instance, scan ringan, tidak ada koordinasi antar-proses. Redis baru masuk kalau ada banyak instance atau butuh job queue yang tahan restart.

**Bagaimana mencegah SQL injection pada sorting?**
`ORDER BY` tidak bisa memakai placeholder, jadi input dipetakan ke ekspresi tetap lewat `switch`; nilai lain jatuh ke default `t.id`.

**Bagaimana transaksi dipakai?**
Pembuatan project + keanggotaan owner harus atomik. Kalau insert member gagal, project tidak boleh tertinggal setengah jadi (`BeginTx` + `defer Rollback` + `Commit`).

## 5. RBAC dalam satu tabel

| Aksi | owner | admin | member |
|---|---|---|---|
| Lihat project & task | ✅ | ✅ | ✅ |
| Update project | ✅ | ✅ | ❌ |
| Tambah member (member) | ✅ | ✅ | ❌ |
| Tambah admin | ✅ | ❌ | ❌ |
| Ubah role | ✅ | ❌ | ❌ |
| Keluarkan member | ✅ | ✅ (kecuali admin lain) | ❌ |
| Hapus project | ✅ | ❌ | ❌ |

Pangkat peran dihitung dengan `roleRank`, sehingga pengecekan "minimal admin" cukup satu perbandingan.

## 6. Bukti kualitas

```bash
go vet ./...                     # bersih
gofmt -l .                       # kosong
go test ./...                    # unit test
TEST_DATABASE_URL=... go test -race ./...   # integration + race detector
```

Integration test memakai database terpisah (`*_test`) di loopback dan menolak database development, sehingga tidak mungkin menghapus data nyata.

## 7. Yang akan dikerjakan berikutnya

1. **Frontend** (Phase 6) — dikerjakan pemilik proyek; API sudah punya CORS configurable dan `openapi.yaml` sebagai kontrak.
2. **Deployment** (Phase 8) — Docker Compose + CI sudah ada; tinggal VPS + reverse proxy HTTPS.
3. **Observability lanjutan** — metrik (Prometheus) dan tracing kalau traffic nyata sudah ada.

## 8. Pertanyaan yang mungkin muncul

- "Bagaimana menangani N+1 query?" → List task memakai satu query dengan join membership; statistik dashboard memakai satu query agregasi.
- "Bagaimana mencegah user meng-assign task ke orang luar?" → `validateAssignment` memeriksa eksistensi user dan keanggotaan project.
- "Apa yang terjadi saat project dihapus?" → `ON DELETE CASCADE` menghapus task dan membership; task yang di-assign ke user lain ikut terhapus karena bagian dari project.
- "Bagaimana worker berhenti?" → context dibatalkan saat shutdown, `Run` keluar dari select, lalu server melakukan graceful shutdown.
