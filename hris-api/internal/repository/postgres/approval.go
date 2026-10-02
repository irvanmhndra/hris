package postgres

import (
	"context"
	"database/sql"

	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/jmoiron/sqlx"
)

type approvalRepository struct{ db *sqlx.DB }

func NewApprovalRepository(db *sqlx.DB) repository.ApprovalRepository {
	return &approvalRepository{db: db}
}

// TeamApprovals lists requests from the manager's direct reports that wait
// for the manager's decision.
func (r *approvalRepository) TeamApprovals(ctx context.Context, companyID, managerID int64) ([]model.TeamRequest, error) {
	v := []model.TeamRequest{}
	err := r.db.SelectContext(ctx, &v, `
		SELECT 'leave' AS "type", l.id, e.name employee_name, l.kind, l.start_date::text start_date,
		       l.end_date::text end_date, '' title, l.reason, '{}'::jsonb data, 0 version, l.created_at
		FROM leave_requests l
		JOIN employees e ON e.company_id = l.company_id AND e.id = l.employee_id
		WHERE l.company_id = $1 AND e.manager_id = $2 AND l.status = 'pending' AND l.stage = 'manager'
		UNION ALL
		SELECT h.module, h.id, e.name, h.module, NULL, NULL, h.title, h.description, h.data, h.version, h.created_at
		FROM hr_items h
		JOIN employees e ON e.company_id = h.company_id AND e.id = h.employee_id
		WHERE h.company_id = $1 AND e.manager_id = $2 AND h.module IN ('overtime', 'corrections')
		  AND h.status = 'pending' AND h.stage = 'manager'
		ORDER BY created_at`,
		companyID, managerID)
	return v, err
}

// requesterOf returns the employee behind a request waiting at the manager
// stage for this manager; anything else is reported as not found.
func requesterOf(ctx context.Context, tx *sqlx.Tx, table string, companyID, managerID, id int64) (int64, error) {
	var employeeID int64
	err := tx.GetContext(ctx, &employeeID, `
		SELECT x.employee_id FROM `+table+` x
		JOIN employees e ON e.company_id = x.company_id AND e.id = x.employee_id
		WHERE x.company_id = $1 AND x.id = $2 AND e.manager_id = $3
		  AND x.status = 'pending' AND x.stage = 'manager'`,
		companyID, id, managerID)
	return employeeID, err
}

// ManagerReviewLeave forwards a leave request to HR (approve) or rejects it.
func (r *approvalRepository) ManagerReviewLeave(ctx context.Context, companyID, managerID, actorID, leaveID int64, approve bool, note string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	employeeID, err := requesterOf(ctx, tx, "leave_requests", companyID, managerID, leaveID)
	if err != nil {
		return err
	}
	if err = lockEmployee(ctx, tx, companyID, employeeID); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE leave_requests
		SET stage = 'hr', manager_reviewed_by = $3, manager_reviewed_at = now(), review_note = $5,
		    status = CASE WHEN $4 THEN 'pending' ELSE 'rejected' END,
		    reviewed_by = CASE WHEN $4 THEN reviewed_by ELSE $3 END,
		    reviewed_at = CASE WHEN $4 THEN reviewed_at ELSE now() END
		WHERE company_id = $1 AND id = $2 AND status = 'pending' AND stage = 'manager'`,
		companyID, leaveID, actorID, approve, note)
	if err != nil {
		return err
	}
	if rowsAffected(res) == 0 {
		return sql.ErrNoRows
	}
	if err = audit(ctx, tx, companyID, actorID, managerAction(approve), "leave", leaveID, "Keputusan atasan atas pengajuan cuti"); err != nil {
		return err
	}
	return tx.Commit()
}

// ManagerReviewHRItem forwards an overtime or correction request to HR
// (approve) or rejects it, guarded by the item version.
func (r *approvalRepository) ManagerReviewHRItem(ctx context.Context, companyID, managerID, actorID int64, module string, id int64,
	approve bool, note string, version int) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	employeeID, err := requesterOf(ctx, tx, "hr_items", companyID, managerID, id)
	if err != nil {
		return err
	}
	if err = lockEmployee(ctx, tx, companyID, employeeID); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE hr_items
		SET stage = 'hr', manager_reviewed_by = $4, manager_reviewed_at = now(), review_note = $6,
		    status = CASE WHEN $5 THEN 'pending' ELSE 'rejected' END,
		    reviewed_by = CASE WHEN $5 THEN reviewed_by ELSE $4 END,
		    version = version + 1, updated_at = now()
		WHERE company_id = $1 AND module = $2 AND id = $3 AND version = $7
		  AND status = 'pending' AND stage = 'manager'`,
		companyID, module, id, actorID, approve, note, version)
	if err != nil {
		return err
	}
	if rowsAffected(res) == 0 {
		return apperror.Conflict("Data sudah berubah. Muat ulang terlebih dahulu.")
	}
	if err = audit(ctx, tx, companyID, actorID, managerAction(approve), module, id, "Keputusan atasan"); err != nil {
		return err
	}
	return tx.Commit()
}

func managerAction(approve bool) string {
	if approve {
		return "manager_approve"
	}
	return "manager_reject"
}
