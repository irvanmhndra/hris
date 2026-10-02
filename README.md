# People HRIS

Aplikasi HRIS dengan API Go, web admin, dan portal karyawan. Arsitektur mengikuti kode lokal NexPOS/FNB: Echo v5 → handler → service → repository → PostgreSQL, serta monorepo React dengan pnpm/Turborepo.

## Fitur

- Login per perusahaan, session token tersimpan sebagai hash, logout, dan pembatasan akses admin/karyawan.
- Dashboard: karyawan aktif, kehadiran hari ini, pengajuan menunggu, dan komposisi departemen.
- Departemen: daftar dan tambah.
- Karyawan: daftar berhalaman dengan pencarian dan filter di server, tambah, edit, nonaktifkan, reset password melalui admin, dan ekspor CSV seluruh hasil filter.
- Akun portal dibuat bersama karyawan dalam transaksi database.
- Absensi: check-in/check-out satu kali per hari, riwayat 100 catatan terbaru, zona waktu Asia/Jakarta.
- Cuti tahunan/sakit/izin: pengajuan berdasarkan hari kerja, saldo per tahun, reservasi pengajuan pending, persetujuan/penolakan, dan pembatalan.
- Kalender kerja dan hari libur perusahaan; kuota cuti dapat diatur per karyawan/tahun.
- Pengajuan lembur dan koreksi absensi dengan persetujuan HR.
- Profil kontak dan kontak darurat yang dapat diperbarui karyawan.
- Pengumuman dan dokumen kebijakan berbasis tautan HTTPS, dengan draft/publikasi/arsip.
- Checklist onboarding, inventaris dan penugasan aset, target kinerja dan progres, serta pipeline kandidat rekrutmen.
- Payroll bulanan: komponen gaji, snapshot draft, penyesuaian, finalisasi, ekspor CSV, slip pribadi yang bisa dicetak, dan pencatatan pembayaran manual.
- Riwayat aktivitas modul HR/payroll, seluruh riwayat dengan pagination.
- Tema indigo, sidebar responsif, dan portal khusus karyawan.
- Semua data dibatasi `company_id`; karyawan hanya dapat melihat absensi dan cutinya sendiri.

## Jalankan lokal

Prasyarat: Go **1.26.6**, Node.js **24**, pnpm **10.6.2**, Docker Compose. Port: PostgreSQL 5440, API 8088, admin 5180, karyawan 5181.

```sh
# Di root repository
make db
# Tunggu database healthy: docker compose ps

# Terapkan migrasi yang belum jalan (aman diulang)
make migrate

# Sekali saja untuk database kosong
SEED_PASSWORD='PeopleDemo2026!' make seed

# Terminal pertama
make api

# Terminal kedua
cd hris-web
pnpm install --frozen-lockfile
pnpm dev
```

Schema dikelola migration runner (`hris-api/cmd/migrate`), bukan lagi oleh `docker-entrypoint-initdb.d`. Setelah menarik perubahan yang berisi migrasi baru, cukup jalankan `make migrate` lagi; jangan hapus volume untuk memperbarui schema. Database yang dibuat dengan setup lama perlu satu langkah awal, lihat [Migrasi database](#migrasi-database).

| Portal | URL | Email demo |
|---|---|---|
| Admin | http://127.0.0.1:5180 | admin@demo.hris |
| Karyawan | http://127.0.0.1:5181 | employee@demo.hris |

Kode perusahaan: `demo`. Password sesuai `SEED_PASSWORD` saat menjalankan seed. Seed menolak menimpa perusahaan demo yang sudah ada. Data demo bukan data karyawan asli.

`make api` menyediakan `DATABASE_URL` lokal melalui Makefile. Untuk menjalankan Go langsung, ekspor variabel sesuai `hris-api/.env.example` (`DATABASE_URL`, `HTTP_ADDR`, opsional `CORS_ALLOWED_ORIGINS` dan `DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS` / `DB_CONN_MAX_LIFETIME`); aplikasi tidak membaca file `.env` otomatis. Bila lingkungan GVM menetapkan `GOROOT` untuk versi lama, hapus variabel tersebut sebelum menjalankan Go 1.26.6.

Alternatif API dalam Docker:

```sh
docker compose up -d --build   # db → migrate (sekali jalan) → api
# Seed hanya jika belum pernah dijalankan:
docker compose run --rm -e SEED_PASSWORD='PeopleDemo2026!' api ./seed
```

Frontend tetap dijalankan dengan `pnpm dev`. Compose hanya mengikat port database/API pada loopback.

## Struktur

```text
hris-api/
  cmd/api, cmd/migrate, cmd/seed   server, migration runner, data demo
  config                 konfigurasi environment
  internal/app           server, middleware global, wiring per domain
  internal/router        registrasi endpoint
  internal/middleware    session authentication dan role
  internal/handler       satu handler per domain: binding dan respons HTTP
  internal/service       satu service per domain: validasi, otorisasi, scope
  internal/repository    kontrak repository per domain
  internal/repository/postgres    implementasi SQL per domain
  internal/model, dto    domain, input repository, dan request
  pkg/apperror, httputil  error berkode dan response envelope
  migrations             migrasi SQL up/down + runner ter-embed
  tests/integration      tes HTTP + PostgreSQL pada schema sementara
hris-web/
  apps/admin             entry point admin (@hris/shared/admin)
  apps/employee          entry point portal karyawan (@hris/shared/employee)
  packages/shared/src/
    portals              navigasi + rute lazy per portal
    pages, layouts, components
    services, stores     API client dan Zustand session
```

## Verifikasi

```sh
make lint     # go vet + golangci-lint
make test     # PostgreSQL harus aktif; Go unit + integration tests + TypeScript
make build    # API binary + production build kedua portal
```

Tes integrasi membuat schema unik, menjalankan migrasi lewat runner, menguji workflow, lalu menghapus schema miliknya sendiri. Data aplikasi tidak dihapus. `go test ./...` tanpa `TEST_DATABASE_URL` melewatkan tes integrasi secara eksplisit. Unit test service (aturan otorisasi dan scope) tidak butuh database. GitHub Actions (`.github/workflows/ci.yml`) menjalankan lint, unit + integration test dengan PostgreSQL, uji migrasi, govulncheck, serta typecheck dan build web pada setiap PR ke `main`.

## Payroll Indonesia

1. **Pengaturan BPJS** (halaman Komponen gaji): kelas risiko JKK dan batas upah Jaminan Pensiun/BPJS Kesehatan. Perbarui batas upah ketika BPJS mengumumkan nilai baru.
2. **Komponen gaji** per karyawan: gaji pokok, tunjangan (tetap/tidak tetap, kena pajak/tidak), potongan rutin, status PTKP, metode PPh 21 (gross, gross-up, atau manual), kepesertaan BPJS, dan hak lembur.
3. **Buat payroll** bulanan, opsional dengan tanggal hari raya untuk THR. Sistem menghitung:
   - prorata gaji pokok dan tunjangan per hari kerja kalender perusahaan untuk karyawan yang masuk atau keluar (tanggal keluar diisi saat karyawan dinonaktifkan);
   - lembur yang disetujui dan belum dibayar: upah per jam 1/173 × (pokok + tunjangan tetap), hari kerja 1,5× jam pertama lalu 2×, hari libur 2×/3×/4× (PP 35/2021, 5 atau 6 hari kerja);
   - THR: 1 bulan upah tetap untuk masa kerja ≥12 bulan, proporsional untuk 1–12 bulan;
   - BPJS Kesehatan 4%+1% (batas upah), JHT 3,7%+2%, JP 2%+1% (batas upah), JKK sesuai kelas, JKM 0,3%;
   - PPh 21 TER bulanan (PP 58/2023) atas bruto termasuk premi JKK/JKM/BPJS Kesehatan perusahaan; Desember atau bulan keluar memakai tarif Pasal 17 setahun dikurangi PPh yang sudah dipotong (bisa menjadi pengembalian). Gross-up menambahkan tunjangan PPh sebesar pajaknya.
4. **Sesuaikan** slip draft (bonus, kasbon, koreksi) — BPJS dan PPh 21 dihitung ulang otomatis.
5. **Finalisasi**, lalu bayar di luar aplikasi dan **Catat sudah dibayar** dengan referensinya.

Profil gaji dan slip yang dibuat sebelum migrasi 000005 tetap memakai metode pajak manual tanpa BPJS, sehingga nominal lamanya tidak berubah. Ubah ke gross/gross-up dan aktifkan BPJS di Komponen gaji untuk memakai perhitungan otomatis. Tabel TER dan tarif dikodekan dari regulasi yang berlaku saat ditulis; verifikasi dengan konsultan pajak sebelum dipakai untuk payroll riil.

Data demo tambahan tersedia melalui `make seed-hr` setelah seed dasar. Nominalnya fiktif. Seed tambahan tidak menimpa komponen gaji yang sudah ada.

## Migrasi database

| Perintah | Fungsi |
|---|---|
| `make migrate` | Terapkan semua migrasi yang belum jalan |
| `make migrate-version` | Tampilkan versi schema saat ini |
| `make migrate-baseline` | Sekali saja: tandai 000001–000004 sudah diterapkan (`migrate force 4`) |
| `cd hris-api && go run ./cmd/migrate down N` | Rollback N langkah; migrasi forward-only ditolak |

**Database dari setup lama.** Database yang dibuat lewat mount `docker-entrypoint-initdb.d` belum punya `schema_migrations`, sehingga `make migrate` akan menolak dengan pesan agar menjalankan `migrate force 4`. Bila keempat migrasi sudah diterapkan (setup Docker lama menerapkan semuanya), jalankan sekali:

```sh
make migrate-baseline
make migrate          # sekarang: tidak ada perubahan
```

Bila database baru sampai 000002, jalankan `cd hris-api && go run ./cmd/migrate force 2`, lalu `make migrate` untuk menerapkan 000003 dan 000004. Untuk Compose, pakai `docker compose run --rm migrate ./migrate force 4` sebelum `docker compose up -d`.

Migrasi 000003/000004/000005 bersifat forward-only. Pengajuan cuti lama mempertahankan hitungan hari kalender; pengajuan baru menyimpan snapshot hari kerja agar perubahan kalender tidak mengubah pengajuan yang sudah ada.

## Batas implementasi saat ini

Payroll menghitung PPh 21, BPJS, prorata, lembur, dan THR secara otomatis dengan pembayaran manual. Belum ada transfer bank, laporan e-Bupot/SPT, koreksi payroll setelah finalisasi, cuti tidak dibayar otomatis, atau saldo PPh dari sistem lain (karyawan yang pindah di tengah tahun dihitung dari periode yang tercatat di HRIS saja). Karyawan keluar tanpa tanggal keluar (data lama) tidak ikut payroll berikutnya.

Approval masih satu tingkat. Kalender kerja belum menjadi penjadwalan shift atau aturan keterlambatan; absensi/koreksi belum mendukung shift lintas tengah malam dan GPS/biometrik. Kuota cuti belum memiliki accrual/carry-over otomatis. Dokumen berupa tautan, tanpa upload/tanda tangan; izin file tetap diatur pada penyedia file. Target kinerja berupa progres, tanpa appraisal/360 review. Rekrutmen mencatat kandidat dan tahapan, tanpa portal lowongan atau konversi otomatis menjadi karyawan.

Session berlaku 12 jam, tanpa refresh token atau lupa-password mandiri. Manajemen perusahaan dilakukan lewat provisioning database; UI registrasi perusahaan belum tersedia. Pagination server tersedia untuk karyawan dan audit log; daftar lain (absensi 100 terbaru, cuti, modul HR) masih dikembalikan utuh.

Konfigurasi Compose dan akun demo ditujukan untuk pengembangan lokal. Untuk deployment, gunakan kredensial terpisah dan reverse proxy HTTPS: `/api` diteruskan ke API dan rute SPA ke `index.html`. `vite preview` tidak menyediakan proxy API pengembangan.

Lihat [pemetaan arsitektur](docs/architecture.md) dan [kontrak API](docs/api.md).
