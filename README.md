# People HRIS

Aplikasi HRIS dengan API Go, web admin, dan portal karyawan. Arsitektur mengikuti kode lokal NexPOS/FNB: Echo v5 → handler → service → repository → PostgreSQL, serta monorepo React dengan pnpm/Turborepo.

## Fitur

- Login per perusahaan, session token tersimpan sebagai hash, logout, dan pembatasan akses admin/karyawan.
- Dashboard: karyawan aktif, kehadiran hari ini, pengajuan menunggu, dan komposisi departemen.
- Departemen: daftar dan tambah.
- Karyawan: daftar, cari, filter, tambah, edit, nonaktifkan, reset password melalui admin, dan ekspor CSV.
- Akun portal dibuat bersama karyawan dalam transaksi database.
- Absensi: check-in/check-out satu kali per hari, riwayat 100 catatan terbaru, zona waktu Asia/Jakarta.
- Cuti tahunan/sakit/izin: pengajuan berdasarkan hari kerja, saldo per tahun, reservasi pengajuan pending, persetujuan/penolakan, dan pembatalan.
- Kalender kerja dan hari libur perusahaan; kuota cuti dapat diatur per karyawan/tahun.
- Pengajuan lembur dan koreksi absensi dengan persetujuan HR.
- Profil kontak dan kontak darurat yang dapat diperbarui karyawan.
- Pengumuman dan dokumen kebijakan berbasis tautan HTTPS, dengan draft/publikasi/arsip.
- Checklist onboarding, inventaris dan penugasan aset, target kinerja dan progres, serta pipeline kandidat rekrutmen.
- Payroll bulanan: komponen gaji, snapshot draft, penyesuaian, finalisasi, ekspor CSV, slip pribadi yang bisa dicetak, dan pencatatan pembayaran manual.
- Riwayat aktivitas modul HR/payroll (250 aktivitas terakhir).
- Tema indigo, sidebar responsif, dan portal khusus karyawan.
- Semua data dibatasi `company_id`; karyawan hanya dapat melihat absensi dan cutinya sendiri.

## Jalankan lokal

Prasyarat: Go **1.26.6**, Node.js **24**, pnpm **10.6.2**, Docker Compose. Port: PostgreSQL 5440, API 8088, admin 5180, karyawan 5181.

```sh
# Di root repository
make db
# Tunggu database healthy: docker compose ps

# Sekali saja untuk database kosong
SEED_PASSWORD='PeopleDemo2026!' make seed

# Terminal pertama
make api

# Terminal kedua
cd hris-web
pnpm install --frozen-lockfile
pnpm dev
```

Database dan tabel dibuat otomatis ketika volume Docker baru diinisialisasi. Jika volume sudah ada, jalankan migrasi baru secara berurutan; jangan hapus volume untuk memperbarui schema.

| Portal | URL | Email demo |
|---|---|---|
| Admin | http://127.0.0.1:5180 | admin@demo.hris |
| Karyawan | http://127.0.0.1:5181 | employee@demo.hris |

Kode perusahaan: `demo`. Password sesuai `SEED_PASSWORD` saat menjalankan seed. Seed menolak menimpa perusahaan demo yang sudah ada. Data demo bukan data karyawan asli.

`make api` menyediakan `DATABASE_URL` lokal melalui Makefile. Untuk menjalankan Go langsung, ekspor `DATABASE_URL` sesuai `hris-api/.env.example`; aplikasi tidak membaca file `.env` otomatis. Bila lingkungan GVM menetapkan `GOROOT` untuk versi lama, hapus variabel tersebut sebelum menjalankan Go 1.26.6.

Alternatif API dalam Docker:

```sh
docker compose up -d --build
# Seed hanya jika belum pernah dijalankan:
docker compose run --rm -e SEED_PASSWORD='PeopleDemo2026!' api ./seed
```

Frontend tetap dijalankan dengan `pnpm dev`. Compose hanya mengikat port database/API pada loopback.

## Struktur

```text
hris-api/
  cmd/api, cmd/seed       entry point dan data demo
  config                 konfigurasi environment
  internal/app           dependency injection
  internal/router        registrasi endpoint
  internal/middleware    session authentication dan role
  internal/handler       binding DTO dan respons HTTP
  internal/service       validasi dan operasi bisnis
  internal/repository    kontrak repository
  internal/repository/postgres
  internal/model, dto    domain dan request
  pkg/apperror, httputil  error dan response envelope
  migrations             migrasi SQL up/down
  tests/integration      tes HTTP + PostgreSQL pada schema sementara
hris-web/
  apps/admin             entry point admin
  apps/employee          entry point portal karyawan
  packages/shared/src/
    pages, layouts, components
    services, stores     API client dan Zustand session
```

## Verifikasi

```sh
make test     # PostgreSQL harus aktif; Go integration tests + TypeScript
make build    # API binary + production build kedua portal
```

Tes integrasi membuat schema unik, menjalankan migrasi, menguji workflow, lalu menghapus schema miliknya sendiri. Data aplikasi tidak dihapus. `go test ./...` tanpa `TEST_DATABASE_URL` melewatkan tes integrasi secara eksplisit.

## Payroll tanpa payment gateway

1. Atur **Komponen gaji** setiap karyawan aktif: gaji pokok, total tunjangan, total potongan, dan catatan.
2. Buat payroll untuk periode bulanan. Sistem menyalin komponen untuk karyawan yang saat ini aktif dan sudah bergabung sebelum akhir periode.
3. Sesuaikan draft, termasuk pajak/BPJS, prorata, atau nominal lembur yang sudah dihitung HR. Gaji bersih dihitung otomatis: pokok + tunjangan − potongan.
4. Finalisasi setelah diverifikasi. Nominal terkunci dan slip tersedia pada portal masing-masing karyawan.
5. Lakukan pembayaran di luar aplikasi, lalu catat referensinya dengan **Catat sudah dibayar**. Tombol ini hanya mencatat pembayaran; tidak mengirim uang.

Data demo tambahan tersedia melalui `make seed-hr` setelah seed dasar. Nominalnya fiktif, bukan hasil perhitungan pajak. Seed tambahan tidak menimpa komponen gaji yang sudah ada.

## Upgrade database awal

Untuk database yang sudah memiliki migrasi 000001 dan 000002, jalankan masing-masing migrasi berikut **sekali** secara berurutan:

```sh
docker compose exec -T db psql -U hris -d hris -v ON_ERROR_STOP=1 --single-transaction < hris-api/migrations/000003_hr_suite.up.sql
docker compose exec -T db psql -U hris -d hris -v ON_ERROR_STOP=1 --single-transaction < hris-api/migrations/000004_payroll.up.sql
```

Database baru menjalankan keempat migrasi otomatis melalui Docker. Migrasi 000003/000004 bersifat forward-only. Pengajuan cuti lama mempertahankan hitungan hari kalender; pengajuan baru menyimpan snapshot hari kerja agar perubahan kalender tidak mengubah pengajuan yang sudah ada.

## Batas implementasi saat ini

Payroll mendukung nominal agregat rupiah utuh dan pembayaran manual. Belum ada kalkulasi otomatis pajak/BPJS, prorata, konversi lembur ke gaji, komponen dinamis, THR, transfer bank, atau koreksi payroll setelah finalisasi. Payroll historis tidak otomatis merekonstruksi karyawan yang sudah keluar.

Approval masih satu tingkat. Kalender kerja belum menjadi penjadwalan shift atau aturan keterlambatan; absensi/koreksi belum mendukung shift lintas tengah malam dan GPS/biometrik. Kuota cuti belum memiliki accrual/carry-over otomatis. Dokumen berupa tautan, tanpa upload/tanda tangan; izin file tetap diatur pada penyedia file. Target kinerja berupa progres, tanpa appraisal/360 review. Rekrutmen mencatat kandidat dan tahapan, tanpa portal lowongan atau konversi otomatis menjadi karyawan.

Session berlaku 12 jam, tanpa refresh token atau lupa-password mandiri. Manajemen perusahaan dilakukan lewat provisioning database; UI registrasi perusahaan belum tersedia. Daftar data belum memakai pagination server.

Konfigurasi Compose dan akun demo ditujukan untuk pengembangan lokal. Untuk deployment, gunakan kredensial terpisah dan reverse proxy HTTPS: `/api` diteruskan ke API dan rute SPA ke `index.html`. `vite preview` tidak menyediakan proxy API pengembangan.

Lihat [pemetaan arsitektur](docs/architecture.md) dan [kontrak API](docs/api.md).
