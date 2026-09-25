package postgres

import (
	"context"
	"database/sql"
	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/jmoiron/sqlx"
)

type Repository struct{ DB *sqlx.DB }

const userFields = `u.id,u.company_id,u.employee_id,u.name,u.email,u.role,u.password_hash,c.name AS company_name`

func (r *Repository) LoginUser(ctx context.Context, company, email string) (*model.User, error) {
	var u model.User
	err := r.DB.GetContext(ctx, &u, `SELECT `+userFields+` FROM users u JOIN companies c ON c.id=u.company_id LEFT JOIN employees e ON e.id=u.employee_id WHERE c.slug=$1 AND u.email=$2 AND (u.employee_id IS NULL OR e.status='active')`, company, email)
	return &u, err
}
func (r *Repository) CreateSession(ctx context.Context, id int64, hash string) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO user_sessions(token_hash,user_id,expires_at) VALUES($1,$2,now()+interval '12 hours')`, hash, id)
	return err
}
func (r *Repository) SessionUser(ctx context.Context, hash string) (*model.User, error) {
	var u model.User
	err := r.DB.GetContext(ctx, &u, `SELECT `+userFields+` FROM user_sessions s JOIN users u ON u.id=s.user_id JOIN companies c ON c.id=u.company_id LEFT JOIN employees e ON e.id=u.employee_id WHERE s.token_hash=$1 AND s.expires_at>now() AND (u.employee_id IS NULL OR e.status='active')`, hash)
	return &u, err
}
func (r *Repository) DeleteSession(ctx context.Context, hash string) error {
	_, err := r.DB.ExecContext(ctx, `DELETE FROM user_sessions WHERE token_hash=$1`, hash)
	return err
}
func (r *Repository) Departments(ctx context.Context, c int64) ([]model.Department, error) {
	v := []model.Department{}
	err := r.DB.SelectContext(ctx, &v, `SELECT d.id,d.name,count(e.id) AS count FROM departments d LEFT JOIN employees e ON e.department_id=d.id AND e.status='active' WHERE d.company_id=$1 GROUP BY d.id ORDER BY d.name`, c)
	return v, err
}
func (r *Repository) CreateDepartment(ctx context.Context, c int64, name string) error {
	_, err := r.DB.ExecContext(ctx, `INSERT INTO departments(company_id,name) VALUES($1,$2)`, c, name)
	return err
}
func (r *Repository) Employees(ctx context.Context, c int64) ([]model.Employee, error) {
	v := []model.Employee{}
	err := r.DB.SelectContext(ctx, &v, `SELECT e.id,e.company_id,e.code,e.name,e.email,e.department_id,d.name AS department,e.position,e.status,e.joined_on::text FROM employees e JOIN departments d ON d.id=e.department_id WHERE e.company_id=$1 ORDER BY e.name`, c)
	return v, err
}
func (r *Repository) SaveEmployee(ctx context.Context, c, id int64, v dto.Employee, hash string) (int64, error) {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	isNew := id == 0
	if isNew {
		err = tx.QueryRowxContext(ctx, `INSERT INTO employees(company_id,code,name,email,department_id,position,status,joined_on) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`, c, v.Code, v.Name, v.Email, v.DepartmentID, v.Position, v.Status, v.JoinedOn).Scan(&id)
	} else {
		var result sql.Result
		result, err = tx.ExecContext(ctx, `UPDATE employees SET code=$3,name=$4,email=$5,department_id=$6,position=$7,status=$8,joined_on=$9 WHERE company_id=$1 AND id=$2`, c, id, v.Code, v.Name, v.Email, v.DepartmentID, v.Position, v.Status, v.JoinedOn)
		if err == nil {
			n, _ := result.RowsAffected()
			if n == 0 {
				return 0, sql.ErrNoRows
			}
		}
	}
	if err != nil {
		return 0, err
	}
	if isNew {
		_, err = tx.ExecContext(ctx, `INSERT INTO users(company_id,employee_id,name,email,password_hash,role) VALUES($1,$2,$3,$4,$5,'employee')`, c, id, v.Name, v.Email, hash)
		if err != nil {
			return 0, err
		}
	}
	// Keep the existing login identity synchronized when an employee email changes.
	_, err = tx.ExecContext(ctx, `UPDATE users SET name=$3,email=$4,password_hash=CASE WHEN $5='' THEN password_hash ELSE $5 END WHERE company_id=$1 AND employee_id=$2`, c, id, v.Name, v.Email, hash)
	if err != nil {
		return 0, err
	}
	if !isNew && (hash != "" || v.Status == "inactive") {
		_, err = tx.ExecContext(ctx, `DELETE FROM user_sessions WHERE user_id IN (SELECT id FROM users WHERE company_id=$1 AND employee_id=$2)`, c, id)
		if err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}
func (r *Repository) Dashboard(ctx context.Context, c int64) (model.Dashboard, error) {
	var v model.Dashboard
	err := r.DB.GetContext(ctx, &v, `SELECT (SELECT count(*) FROM employees WHERE company_id=$1 AND status='active') employees,(SELECT count(*) FROM attendances WHERE company_id=$1 AND date=(now() AT TIME ZONE 'Asia/Jakarta')::date) present,(SELECT count(*) FROM leave_requests WHERE company_id=$1 AND status='pending') pending,(SELECT count(*) FROM departments WHERE company_id=$1) departments`, c)
	return v, err
}
func (r *Repository) Attendances(ctx context.Context, c int64, e *int64) ([]model.Attendance, error) {
	v := []model.Attendance{}
	err := r.DB.SelectContext(ctx, &v, `SELECT a.id,a.employee_id,e.name,a.date::text,a.check_in,a.check_out FROM attendances a JOIN employees e ON e.id=a.employee_id WHERE a.company_id=$1 AND ($2::bigint IS NULL OR a.employee_id=$2) ORDER BY a.date DESC,a.check_in DESC LIMIT 100`, c, e)
	return v, err
}
func (r *Repository) Clock(ctx context.Context, c, e int64, out bool) error {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = lockEmployee(ctx, tx, c, e); err != nil {
		return err
	}
	if !out {
		_, err = tx.ExecContext(ctx, `INSERT INTO attendances(company_id,employee_id,date) VALUES($1,$2,(now() AT TIME ZONE 'Asia/Jakarta')::date)`, c, e)
	} else {
		var res sql.Result
		res, err = tx.ExecContext(ctx, `UPDATE attendances SET check_out=now() WHERE company_id=$1 AND employee_id=$2 AND date=(now() AT TIME ZONE 'Asia/Jakarta')::date AND check_out IS NULL`, c, e)
		if err == nil {
			n, _ := res.RowsAffected()
			if n == 0 {
				return sql.ErrNoRows
			}
		}
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) Leaves(ctx context.Context, c int64, e *int64) ([]model.Leave, error) {
	v := []model.Leave{}
	err := r.DB.SelectContext(ctx, &v, `SELECT l.id,l.employee_id,e.name,l.kind,l.start_date::text,l.end_date::text,l.reason,l.status,l.calculation,(SELECT count(*) FROM leave_days ld WHERE ld.company_id=l.company_id AND ld.leave_id=l.id) days FROM leave_requests l JOIN employees e ON e.id=l.employee_id WHERE l.company_id=$1 AND ($2::bigint IS NULL OR l.employee_id=$2) ORDER BY l.created_at DESC`, c, e)
	return v, err
}
