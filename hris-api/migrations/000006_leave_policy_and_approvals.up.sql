-- Unpaid leave, leave accrual/carry-over policy, and two-step approval
-- (direct manager, then HR) for leave, overtime, and attendance corrections.

ALTER TABLE leave_requests DROP CONSTRAINT leave_requests_kind_check;
ALTER TABLE leave_requests ADD CONSTRAINT leave_requests_kind_check CHECK(kind IN ('annual','sick','personal','unpaid'));

ALTER TABLE work_calendars
 ADD COLUMN leave_accrual TEXT NOT NULL DEFAULT 'annual' CHECK(leave_accrual IN ('annual','monthly')),
 ADD COLUMN carry_over_max INTEGER NOT NULL DEFAULT 0 CHECK(carry_over_max BETWEEN 0 AND 366),
 ADD COLUMN leave_eligibility_months INTEGER NOT NULL DEFAULT 0 CHECK(leave_eligibility_months BETWEEN 0 AND 24);

ALTER TABLE employees
 ADD COLUMN manager_id BIGINT,
 ADD CONSTRAINT employees_manager_fk FOREIGN KEY(company_id,manager_id) REFERENCES employees(company_id,id),
 ADD CONSTRAINT employees_manager_not_self CHECK(manager_id IS NULL OR manager_id <> id);
CREATE INDEX employees_manager ON employees(company_id,manager_id) WHERE manager_id IS NOT NULL;

-- stage: who must act next on a pending request. Existing requests go to HR.
ALTER TABLE leave_requests
 ADD COLUMN stage TEXT NOT NULL DEFAULT 'hr' CHECK(stage IN ('manager','hr')),
 ADD COLUMN manager_reviewed_by BIGINT,
 ADD COLUMN manager_reviewed_at TIMESTAMPTZ,
 ADD COLUMN review_note TEXT NOT NULL DEFAULT '',
 ADD CONSTRAINT leave_manager_reviewer_fk FOREIGN KEY(company_id,manager_reviewed_by) REFERENCES users(company_id,id);
ALTER TABLE hr_items
 ADD COLUMN stage TEXT NOT NULL DEFAULT 'hr' CHECK(stage IN ('manager','hr')),
 ADD COLUMN manager_reviewed_by BIGINT,
 ADD COLUMN manager_reviewed_at TIMESTAMPTZ,
 ADD CONSTRAINT hr_items_manager_reviewer_fk FOREIGN KEY(company_id,manager_reviewed_by) REFERENCES users(company_id,id);

-- Approved unpaid leave reduces the days paid in a payroll period.
ALTER TABLE payroll_entries ADD COLUMN unpaid_leave_days INTEGER NOT NULL DEFAULT 0 CHECK(unpaid_leave_days >= 0);
