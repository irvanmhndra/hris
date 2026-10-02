-- Automatic Indonesian payroll: PPh 21 (TER), BPJS, overtime, THR, prorata,
-- and itemised salary components and payslip lines.

ALTER TABLE employees ADD COLUMN left_on DATE;
ALTER TABLE employees ADD CONSTRAINT employees_left_after_join CHECK(left_on IS NULL OR left_on >= joined_on);
ALTER TABLE hr_items ADD CONSTRAINT hr_items_tenant_identity UNIQUE(company_id,id);

CREATE TABLE payroll_settings (
 company_id BIGINT PRIMARY KEY REFERENCES companies(id),
 jkk_rate INTEGER NOT NULL DEFAULT 24 CHECK(jkk_rate IN (24,54,89,127,174)),
 jp_wage_cap BIGINT NOT NULL DEFAULT 10547400 CHECK(jp_wage_cap BETWEEN 0 AND 1000000000000),
 kes_wage_cap BIGINT NOT NULL DEFAULT 12000000 CHECK(kes_wage_cap BETWEEN 0 AND 1000000000000)
);

-- Tax and BPJS settings per employee. Profiles that already exist keep their
-- manual behaviour (HR entered tax/BPJS as deductions), so nothing is
-- counted twice; new profiles default to automatic calculation.
ALTER TABLE salary_profiles
 ADD COLUMN ptkp_status TEXT NOT NULL DEFAULT 'TK/0' CHECK(ptkp_status IN ('TK/0','TK/1','TK/2','TK/3','K/0','K/1','K/2','K/3')),
 ADD COLUMN tax_method TEXT NOT NULL DEFAULT 'gross' CHECK(tax_method IN ('gross','gross_up','none')),
 ADD COLUMN bpjs_kesehatan BOOLEAN NOT NULL DEFAULT true,
 ADD COLUMN bpjs_ketenagakerjaan BOOLEAN NOT NULL DEFAULT true,
 ADD COLUMN bpjs_pensiun BOOLEAN NOT NULL DEFAULT true,
 ADD COLUMN overtime_eligible BOOLEAN NOT NULL DEFAULT true;
UPDATE salary_profiles SET tax_method='none', bpjs_kesehatan=false, bpjs_ketenagakerjaan=false, bpjs_pensiun=false;

CREATE TABLE salary_components (
 id BIGSERIAL PRIMARY KEY, company_id BIGINT NOT NULL, employee_id BIGINT NOT NULL,
 kind TEXT NOT NULL CHECK(kind IN ('allowance','deduction')),
 name TEXT NOT NULL CHECK(length(name) BETWEEN 1 AND 80),
 amount BIGINT NOT NULL CHECK(amount BETWEEN 0 AND 1000000000000),
 fixed BOOLEAN NOT NULL DEFAULT false, taxable BOOLEAN NOT NULL DEFAULT true, position INTEGER NOT NULL DEFAULT 0,
 FOREIGN KEY(company_id,employee_id) REFERENCES salary_profiles(company_id,employee_id) ON DELETE CASCADE,
 CHECK(kind='allowance' OR (NOT fixed AND taxable))
);
CREATE INDEX salary_components_owner ON salary_components(company_id,employee_id,position);
INSERT INTO salary_components(company_id,employee_id,kind,name,amount,fixed,taxable,position)
 SELECT company_id,employee_id,'allowance','Tunjangan',allowance,true,true,0 FROM salary_profiles WHERE allowance>0;
INSERT INTO salary_components(company_id,employee_id,kind,name,amount,fixed,taxable,position)
 SELECT company_id,employee_id,'deduction','Potongan',deduction,false,true,1 FROM salary_profiles WHERE deduction>0;
ALTER TABLE salary_profiles DROP COLUMN allowance, DROP COLUMN deduction;

-- Runs snapshot the company settings they were calculated with.
ALTER TABLE payroll_runs
 ADD COLUMN jkk_rate INTEGER NOT NULL DEFAULT 24,
 ADD COLUMN jp_wage_cap BIGINT NOT NULL DEFAULT 10547400,
 ADD COLUMN kes_wage_cap BIGINT NOT NULL DEFAULT 12000000,
 ADD COLUMN thr_date DATE;

-- Entries keep their aggregate columns (basic, other earnings, deductions) as
-- totals of their lines, plus the inputs needed to recalculate a draft.
-- Existing slips become tax_method 'none' with no BPJS: amounts unchanged.
ALTER TABLE payroll_entries
 ADD CONSTRAINT payroll_entries_tenant_identity UNIQUE(company_id,id),
 ADD COLUMN ptkp_status TEXT NOT NULL DEFAULT 'TK/0',
 ADD COLUMN tax_method TEXT NOT NULL DEFAULT 'none',
 ADD COLUMN bpjs_kesehatan BOOLEAN NOT NULL DEFAULT false,
 ADD COLUMN bpjs_ketenagakerjaan BOOLEAN NOT NULL DEFAULT false,
 ADD COLUMN bpjs_pensiun BOOLEAN NOT NULL DEFAULT false,
 ADD COLUMN final_period BOOLEAN NOT NULL DEFAULT false,
 ADD COLUMN worked_days INTEGER NOT NULL DEFAULT 0,
 ADD COLUMN period_days INTEGER NOT NULL DEFAULT 0,
 ADD COLUMN taxable_gross BIGINT NOT NULL DEFAULT 0,
 ADD COLUMN pension_deduction BIGINT NOT NULL DEFAULT 0,
 ADD COLUMN pph21 BIGINT NOT NULL DEFAULT 0,
 ADD COLUMN employer_cost BIGINT NOT NULL DEFAULT 0 CHECK(employer_cost >= 0);

CREATE TABLE payroll_lines (
 id BIGSERIAL PRIMARY KEY, company_id BIGINT NOT NULL, entry_id BIGINT NOT NULL,
 kind TEXT NOT NULL CHECK(kind IN ('earning','deduction','employer')),
 code TEXT NOT NULL, name TEXT NOT NULL CHECK(length(name) BETWEEN 1 AND 80),
 amount BIGINT NOT NULL CHECK(amount BETWEEN 0 AND 1000000000000),
 taxable BOOLEAN NOT NULL DEFAULT false, fixed BOOLEAN NOT NULL DEFAULT false, position INTEGER NOT NULL,
 FOREIGN KEY(company_id,entry_id) REFERENCES payroll_entries(company_id,id) ON DELETE CASCADE
);
CREATE INDEX payroll_lines_entry ON payroll_lines(company_id,entry_id,position);
INSERT INTO payroll_lines(company_id,entry_id,kind,code,name,amount,taxable,fixed,position)
 SELECT company_id,id,'earning','BASIC','Gaji pokok',basic_salary,true,true,0 FROM payroll_entries;
INSERT INTO payroll_lines(company_id,entry_id,kind,code,name,amount,taxable,fixed,position)
 SELECT company_id,id,'earning','ALLOWANCE','Tunjangan',allowance,true,true,1 FROM payroll_entries WHERE allowance>0;
INSERT INTO payroll_lines(company_id,entry_id,kind,code,name,amount,taxable,fixed,position)
 SELECT company_id,id,'deduction','DEDUCTION','Potongan',deduction,false,false,2 FROM payroll_entries WHERE deduction>0;

-- Approved overtime is paid once: the link is removed if its run is voided.
CREATE TABLE payroll_overtime (
 company_id BIGINT NOT NULL, hr_item_id BIGINT NOT NULL, entry_id BIGINT NOT NULL,
 PRIMARY KEY(company_id,hr_item_id),
 FOREIGN KEY(company_id,hr_item_id) REFERENCES hr_items(company_id,id),
 FOREIGN KEY(company_id,entry_id) REFERENCES payroll_entries(company_id,id) ON DELETE CASCADE
);
