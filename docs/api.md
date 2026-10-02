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
| 413 | PAYLOAD_TOO_LARGE | Body lebih dari 1 MB (unggahan file: 10 MB) |
| 422 | VALIDATION_ERROR | Input tidak lolos validasi, termasuk body JSON yang tidak dapat dibaca |
| 429 | TOO_MANY_REQUESTS | Rate limit login dan endpoint akun mandiri |
| 500 | INTERNAL_ERROR | Kesalahan server; detail hanya di log |
| 503 | UNAVAILABLE | `/health`: database tidak terjangkau |

Klien sebaiknya bercabang berdasarkan `error_code`, bukan teks `message`. Setiap respons membawa header `X-Request-Id` yang juga tercatat di log server.

Login: `POST /auth/login`, body `{ "company": "demo", "email": "admin@demo.hris", "password": "..." }`, respons `{token,user}`. `user.is_manager` bernilai true bila ada karyawan aktif yang melapor kepadanya. Kirim `Authorization: Bearer <token>` pada endpoint berikut.

Akun mandiri (tanpa login, rate limit 5 permintaan lalu 1 per 10 detik per IP):

| Method | Endpoint | Fungsi |
|---|---|---|
| GET | /auth/config | `{signup_enabled}` |
| POST | /auth/forgot-password | `{company,email}`; selalu 200. Bila akun aktif ada, email berisi tautan `<portal>/reset-password?token=…` (1 jam, sekali pakai) |
| POST | /auth/reset-password | `{token,password}`; password 12–72 karakter; semua session lama dicabut |
| POST | /auth/register | `{company_name,company_slug,name,email,password}`; hanya bila `SIGNUP_ENABLED=true`; membuat perusahaan, kalender default, departemen "Umum", admin; respons `{token,user}` |

| Method | Endpoint | Akses | Fungsi |
|---|---|---|---|
| GET | /auth/me | semua | Identitas session |
| POST | /auth/logout | semua | Cabut session |
| GET | /dashboard | admin | Ringkasan perusahaan |
| GET, POST | /departments | admin | Daftar / tambah `{name}` |
| GET, POST | /employees | admin | Daftar (lihat [pagination](#pagination)) / buat karyawan + akun |
| PUT | /employees/:id | admin | Edit karyawan + akun |
| GET | /attendances | semua | Terbaru dulu; tanpa `page` 100 catatan terbaru; filter `date`; karyawan hanya diri sendiri |
| GET | /attendance/today | karyawan | Jadwal hari ini, `require_location`, jumlah lokasi |
| POST | /attendance/in | karyawan | Check-in; body opsional `{latitude,longitude}` |
| POST | /attendance/out | karyawan | Check-out absensi terbuka (≤ 20 jam, termasuk shift malam) |
| GET | /leaves | semua | Pengajuan sesuai scope; filter `status`, pagination |
| POST | /leaves | karyawan | Ajukan cuti |
| PATCH | /leaves/:id | admin | Putuskan pengajuan pending |

Body karyawan:

```json
{"code":"EMP-013","name":"Nama Karyawan","email":"nama@example.test","department_id":1,"position":"Engineer","status":"active","joined_on":"2026-09-20","password":"Minimal12Karakter","manager_id":3,"shift_id":null,"left_on":""}
```

Untuk edit, password kosong mempertahankan password lama. Password baru mencabut session sebelumnya. Status `inactive` memblokir login dan mencabut session; `left_on` (hari kerja terakhir, default hari ini) dipakai payroll untuk prorata dan perhitungan PPh tahunan. `manager_id` adalah atasan langsung (tidak boleh melingkar); `shift_id` shift default.

Body cuti: `{ "kind":"annual", "start_date":"2026-11-10", "end_date":"2026-11-11", "reason":"Keperluan keluarga" }`. `kind` menerima annual/sick/personal/unpaid. Cuti `unpaid` tidak memakai saldo dan mengurangi hari dibayar di payroll. Body keputusan `{ "status":"approved" }` atau `{ "status":"rejected" }`; HR dapat memutuskan di tahap mana pun.

### Persetujuan dua tingkat

Pengajuan cuti, lembur, dan koreksi dari karyawan yang memiliki atasan aktif (dengan akun portal) dimulai pada `stage: "manager"`, lainnya langsung `stage: "hr"`.

| Method | Endpoint | Akses | Fungsi |
|---|---|---|---|
| GET | /team/approvals | karyawan | Pengajuan bawahan langsung yang menunggu atasan |
| POST | /team/approvals/:type/:id | karyawan (atasan) | `type` leave/overtime/corrections; `{action:"approve"\|"reject",note,version}` — approve meneruskan ke HR, reject wajib alasan ≥5 karakter; `version` wajib untuk lembur/koreksi |

`GET /health` berada di luar prefix dan mengecek readiness: 200 `{status:"ok",database:"connected"}` bila database menjawab ping dalam 2 detik, selain itu 503 `UNAVAILABLE`.

## Pagination

`GET /employees`, `GET /audit-logs`, `GET /attendances`, `GET /leaves`, dan `GET /hr/:module` menerima `page` (mulai 1) dan `per_page` (default 20, maksimal 100). Respons berhalaman:

```json
{
  "success": true,
  "data": [ ... ],
  "meta": { "pagination": { "current_page": 2, "per_page": 25, "total_records": 52, "total_pages": 3 } }
}
```

`/employees` juga menerima filter `search` (nama, NIK, email, atau departemen; tanpa beda huruf besar/kecil; `%` dan `_` dicari sebagai karakter biasa; maks. 120 karakter), `status` (`active`/`inactive`), dan `department_id`. Filter berlaku dengan atau tanpa `page`.

`/leaves` dan `/hr/:module` menerima `status`; `/attendances` menerima `date` (YYYY-MM-DD).

Tanpa `page`, endpoint mempertahankan perilaku lama agar klien lama tidak rusak: `/employees` mengembalikan seluruh data yang cocok (dipakai ekspor CSV dan pemilih karyawan), `/audit-logs` mengembalikan 250 aktivitas terbaru, `/attendances` 100 terbaru, `/leaves` dan `/hr/:module` seluruhnya. Semuanya tanpa `meta`.

## Kalender, saldo, dan profil

| Method | Endpoint | Akses | Fungsi |
|---|---|---|---|
| GET / PUT | /calendar | semua / admin | Kalender kerja |
| GET / POST | /holidays | semua / admin | Daftar / tambah `{date,name}` |
| DELETE | /holidays/:id | admin | Hapus hari libur |
| GET | /leave-balances?year=2026 | semua | Kuota, accrued, carried_over, used, reserved, available; karyawan hanya sendiri |
| PUT | /leave-balances/:employeeId | admin | `{year,allowance}`; hak setahun + carry-over tidak boleh di bawah pemakaian + reservasi |
| POST | /leaves/:id/cancel | karyawan | Batalkan milik sendiri; approved hanya sebelum mulai |
| GET / PUT | /profile | karyawan | Kontak pribadi |
| GET / PUT | /employees/:id/profile | admin | Kontak karyawan dalam perusahaan |
| GET | /audit-logs | admin | Aktivitas modul HR/payroll, terbaru dulu (lihat [pagination](#pagination)) |

Body kalender: `{"workdays":[1,2,3,4,5],"annual_allowance":12,"start_time":"09:00","end_time":"18:00","leave_accrual":"annual","carry_over_max":0,"leave_eligibility_months":0,"require_location":false}`. Minggu=0, Sabtu=6. Jam kalender adalah jadwal default ("Jam kantor") untuk karyawan tanpa shift. `leave_accrual` `annual` (penuh di awal tahun) atau `monthly` (1/12 per bulan sampai bulan cuti); `carry_over_max` hari sisa hak tahun lalu yang terbawa (0 = hangus; hanya dari hak tahun lalu sendiri); `leave_eligibility_months` masa kerja sebelum berhak cuti tahunan (0–24). `require_location` menolak absensi di luar radius lokasi. `carry_over_expiry_months` (0–12; 0 = tanpa batas): hari carry-over hanya dapat dipakai untuk cuti sampai akhir bulan tersebut; saldo menampilkan `carry_expires_on`. Body profil menerima `phone`, `address`, `emergency_name`, `emergency_phone`, `emergency_relation`; field identitas dari client tidak dipakai untuk mengubah data karyawan.

Cuti baru menghitung hari kerja di luar libur perusahaan dan menyimpan snapshot. Cuti annual pending mencadangkan saldo; approved memakai saldo; rejected/cancelled melepaskannya. Cuti lintas tahun memeriksa kuota masing-masing tahun. `calculation` menunjukkan `working_days` atau `legacy_calendar_days` untuk pengajuan lama.

## Workflow HR

`GET /hr/:module`, `POST /hr/:module`, `PUT /hr/:module/:id`, `PATCH /hr/:module/:id/action`.

Body create/edit: `{employee_id,title,description,status,due_date,data,version}`. `due_date` opsional dengan format YYYY-MM-DD. `version` wajib untuk edit dan tindakan, ambil dari respons daftar terbaru. Konflik edit basi menghasilkan 409. Karyawan tidak dapat mengubah penugasan dirinya.

| Module | Data utama | Akses/tindakan |
|---|---|---|
| overtime | start_at, end_at (RFC3339) | Karyawan buat pending/cancel; admin approve/reject. Durasi 15 menit–16 jam, tanpa overlap aktif |
| corrections | date, check_in, check_out (HH:mm WIB) | Karyawan buat/cancel; admin approve/reject. Approval memperbarui absensi bila data asal belum berubah |
| announcements | category | Admin kelola draft/published/archived; karyawan baca published |
| documents | url HTTPS **atau** file_id, category | Sama dengan pengumuman; file diunggah lewat `POST /files` |
| onboarding | — | Admin buat/assign; pemilik atau admin start/complete |
| assets | code unik, serial | Admin kelola available/assigned/maintenance/retired; karyawan baca aset sendiri |
| goals | target, progress 0–100 | Admin buat/assign; pemilik atau admin memperbarui progress |
| recruitment | email, position | Admin saja; applied/screening/interview/offer/hired/rejected. `POST /hr/recruitment/:id/convert` dengan body karyawan membuat karyawan + akun dari kandidat offer/hired (sekali), lalu menautkan kandidat (`employee_id`) dan menandainya hired |

Pengumuman dan dokumen dapat membawa `data.file_id`; server mengisi `data.file_name`. Lembur dan koreksi menampilkan `stage` (lihat persetujuan dua tingkat). Koreksi dengan jam pulang ≤ jam masuk berarti keesokan hari (maks. 20 jam).

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
| POST | /payroll/:id/correction | admin | Buat run koreksi (draft) untuk payroll reguler final/paid pada periode terakhir tahun berjalan |
| GET | /payslips | karyawan | Slip sendiri yang finalized atau paid |

Body profil gaji:

```json
{"basic_salary":8000000,"ptkp_status":"K/1","tax_method":"gross",
 "bpjs_kesehatan":true,"bpjs_ketenagakerjaan":true,"bpjs_pensiun":true,"overtime_eligible":true,
 "note":"","components":[
  {"kind":"allowance","name":"Tunjangan jabatan","amount":750000,"fixed":true,"taxable":true},
  {"kind":"deduction","name":"Koperasi","amount":100000}]}
```

Profil gaji juga menerima `nik` (16 digit), `npwp` (15/16 digit), `bpjs_kesehatan_number`, dan `bpjs_ketenagakerjaan_number` (8–20 digit), boleh kosong; titik/tanda hubung diabaikan. NIK/NPWP disalin ke slip. `ptkp_status`: TK/0–TK/3, K/0–K/3. `tax_method`: `gross` (dipotong dari gaji), `gross_up` (perusahaan memberi tunjangan PPh sebesar pajaknya), `none` (PPh 21 tidak dihitung; dipakai profil lama). Jaminan Pensiun membutuhkan BPJS Ketenagakerjaan. Tunjangan `fixed` (tetap) menjadi dasar BPJS, upah lembur, dan THR. Nominal integer rupiah 0–1 triliun.

Settings: `{"jkk_rate":24,"jp_wage_cap":10547400,"kes_wage_cap":12000000,"late_deduction":"none","late_deduction_amount":0,"deduct_absence":false}`. `late_deduction`: `none`, `per_minute` (upah tetap/173 per jam × menit terlambat), atau `per_occurrence` (`late_deduction_amount` per check-in terlambat), menjadi baris `LATE` yang dapat disesuaikan. `deduct_absence` mengurangi hari dibayar untuk hari terjadwal tanpa absensi maupun cuti disetujui (dihitung sampai hari payroll dibuat). `jkk_rate` dalam perseratus persen (24 = 0,24%; kelas 24/54/89/127/174).

Penyesuaian slip: `{"version":1,"note":"","lines":[{"kind":"earning","code":"ADJUSTMENT","name":"Bonus","amount":1000000,"taxable":true,"fixed":false}, …]}`. Kirim seluruh baris non-statutori (kode `BASIC`, `ALLOWANCE`, `OVERTIME`, `THR`, `ADJUSTMENT`, `DEDUCTION`, `LATE`); baris BPJS, PPh 21, tunjangan PPh, dan pengembalian PPh selalu dihitung ulang dan ditolak bila dikirim.

Slip menyertakan `ter_rate` (perseratus persen; 0 untuk perhitungan tahunan/manual/koreksi), `late_count`, `late_minutes`, `absent_days`, `nik`, `npwp`, nomor BPJS, dan `run_kind`. Run koreksi (`kind: "correction"`, `corrects_run_id`) menyalin snapshot pajak setiap slip dengan nominal nol; baris yang ditambahkan HR dihitung bersama slip terkunci periode yang sama sehingga PPh 21 slip koreksi adalah selisihnya (bisa berupa pengembalian). Satu draft koreksi per periode; payroll periode berikutnya menunggu koreksi difinalisasi.

Satu periode hanya boleh memiliki satu payroll reguler non-void; payroll periode sebelumnya harus sudah final/paid, dan periode baru tidak boleh disisipkan sebelum payroll lain pada tahun yang sama. Pembuatan membutuhkan profil gaji untuk seluruh karyawan aktif yang bergabung sebelum akhir periode serta karyawan yang keluar pada/sesudah awal periode. Draft dapat disesuaikan atau di-void (lembur di dalamnya kembali tersedia). Finalisasi mengunci nominal dan menerbitkan slip. Final/paid tidak dapat dibatalkan atau diedit.

Body tindakan: `{"action":"finalize"}`, `{"action":"void"}`, atau `{"action":"paid","reference":"TRANSFER-MANUAL-001"}`. Referensi paid wajib 5–150 karakter. Tindakan paid **hanya mencatat pembayaran yang dilakukan di luar aplikasi**. Tidak ada pemanggilan payment gateway atau transfer bank.

## Shift, jadwal, dan lokasi absensi

| Method | Endpoint | Akses | Fungsi |
|---|---|---|---|
| GET / POST | /shifts | admin | Daftar / buat `{name,start_time,end_time,grace_minutes}` (HH:MM; selesai ≤ mulai = shift malam) |
| PUT | /shifts/:id | admin | Edit, termasuk `active` |
| GET | /schedule?from=&to= | admin | Jadwal terselesaikan per karyawan per tanggal (maks. 42 hari): `off`, `shift_name`, `start`, `end`, `assigned` |
| PUT | /schedule | admin | `{employee_ids,from,to,shift_id,clear}` maks. 92 hari: `shift_id` null = libur; `clear` kembali ke default |
| GET | /attendance-summary?month=YYYY-MM | admin | Rekap s.d. hari ini: hari terjadwal, hadir, terlambat (kali/menit), pulang cepat, cuti, tanpa keterangan |
| GET / POST | /attendance-locations | admin | `{name,latitude,longitude,radius_m}` (radius 10–5000 m) |
| PUT / DELETE | /attendance-locations/:id | admin | Edit / hapus |

Urutan jadwal: penugasan per tanggal → shift default karyawan (hanya pada hari kerja kalender, di luar libur) → jam kantor kalender. Check-in menyimpan snapshot jadwal, `late_minutes` (setelah toleransi), serta jarak ke lokasi terdekat; check-out menyimpan `early_leave_minutes`.

## File

| Method | Endpoint | Akses | Fungsi |
|---|---|---|---|
| POST | /files | admin | Multipart field `file`, maks. 10 MB; PDF, PNG, JPG, WEBP, DOCX, XLSX (ekstensi dicocokkan dengan isi) |
| GET | /files/:id | semua | Unduh sebagai attachment; karyawan hanya file pada pengumuman/dokumen published |

File disimpan di `UPLOAD_DIR` (disk lokal) atau bucket S3-compatible (`STORAGE_DRIVER=s3`: AWS S3, Cloudflare R2, MinIO) dengan nama acak; metadata di tabel `files`.
