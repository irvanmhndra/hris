-- Attendance-based deductions, carry-over expiry, tax/BPJS identifiers for
-- reports, and correction runs for finalized payroll periods.

ALTER TABLE payroll_settings
 ADD COLUMN late_deduction TEXT NOT NULL DEFAULT 'none' CHECK(late_deduction IN ('none','per_minute','per_occurrence')),
 ADD COLUMN late_deduction_amount BIGINT NOT NULL DEFAULT 0 CHECK(late_deduction_amount BETWEEN 0 AND 1000000000),
 ADD COLUMN deduct_absence BOOLEAN NOT NULL DEFAULT false;

-- Carried-over leave can only be taken up to the end of this month of the
-- next year (0 = never expires).
ALTER TABLE work_calendars ADD COLUMN carry_over_expiry_months INTEGER NOT NULL DEFAULT 0 CHECK(carry_over_expiry_months BETWEEN 0 AND 12);

ALTER TABLE salary_profiles
 ADD COLUMN nik TEXT NOT NULL DEFAULT '' CHECK(nik = '' OR nik ~ '^[0-9]{16}$'),
 ADD COLUMN npwp TEXT NOT NULL DEFAULT '' CHECK(npwp = '' OR npwp ~ '^[0-9]{15,16}$'),
 ADD COLUMN bpjs_kesehatan_number TEXT NOT NULL DEFAULT '' CHECK(bpjs_kesehatan_number = '' OR bpjs_kesehatan_number ~ '^[0-9]{8,20}$'),
 ADD COLUMN bpjs_ketenagakerjaan_number TEXT NOT NULL DEFAULT '' CHECK(bpjs_ketenagakerjaan_number = '' OR bpjs_ketenagakerjaan_number ~ '^[0-9]{8,20}$');

ALTER TABLE payroll_entries
 ADD COLUMN nik TEXT NOT NULL DEFAULT '',
 ADD COLUMN npwp TEXT NOT NULL DEFAULT '',
 ADD COLUMN late_count INTEGER NOT NULL DEFAULT 0 CHECK(late_count >= 0),
 ADD COLUMN late_minutes INTEGER NOT NULL DEFAULT 0 CHECK(late_minutes >= 0),
 ADD COLUMN absent_days INTEGER NOT NULL DEFAULT 0 CHECK(absent_days >= 0);

-- A correction run adds adjustments (and the PPh 21 difference) to a
-- finalized period without reopening it.
ALTER TABLE payroll_runs
 ADD COLUMN kind TEXT NOT NULL DEFAULT 'regular' CHECK(kind IN ('regular','correction')),
 ADD COLUMN corrects_run_id BIGINT,
 ADD CONSTRAINT payroll_runs_corrects_fk FOREIGN KEY(company_id,corrects_run_id) REFERENCES payroll_runs(company_id,id),
 ADD CONSTRAINT payroll_runs_correction_target CHECK((kind = 'correction') = (corrects_run_id IS NOT NULL));
DROP INDEX payroll_active_period;
CREATE UNIQUE INDEX payroll_active_period ON payroll_runs(company_id,period) WHERE status <> 'void' AND kind = 'regular';
-- One open correction draft per period at a time.
CREATE UNIQUE INDEX payroll_open_correction ON payroll_runs(company_id,period) WHERE status = 'draft' AND kind = 'correction';
