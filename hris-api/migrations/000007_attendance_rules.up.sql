-- Shifts (including overnight), per-date schedules, lateness/early-leave
-- tracking, and GPS geofenced check-in.

CREATE TABLE shifts (
 id BIGSERIAL PRIMARY KEY, company_id BIGINT NOT NULL REFERENCES companies(id),
 name TEXT NOT NULL CHECK(length(name) BETWEEN 1 AND 80),
 start_time TEXT NOT NULL CHECK(start_time ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
 end_time TEXT NOT NULL CHECK(end_time ~ '^([01][0-9]|2[0-3]):[0-5][0-9]$'),
 grace_minutes INTEGER NOT NULL DEFAULT 0 CHECK(grace_minutes BETWEEN 0 AND 240),
 active BOOLEAN NOT NULL DEFAULT true,
 UNIQUE(company_id,id), UNIQUE(company_id,name)
);
ALTER TABLE employees
 ADD COLUMN shift_id BIGINT,
 ADD CONSTRAINT employees_shift_fk FOREIGN KEY(company_id,shift_id) REFERENCES shifts(company_id,id);

-- A row pins one date: a shift, or a day off when shift_id is NULL.
CREATE TABLE shift_assignments (
 company_id BIGINT NOT NULL, employee_id BIGINT NOT NULL, date DATE NOT NULL, shift_id BIGINT,
 PRIMARY KEY(company_id,employee_id,date),
 FOREIGN KEY(company_id,employee_id) REFERENCES employees(company_id,id),
 FOREIGN KEY(company_id,shift_id) REFERENCES shifts(company_id,id)
);

CREATE TABLE attendance_locations (
 id BIGSERIAL PRIMARY KEY, company_id BIGINT NOT NULL REFERENCES companies(id),
 name TEXT NOT NULL CHECK(length(name) BETWEEN 1 AND 80),
 latitude DOUBLE PRECISION NOT NULL CHECK(latitude BETWEEN -90 AND 90),
 longitude DOUBLE PRECISION NOT NULL CHECK(longitude BETWEEN -180 AND 180),
 radius_m INTEGER NOT NULL CHECK(radius_m BETWEEN 10 AND 5000),
 UNIQUE(company_id,id)
);
ALTER TABLE work_calendars ADD COLUMN require_location BOOLEAN NOT NULL DEFAULT false;

-- Each attendance keeps the schedule it was measured against.
ALTER TABLE attendances
 ADD COLUMN shift_name TEXT NOT NULL DEFAULT '',
 ADD COLUMN scheduled_start TIMESTAMPTZ,
 ADD COLUMN scheduled_end TIMESTAMPTZ,
 ADD COLUMN grace_minutes INTEGER NOT NULL DEFAULT 0,
 ADD COLUMN late_minutes INTEGER NOT NULL DEFAULT 0 CHECK(late_minutes >= 0),
 ADD COLUMN early_leave_minutes INTEGER NOT NULL DEFAULT 0 CHECK(early_leave_minutes >= 0),
 ADD COLUMN check_in_latitude DOUBLE PRECISION, ADD COLUMN check_in_longitude DOUBLE PRECISION,
 ADD COLUMN check_in_distance_m INTEGER,
 ADD COLUMN check_out_latitude DOUBLE PRECISION, ADD COLUMN check_out_longitude DOUBLE PRECISION,
 ADD COLUMN check_out_distance_m INTEGER;
CREATE INDEX attendances_open ON attendances(company_id,employee_id) WHERE check_out IS NULL;
