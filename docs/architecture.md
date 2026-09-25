# Arsitektur dan acuan

Acuan dibaca dari `/Users/irvan/Projects/nexpos-api`, `nexpos-web`, `fnb-api`, dan `fnb-web`. Tidak ada file atau data proyek tersebut yang diubah.

| Aspek | NexPOS/FNB | HRIS |
|---|---|---|
| Backend | Go 1.26.6, Echo v5 | Sama |
| Penyimpanan | PostgreSQL, sqlx, lib/pq | Sama |
| Struktur | cmd/config/internal/app/router/handler/service/repository/model/dto/pkg, dipecah per domain | Sama: repository, service, dan handler per domain |
| Tenant | company_id | Scope query dan foreign key komposit |
| Autentikasi aktual | Opaque access token yang diverifikasi lewat tabel session | Opaque token 256-bit; SHA-256 di database; masa aktif 12 jam |
| Frontend | pnpm + Turborepo, aplikasi terpisah, shared package | admin + employee + shared; tiap portal punya rute lazy sendiri |
| UI state | React Query, Zustand, React Router | Sama |
| Build | React + TypeScript + Vite | Sama; lockfile mengunci resolusi dependency |
| Styling | Tailwind dan shared components | CSS bersama untuk desain HRIS |
| Migrasi | golang-migrate | golang-migrate, file SQL di-embed ke binary |
| Domain | POS, menu, transaksi, outlet | karyawan, cuti, absensi, workflow HR, payroll |

API dan web diletakkan sebagai dua direktori di satu repository karena workspace HRIS awal kosong. Keduanya tetap memiliki manifest/build terpisah dan dapat dipisahkan menjadi repository tersendiri.

## Lapisan backend

Request melewati router dan middleware auth, lalu handler mengubah JSON menjadi DTO, service memvalidasi dan memutuskan otorisasi, dan repository menjalankan SQL dengan scope tenant. Identitas perusahaan/karyawan selalu berasal dari session, bukan dari request body.

Setiap domain punya kontrak repository sendiri di `internal/repository/interface.go` (auth, employee, attendance, leave, profile, HR item, payroll, audit, dashboard), satu service, dan satu handler; `internal/app/wiring.go` merangkainya. Pembagiannya:

- **Handler**: parsing path/query/body dan pemilihan bentuk respons (misalnya berhalaman atau tidak). Tidak ada aturan bisnis.
- **Service**: validasi, otorisasi, dan scope data dari user session: karyawan hanya menjangkau record miliknya (`ownScope`), modul request (lembur/koreksi) selalu diajukan atas nama pemohon dengan status `pending`, pengumuman/dokumen bersifat company-wide, dan sebagainya. Aturan ini diuji tanpa database memakai fake repository (`internal/service/scope_test.go`).
- **Repository**: SQL saja. Menerima nilai eksplisit (`companyID`, `actorID`, scope, struct input di `internal/model`) dan tidak mengenal DTO maupun `*model.User`. Operasi yang kebenarannya bergantung pada lock (overlap cuti, kuota, snapshot payroll, approval koreksi) tetap berada dalam satu transaksi repository, bersama penulisan audit.

Error memakai `apperror.Error{Status, Code, Message}`; `httputil.Error` memetakan error PostgreSQL (unique, foreign key, check, serialization/deadlock) ke 409/422 dan menyembunyikan detail error tak dikenal di balik 500 yang tercatat di log bersama request ID. Error handler Echo merender 404/405/413/429 dalam envelope yang sama.

Middleware global: request ID, log request terstruktur (`slog`), recover, CORS opsional (`CORS_ALLOWED_ORIGINS`; kosong berarti same-origin), dan batas body 1 MB. Login dibatasi rate limiter. Email yang tidak terdaftar tetap menjalankan bcrypt agar waktu respons tidak membocorkan akun mana yang ada. Kegagalan database saat membaca session menghasilkan 500, bukan 401, supaya gangguan database tidak terlihat seperti semua user logout.

Pembuatan/perubahan karyawan dan akun login memakai transaksi. Reset password atau penonaktifan mencabut session terkait. Unique constraint mencegah absensi ganda. Pembuatan cuti mengunci baris karyawan untuk mencegah dua pengajuan overlap yang bersamaan. Review memakai conditional update `status='pending'` sehingga hanya keputusan pertama diterima. Foreign key komposit menolak referensi departemen/karyawan lintas perusahaan.

Tanggal absensi mengikuti Asia/Jakarta; timestamp disimpan dengan timezone. Cuti menggunakan DATE dan snapshot hari kerja sesuai kalender perusahaan. Pengajuan lama mempertahankan perhitungan hari kalender. Frontend memanggil API same-origin lewat proxy Vite; token hanya bertahan dalam sessionStorage tab. Backend tetap menegakkan otorisasi meskipun rute frontend dimanipulasi.

## Migrasi

Migrasi `000001` membuat tabel dasar, `000002` memperkuat relasi akun-karyawan, `000003` menambah kalender/saldo/profil/workflow/audit, dan `000004` menambah payroll. File SQL di-embed ke binary dan dijalankan golang-migrate (`cmd/migrate`), yang mencatat versi di `schema_migrations`. Di Compose, service `migrate` berjalan sekali setelah database healthy dan API baru start setelah migrasi sukses; migrasi baru otomatis sampai ke database yang sudah ada.

Dua pengaman: `up` menolak schema yang sudah berisi tabel tetapi belum punya `schema_migrations` (hasil setup initdb.d lama) alih-alih menjalankan ulang `000001` dan meninggalkan status dirty; `down` memeriksa setiap langkah lebih dulu dan menolak migrasi yang ditandai `-- Forward-only` (000003, 000004) sebelum menyentuh database. Tes integrasi menjalankan runner yang sama dua kali untuk memastikan idempoten.

## Frontend

`@hris/shared` berisi Shell, halaman, komponen, API client, dan store. Shell tidak lagi mengimpor halaman; portal `@hris/shared/admin` dan `@hris/shared/employee` masing-masing mendeklarasikan navigasi dan rute lazy-nya sendiri, dan `apps/admin` / `apps/employee` hanya mengimpor portalnya. Hasilnya build karyawan tidak berisi halaman payroll, karyawan, audit, atau ringkasan admin, dan setiap halaman dimuat saat pertama dikunjungi. Daftar karyawan dan audit log memakai pagination server; pencarian di-debounce dan ekspor CSV mengambil seluruh data yang cocok dengan filter.

## Workflow dan konsistensi

`hr_items` menyimpan workflow ringan dalam satu tabel, dengan whitelist modul/status, data JSONB yang dipetakan ke DTO bertipe, serta scope tenant dan pemilik. Modul finansial memakai tabel khusus: `salary_profiles`, `payroll_runs`, dan `payroll_entries`. Nominal rupiah berupa integer, bukan floating point.

Saldo cuti tahunan adalah kuota dikurangi hari approved dan pending. Transaksi mengunci karyawan, memeriksa overlap dan kuota per tahun, lalu menyimpan `leave_days`. Penolakan/pembatalan melepaskan reservasi. Perubahan kalender tidak menghitung ulang snapshot cuti. Alokasi yang sudah dipakai dibekukan agar perubahan kuota default tidak mengubah hak sebelumnya.

Workflow memakai kolom versi untuk menolak edit basi. Lembur memeriksa overlap; koreksi absensi menyimpan fingerprint data asal dan menolak approval jika absensi sudah berubah. Approval koreksi dan perubahan absensi berlangsung dalam transaksi yang sama.

Pembuatan payroll menggunakan snapshot transaksi repeatable-read. Pengeditan slip mengunci periode dan memeriksa versi; finalisasi mengunci nominal dan menerbitkan slip. Karyawan hanya dapat membaca slip miliknya yang final/paid. Status paid membutuhkan referensi pembayaran manual, tanpa integrasi transfer. Audit modul baru ditulis dalam transaksi perubahan.
