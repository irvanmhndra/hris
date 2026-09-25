package postgres

import (
	"context"
	"database/sql"

	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/jmoiron/sqlx"
)

type payrollRepository struct{ db *sqlx.DB }

func NewPayrollRepository(db *sqlx.DB) repository.PayrollRepository {
	return &payrollRepository{db: db}
}

func (r *payrollRepository) Salaries(ctx context.Context, companyID int64) ([]model.Salary, error) {
	v := []model.Salary{}
	err := r.db.SelectContext(ctx, &v, `
		SELECT e.id employee_id, e.name, e.code, (s.employee_id IS NOT NULL) configured,
		       COALESCE(s.basic_salary, 0) basic_salary,
		       COALESCE(s.allowance, 0) allowance,
		       COALESCE(s.deduction, 0) deduction,
		       COALESCE(s.note, '') note
		FROM employees e
		LEFT JOIN salary_profiles s ON s.company_id = e.company_id AND s.employee_id = e.id
		WHERE e.company_id = $1 AND e.status = 'active'
		ORDER BY e.name`,
		companyID)
	return v, err
}

func (r *payrollRepository) SaveSalary(ctx context.Context, companyID, actorID, employeeID int64, v model.SalaryInput) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if err = lockEmployee(ctx, tx, companyID, employeeID); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO salary_profiles (company_id, employee_id, basic_salary, allowance, deduction, note)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (company_id, employee_id) DO UPDATE
		SET basic_salary = $3, allowance = $4, deduction = $5, note = $6`,
		companyID, employeeID, v.BasicSalary, v.Allowance, v.Deduction, v.Note)
	if err != nil {
		return err
	}
	if err = audit(ctx, tx, companyID, actorID, "update", "salary_profile", employeeID, "Memperbarui komponen gaji"); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *payrollRepository) PayrollRuns(ctx context.Context, companyID int64) ([]model.PayrollRun, error) {
	v := []model.PayrollRun{}
	err := r.db.SelectContext(ctx, &v, `
		SELECT r.id, to_char(r.period, 'YYYY-MM') period, r.status, count(e.id) employees,
		       COALESCE(sum(e.basic_salary + e.allowance - e.deduction), 0) total,
		       r.payment_reference, r.created_at
		FROM payroll_runs r
		LEFT JOIN payroll_entries e ON e.run_id = r.id AND e.company_id = r.company_id
		WHERE r.company_id = $1
		GROUP BY r.id
		ORDER BY r.period DESC, r.id DESC`,
		companyID)
	return v, err
}

// CreatePayroll snapshots salary profiles of employees active in the period
// into a draft run. Repeatable read gives the count check and the copy one
// consistent view, so a salary change mid-creation cannot split the snapshot.
func (r *payrollRepository) CreatePayroll(ctx context.Context, companyID, actorID int64, periodStart string) (int64, error) {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return 0, err
	}
	defer rollback(tx)
	var count, missing int
	err = tx.QueryRowxContext(ctx, `
		SELECT count(*), count(*) FILTER (WHERE s.employee_id IS NULL)
		FROM employees e
		LEFT JOIN salary_profiles s ON s.company_id = e.company_id AND s.employee_id = e.id
		WHERE e.company_id = $1 AND e.status = 'active' AND e.joined_on < ($2::date + interval '1 month')`,
		companyID, periodStart).Scan(&count, &missing)
	if err != nil {
		return 0, err
	}
	if count == 0 || missing > 0 {
		return 0, apperror.Conflict("Lengkapi komponen gaji seluruh karyawan aktif pada periode tersebut sebelum membuat payroll")
	}
	var id int64
	err = tx.QueryRowxContext(ctx, `
		INSERT INTO payroll_runs (company_id, period, created_by) VALUES ($1, $2, $3) RETURNING id`,
		companyID, periodStart, actorID).Scan(&id)
	if err != nil {
		return 0, err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO payroll_entries
		       (company_id, run_id, employee_id, employee_name, employee_code, position,
		        basic_salary, allowance, deduction, note)
		SELECT e.company_id, $2, e.id, e.name, e.code, e.position,
		       s.basic_salary, s.allowance, s.deduction, s.note
		FROM employees e
		JOIN salary_profiles s ON s.company_id = e.company_id AND s.employee_id = e.id
		WHERE e.company_id = $1 AND e.status = 'active' AND e.joined_on < ($3::date + interval '1 month')`,
		companyID, id, periodStart)
	if err != nil {
		return 0, err
	}
	if err = audit(ctx, tx, companyID, actorID, "create", "payroll", id, "Membuat draft payroll "+periodStart); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (r *payrollRepository) Payslips(ctx context.Context, companyID int64, employeeID *int64, runID int64) ([]model.Payslip, error) {
	v := []model.Payslip{}
	err := r.db.SelectContext(ctx, &v, `
		SELECT e.id, e.run_id, e.employee_id, e.employee_name, e.employee_code, e.position,
		       to_char(r.period, 'YYYY-MM') period, r.status, e.basic_salary, e.allowance, e.deduction,
		       (e.basic_salary + e.allowance - e.deduction) net, e.note, e.version
		FROM payroll_entries e
		JOIN payroll_runs r ON r.id = e.run_id AND r.company_id = e.company_id
		WHERE e.company_id = $1
		  AND ($2::bigint IS NULL OR (e.employee_id = $2 AND r.status IN ('finalized', 'paid')))
		  AND ($3::bigint = 0 OR e.run_id = $3)
		ORDER BY r.period DESC, e.employee_name`,
		companyID, employeeID, runID)
	return v, err
}

// SavePayslip adjusts one slip of a draft run. The run row is locked so a
// concurrent finalisation cannot interleave, and version guards stale edits.
func (r *payrollRepository) SavePayslip(ctx context.Context, companyID, actorID, entryID int64, v model.SalaryInput) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	var runID int64
	if err = tx.GetContext(ctx, &runID, `
		SELECT run_id FROM payroll_entries WHERE company_id = $1 AND id = $2`,
		companyID, entryID); err != nil {
		return err
	}
	var status string
	if err = tx.GetContext(ctx, &status, `
		SELECT status FROM payroll_runs WHERE company_id = $1 AND id = $2 FOR UPDATE`,
		companyID, runID); err != nil {
		return err
	}
	if status != "draft" {
		return apperror.Conflict("Payroll yang sudah difinalisasi tidak dapat diubah")
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE payroll_entries
		SET basic_salary = $3, allowance = $4, deduction = $5, note = $6, version = version + 1
		WHERE company_id = $1 AND id = $2 AND version = $7`,
		companyID, entryID, v.BasicSalary, v.Allowance, v.Deduction, v.Note, v.Version)
	if err != nil {
		return err
	}
	if rowsAffected(res) == 0 {
		return apperror.Conflict("Slip sudah berubah; muat ulang terlebih dahulu")
	}
	if err = audit(ctx, tx, companyID, actorID, "update", "payslip", entryID, "Penyesuaian komponen slip draft"); err != nil {
		return err
	}
	return tx.Commit()
}

// PayrollAction moves a run through draft → finalized → paid (or draft → void).
func (r *payrollRepository) PayrollAction(ctx context.Context, companyID, actorID, runID int64, action, reference string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	var status string
	if err = tx.GetContext(ctx, &status, `
		SELECT status FROM payroll_runs WHERE company_id = $1 AND id = $2 FOR UPDATE`,
		companyID, runID); err != nil {
		return err
	}
	var next string
	switch {
	case action == "finalize" && status == "draft":
		next = "finalized"
	case action == "paid" && status == "finalized":
		next = "paid"
	case action == "void" && status == "draft":
		next = "void"
	default:
		return apperror.Conflict("Perubahan status payroll tidak diizinkan")
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE payroll_runs
		SET status = $3,
		    finalized_at = CASE WHEN $3 = 'finalized' THEN now() ELSE finalized_at END,
		    paid_at = CASE WHEN $3 = 'paid' THEN now() ELSE paid_at END,
		    payment_reference = CASE WHEN $3 = 'paid' THEN $4 ELSE payment_reference END
		WHERE company_id = $1 AND id = $2`,
		companyID, runID, next, reference)
	if err != nil {
		return err
	}
	if err = audit(ctx, tx, companyID, actorID, action, "payroll", runID, "Status payroll: "+next); err != nil {
		return err
	}
	return tx.Commit()
}
