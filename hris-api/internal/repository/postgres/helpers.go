// Package postgres implements the repository contracts on PostgreSQL via sqlx.
package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/jmoiron/sqlx"
)

// rollback is deferred right after BeginTxx; it is a no-op once committed.
func rollback(tx *sqlx.Tx) { _ = tx.Rollback() }

func rowsAffected(res sql.Result) int64 {
	n, _ := res.RowsAffected()
	return n
}

// audit records a change inside the caller's transaction, so the audit row
// commits or rolls back together with the change it describes.
func audit(ctx context.Context, tx *sqlx.Tx, companyID, actorID int64, action, resource string, resourceID int64, summary string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO audit_logs (company_id, actor_id, action, resource, resource_id, summary)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		companyID, actorID, action, resource, resourceID, summary)
	return err
}

// lockEmployee row-locks the employee, serialising every request that touches
// the same person's leave, attendance, profile, salary, or workflow items.
func lockEmployee(ctx context.Context, tx *sqlx.Tx, companyID, employeeID int64) error {
	var id int64
	return tx.GetContext(ctx, &id, `
		SELECT id FROM employees WHERE company_id = $1 AND id = $2 FOR UPDATE`,
		companyID, employeeID)
}

// attendanceFingerprint snapshots an attendance row so approving a correction
// can detect that the record changed after the correction was requested.
func attendanceFingerprint(ctx context.Context, tx *sqlx.Tx, companyID, employeeID int64, date string) (string, error) {
	var v string
	err := tx.GetContext(ctx, &v, `
		SELECT check_in::text || '|' || COALESCE(check_out::text, '')
		FROM attendances
		WHERE company_id = $1 AND employee_id = $2 AND date = $3`,
		companyID, employeeID, date)
	if errors.Is(err, sql.ErrNoRows) {
		return "missing", nil
	}
	return v, err
}

// approvalStage is a SQL expression for the first approval stage of a new
// request by an employee: "manager" when they report to an active manager
// with a portal account, otherwise straight to "hr". Arguments are the
// placeholders holding the company and employee ids.
func approvalStage(company, employee string) string {
	return `(CASE WHEN EXISTS (
		SELECT 1 FROM employees r
		JOIN employees m ON m.company_id = r.company_id AND m.id = r.manager_id AND m.status = 'active'
		JOIN users mu ON mu.company_id = m.company_id AND mu.employee_id = m.id
		WHERE r.company_id = ` + company + ` AND r.id = ` + employee + `) THEN 'manager' ELSE 'hr' END)`
}

// limitOf maps a filter's limit to SQL: LIMIT NULL means no limit.
func limitOf(f model.ListFilter) sql.NullInt64 {
	return sql.NullInt64{Int64: int64(f.Limit), Valid: f.Limit > 0}
}
