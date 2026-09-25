package service

import (
	"context"

	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
)

type AttendanceService struct {
	repo repository.AttendanceRepository
}

func NewAttendanceService(repo repository.AttendanceRepository) *AttendanceService {
	return &AttendanceService{repo: repo}
}

// Attendances returns the latest records: company-wide for admins, own for employees.
func (s *AttendanceService) Attendances(ctx context.Context, u *model.User) ([]model.Attendance, error) {
	return s.repo.Attendances(ctx, u.CompanyID, ownScope(u))
}

func (s *AttendanceService) Clock(ctx context.Context, u *model.User, action string) error {
	if action != "in" && action != "out" {
		return apperror.Invalid("Tindakan tidak valid")
	}
	if u.EmployeeID == nil {
		return apperror.Forbidden("Akses tidak diizinkan")
	}
	return s.repo.Clock(ctx, u.CompanyID, *u.EmployeeID, action == "out")
}
