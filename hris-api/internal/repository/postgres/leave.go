package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/irvanmhndra/hris-api/internal/leavepolicy"
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
	return workCalendar(ctx, r.db, companyID)
}

func workCalendar(ctx context.Context, q sqlx.QueryerContext, companyID int64) (model.WorkCalendar, error) {
	v := model.WorkCalendar{Workdays: pq.Int64Array{1, 2, 3, 4, 5}, AnnualAllowance: 12, StartTime: "09:00", EndTime: "18:00",
		LeaveAccrual: leavepolicy.Annual}
	err := sqlx.GetContext(ctx, q, &v, `
		SELECT workdays, annual_allowance, start_time, end_time, leave_accrual, carry_over_max, leave_eligibility_months,
		       require_location, carry_over_expiry_months
		FROM work_calendars WHERE company_id = $1`,
		companyID)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return v, err
}

func leavePolicy(c model.WorkCalendar) leavepolicy.Policy {
	return leavepolicy.Policy{Accrual: c.LeaveAccrual, CarryOverMax: c.CarryOverMax,
		EligibilityMonths: c.LeaveEligibilityMonths, CarryOverExpiryMonths: c.CarryOverExpiryMonths}
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
		INSERT INTO work_calendars (company_id, workdays, annual_allowance, start_time, end_time,
		                            leave_accrual, carry_over_max, leave_eligibility_months, require_location,
		                            carry_over_expiry_months)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (company_id) DO UPDATE
		SET workdays = $2, annual_allowance = $3, start_time = $4, end_time = $5,
		    leave_accrual = $6, carry_over_max = $7, leave_eligibility_months = $8, require_location = $9,
		    carry_over_expiry_months = $10`,
		companyID, v.Workdays, v.AnnualAllowance, v.StartTime, v.EndTime,
		v.LeaveAccrual, v.CarryOverMax, v.LeaveEligibilityMonths, v.RequireLocation, v.CarryOverExpiryMonths)
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

// balancesSQL loads the raw annual-leave figures of a year: allowance
// (per-employee allocation, else the company default, else 12), approved
// (used) and pending (reserved) days, and the previous year's allowance and
// usage for carry-over, and days spent by the carry-over expiry date.
// Parameters: company, employee (nullable), year, expiry date.
const balancesSQL = `
	SELECT e.id employee_id, e.name, e.joined_on::text joined_on, $3::integer AS "year",
	       COALESCE(a.allowance, w.annual_allowance, 12) allowance,
	       COALESCE(pa.allowance, w.annual_allowance, 12) prev_allowance,
	       count(d.date) FILTER (WHERE l.status = 'approved' AND EXTRACT(year FROM d.date) = $3) used,
	       count(d.date) FILTER (WHERE l.status = 'pending' AND EXTRACT(year FROM d.date) = $3) reserved,
	       count(d.date) FILTER (WHERE l.status = 'approved' AND EXTRACT(year FROM d.date) = $3 - 1) prev_used,
	       count(d.date) FILTER (WHERE l.status = 'approved' AND EXTRACT(year FROM d.date) = $3 AND d.date <= $4::date) used_early,
	       count(d.date) FILTER (WHERE l.status = 'pending' AND EXTRACT(year FROM d.date) = $3 AND d.date <= $4::date) reserved_early
	FROM employees e
	LEFT JOIN work_calendars w ON w.company_id = e.company_id
	LEFT JOIN leave_allocations a ON a.company_id = e.company_id AND a.employee_id = e.id AND a.year = $3
	LEFT JOIN leave_allocations pa ON pa.company_id = e.company_id AND pa.employee_id = e.id AND pa.year = $3 - 1
	LEFT JOIN leave_requests l ON l.company_id = e.company_id AND l.employee_id = e.id AND l.kind = 'annual'
	LEFT JOIN leave_days d ON d.company_id = l.company_id AND d.leave_id = l.id
	WHERE e.company_id = $1 AND ($2::bigint IS NULL OR e.id = $2)
	GROUP BY e.id, a.allowance, pa.allowance, w.annual_allowance
	ORDER BY e.name`

// balances computes entitlement under the company policy. asOf picks the date
// accrual is counted to for each balance's year.
func balances(ctx context.Context, q sqlx.QueryerContext, companyID int64, employeeID *int64, year int,
	asOf func(year int) time.Time) ([]model.Balance, error) {
	cal, err := workCalendar(ctx, q, companyID)
	if err != nil {
		return nil, err
	}
	p := leavePolicy(cal)
	expiry, expires := leavepolicy.CarryExpiry(p, year)
	if !expires {
		expiry = yearEnd(year)
	}
	v := []model.Balance{}
	if err = sqlx.SelectContext(ctx, q, &v, balancesSQL, companyID, employeeID, year, expiry.Format(time.DateOnly)); err != nil {
		return nil, err
	}
	for i := range v {
		b := &v[i]
		joined, err := time.Parse(time.DateOnly, b.JoinedOn)
		if err != nil {
			return nil, err
		}
		b.Accrued = leavepolicy.Accrued(p, b.Allowance, joined, year, asOf(year))
		b.CarriedOver = leavepolicy.Carry(p, b.PrevAllowance, b.PrevUsed, joined, year)
		if expires && b.CarriedOver > 0 {
			d := expiry.Format(time.DateOnly)
			b.CarryExpiresOn = &d
		}
		usable := leavepolicy.UsableCarry(p, b.CarriedOver, b.UsedEarly+b.ReservedEarly, year, asOf(year))
		b.Available = b.Accrued + usable - b.Used - b.Reserved
	}
	return v, nil
}

// wib is Asia/Jakarta without depending on the host's tz database.
var wib = time.FixedZone("WIB", 7*3600)

// asOfToday counts accrual to today for the current year and to year end for
// other years.
func asOfToday(year int) time.Time {
	now := time.Now().In(wib)
	if now.Year() == year {
		return time.Date(year, now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	}
	return yearEnd(year)
}

func yearEnd(year int) time.Time { return time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC) }

func (r *leaveRepository) Balances(ctx context.Context, companyID int64, employeeID *int64, year int) ([]model.Balance, error) {
	return balances(ctx, r.db, companyID, employeeID, year, asOfToday)
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
	rows, err := balances(ctx, tx, companyID, &employeeID, year, yearEnd)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return sql.ErrNoRows
	}
	b := rows[0]
	cal, err := workCalendar(ctx, tx, companyID)
	if err != nil {
		return err
	}
	joined, _ := time.Parse(time.DateOnly, b.JoinedOn)
	if leavepolicy.Accrued(leavePolicy(cal), allowance, joined, year, yearEnd(year))+b.CarriedOver < b.Used+b.Reserved {
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

// checkAnnualBalance verifies an employee's annual-leave balance for every
// year the given leave touches, counting accrual up to the leave's last day
// in that year. withReserved also counts pending requests (new requests);
// approvals only need the approved days to fit.
func checkAnnualBalance(ctx context.Context, tx *sqlx.Tx, companyID, employeeID, leaveID int64, withReserved bool) error {
	var years []struct {
		Year int    `db:"year"`
		Last string `db:"last"`
	}
	if err := tx.SelectContext(ctx, &years, `
		SELECT EXTRACT(year FROM date)::int AS "year", max(date)::text AS "last" FROM leave_days
		WHERE company_id = $1 AND leave_id = $2 GROUP BY 1`,
		companyID, leaveID); err != nil {
		return err
	}
	for _, y := range years {
		last, err := time.Parse(time.DateOnly, y.Last)
		if err != nil {
			return err
		}
		rows, err := balances(ctx, tx, companyID, &employeeID, y.Year, func(int) time.Time { return last })
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return sql.ErrNoRows
		}
		b := rows[0]
		spent, early := b.Used, b.UsedEarly
		if withReserved {
			spent, early = spent+b.Reserved, early+b.ReservedEarly
		}
		cal, err := workCalendar(ctx, tx, companyID)
		if err != nil {
			return err
		}
		carry := leavepolicy.UsableCarry(leavePolicy(cal), b.CarriedOver, early, y.Year, last)
		if b.Accrued+carry < spent {
			return apperror.Conflict(fmt.Sprintf("Saldo cuti tahun %d tidak mencukupi (termasuk akrual sampai %s)", y.Year, y.Last))
		}
	}
	return nil
}

func (r *leaveRepository) Leaves(ctx context.Context, companyID int64, employeeID *int64, f model.ListFilter) ([]model.Leave, int, error) {
	var total int
	if err := r.db.GetContext(ctx, &total, `
		SELECT count(*) FROM leave_requests
		WHERE company_id = $1 AND ($2::bigint IS NULL OR employee_id = $2) AND ($3 = '' OR status = $3)`,
		companyID, employeeID, f.Status); err != nil {
		return nil, 0, err
	}
	v := []model.Leave{}
	err := r.db.SelectContext(ctx, &v, `
		SELECT l.id, l.employee_id, e.name, l.kind, l.start_date::text, l.end_date::text,
		       l.reason, l.status, l.stage, l.review_note, l.calculation,
		       (SELECT count(*) FROM leave_days ld WHERE ld.company_id = l.company_id AND ld.leave_id = l.id) days
		FROM leave_requests l
		JOIN employees e ON e.id = l.employee_id
		WHERE l.company_id = $1 AND ($2::bigint IS NULL OR l.employee_id = $2) AND ($3 = '' OR l.status = $3)
		ORDER BY l.created_at DESC, l.id DESC
		LIMIT $4 OFFSET $5`,
		companyID, employeeID, f.Status, limitOf(f), f.Offset)
	return v, total, err
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
		INSERT INTO leave_requests (company_id, employee_id, kind, start_date, end_date, reason, stage)
		VALUES ($1, $2, $3, $4, $5, $6, `+approvalStage("$1", "$2")+`)
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
		for _, year := range []string{v.StartDate[:4], v.EndDate[:4]} {
			// Freeze the year's allowance so later default changes keep it.
			if _, err = tx.ExecContext(ctx, `
				INSERT INTO leave_allocations (company_id, employee_id, year, allowance)
				VALUES ($1, $2, $3::int, COALESCE((SELECT annual_allowance FROM work_calendars WHERE company_id = $1), 12))
				ON CONFLICT DO NOTHING`,
				companyID, employeeID, year); err != nil {
				return err
			}
		}
		if err = checkAnnualBalance(ctx, tx, companyID, employeeID, id, true); err != nil {
			return err
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
	var kind string
	if err = tx.QueryRowxContext(ctx, `
		SELECT employee_id, kind FROM leave_requests WHERE company_id = $1 AND id = $2`,
		companyID, leaveID).Scan(&employeeID, &kind); err != nil {
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
	if status == "approved" && kind == "annual" {
		if err = checkAnnualBalance(ctx, tx, companyID, employeeID, leaveID, false); err != nil {
			return err
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
