ALTER TABLE users ADD CONSTRAINT employee_user_identity UNIQUE(company_id,employee_id);
ALTER TABLE users ADD CONSTRAINT employee_user_requires_employee CHECK(role != 'employee' OR employee_id IS NOT NULL);
