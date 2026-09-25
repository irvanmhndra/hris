package service

import (
	"context"

	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
)

type DashboardService struct {
	repo repository.DashboardRepository
}

func NewDashboardService(repo repository.DashboardRepository) *DashboardService {
	return &DashboardService{repo: repo}
}

func (s *DashboardService) Dashboard(ctx context.Context, companyID int64) (model.Dashboard, error) {
	return s.repo.Dashboard(ctx, companyID)
}
