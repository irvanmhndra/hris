package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func audit(ctx context.Context, tx *sqlx.Tx, c, u int64, action, resource string, id int64, summary string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO audit_logs(company_id,actor_id,action,resource,resource_id,summary) VALUES($1,$2,$3,$4,$5,$6)`, c, u, action, resource, id, summary)
	return err
}
func lockEmployee(ctx context.Context, tx *sqlx.Tx, c, e int64) error {
	var id int64
	return tx.GetContext(ctx, &id, `SELECT id FROM employees WHERE company_id=$1 AND id=$2 FOR UPDATE`, c, e)
}
func (r *Repository) WorkCalendar(ctx context.Context, c int64) (model.WorkCalendar, error) {
	v := model.WorkCalendar{Workdays: pq.Int64Array{1, 2, 3, 4, 5}, AnnualAllowance: 12, StartTime: "09:00", EndTime: "18:00"}
	err := r.DB.GetContext(ctx, &v, `SELECT workdays,annual_allowance,start_time,end_time FROM work_calendars WHERE company_id=$1`, c)
	if err == sql.ErrNoRows {
		err = nil
	}
	return v, err
}
func (r *Repository) SaveCalendar(ctx context.Context, u *model.User, v model.WorkCalendar) error {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Freeze entitlements for legacy requests before changing the default.
	_, err = tx.ExecContext(ctx, `INSERT INTO leave_allocations(company_id,employee_id,year,allowance) SELECT DISTINCT l.company_id,l.employee_id,EXTRACT(year FROM d.date)::integer,COALESCE(w.annual_allowance,12) FROM leave_requests l JOIN leave_days d ON d.company_id=l.company_id AND d.leave_id=l.id LEFT JOIN work_calendars w ON w.company_id=l.company_id WHERE l.company_id=$1 AND l.kind='annual' ON CONFLICT DO NOTHING`, u.CompanyID)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO work_calendars(company_id,workdays,annual_allowance,start_time,end_time) VALUES($1,$2,$3,$4,$5) ON CONFLICT(company_id) DO UPDATE SET workdays=$2,annual_allowance=$3,start_time=$4,end_time=$5`, u.CompanyID, v.Workdays, v.AnnualAllowance, v.StartTime, v.EndTime)
	if err != nil {
		return err
	}
	if err = audit(ctx, tx, u.CompanyID, u.ID, "update", "calendar", u.CompanyID, "Memperbarui kalender dan kuota default untuk alokasi baru"); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) Holidays(ctx context.Context, c int64) ([]model.Holiday, error) {
	v := []model.Holiday{}
	err := r.DB.SelectContext(ctx, &v, `SELECT id,date::text,name FROM holidays WHERE company_id=$1 ORDER BY date`, c)
	return v, err
}
func (r *Repository) SaveHoliday(ctx context.Context, u *model.User, v model.Holiday) error {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id int64
	err = tx.QueryRowxContext(ctx, `INSERT INTO holidays(company_id,date,name) VALUES($1,$2,$3) RETURNING id`, u.CompanyID, v.Date, v.Name).Scan(&id)
	if err != nil {
		return err
	}
	if err = audit(ctx, tx, u.CompanyID, u.ID, "create", "holiday", id, v.Name); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) DeleteHoliday(ctx context.Context, u *model.User, id int64) error {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var name string
	err = tx.QueryRowxContext(ctx, `DELETE FROM holidays WHERE company_id=$1 AND id=$2 RETURNING name`, u.CompanyID, id).Scan(&name)
	if err != nil {
		return err
	}
	if err = audit(ctx, tx, u.CompanyID, u.ID, "delete", "holiday", id, name); err != nil {
		return err
	}
	return tx.Commit()
}

const balancesSQL = `WITH totals AS (
 SELECT e.id employee_id,e.name,$3::integer AS "year",COALESCE(a.allowance,w.annual_allowance,12) allowance,
 (SELECT count(*) FROM leave_days d JOIN leave_requests l ON l.id=d.leave_id AND l.company_id=d.company_id WHERE l.company_id=e.company_id AND l.employee_id=e.id AND l.kind='annual' AND l.status='approved' AND EXTRACT(year FROM d.date)=$3) used,
 (SELECT count(*) FROM leave_days d JOIN leave_requests l ON l.id=d.leave_id AND l.company_id=d.company_id WHERE l.company_id=e.company_id AND l.employee_id=e.id AND l.kind='annual' AND l.status='pending' AND EXTRACT(year FROM d.date)=$3) reserved
 FROM employees e LEFT JOIN work_calendars w ON w.company_id=e.company_id LEFT JOIN leave_allocations a ON a.company_id=e.company_id AND a.employee_id=e.id AND a.year=$3
 WHERE e.company_id=$1 AND ($2::bigint IS NULL OR e.id=$2)
) SELECT *,allowance-used-reserved available FROM totals ORDER BY name`

func (r *Repository) Balances(ctx context.Context, c int64, e *int64, year int) ([]model.Balance, error) {
	v := []model.Balance{}
	err := r.DB.SelectContext(ctx, &v, balancesSQL, c, e, year)
	return v, err
}
func (r *Repository) SetAllowance(ctx context.Context, u *model.User, e int64, year, allowance int) error {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = lockEmployee(ctx, tx, u.CompanyID, e); err != nil {
		return err
	}
	var b model.Balance
	if err = tx.GetContext(ctx, &b, balancesSQL, u.CompanyID, e, year); err != nil {
		return err
	}
	if allowance < b.Used+b.Reserved {
		return &apperror.Error{409, "Kuota tidak boleh lebih kecil dari cuti terpakai dan yang sedang diajukan"}
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO leave_allocations(company_id,employee_id,year,allowance) VALUES($1,$2,$3,$4) ON CONFLICT(company_id,employee_id,year) DO UPDATE SET allowance=$4`, u.CompanyID, e, year, allowance)
	if err != nil {
		return err
	}
	if err = audit(ctx, tx, u.CompanyID, u.ID, "allocate", "leave_balance", e, fmt.Sprintf("Kuota %d: %d hari", year, allowance)); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) CreateLeave(ctx context.Context, c, e int64, v dto.Leave) error {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = lockEmployee(ctx, tx, c, e); err != nil {
		return err
	}
	var overlap bool
	err = tx.GetContext(ctx, &overlap, `SELECT EXISTS(SELECT 1 FROM leave_requests WHERE company_id=$1 AND employee_id=$2 AND status IN ('pending','approved') AND start_date<=$4::date AND end_date>=$3::date)`, c, e, v.StartDate, v.EndDate)
	if err != nil {
		return err
	}
	if overlap {
		return apperror.ErrOverlap
	}
	var id int64
	err = tx.QueryRowxContext(ctx, `INSERT INTO leave_requests(company_id,employee_id,kind,start_date,end_date,reason) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`, c, e, v.Kind, v.StartDate, v.EndDate, v.Reason).Scan(&id)
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO leave_days(company_id,leave_id,date) SELECT $1,$2,d::date FROM generate_series($3::date,$4::date,interval '1 day') d WHERE EXTRACT(dow FROM d)::int = ANY(COALESCE((SELECT workdays FROM work_calendars WHERE company_id=$1),ARRAY[1,2,3,4,5])) AND NOT EXISTS(SELECT 1 FROM holidays WHERE company_id=$1 AND date=d::date)`, c, id, v.StartDate, v.EndDate)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apperror.Invalid("Rentang cuti tidak memiliki hari kerja")
	}
	if v.Kind == "annual" {
		var years []int
		if err = tx.SelectContext(ctx, &years, `SELECT DISTINCT EXTRACT(year FROM date)::int FROM leave_days WHERE company_id=$1 AND leave_id=$2`, c, id); err != nil {
			return err
		}
		for _, year := range years {
			_, err = tx.ExecContext(ctx, `INSERT INTO leave_allocations(company_id,employee_id,year,allowance) VALUES($1,$2,$3,COALESCE((SELECT annual_allowance FROM work_calendars WHERE company_id=$1),12)) ON CONFLICT DO NOTHING`, c, e, year)
			if err != nil {
				return err
			}
			var b model.Balance
			if err = tx.GetContext(ctx, &b, balancesSQL, c, e, year); err != nil {
				return err
			}
			if b.Available < 0 {
				return &apperror.Error{409, fmt.Sprintf("Saldo cuti tahun %d tidak mencukupi", year)}
			}
		}
	}
	var actor int64
	if err = tx.GetContext(ctx, &actor, `SELECT id FROM users WHERE company_id=$1 AND employee_id=$2`, c, e); err != nil {
		return err
	}
	if err = audit(ctx, tx, c, actor, "create", "leave", id, "Mengajukan cuti "+v.StartDate+" s.d. "+v.EndDate); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) ReviewLeave(ctx context.Context, c, id, u int64, status string) error {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var employee int64
	if err = tx.GetContext(ctx, &employee, `SELECT employee_id FROM leave_requests WHERE company_id=$1 AND id=$2`, c, id); err != nil {
		return err
	}
	if err = lockEmployee(ctx, tx, c, employee); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE leave_requests SET status=$4,reviewed_by=$3,reviewed_at=now() WHERE company_id=$1 AND id=$2 AND status='pending'`, c, id, u, status)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	if status == "approved" {
		var excess bool
		err = tx.GetContext(ctx, &excess, `SELECT EXISTS(SELECT 1 FROM leave_days d JOIN leave_requests l ON l.id=d.leave_id LEFT JOIN leave_allocations a ON a.company_id=l.company_id AND a.employee_id=l.employee_id AND a.year=EXTRACT(year FROM d.date) LEFT JOIN work_calendars w ON w.company_id=l.company_id WHERE l.company_id=$1 AND l.employee_id=$2 AND l.kind='annual' AND l.status='approved' GROUP BY EXTRACT(year FROM d.date),a.allowance,w.annual_allowance HAVING count(*)>COALESCE(a.allowance,w.annual_allowance,12))`, c, employee)
		if err != nil {
			return err
		}
		if excess {
			return &apperror.Error{409, "Saldo cuti tidak mencukupi; periksa alokasi tahunan"}
		}
	}
	if err = audit(ctx, tx, c, u, status, "leave", id, "Keputusan pengajuan cuti"); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) CancelLeave(ctx context.Context, u *model.User, id int64) error {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = lockEmployee(ctx, tx, u.CompanyID, *u.EmployeeID); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE leave_requests SET status='cancelled' WHERE company_id=$1 AND employee_id=$2 AND id=$3 AND (status='pending' OR (status='approved' AND start_date>(now() AT TIME ZONE 'Asia/Jakarta')::date))`, u.CompanyID, u.EmployeeID, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return &apperror.Error{409, "Cuti tidak dapat dibatalkan atau tidak ditemukan"}
	}
	if err = audit(ctx, tx, u.CompanyID, u.ID, "cancel", "leave", id, "Membatalkan pengajuan dan mengembalikan saldo"); err != nil {
		return err
	}
	return tx.Commit()
}
