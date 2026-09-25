CREATE TABLE salary_profiles (
 company_id BIGINT NOT NULL,employee_id BIGINT NOT NULL,basic_salary BIGINT NOT NULL CHECK(basic_salary BETWEEN 0 AND 1000000000000),
 allowance BIGINT NOT NULL DEFAULT 0 CHECK(allowance BETWEEN 0 AND 1000000000000),deduction BIGINT NOT NULL DEFAULT 0 CHECK(deduction BETWEEN 0 AND 1000000000000),
 note TEXT NOT NULL DEFAULT '',PRIMARY KEY(company_id,employee_id),FOREIGN KEY(company_id,employee_id) REFERENCES employees(company_id,id),CHECK(deduction<=basic_salary+allowance)
);
CREATE TABLE payroll_runs (
 id BIGSERIAL PRIMARY KEY,company_id BIGINT NOT NULL REFERENCES companies(id),period DATE NOT NULL CHECK(EXTRACT(day FROM period)=1),
 status TEXT NOT NULL DEFAULT 'draft' CHECK(status IN ('draft','finalized','paid','void')),created_by BIGINT NOT NULL,
 payment_reference TEXT NOT NULL DEFAULT '',created_at TIMESTAMPTZ NOT NULL DEFAULT now(),finalized_at TIMESTAMPTZ,paid_at TIMESTAMPTZ,
 UNIQUE(company_id,id),FOREIGN KEY(company_id,created_by) REFERENCES users(company_id,id)
);
CREATE UNIQUE INDEX payroll_active_period ON payroll_runs(company_id,period) WHERE status!='void';
CREATE TABLE payroll_entries (
 id BIGSERIAL PRIMARY KEY,company_id BIGINT NOT NULL,run_id BIGINT NOT NULL,employee_id BIGINT NOT NULL,
 employee_name TEXT NOT NULL,employee_code TEXT NOT NULL,position TEXT NOT NULL,
 basic_salary BIGINT NOT NULL CHECK(basic_salary BETWEEN 0 AND 1000000000000),allowance BIGINT NOT NULL DEFAULT 0 CHECK(allowance BETWEEN 0 AND 1000000000000),
 deduction BIGINT NOT NULL DEFAULT 0 CHECK(deduction BETWEEN 0 AND 1000000000000),note TEXT NOT NULL DEFAULT '',version INTEGER NOT NULL DEFAULT 1,
 UNIQUE(company_id,run_id,employee_id),FOREIGN KEY(company_id,run_id) REFERENCES payroll_runs(company_id,id),FOREIGN KEY(company_id,employee_id) REFERENCES employees(company_id,id),CHECK(deduction<=basic_salary+allowance)
);
