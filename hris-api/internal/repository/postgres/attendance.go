package postgres

import (
	"context"
	"database/sql"

	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type attendanceRepository struct{ db *sqlx.DB }

func NewAttendanceRepository(db *sqlx.DB) repository.AttendanceRepository {
	return &attendanceRepository{db: db}
}

const attendanceFields = `a.id, a.employee_id, e.name, a.date::text, a.check_in, a.check_out,
	a.shift_name, a.scheduled_start, a.scheduled_end, a.late_minutes, a.early_leave_minutes,
	a.check_in_distance_m, a.check_out_distance_m`

func (r *attendanceRepository) Attendances(ctx context.Context, companyID int64, employeeID *int64, f model.ListFilter) ([]model.Attendance, int, error) {
	const where = `
		FROM attendances a
		JOIN employees e ON e.id = a.employee_id
		WHERE a.company_id = $1 AND ($2::bigint IS NULL OR a.employee_id = $2)
		  AND ($3 = '' OR a.date = NULLIF($3, '')::date)`
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT count(*) `+where, companyID, employeeID, f.Date); err != nil {
		return nil, 0, err
	}
	v := []model.Attendance{}
	err := r.db.SelectContext(ctx, &v, `
		SELECT `+attendanceFields+where+`
		ORDER BY a.date DESC, a.check_in DESC
		LIMIT $4 OFFSET $5`,
		companyID, employeeID, f.Date, limitOf(f), f.Offset)
	return v, total, err
}

// ScheduleSource loads calendars, shifts, assignments, and locations for the
// given employees (nil = every employee) between from and to. With
// withRecords it also loads attendances and approved leave days.
func (r *attendanceRepository) ScheduleSource(ctx context.Context, companyID int64, employeeID *int64, from, to string, withRecords bool) (model.ScheduleSource, error) {
	var src model.ScheduleSource
	var err error
	if src.Calendar, err = workCalendar(ctx, r.db, companyID); err != nil {
		return src, err
	}
	if err = r.db.SelectContext(ctx, &src.Holidays, `
		SELECT date::text FROM holidays WHERE company_id = $1 AND date BETWEEN $2 AND $3`,
		companyID, from, to); err != nil {
		return src, err
	}
	if err = r.db.SelectContext(ctx, &src.Shifts, `
		SELECT id, name, start_time, end_time, grace_minutes, active FROM shifts WHERE company_id = $1 ORDER BY name`,
		companyID); err != nil {
		return src, err
	}
	if err = r.db.SelectContext(ctx, &src.Employees, `
		SELECT id, name, shift_id, joined_on::text joined_on, left_on::text left_on
		FROM employees
		WHERE company_id = $1 AND ($2::bigint IS NULL OR id = $2)
		  AND joined_on <= $4::date AND (status = 'active' OR left_on >= $3::date)
		ORDER BY name, id`,
		companyID, employeeID, from, to); err != nil {
		return src, err
	}
	if err = r.db.SelectContext(ctx, &src.Assignments, `
		SELECT employee_id, date::text date, shift_id FROM shift_assignments
		WHERE company_id = $1 AND ($2::bigint IS NULL OR employee_id = $2) AND date BETWEEN $3 AND $4`,
		companyID, employeeID, from, to); err != nil {
		return src, err
	}
	if err = r.db.SelectContext(ctx, &src.Locations, `
		SELECT id, name, latitude, longitude, radius_m FROM attendance_locations WHERE company_id = $1 ORDER BY name`,
		companyID); err != nil {
		return src, err
	}
	if !withRecords {
		return src, nil
	}
	if err = r.db.SelectContext(ctx, &src.Attendances, `
		SELECT `+attendanceFields+`
		FROM attendances a JOIN employees e ON e.id = a.employee_id
		WHERE a.company_id = $1 AND ($2::bigint IS NULL OR a.employee_id = $2) AND a.date BETWEEN $3 AND $4`,
		companyID, employeeID, from, to); err != nil {
		return src, err
	}
	err = r.db.SelectContext(ctx, &src.LeaveDays, `
		SELECT l.employee_id, d.date::text date
		FROM leave_days d JOIN leave_requests l ON l.company_id = d.company_id AND l.id = d.leave_id
		WHERE l.company_id = $1 AND ($2::bigint IS NULL OR l.employee_id = $2) AND l.status = 'approved'
		  AND d.date BETWEEN $3 AND $4`,
		companyID, employeeID, from, to)
	return src, err
}

// Clock records a check-in with its schedule snapshot, or checks out the
// latest open attendance started within 20 hours, so an overnight shift can
// end after midnight. One check-in per date is enforced by a unique key.
func (r *attendanceRepository) Clock(ctx context.Context, companyID, employeeID int64, v model.ClockInput) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if err = lockEmployee(ctx, tx, companyID, employeeID); err != nil {
		return err
	}
	if !v.Out {
		var open bool
		if err = tx.GetContext(ctx, &open, `
			SELECT EXISTS (SELECT 1 FROM attendances WHERE company_id = $1 AND employee_id = $2
			               AND check_out IS NULL AND check_in > $3::timestamptz - interval '20 hours')`,
			companyID, employeeID, v.At); err != nil {
			return err
		}
		if open {
			return apperror.Conflict("Masih ada absensi yang belum check-out")
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO attendances (company_id, employee_id, date, check_in, shift_name, scheduled_start,
			                         scheduled_end, grace_minutes, late_minutes, check_in_latitude,
			                         check_in_longitude, check_in_distance_m)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			companyID, employeeID, v.Date, v.At, v.ShiftName, v.ScheduledStart, v.ScheduledEnd,
			v.GraceMinutes, v.LateMinutes, v.Latitude, v.Longitude, v.DistanceM)
		if err != nil {
			return err
		}
		return tx.Commit()
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE attendances
		SET check_out = $3,
		    early_leave_minutes = CASE WHEN scheduled_end IS NULL THEN 0
		        ELSE GREATEST(0, floor(extract(epoch FROM scheduled_end - $3::timestamptz) / 60))::int END,
		    check_out_latitude = $4, check_out_longitude = $5, check_out_distance_m = $6
		WHERE id = (SELECT id FROM attendances
		            WHERE company_id = $1 AND employee_id = $2 AND check_out IS NULL
		              AND check_in > $3::timestamptz - interval '20 hours'
		            ORDER BY check_in DESC LIMIT 1)`,
		companyID, employeeID, v.At, v.Latitude, v.Longitude, v.DistanceM)
	if err != nil {
		return err
	}
	if rowsAffected(res) == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

func (r *attendanceRepository) Shifts(ctx context.Context, companyID int64) ([]model.Shift, error) {
	v := []model.Shift{}
	err := r.db.SelectContext(ctx, &v, `
		SELECT id, name, start_time, end_time, grace_minutes, active FROM shifts WHERE company_id = $1 ORDER BY name`,
		companyID)
	return v, err
}

func (r *attendanceRepository) SaveShift(ctx context.Context, companyID, actorID, id int64, v model.Shift) (int64, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer rollback(tx)
	if id == 0 {
		err = tx.GetContext(ctx, &id, `
			INSERT INTO shifts (company_id, name, start_time, end_time, grace_minutes, active)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
			companyID, v.Name, v.StartTime, v.EndTime, v.GraceMinutes, v.Active)
	} else {
		var res sql.Result
		res, err = tx.ExecContext(ctx, `
			UPDATE shifts SET name = $3, start_time = $4, end_time = $5, grace_minutes = $6, active = $7
			WHERE company_id = $1 AND id = $2`,
			companyID, id, v.Name, v.StartTime, v.EndTime, v.GraceMinutes, v.Active)
		if err == nil && rowsAffected(res) == 0 {
			err = sql.ErrNoRows
		}
	}
	if err != nil {
		return 0, err
	}
	if err = audit(ctx, tx, companyID, actorID, "save", "shift", id, v.Name); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

// SetAssignments replaces the assignments of the employees in [from, to]:
// with clear the range falls back to default schedules, otherwise every date
// gets shiftID (nil = day off).
func (r *attendanceRepository) SetAssignments(ctx context.Context, companyID, actorID int64, employeeIDs []int64, from, to string, shiftID *int64, clear bool) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	var found int
	if err = tx.GetContext(ctx, &found, `
		SELECT count(*) FROM employees WHERE company_id = $1 AND id = ANY($2)`,
		companyID, pq.Array(employeeIDs)); err != nil {
		return err
	}
	if found != len(employeeIDs) {
		return sql.ErrNoRows
	}
	if shiftID != nil {
		var active bool
		if err = tx.GetContext(ctx, &active, `SELECT active FROM shifts WHERE company_id = $1 AND id = $2`,
			companyID, *shiftID); err != nil {
			return err
		}
		if !active {
			return apperror.Invalid("Shift sudah nonaktif")
		}
	}
	if _, err = tx.ExecContext(ctx, `
		DELETE FROM shift_assignments WHERE company_id = $1 AND employee_id = ANY($2) AND date BETWEEN $3 AND $4`,
		companyID, pq.Array(employeeIDs), from, to); err != nil {
		return err
	}
	if !clear {
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO shift_assignments (company_id, employee_id, date, shift_id)
			SELECT $1, e, d::date, $5::bigint FROM unnest($2::bigint[]) e
			CROSS JOIN generate_series($3::date, $4::date, interval '1 day') d`,
			companyID, pq.Array(employeeIDs), from, to, shiftID); err != nil {
			return err
		}
	}
	if err = audit(ctx, tx, companyID, actorID, "assign", "shift_schedule", companyID,
		"Jadwal shift "+from+" s.d. "+to); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *attendanceRepository) Locations(ctx context.Context, companyID int64) ([]model.AttendanceLocation, error) {
	v := []model.AttendanceLocation{}
	err := r.db.SelectContext(ctx, &v, `
		SELECT id, name, latitude, longitude, radius_m FROM attendance_locations WHERE company_id = $1 ORDER BY name`,
		companyID)
	return v, err
}

func (r *attendanceRepository) SaveLocation(ctx context.Context, companyID, actorID, id int64, v model.AttendanceLocation) (int64, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer rollback(tx)
	if id == 0 {
		err = tx.GetContext(ctx, &id, `
			INSERT INTO attendance_locations (company_id, name, latitude, longitude, radius_m)
			VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			companyID, v.Name, v.Latitude, v.Longitude, v.RadiusM)
	} else {
		var res sql.Result
		res, err = tx.ExecContext(ctx, `
			UPDATE attendance_locations SET name = $3, latitude = $4, longitude = $5, radius_m = $6
			WHERE company_id = $1 AND id = $2`,
			companyID, id, v.Name, v.Latitude, v.Longitude, v.RadiusM)
		if err == nil && rowsAffected(res) == 0 {
			err = sql.ErrNoRows
		}
	}
	if err != nil {
		return 0, err
	}
	if err = audit(ctx, tx, companyID, actorID, "save", "attendance_location", id, v.Name); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (r *attendanceRepository) DeleteLocation(ctx context.Context, companyID, actorID, id int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	var name string
	if err = tx.GetContext(ctx, &name, `
		DELETE FROM attendance_locations WHERE company_id = $1 AND id = $2 RETURNING name`,
		companyID, id); err != nil {
		return err
	}
	if err = audit(ctx, tx, companyID, actorID, "delete", "attendance_location", id, name); err != nil {
		return err
	}
	return tx.Commit()
}
