package postgres

import (
	"context"
	"database/sql"
	"strings"

	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/jmoiron/sqlx"
)

type employeeRepository struct{ db *sqlx.DB }

func NewEmployeeRepository(db *sqlx.DB) repository.EmployeeRepository {
	return &employeeRepository{db: db}
}

func (r *employeeRepository) Departments(ctx context.Context, companyID int64) ([]model.Department, error) {
	v := []model.Department{}
	err := r.db.SelectContext(ctx, &v, `
		SELECT d.id, d.name, count(e.id) AS count
		FROM departments d
		LEFT JOIN employees e ON e.department_id = d.id AND e.status = 'active'
		WHERE d.company_id = $1
		GROUP BY d.id
		ORDER BY d.name`,
		companyID)
	return v, err
}

func (r *employeeRepository) CreateDepartment(ctx context.Context, companyID int64, name string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO departments (company_id, name) VALUES ($1, $2)`, companyID, name)
	return err
}

// employeeFilterSQL is shared by the page and count queries so they always
// agree. The search matches the same fields the web client searched before.
const employeeFilterSQL = `
	FROM employees e
	JOIN departments d ON d.id = e.department_id
	WHERE e.company_id = $1
	  AND ($2 = '' OR e.name ILIKE $2 OR e.code ILIKE $2 OR e.email ILIKE $2 OR d.name ILIKE $2)
	  AND ($3 = '' OR e.status = $3)
	  AND ($4::bigint = 0 OR e.department_id = $4)`

func (r *employeeRepository) Employees(ctx context.Context, companyID int64, f model.EmployeeFilter) ([]model.Employee, int, error) {
	pattern := ""
	if s := strings.TrimSpace(f.Search); s != "" {
		pattern = "%" + likeEscaper.Replace(s) + "%"
	}
	args := []any{companyID, pattern, f.Status, f.DepartmentID}

	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT count(*) `+employeeFilterSQL, args...); err != nil {
		return nil, 0, err
	}
	// LIMIT NULL means "no limit": callers that need every row (CSV export,
	// employee pickers) omit paging.
	var limit sql.NullInt64
	if f.Limit > 0 {
		limit = sql.NullInt64{Int64: int64(f.Limit), Valid: true}
	}
	v := []model.Employee{}
	err := r.db.SelectContext(ctx, &v, `
		SELECT e.id, e.company_id, e.code, e.name, e.email, e.department_id,
		       d.name AS department, e.position, e.status, e.joined_on::text
		`+employeeFilterSQL+`
		ORDER BY e.name, e.id
		LIMIT $5 OFFSET $6`,
		append(args, limit, f.Offset)...)
	return v, total, err
}

// likeEscaper neutralises LIKE wildcards in user input (backslash is the
// default ESCAPE character in PostgreSQL).
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func (r *employeeRepository) SaveEmployee(ctx context.Context, companyID, id int64, v model.EmployeeInput) (int64, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer rollback(tx)

	isNew := id == 0
	if isNew {
		err = tx.QueryRowxContext(ctx, `
			INSERT INTO employees (company_id, code, name, email, department_id, position, status, joined_on)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id`,
			companyID, v.Code, v.Name, v.Email, v.DepartmentID, v.Position, v.Status, v.JoinedOn).Scan(&id)
		if err != nil {
			return 0, err
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO users (company_id, employee_id, name, email, password_hash, role)
			VALUES ($1, $2, $3, $4, $5, 'employee')`,
			companyID, id, v.Name, v.Email, v.PasswordHash)
		if err != nil {
			return 0, err
		}
	} else {
		res, err := tx.ExecContext(ctx, `
			UPDATE employees
			SET code = $3, name = $4, email = $5, department_id = $6, position = $7, status = $8, joined_on = $9
			WHERE company_id = $1 AND id = $2`,
			companyID, id, v.Code, v.Name, v.Email, v.DepartmentID, v.Position, v.Status, v.JoinedOn)
		if err != nil {
			return 0, err
		}
		if rowsAffected(res) == 0 {
			return 0, sql.ErrNoRows
		}
	}

	// Keep the login identity in sync with the employee record.
	_, err = tx.ExecContext(ctx, `
		UPDATE users
		SET name = $3, email = $4,
		    password_hash = CASE WHEN $5 = '' THEN password_hash ELSE $5 END
		WHERE company_id = $1 AND employee_id = $2`,
		companyID, id, v.Name, v.Email, v.PasswordHash)
	if err != nil {
		return 0, err
	}
	// A password reset or deactivation must end existing sessions immediately.
	if !isNew && (v.PasswordHash != "" || v.Status == "inactive") {
		_, err = tx.ExecContext(ctx, `
			DELETE FROM user_sessions
			WHERE user_id IN (SELECT id FROM users WHERE company_id = $1 AND employee_id = $2)`,
			companyID, id)
		if err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}
