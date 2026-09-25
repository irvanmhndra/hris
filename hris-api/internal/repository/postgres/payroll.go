package postgres

import (
	"context"
	"database/sql"
	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
)

func (r *Repository) Salaries(ctx context.Context, c int64) ([]model.Salary, error) {
	v := []model.Salary{}
	err := r.DB.SelectContext(ctx, &v, `SELECT e.id employee_id,e.name,e.code,(s.employee_id IS NOT NULL) configured,COALESCE(s.basic_salary,0) basic_salary,COALESCE(s.allowance,0) allowance,COALESCE(s.deduction,0) deduction,COALESCE(s.note,'') note FROM employees e LEFT JOIN salary_profiles s ON s.company_id=e.company_id AND s.employee_id=e.id WHERE e.company_id=$1 AND e.status='active' ORDER BY e.name`, c)
	return v, err
}
func (r *Repository) SaveSalary(ctx context.Context, u *model.User, e int64, v dto.Salary) error {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = lockEmployee(ctx, tx, u.CompanyID, e); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO salary_profiles(company_id,employee_id,basic_salary,allowance,deduction,note) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(company_id,employee_id) DO UPDATE SET basic_salary=$3,allowance=$4,deduction=$5,note=$6`, u.CompanyID, e, v.BasicSalary, v.Allowance, v.Deduction, v.Note)
	if err != nil {
		return err
	}
	if err = audit(ctx, tx, u.CompanyID, u.ID, "update", "salary_profile", e, "Memperbarui komponen gaji"); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) PayrollRuns(ctx context.Context, c int64) ([]model.PayrollRun, error) {
	v := []model.PayrollRun{}
	err := r.DB.SelectContext(ctx, &v, `SELECT r.id,to_char(r.period,'YYYY-MM') period,r.status,count(e.id) employees,COALESCE(sum(e.basic_salary+e.allowance-e.deduction),0) total,r.payment_reference,r.created_at FROM payroll_runs r LEFT JOIN payroll_entries e ON e.run_id=r.id AND e.company_id=r.company_id WHERE r.company_id=$1 GROUP BY r.id ORDER BY r.period DESC,r.id DESC`, c)
	return v, err
}
func (r *Repository) CreatePayroll(ctx context.Context, u *model.User, period string) (int64, error) {
	tx, err := r.DB.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var count, missing int
	err = tx.QueryRowxContext(ctx, `SELECT count(*),count(*) FILTER(WHERE s.employee_id IS NULL) FROM employees e LEFT JOIN salary_profiles s ON s.company_id=e.company_id AND s.employee_id=e.id WHERE e.company_id=$1 AND e.status='active' AND e.joined_on<($2::date+interval '1 month')`, u.CompanyID, period).Scan(&count, &missing)
	if err != nil {
		return 0, err
	}
	if count == 0 || missing > 0 {
		return 0, &apperror.Error{409, "Lengkapi komponen gaji seluruh karyawan aktif pada periode tersebut sebelum membuat payroll"}
	}
	var id int64
	err = tx.QueryRowxContext(ctx, `INSERT INTO payroll_runs(company_id,period,created_by) VALUES($1,$2,$3) RETURNING id`, u.CompanyID, period, u.ID).Scan(&id)
	if err != nil {
		return 0, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO payroll_entries(company_id,run_id,employee_id,employee_name,employee_code,position,basic_salary,allowance,deduction,note) SELECT e.company_id,$2,e.id,e.name,e.code,e.position,s.basic_salary,s.allowance,s.deduction,s.note FROM employees e JOIN salary_profiles s ON s.company_id=e.company_id AND s.employee_id=e.id WHERE e.company_id=$1 AND e.status='active' AND e.joined_on<($3::date+interval '1 month')`, u.CompanyID, id, period)
	if err != nil {
		return 0, err
	}
	if err = audit(ctx, tx, u.CompanyID, u.ID, "create", "payroll", id, "Membuat draft payroll "+period); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}
func (r *Repository) Payslips(ctx context.Context, u *model.User, run int64) ([]model.Payslip, error) {
	v := []model.Payslip{}
	var employee *int64
	if u.Role == "employee" {
		employee = u.EmployeeID
	}
	err := r.DB.SelectContext(ctx, &v, `SELECT e.id,e.run_id,e.employee_id,e.employee_name,e.employee_code,e.position,to_char(r.period,'YYYY-MM') period,r.status,e.basic_salary,e.allowance,e.deduction,(e.basic_salary+e.allowance-e.deduction) net,e.note,e.version FROM payroll_entries e JOIN payroll_runs r ON r.id=e.run_id AND r.company_id=e.company_id WHERE e.company_id=$1 AND ($2::bigint IS NULL OR (e.employee_id=$2 AND r.status IN ('finalized','paid'))) AND ($3::bigint=0 OR e.run_id=$3) ORDER BY r.period DESC,e.employee_name`, u.CompanyID, employee, run)
	return v, err
}
func (r *Repository) SavePayslip(ctx context.Context, u *model.User, id int64, v dto.Salary) error {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var run int64
	if err = tx.GetContext(ctx, &run, `SELECT run_id FROM payroll_entries WHERE company_id=$1 AND id=$2`, u.CompanyID, id); err != nil {
		return err
	}
	var status string
	if err = tx.GetContext(ctx, &status, `SELECT status FROM payroll_runs WHERE company_id=$1 AND id=$2 FOR UPDATE`, u.CompanyID, run); err != nil {
		return err
	}
	if status != "draft" {
		return &apperror.Error{409, "Payroll yang sudah difinalisasi tidak dapat diubah"}
	}
	res, err := tx.ExecContext(ctx, `UPDATE payroll_entries SET basic_salary=$3,allowance=$4,deduction=$5,note=$6,version=version+1 WHERE company_id=$1 AND id=$2 AND version=$7`, u.CompanyID, id, v.BasicSalary, v.Allowance, v.Deduction, v.Note, v.Version)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return &apperror.Error{409, "Slip sudah berubah; muat ulang terlebih dahulu"}
	}
	if err = audit(ctx, tx, u.CompanyID, u.ID, "update", "payslip", id, "Penyesuaian komponen slip draft"); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) PayrollAction(ctx context.Context, u *model.User, id int64, v dto.PayrollAction) error {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var status string
	if err = tx.GetContext(ctx, &status, `SELECT status FROM payroll_runs WHERE company_id=$1 AND id=$2 FOR UPDATE`, u.CompanyID, id); err != nil {
		return err
	}
	next := ""
	switch {
	case v.Action == "finalize" && status == "draft":
		next = "finalized"
	case v.Action == "paid" && status == "finalized":
		next = "paid"
	case v.Action == "void" && status == "draft":
		next = "void"
	default:
		return &apperror.Error{409, "Perubahan status payroll tidak diizinkan"}
	}
	_, err = tx.ExecContext(ctx, `UPDATE payroll_runs SET status=$3,finalized_at=CASE WHEN $3='finalized' THEN now() ELSE finalized_at END,paid_at=CASE WHEN $3='paid' THEN now() ELSE paid_at END,payment_reference=CASE WHEN $3='paid' THEN $4 ELSE payment_reference END WHERE company_id=$1 AND id=$2`, u.CompanyID, id, next, v.Reference)
	if err != nil {
		return err
	}
	if err = audit(ctx, tx, u.CompanyID, u.ID, v.Action, "payroll", id, "Status payroll: "+next); err != nil {
		return err
	}
	return tx.Commit()
}
