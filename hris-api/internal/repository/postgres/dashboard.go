package postgres

import (
	"context"

	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/jmoiron/sqlx"
)

type dashboardRepository struct{ db *sqlx.DB }

func NewDashboardRepository(db *sqlx.DB) repository.DashboardRepository {
	return &dashboardRepository{db: db}
}

func (r *dashboardRepository) Dashboard(ctx context.Context, companyID int64) (model.Dashboard, error) {
	var v model.Dashboard
	err := r.db.GetContext(ctx, &v, `
		SELECT
		  (SELECT count(*) FROM employees WHERE company_id = $1 AND status = 'active') employees,
		  (SELECT count(*) FROM attendances
		   WHERE company_id = $1 AND date = (now() AT TIME ZONE 'Asia/Jakarta')::date) present,
		  (SELECT count(*) FROM leave_requests WHERE company_id = $1 AND status = 'pending') pending,
		  (SELECT count(*) FROM departments WHERE company_id = $1) departments`,
		companyID)
	return v, err
}
