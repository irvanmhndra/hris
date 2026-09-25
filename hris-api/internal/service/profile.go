package service

import (
	"context"

	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
)

type ProfileService struct{ repo repository.ProfileRepository }

func NewProfileService(repo repository.ProfileRepository) *ProfileService {
	return &ProfileService{repo: repo}
}

// profileTarget: an employee may only ever address their own profile; an
// admin addresses the employee named in the route.
func profileTarget(u *model.User, employeeID int64) (int64, error) {
	if u.Role == model.RoleEmployee {
		if u.EmployeeID == nil {
			return 0, apperror.Forbidden("Akses tidak diizinkan")
		}
		return *u.EmployeeID, nil
	}
	if employeeID <= 0 {
		return 0, apperror.Invalid("ID tidak valid")
	}
	return employeeID, nil
}

func (s *ProfileService) Profile(ctx context.Context, u *model.User, employeeID int64) (model.Profile, error) {
	id, err := profileTarget(u, employeeID)
	if err != nil {
		return model.Profile{}, err
	}
	return s.repo.Profile(ctx, u.CompanyID, id)
}

func (s *ProfileService) SaveProfile(ctx context.Context, u *model.User, employeeID int64, v model.Profile) error {
	id, err := profileTarget(u, employeeID)
	if err != nil {
		return err
	}
	if len(v.Phone) > 30 || len(v.Address) > 500 || len(v.EmergencyName) > 120 ||
		len(v.EmergencyPhone) > 30 || len(v.EmergencyRelation) > 80 {
		return apperror.Invalid("Data kontak terlalu panjang")
	}
	return s.repo.SaveProfile(ctx, u.CompanyID, u.ID, id, v)
}
