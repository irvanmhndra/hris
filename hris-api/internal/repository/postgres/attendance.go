package postgres

import (
	"context"
	"database/sql"

	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/jmoiron/sqlx"
)

type attendanceRepository struct{ db *sqlx.DB }

func NewAttendanceRepository(db *sqlx.DB) repository.AttendanceRepository {
	return &attendanceRepository{db: db}
}

func (r *attendanceRepository) Attendances(ctx context.Context, companyID int64, employeeID *int64) ([]model.Attendance, error) {
	v := []model.Attendance{}
	err := r.db.SelectContext(ctx, &v, `
		SELECT a.id, a.employee_id, e.name, a.date::text, a.check_in, a.check_out
		FROM attendances a
		JOIN employees e ON e.id = a.employee_id
		WHERE a.company_id = $1 AND ($2::bigint IS NULL OR a.employee_id = $2)
		ORDER BY a.date DESC, a.check_in DESC
		LIMIT 100`,
		companyID, employeeID)
	return v, err
}

// Clock records check-in or check-out for today in Asia/Jakarta. A unique
// constraint allows one check-in per day; check-out needs an open check-in.
func (r *attendanceRepository) Clock(ctx context.Context, companyID, employeeID int64, out bool) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if err = lockEmployee(ctx, tx, companyID, employeeID); err != nil {
		return err
	}
	if !out {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO attendances (company_id, employee_id, date)
			VALUES ($1, $2, (now() AT TIME ZONE 'Asia/Jakarta')::date)`,
			companyID, employeeID)
		if err != nil {
			return err
		}
		return tx.Commit()
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE attendances SET check_out = now()
		WHERE company_id = $1 AND employee_id = $2
		  AND date = (now() AT TIME ZONE 'Asia/Jakarta')::date
		  AND check_out IS NULL`,
		companyID, employeeID)
	if err != nil {
		return err
	}
	if rowsAffected(res) == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}
