# Arsitektur dan acuan

Acuan dibaca dari `/Users/irvan/Projects/nexpos-api`, `nexpos-web`, `fnb-api`, dan `fnb-web`. Tidak ada file atau data proyek tersebut yang diubah.

| Aspek | NexPOS/FNB | HRIS |
|---|---|---|
| Backend | Go 1.26.6, Echo v5 | Sama |
| Penyimpanan | PostgreSQL, sqlx, lib/pq | Sama |
| Struktur | cmd/config/internal/app/router/handler/service/repository/model/dto/pkg | Sama |
| Tenant | company_id | Scope query dan foreign key komposit |
| Autentikasi aktual | Opaque access token yang diverifikasi lewat tabel session | Opaque token 256-bit; SHA-256 di database; masa aktif 12 jam |
| Frontend | pnpm + Turborepo, aplikasi terpisah, shared package | admin + employee + shared |
| UI state | React Query, Zustand, React Router | Sama |
| Build | React + TypeScript + Vite | Sama; lockfile mengunci resolusi dependency |
| Styling | Tailwind dan shared components | CSS bersama untuk desain HRIS |
| Domain | POS, menu, transaksi, outlet | karyawan, cuti, absensi, workflow HR, payroll |

API dan web diletakkan sebagai dua direktori di satu repository karena workspace HRIS awal kosong. Keduanya tetap memiliki manifest/build terpisah dan dapat dipisahkan menjadi repository tersendiri.

Request melewati router dan middleware auth, lalu handler mengubah JSON menjadi DTO, service memvalidasi, dan repository menjalankan SQL dengan scope tenant. Identitas perusahaan/karyawan selalu berasal dari session, bukan dari request body.

Pembuatan/perubahan karyawan dan akun login memakai transaksi. Reset password atau penonaktifan mencabut session terkait. Unique constraint mencegah absensi ganda. Pembuatan cuti mengunci baris karyawan untuk mencegah dua pengajuan overlap yang bersamaan. Review memakai conditional update `status='pending'` sehingga hanya keputusan pertama diterima. Foreign key komposit menolak referensi departemen/karyawan lintas perusahaan.

Tanggal absensi mengikuti Asia/Jakarta; timestamp disimpan dengan timezone. Cuti menggunakan DATE dan snapshot hari kerja sesuai kalender perusahaan. Pengajuan lama mempertahankan perhitungan hari kalender. Frontend memanggil API same-origin lewat proxy Vite; token hanya bertahan dalam sessionStorage tab. Backend tetap menegakkan otorisasi meskipun rute frontend dimanipulasi.

Migrasi `000001` membuat tabel dasar, `000002` memperkuat relasi akun-karyawan, `000003` menambah kalender/saldo/profil/workflow/audit, dan `000004` menambah payroll. Bootstrap Docker hanya menjalankan migrasi ketika volume baru dibuat; update database lama harus dilakukan secara eksplisit dengan migration runner atau psql.

## Workflow dan konsistensi

`hr_items` menyimpan workflow ringan dalam satu tabel, dengan whitelist modul/status, data JSONB yang dipetakan ke DTO bertipe, serta scope tenant dan pemilik. Modul finansial memakai tabel khusus: `salary_profiles`, `payroll_runs`, dan `payroll_entries`. Nominal rupiah berupa integer, bukan floating point.

Saldo cuti tahunan adalah kuota dikurangi hari approved dan pending. Transaksi mengunci karyawan, memeriksa overlap dan kuota per tahun, lalu menyimpan `leave_days`. Penolakan/pembatalan melepaskan reservasi. Perubahan kalender tidak menghitung ulang snapshot cuti. Alokasi yang sudah dipakai dibekukan agar perubahan kuota default tidak mengubah hak sebelumnya.

Workflow memakai kolom versi untuk menolak edit basi. Lembur memeriksa overlap; koreksi absensi menyimpan fingerprint data asal dan menolak approval jika absensi sudah berubah. Approval koreksi dan perubahan absensi berlangsung dalam transaksi yang sama.

Pembuatan payroll menggunakan snapshot transaksi repeatable-read. Pengeditan slip mengunci periode dan memeriksa versi; finalisasi mengunci nominal dan menerbitkan slip. Karyawan hanya dapat membaca slip miliknya yang final/paid. Status paid membutuhkan referensi pembayaran manual, tanpa integrasi transfer. Audit modul baru ditulis dalam transaksi perubahan.
