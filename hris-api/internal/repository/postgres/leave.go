package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type leaveRepository struct{ db *sqlx.DB }

func NewLeaveRepository(db *sqlx.DB) repository.LeaveRepository {
	return &leaveRepository{db: db}
}

func (r *leaveRepository) WorkCalendar(ctx context.Context, companyID int64) (model.WorkCalendar, error) {
	// Defaults apply until the company saves its own calendar.
	v := model.WorkCalendar{Workdays: pq.Int64Array{1, 2, 3, 4, 5}, AnnualAllowance: 12, StartTime: "09:00", EndTime: "18:00"}
	err := r.db.GetContext(ctx, &v, `
		SELECT workdays, annual_allowance, start_time, end_time
		FROM work_calendars WHERE company_id = $1`,
		companyID)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return v, err
}

func (r *leaveRepository) SaveCalendar(ctx context.Context, companyID, actorID int64, v model.WorkCalendar) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	// Freeze entitlements already in use before the default quota changes, so
	// existing requests keep the allowance they were made under.
	_, err = tx.ExecContext(ctx, `
		INSERT INTO leave_allocations (company_id, employee_id, year, allowance)
		SELECT DISTINCT l.company_id, l.employee_id, EXTRACT(year FROM d.date)::integer,
		       COALESCE(w.annual_allowance, 12)
		FROM leave_requests l
		JOIN leave_days d ON d.company_id = l.company_id AND d.leave_id = l.id
		LEFT JOIN work_calendars w ON w.company_id = l.company_id
		WHERE l.company_id = $1 AND l.kind = 'annual'
		ON CONFLICT DO NOTHING`,
		companyID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO work_calendars (company_id, workdays, annual_allowance, start_time, end_time)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (company_id) DO UPDATE
		SET workdays = $2, annual_allowance = $3, start_time = $4, end_time = $5`,
		companyID, v.Workdays, v.AnnualAllowance, v.StartTime, v.EndTime)
	if err != nil {
		return err
	}
	if err = audit(ctx, tx, companyID, actorID, "update", "calendar", companyID,
		"Memperbarui kalender dan kuota default untuk alokasi baru"); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *leaveRepository) Holidays(ctx context.Context, companyID int64) ([]model.Holiday, error) {
	v := []model.Holiday{}
	err := r.db.SelectContext(ctx, &v, `
		SELECT id, date::text, name FROM holidays WHERE company_id = $1 ORDER BY date`,
		companyID)
	return v, err
}

func (r *leaveRepository) SaveHoliday(ctx context.Context, companyID, actorID int64, v model.Holiday) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	var id int64
	err = tx.QueryRowxContext(ctx, `
		INSERT INTO holidays (company_id, date, name) VALUES ($1, $2, $3) RETURNING id`,
		companyID, v.Date, v.Name).Scan(&id)
	if err != nil {
		return err
	}
	if err = audit(ctx, tx, companyID, actorID, "create", "holiday", id, v.Name); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *leaveRepository) DeleteHoliday(ctx context.Context, companyID, actorID, id int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	var name string
	err = tx.QueryRowxContext(ctx, `
		DELETE FROM holidays WHERE company_id = $1 AND id = $2 RETURNING name`,
		companyID, id).Scan(&name)
	if err != nil {
		return err
	}
	if err = audit(ctx, tx, companyID, actorID, "delete", "holiday", id, name); err != nil {
		return err
	}
	return tx.Commit()
}

// balancesSQL computes annual-leave balances: allowance (per-employee
// allocation, else the company default, else 12) minus approved days (used)
// minus pending days (reserved). Parameters: company, employee (nullable), year.
const balancesSQL = `
	WITH totals AS (
		SELECT e.id employee_id, e.name, $3::integer AS "year",
		       COALESCE(a.allowance, w.annual_allowance, 12) allowance,
		       (SELECT count(*)
		        FROM leave_days d
		        JOIN leave_requests l ON l.id = d.leave_id AND l.company_id = d.company_id
		        WHERE l.company_id = e.company_id AND l.employee_id = e.id AND l.kind = 'annual'
		          AND l.status = 'approved' AND EXTRACT(year FROM d.date) = $3) used,
		       (SELECT count(*)
		        FROM leave_days d
		        JOIN leave_requests l ON l.id = d.leave_id AND l.company_id = d.company_id
		        WHERE l.company_id = e.company_id AND l.employee_id = e.id AND l.kind = 'annual'
		          AND l.status = 'pending' AND EXTRACT(year FROM d.date) = $3) reserved
		FROM employees e
		LEFT JOIN work_calendars w ON w.company_id = e.company_id
		LEFT JOIN leave_allocations a ON a.company_id = e.company_id AND a.employee_id = e.id AND a.year = $3
		WHERE e.company_id = $1 AND ($2::bigint IS NULL OR e.id = $2)
	)
	SELECT *, allowance - used - reserved available FROM totals ORDER BY name`

func (r *leaveRepository) Balances(ctx context.Context, companyID int64, employeeID *int64, year int) ([]model.Balance, error) {
	v := []model.Balance{}
	err := r.db.SelectContext(ctx, &v, balancesSQL, companyID, employeeID, year)
	return v, err
}

func (r *leaveRepository) SetAllowance(ctx context.Context, companyID, actorID, employeeID int64, year, allowance int) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if err = lockEmployee(ctx, tx, companyID, employeeID); err != nil {
		return err
	}
	var b model.Balance
	if err = tx.GetContext(ctx, &b, balancesSQL, companyID, employeeID, year); err != nil {
		return err
	}
	if allowance < b.Used+b.Reserved {
		return apperror.Conflict("Kuota tidak boleh lebih kecil dari cuti terpakai dan yang sedang diajukan")
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO leave_allocations (company_id, employee_id, year, allowance)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (company_id, employee_id, year) DO UPDATE SET allowance = $4`,
		companyID, employeeID, year, allowance)
	if err != nil {
		return err
	}
	if err = audit(ctx, tx, companyID, actorID, "allocate", "leave_balance", employeeID,
		fmt.Sprintf("Kuota %d: %d hari", year, allowance)); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *leaveRepository) Leaves(ctx context.Context, companyID int64, employeeID *int64) ([]model.Leave, error) {
	v := []model.Leave{}
	err := r.db.SelectContext(ctx, &v, `
		SELECT l.id, l.employee_id, e.name, l.kind, l.start_date::text, l.end_date::text,
		       l.reason, l.status, l.calculation,
		       (SELECT count(*) FROM leave_days ld WHERE ld.company_id = l.company_id AND ld.leave_id = l.id) days
		FROM leave_requests l
		JOIN employees e ON e.id = l.employee_id
		WHERE l.company_id = $1 AND ($2::bigint IS NULL OR l.employee_id = $2)
		ORDER BY l.created_at DESC`,
		companyID, employeeID)
	return v, err
}

// CreateLeave locks the employee, rejects overlaps, snapshots the working days
// (company workdays minus holidays) into leave_days, and for annual leave
// allocates each touched year and verifies the balance — all atomically, so two
// concurrent requests cannot both spend the same allowance.
func (r *leaveRepository) CreateLeave(ctx context.Context, companyID, employeeID, actorID int64, v model.LeaveInput) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if err = lockEmployee(ctx, tx, companyID, employeeID); err != nil {
		return err
	}
	var overlap bool
	err = tx.GetContext(ctx, &overlap, `
		SELECT EXISTS (
			SELECT 1 FROM leave_requests
			WHERE company_id = $1 AND employee_id = $2 AND status IN ('pending', 'approved')
			  AND start_date <= $4::date AND end_date >= $3::date)`,
		companyID, employeeID, v.StartDate, v.EndDate)
	if err != nil {
		return err
	}
	if overlap {
		return apperror.ErrOverlap
	}
	var id int64
	err = tx.QueryRowxContext(ctx, `
		INSERT INTO leave_requests (company_id, employee_id, kind, start_date, end_date, reason)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		companyID, employeeID, v.Kind, v.StartDate, v.EndDate, v.Reason).Scan(&id)
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `
		INSERT INTO leave_days (company_id, leave_id, date)
		SELECT $1, $2, d::date
		FROM generate_series($3::date, $4::date, interval '1 day') d
		WHERE EXTRACT(dow FROM d)::int = ANY(COALESCE(
		        (SELECT workdays FROM work_calendars WHERE company_id = $1), ARRAY[1, 2, 3, 4, 5]))
		  AND NOT EXISTS (SELECT 1 FROM holidays WHERE company_id = $1 AND date = d::date)`,
		companyID, id, v.StartDate, v.EndDate)
	if err != nil {
		return err
	}
	if rowsAffected(res) == 0 {
		return apperror.Invalid("Rentang cuti tidak memiliki hari kerja")
	}
	if v.Kind == "annual" {
		var years []int
		if err = tx.SelectContext(ctx, &years, `
			SELECT DISTINCT EXTRACT(year FROM date)::int FROM leave_days
			WHERE company_id = $1 AND leave_id = $2`,
			companyID, id); err != nil {
			return err
		}
		for _, year := range years {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO leave_allocations (company_id, employee_id, year, allowance)
				VALUES ($1, $2, $3, COALESCE((SELECT annual_allowance FROM work_calendars WHERE company_id = $1), 12))
				ON CONFLICT DO NOTHING`,
				companyID, employeeID, year)
			if err != nil {
				return err
			}
			var b model.Balance
			if err = tx.GetContext(ctx, &b, balancesSQL, companyID, employeeID, year); err != nil {
				return err
			}
			if b.Available < 0 {
				return apperror.Conflict(fmt.Sprintf("Saldo cuti tahun %d tidak mencukupi", year))
			}
		}
	}
	if err = audit(ctx, tx, companyID, actorID, "create", "leave", id,
		"Mengajukan cuti "+v.StartDate+" s.d. "+v.EndDate); err != nil {
		return err
	}
	return tx.Commit()
}

// ReviewLeave decides a pending request. The conditional update on
// status = 'pending' guarantees only the first decision wins.
func (r *leaveRepository) ReviewLeave(ctx context.Context, companyID, leaveID, reviewerID int64, status string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	var employeeID int64
	if err = tx.GetContext(ctx, &employeeID, `
		SELECT employee_id FROM leave_requests WHERE company_id = $1 AND id = $2`,
		companyID, leaveID); err != nil {
		return err
	}
	if err = lockEmployee(ctx, tx, companyID, employeeID); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE leave_requests SET status = $4, reviewed_by = $3, reviewed_at = now()
		WHERE company_id = $1 AND id = $2 AND status = 'pending'`,
		companyID, leaveID, reviewerID, status)
	if err != nil {
		return err
	}
	if rowsAffected(res) == 0 {
		return sql.ErrNoRows
	}
	if status == "approved" {
		var excess bool
		err = tx.GetContext(ctx, &excess, `
			SELECT EXISTS (
				SELECT 1
				FROM leave_days d
				JOIN leave_requests l ON l.id = d.leave_id
				LEFT JOIN leave_allocations a
				       ON a.company_id = l.company_id AND a.employee_id = l.employee_id
				      AND a.year = EXTRACT(year FROM d.date)
				LEFT JOIN work_calendars w ON w.company_id = l.company_id
				WHERE l.company_id = $1 AND l.employee_id = $2 AND l.kind = 'annual' AND l.status = 'approved'
				GROUP BY EXTRACT(year FROM d.date), a.allowance, w.annual_allowance
				HAVING count(*) > COALESCE(a.allowance, w.annual_allowance, 12))`,
			companyID, employeeID)
		if err != nil {
			return err
		}
		if excess {
			return apperror.Conflict("Saldo cuti tidak mencukupi; periksa alokasi tahunan")
		}
	}
	if err = audit(ctx, tx, companyID, reviewerID, status, "leave", leaveID, "Keputusan pengajuan cuti"); err != nil {
		return err
	}
	return tx.Commit()
}

// CancelLeave lets an employee withdraw a pending request, or an approved one
// that has not started yet; the reserved/used days return to the balance.
func (r *leaveRepository) CancelLeave(ctx context.Context, companyID, employeeID, actorID, leaveID int64) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if err = lockEmployee(ctx, tx, companyID, employeeID); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE leave_requests SET status = 'cancelled'
		WHERE company_id = $1 AND employee_id = $2 AND id = $3
		  AND (status = 'pending'
		       OR (status = 'approved' AND start_date > (now() AT TIME ZONE 'Asia/Jakarta')::date))`,
		companyID, employeeID, leaveID)
	if err != nil {
		return err
	}
	if rowsAffected(res) == 0 {
		return apperror.Conflict("Cuti tidak dapat dibatalkan atau tidak ditemukan")
	}
	if err = audit(ctx, tx, companyID, actorID, "cancel", "leave", leaveID,
		"Membatalkan pengajuan dan mengembalikan saldo"); err != nil {
		return err
	}
	return tx.Commit()
}
