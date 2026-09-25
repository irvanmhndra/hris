package postgres

import (
	"context"

	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/jmoiron/sqlx"
)

type auditRepository struct{ db *sqlx.DB }

func NewAuditRepository(db *sqlx.DB) repository.AuditRepository {
	return &auditRepository{db: db}
}

func (r *auditRepository) AuditLogs(ctx context.Context, companyID int64, limit, offset int) ([]model.AuditLog, int, error) {
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT count(*) FROM audit_logs WHERE company_id = $1`, companyID); err != nil {
		return nil, 0, err
	}
	v := []model.AuditLog{}
	err := r.db.SelectContext(ctx, &v, `
		SELECT a.id, u.name actor, a.action, a.resource, a.resource_id, a.summary, a.created_at
		FROM audit_logs a
		JOIN users u ON u.id = a.actor_id
		WHERE a.company_id = $1
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT $2 OFFSET $3`,
		companyID, limit, offset)
	return v, total, err
}
