CREATE TABLE companies (id BIGSERIAL PRIMARY KEY, name TEXT NOT NULL, slug TEXT NOT NULL UNIQUE);
CREATE TABLE departments (id BIGSERIAL PRIMARY KEY, company_id BIGINT NOT NULL REFERENCES companies(id), name TEXT NOT NULL, UNIQUE(company_id,name), UNIQUE(company_id,id));
CREATE TABLE employees (
 id BIGSERIAL PRIMARY KEY, company_id BIGINT NOT NULL REFERENCES companies(id), code TEXT NOT NULL,
 name TEXT NOT NULL, email TEXT NOT NULL, department_id BIGINT NOT NULL, position TEXT NOT NULL,
 status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','inactive')), joined_on DATE NOT NULL,
 UNIQUE(company_id,code), UNIQUE(company_id,email), UNIQUE(company_id,id),
 FOREIGN KEY(company_id,department_id) REFERENCES departments(company_id,id)
);
CREATE TABLE users (
 id BIGSERIAL PRIMARY KEY, company_id BIGINT NOT NULL REFERENCES companies(id), employee_id BIGINT,
 name TEXT NOT NULL, email TEXT NOT NULL, password_hash TEXT NOT NULL,
 role TEXT NOT NULL CHECK(role IN ('admin','employee')), UNIQUE(company_id,email), UNIQUE(company_id,id),
 FOREIGN KEY(company_id,employee_id) REFERENCES employees(company_id,id)
);
CREATE TABLE user_sessions (token_hash TEXT PRIMARY KEY, user_id BIGINT NOT NULL REFERENCES users(id), expires_at TIMESTAMPTZ NOT NULL);
CREATE INDEX sessions_expiry ON user_sessions(expires_at);
CREATE TABLE attendances (
 id BIGSERIAL PRIMARY KEY, company_id BIGINT NOT NULL, employee_id BIGINT NOT NULL, date DATE NOT NULL,
 check_in TIMESTAMPTZ NOT NULL DEFAULT now(), check_out TIMESTAMPTZ,
 UNIQUE(company_id,employee_id,date), FOREIGN KEY(company_id,employee_id) REFERENCES employees(company_id,id),
 CHECK(check_out IS NULL OR check_out >= check_in)
);
CREATE TABLE leave_requests (
 id BIGSERIAL PRIMARY KEY, company_id BIGINT NOT NULL, employee_id BIGINT NOT NULL,
 kind TEXT NOT NULL CHECK(kind IN ('annual','sick','personal')), start_date DATE NOT NULL, end_date DATE NOT NULL,
 reason TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','approved','rejected')),
 reviewed_by BIGINT, reviewed_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 CHECK(end_date >= start_date), FOREIGN KEY(company_id,employee_id) REFERENCES employees(company_id,id),
 FOREIGN KEY(company_id,reviewed_by) REFERENCES users(company_id,id)
);
CREATE INDEX leave_company_status ON leave_requests(company_id,status);
