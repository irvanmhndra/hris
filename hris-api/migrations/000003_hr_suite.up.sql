CREATE TABLE work_calendars (
 company_id BIGINT PRIMARY KEY REFERENCES companies(id),
 workdays INTEGER[] NOT NULL DEFAULT ARRAY[1,2,3,4,5],
 annual_allowance INTEGER NOT NULL DEFAULT 12 CHECK(annual_allowance BETWEEN 0 AND 366),
 start_time TEXT NOT NULL DEFAULT '09:00', end_time TEXT NOT NULL DEFAULT '18:00',
 CHECK(cardinality(workdays)>0 AND workdays <@ ARRAY[0,1,2,3,4,5,6])
);
INSERT INTO work_calendars(company_id) SELECT id FROM companies;
CREATE TABLE holidays (id BIGSERIAL PRIMARY KEY, company_id BIGINT NOT NULL REFERENCES companies(id), date DATE NOT NULL, name TEXT NOT NULL, UNIQUE(company_id,date));
CREATE TABLE leave_allocations (company_id BIGINT NOT NULL, employee_id BIGINT NOT NULL, year INTEGER NOT NULL CHECK(year BETWEEN 2000 AND 2200), allowance INTEGER NOT NULL CHECK(allowance BETWEEN 0 AND 366), PRIMARY KEY(company_id,employee_id,year), FOREIGN KEY(company_id,employee_id) REFERENCES employees(company_id,id));
ALTER TABLE leave_requests DROP CONSTRAINT leave_requests_status_check;
ALTER TABLE leave_requests ADD CONSTRAINT leave_requests_status_check CHECK(status IN ('pending','approved','rejected','cancelled'));
ALTER TABLE leave_requests ADD CONSTRAINT leave_tenant_identity UNIQUE(company_id,id);
ALTER TABLE leave_requests ADD COLUMN calculation TEXT NOT NULL DEFAULT 'working_days';
UPDATE leave_requests SET calculation='legacy_calendar_days';
CREATE TABLE leave_days (company_id BIGINT NOT NULL, leave_id BIGINT NOT NULL, date DATE NOT NULL, PRIMARY KEY(company_id,leave_id,date), FOREIGN KEY(company_id,leave_id) REFERENCES leave_requests(company_id,id));
-- Preserve the charges displayed before this migration. New requests snapshot working days.
INSERT INTO leave_days(company_id,leave_id,date) SELECT company_id,id,d::date FROM leave_requests CROSS JOIN LATERAL generate_series(start_date,end_date,interval '1 day') d;
CREATE TABLE employee_profiles (
 company_id BIGINT NOT NULL,employee_id BIGINT NOT NULL,phone TEXT NOT NULL DEFAULT '',address TEXT NOT NULL DEFAULT '',
 emergency_name TEXT NOT NULL DEFAULT '',emergency_phone TEXT NOT NULL DEFAULT '',emergency_relation TEXT NOT NULL DEFAULT '',
 PRIMARY KEY(company_id,employee_id), FOREIGN KEY(company_id,employee_id) REFERENCES employees(company_id,id)
);
CREATE TABLE hr_items (
 id BIGSERIAL PRIMARY KEY, company_id BIGINT NOT NULL REFERENCES companies(id),
 module TEXT NOT NULL CHECK(module IN ('overtime','corrections','announcements','documents','onboarding','assets','goals','recruitment')),
 employee_id BIGINT, title TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', status TEXT NOT NULL,
 due_date DATE, data JSONB NOT NULL DEFAULT '{}', version INTEGER NOT NULL DEFAULT 1,
 created_by BIGINT NOT NULL, reviewed_by BIGINT, review_note TEXT NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 FOREIGN KEY(company_id,employee_id) REFERENCES employees(company_id,id),
 FOREIGN KEY(company_id,created_by) REFERENCES users(company_id,id),FOREIGN KEY(company_id,reviewed_by) REFERENCES users(company_id,id),
 CHECK(module NOT IN ('overtime','corrections','onboarding','goals') OR employee_id IS NOT NULL),
 CHECK((module IN ('overtime','corrections') AND status IN ('pending','approved','rejected','cancelled')) OR
 (module IN ('announcements','documents') AND status IN ('draft','published','archived')) OR
 (module='onboarding' AND status IN ('todo','in_progress','done')) OR
 (module='assets' AND status IN ('available','assigned','maintenance','retired')) OR
 (module='goals' AND status IN ('active','done')) OR
 (module='recruitment' AND status IN ('applied','screening','interview','offer','hired','rejected')))
);
CREATE UNIQUE INDEX asset_code_unique ON hr_items(company_id,(data->>'code')) WHERE module='assets';
CREATE INDEX hr_items_scope ON hr_items(company_id,module,employee_id,status);
CREATE TABLE audit_logs (id BIGSERIAL PRIMARY KEY,company_id BIGINT NOT NULL REFERENCES companies(id),actor_id BIGINT NOT NULL,action TEXT NOT NULL,resource TEXT NOT NULL,resource_id BIGINT NOT NULL,summary TEXT NOT NULL,created_at TIMESTAMPTZ NOT NULL DEFAULT now(),FOREIGN KEY(company_id,actor_id) REFERENCES users(company_id,id));
CREATE INDEX audit_scope ON audit_logs(company_id,created_at DESC);
