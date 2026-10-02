DROP INDEX attendances_open;
ALTER TABLE attendances
 DROP COLUMN shift_name, DROP COLUMN scheduled_start, DROP COLUMN scheduled_end, DROP COLUMN grace_minutes,
 DROP COLUMN late_minutes, DROP COLUMN early_leave_minutes,
 DROP COLUMN check_in_latitude, DROP COLUMN check_in_longitude, DROP COLUMN check_in_distance_m,
 DROP COLUMN check_out_latitude, DROP COLUMN check_out_longitude, DROP COLUMN check_out_distance_m;
ALTER TABLE work_calendars DROP COLUMN require_location;
DROP TABLE attendance_locations;
DROP TABLE shift_assignments;
ALTER TABLE employees DROP CONSTRAINT employees_shift_fk, DROP COLUMN shift_id;
DROP TABLE shifts;
