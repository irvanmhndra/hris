-- Optional fictitious demo data. Existing records and salary configuration are preserved.
INSERT INTO salary_profiles(company_id,employee_id,basic_salary,note,ptkp_status)
SELECT e.company_id,e.id,8000000+((row_number() OVER(ORDER BY e.id)-1)%5)*500000,'Data demo fiktif.',
       (ARRAY['TK/0','K/0','K/1','TK/1','K/2'])[((row_number() OVER(ORDER BY e.id)-1)%5)+1]
FROM employees e JOIN companies c ON c.id=e.company_id WHERE c.slug='demo'
ON CONFLICT DO NOTHING;
-- Components only for profiles that have none yet, so reruns add nothing.
INSERT INTO salary_components(company_id,employee_id,kind,name,amount,fixed,taxable,position)
SELECT s.company_id,s.employee_id,v.kind,v.name,v.amount,v.fixed,true,v.position
FROM salary_profiles s JOIN companies c ON c.id=s.company_id AND c.slug='demo'
CROSS JOIN (VALUES ('allowance','Tunjangan jabatan',500000,true,0),('allowance','Uang makan',250000,false,1),('deduction','Iuran koperasi',100000,false,2))
 AS v(kind,name,amount,fixed,position)
WHERE s.note='Data demo fiktif.'
  AND NOT EXISTS (SELECT 1 FROM salary_components x WHERE x.company_id=s.company_id AND x.employee_id=s.employee_id);

INSERT INTO hr_items(company_id,module,employee_id,title,description,status,due_date,data,created_by)
SELECT c.id,v.module,CASE WHEN v.module IN ('onboarding','assets','goals') THEN e.id ELSE NULL END,v.title,v.description,v.status,
CASE WHEN v.module IN ('onboarding','goals','recruitment') THEN (now() AT TIME ZONE 'Asia/Jakarta')::date+14 ELSE NULL END,v.data::jsonb,u.id
FROM companies c JOIN users u ON u.company_id=c.id AND u.email='admin@demo.hris'
JOIN employees e ON e.company_id=c.id AND e.email='employee@demo.hris'
CROSS JOIN (VALUES
('announcements','Selamat datang di workspace People','Sekarang pengajuan cuti, absensi, target kerja, dan slip gaji tersedia dalam satu workspace. Silakan lengkapi kontak darurat pada halaman Profil saya.','published','{}'),
('announcements','Panduan penggunaan data demo','Seluruh nama, angka gaji, dan catatan di workspace demo merupakan contoh fiktif untuk mencoba fitur aplikasi.','published','{}'),
('onboarding','Lengkapi profil dan kontak darurat','Buka Profil saya dan tambahkan nomor telepon serta kontak yang bisa dihubungi saat darurat.','todo','{}'),
('onboarding','Orientasi bersama tim People','Jadwalkan perkenalan dengan tim dan pelajari alur cuti serta koreksi kehadiran.','in_progress','{}'),
('assets','Laptop kerja · Demo','Perangkat contoh untuk menguji peminjaman dan pengembalian inventaris.','assigned','{"code":"DEMO-LAP-001","category":"Laptop","serial":"DEMO-2026-001"}'),
('goals','Selesaikan orientasi HRIS','Kenali workflow aplikasi dan sampaikan masukan kepada HR.','active','{"target":"Menyelesaikan dua tugas onboarding dan memperbarui profil","progress":25}'),
('recruitment','Kandidat Demo','Kandidat fiktif untuk mencoba perpindahan tahap rekrutmen.','screening','{"email":"candidate@example.test","position":"People Operations Specialist"}')
) AS v(module,title,description,status,data)
WHERE c.slug='demo' AND NOT EXISTS(SELECT 1 FROM hr_items h WHERE h.company_id=c.id AND h.module=v.module AND h.title=v.title);
