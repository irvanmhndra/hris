# Kontrak API v1

Prefix `/api/v1`. JSON sukses: `{ "success": true, "data": ... }`. JSON error aplikasi: `{ "success": false, "message": "..." }`. Status validasi 422, konflik 409, tidak ditemukan/sudah diproses 404, tidak terautentikasi 401, role salah 403. Middleware Echo dapat mengembalikan envelope default untuk 404/413/429.

Login: `POST /auth/login`, body `{ "company": "demo", "email": "admin@demo.hris", "password": "..." }`, respons `{token,user}`. Kirim `Authorization: Bearer <token>` pada endpoint berikut.

| Method | Endpoint | Akses | Fungsi |
|---|---|---|---|
| GET | /auth/me | semua | Identitas session |
| POST | /auth/logout | semua | Cabut session |
| GET | /dashboard | admin | Ringkasan perusahaan |
| GET, POST | /departments | admin | Daftar / tambah `{name}` |
| GET, POST | /employees | admin | Daftar / buat karyawan + akun |
| PUT | /employees/:id | admin | Edit karyawan + akun |
| GET | /attendances | semua | 100 catatan terbaru; karyawan hanya diri sendiri |
| POST | /attendance/in | karyawan | Check-in hari ini |
| POST | /attendance/out | karyawan | Check-out hari ini |
| GET | /leaves | semua | Pengajuan sesuai scope |
| POST | /leaves | karyawan | Ajukan cuti |
| PATCH | /leaves/:id | admin | Putuskan pengajuan pending |

Body karyawan:

```json
{"code":"EMP-013","name":"Nama Karyawan","email":"nama@example.test","department_id":1,"position":"Engineer","status":"active","joined_on":"2026-09-20","password":"Minimal12Karakter"}
```

Untuk edit, password kosong mempertahankan password lama. Password baru mencabut session sebelumnya. Status `inactive` memblokir login dan mencabut session.

Body cuti: `{ "kind":"annual", "start_date":"2026-11-10", "end_date":"2026-11-11", "reason":"Keperluan keluarga" }`. `kind` menerima annual/sick/personal. Body keputusan `{ "status":"approved" }` atau `{ "status":"rejected" }`.

`GET /health` berada di luar prefix dan mengecek liveness HTTP, bukan readiness database.

## Kalender, saldo, dan profil

| Method | Endpoint | Akses | Fungsi |
|---|---|---|---|
| GET / PUT | /calendar | semua / admin | Kalender kerja |
| GET / POST | /holidays | semua / admin | Daftar / tambah `{date,name}` |
| DELETE | /holidays/:id | admin | Hapus hari libur |
| GET | /leave-balances?year=2026 | semua | Kuota, used, reserved, available; karyawan hanya sendiri |
| PUT | /leave-balances/:employeeId | admin | `{year,allowance}`; tidak boleh di bawah pemakaian + reservasi |
| POST | /leaves/:id/cancel | karyawan | Batalkan milik sendiri; approved hanya sebelum mulai |
| GET / PUT | /profile | karyawan | Kontak pribadi |
| GET / PUT | /employees/:id/profile | admin | Kontak karyawan dalam perusahaan |
| GET | /audit-logs | admin | 250 aktivitas terbaru modul HR/payroll |

Body kalender: `{"workdays":[1,2,3,4,5],"annual_allowance":12,"start_time":"09:00","end_time":"18:00"}`. Minggu=0, Sabtu=6. Kalender tidak otomatis menilai keterlambatan. Body profil menerima `phone`, `address`, `emergency_name`, `emergency_phone`, `emergency_relation`; field identitas dari client tidak dipakai untuk mengubah data karyawan.

Cuti baru menghitung hari kerja di luar libur perusahaan dan menyimpan snapshot. Cuti annual pending mencadangkan saldo; approved memakai saldo; rejected/cancelled melepaskannya. Cuti lintas tahun memeriksa kuota masing-masing tahun. `calculation` menunjukkan `working_days` atau `legacy_calendar_days` untuk pengajuan lama.

## Workflow HR

`GET /hr/:module`, `POST /hr/:module`, `PUT /hr/:module/:id`, `PATCH /hr/:module/:id/action`.

Body create/edit: `{employee_id,title,description,status,due_date,data,version}`. `due_date` opsional dengan format YYYY-MM-DD. `version` wajib untuk edit dan tindakan, ambil dari respons daftar terbaru. Konflik edit basi menghasilkan 409. Karyawan tidak dapat mengubah penugasan dirinya.

| Module | Data utama | Akses/tindakan |
|---|---|---|
| overtime | start_at, end_at (RFC3339) | Karyawan buat pending/cancel; admin approve/reject. Durasi 15 menit–16 jam, tanpa overlap aktif |
| corrections | date, check_in, check_out (HH:mm WIB) | Karyawan buat/cancel; admin approve/reject. Approval memperbarui absensi bila data asal belum berubah |
| announcements | category | Admin kelola draft/published/archived; karyawan baca published |
| documents | url HTTPS, category | Sama dengan pengumuman; hanya tautan, bukan upload |
| onboarding | — | Admin buat/assign; pemilik atau admin start/complete |
| assets | code unik, serial | Admin kelola available/assigned/maintenance/retired; karyawan baca aset sendiri |
| goals | target, progress 0–100 | Admin buat/assign; pemilik atau admin memperbarui progress |
| recruitment | email, position | Admin saja; applied/screening/interview/offer/hired/rejected |

Body action: `{"action":"approve","version":1}`. Penolakan memerlukan `note` minimal 5 karakter. Target memakai `{"action":"progress","progress":60,"version":1}`. Semua scope perusahaan/pemilik ditegakkan API, bukan hanya UI.

## Payroll

| Method | Endpoint | Akses | Fungsi |
|---|---|---|---|
| GET | /salaries | admin | Komponen gaji karyawan |
| PUT | /salaries/:employeeId | admin | Simpan komponen default |
| GET / POST | /payroll | admin | Daftar / buat `{period:"2026-09"}` |
| GET | /payroll/:id/slips | admin | Snapshot slip suatu periode |
| PUT | /payroll/slips/:id | admin | Edit snapshot selama draft, wajib version |
| PATCH | /payroll/:id/action | admin | finalize / paid / void |
| GET | /payslips | karyawan | Slip sendiri yang finalized atau paid |

Body komponen: `{"basic_salary":8000000,"allowance":750000,"deduction":250000,"note":"Rincian penyesuaian HR","version":1}`. Nominal integer rupiah non-negatif, maksimal 1 triliun per komponen; potongan tidak boleh melebihi pokok+tunjangan. Version digunakan pada edit snapshot slip. Perhitungan net otomatis; kalkulasi pajak/BPJS/prorata/lembur belum otomatis.

Satu periode hanya boleh memiliki satu payroll non-void. Pembuatan membutuhkan konfigurasi gaji seluruh karyawan aktif yang bergabung sebelum akhir periode. Komponen disalin; perubahan default kemudian tidak mengubah snapshot. Draft dapat diedit atau void. Finalisasi mengunci nominal dan menerbitkan slip. Final/paid tidak dapat dibatalkan atau diedit.

Body tindakan: `{"action":"finalize"}`, `{"action":"void"}`, atau `{"action":"paid","reference":"TRANSFER-MANUAL-001"}`. Referensi paid wajib 5–150 karakter. Tindakan paid **hanya mencatat pembayaran yang dilakukan di luar aplikasi**. Tidak ada pemanggilan payment gateway atau transfer bank.
