# Kontrak API v1

Prefix `/api/v1`. JSON sukses: `{ "success": true, "data": ... }`; daftar berhalaman menambah `meta.pagination`. Setiap error, termasuk yang berasal dari Echo (rute tidak ada, method salah, body > 1 MB, rate limit), memakai envelope yang sama:

```json
{ "success": false, "error_code": "VALIDATION_ERROR", "message": "Status tidak valid" }
```

| HTTP | error_code | Arti |
|---|---|---|
| 400 | BAD_REQUEST | Permintaan ditolak Echo sebelum mencapai handler |
| 401 | UNAUTHORIZED | Belum login / session berakhir |
| 403 | FORBIDDEN | Role tidak berhak |
| 404 | NOT_FOUND | Data tidak ada, di luar scope, atau sudah diproses; juga rute tidak dikenal |
| 405 | METHOD_NOT_ALLOWED | Method tidak didukung rute |
| 409 | CONFLICT | Duplikat, versi basi, atau transaksi bersamaan |
| 413 | PAYLOAD_TOO_LARGE | Body lebih dari 1 MB |
| 422 | VALIDATION_ERROR | Input tidak lolos validasi, termasuk body JSON yang tidak dapat dibaca |
| 429 | TOO_MANY_REQUESTS | Rate limit login |
| 500 | INTERNAL_ERROR | Kesalahan server; detail hanya di log |
| 503 | UNAVAILABLE | `/health`: database tidak terjangkau |

Klien sebaiknya bercabang berdasarkan `error_code`, bukan teks `message`. Setiap respons membawa header `X-Request-Id` yang juga tercatat di log server.

Login: `POST /auth/login`, body `{ "company": "demo", "email": "admin@demo.hris", "password": "..." }`, respons `{token,user}`. Kirim `Authorization: Bearer <token>` pada endpoint berikut.

| Method | Endpoint | Akses | Fungsi |
|---|---|---|---|
| GET | /auth/me | semua | Identitas session |
| POST | /auth/logout | semua | Cabut session |
| GET | /dashboard | admin | Ringkasan perusahaan |
| GET, POST | /departments | admin | Daftar / tambah `{name}` |
| GET, POST | /employees | admin | Daftar (lihat [pagination](#pagination)) / buat karyawan + akun |
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

`GET /health` berada di luar prefix dan mengecek readiness: 200 `{status:"ok",database:"connected"}` bila database menjawab ping dalam 2 detik, selain itu 503 `UNAVAILABLE`.

## Pagination

`GET /employees` dan `GET /audit-logs` menerima `page` (mulai 1) dan `per_page` (default 20, maksimal 100). Respons berhalaman:

```json
{
  "success": true,
  "data": [ ... ],
  "meta": { "pagination": { "current_page": 2, "per_page": 25, "total_records": 52, "total_pages": 3 } }
}
```

`/employees` juga menerima filter `search` (nama, NIK, email, atau departemen; tanpa beda huruf besar/kecil; `%` dan `_` dicari sebagai karakter biasa; maks. 120 karakter), `status` (`active`/`inactive`), dan `department_id`. Filter berlaku dengan atau tanpa `page`.

Tanpa `page`, kedua endpoint mempertahankan perilaku lama agar klien lama tidak rusak: `/employees` mengembalikan seluruh data yang cocok (dipakai ekspor CSV dan pemilih karyawan), `/audit-logs` mengembalikan 250 aktivitas terbaru. Keduanya tanpa `meta`.

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
| GET | /audit-logs | admin | Aktivitas modul HR/payroll, terbaru dulu (lihat [pagination](#pagination)) |

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
| GET | /salaries | admin | Profil gaji karyawan aktif + komponen |
| PUT | /salaries/:employeeId | admin | Simpan profil gaji (lihat body di bawah) |
| GET / PUT | /payroll/settings | admin | Kelas JKK dan batas upah BPJS perusahaan |
| GET / POST | /payroll | admin | Daftar / buat `{"period":"2026-09","thr_date":"2026-03-20"}` (`thr_date` opsional) |
| GET | /payroll/:id/slips | admin | Slip suatu periode, termasuk `lines` |
| PUT | /payroll/slips/:id | admin | Sesuaikan baris slip draft lalu hitung ulang, wajib version |
| PATCH | /payroll/:id/action | admin | finalize / paid / void |
| GET | /payslips | karyawan | Slip sendiri yang finalized atau paid |

Body profil gaji:

```json
{"basic_salary":8000000,"ptkp_status":"K/1","tax_method":"gross",
 "bpjs_kesehatan":true,"bpjs_ketenagakerjaan":true,"bpjs_pensiun":true,"overtime_eligible":true,
 "note":"","components":[
  {"kind":"allowance","name":"Tunjangan jabatan","amount":750000,"fixed":true,"taxable":true},
  {"kind":"deduction","name":"Koperasi","amount":100000}]}
```

`ptkp_status`: TK/0–TK/3, K/0–K/3. `tax_method`: `gross` (dipotong dari gaji), `gross_up` (perusahaan memberi tunjangan PPh sebesar pajaknya), `none` (PPh 21 tidak dihitung; dipakai profil lama). Jaminan Pensiun membutuhkan BPJS Ketenagakerjaan. Tunjangan `fixed` (tetap) menjadi dasar BPJS, upah lembur, dan THR. Nominal integer rupiah 0–1 triliun.

Settings: `{"jkk_rate":24,"jp_wage_cap":10547400,"kes_wage_cap":12000000}`. `jkk_rate` dalam perseratus persen (24 = 0,24%; kelas 24/54/89/127/174).

Penyesuaian slip: `{"version":1,"note":"","lines":[{"kind":"earning","code":"ADJUSTMENT","name":"Bonus","amount":1000000,"taxable":true,"fixed":false}, …]}`. Kirim seluruh baris non-statutori (kode `BASIC`, `ALLOWANCE`, `OVERTIME`, `THR`, `ADJUSTMENT`, `DEDUCTION`); baris BPJS, PPh 21, tunjangan PPh, dan pengembalian PPh selalu dihitung ulang dan ditolak bila dikirim.

Satu periode hanya boleh memiliki satu payroll non-void; payroll periode sebelumnya harus sudah final/paid, dan periode baru tidak boleh disisipkan sebelum payroll lain pada tahun yang sama. Pembuatan membutuhkan profil gaji untuk seluruh karyawan aktif yang bergabung sebelum akhir periode serta karyawan yang keluar pada/sesudah awal periode. Draft dapat disesuaikan atau di-void (lembur di dalamnya kembali tersedia). Finalisasi mengunci nominal dan menerbitkan slip. Final/paid tidak dapat dibatalkan atau diedit.

Body tindakan: `{"action":"finalize"}`, `{"action":"void"}`, atau `{"action":"paid","reference":"TRANSFER-MANUAL-001"}`. Referensi paid wajib 5–150 karakter. Tindakan paid **hanya mencatat pembayaran yang dilakukan di luar aplikasi**. Tidak ada pemanggilan payment gateway atau transfer bank.
