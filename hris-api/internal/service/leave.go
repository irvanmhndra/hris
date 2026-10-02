package service

import (
	"context"
	"strings"
	"time"

	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/leavepolicy"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
)

type LeaveService struct{ repo repository.LeaveRepository }

func NewLeaveService(repo repository.LeaveRepository) *LeaveService {
	return &LeaveService{repo: repo}
}

func (s *LeaveService) WorkCalendar(ctx context.Context, companyID int64) (model.WorkCalendar, error) {
	return s.repo.WorkCalendar(ctx, companyID)
}

func (s *LeaveService) SaveCalendar(ctx context.Context, u *model.User, v model.WorkCalendar) error {
	if len(v.Workdays) < 1 || len(v.Workdays) > 7 || v.AnnualAllowance < 0 || v.AnnualAllowance > 366 {
		return apperror.Invalid("Hari kerja atau kuota cuti tidak valid")
	}
	seen := map[int64]bool{}
	for _, d := range v.Workdays {
		if d < 0 || d > 6 || seen[d] {
			return apperror.Invalid("Hari kerja harus unik (0–6)")
		}
		seen[d] = true
	}
	if v.LeaveAccrual == "" {
		v.LeaveAccrual = leavepolicy.Annual
	}
	if v.LeaveAccrual != leavepolicy.Annual && v.LeaveAccrual != leavepolicy.Monthly {
		return apperror.Invalid("Akrual cuti harus tahunan atau bulanan")
	}
	if v.CarryOverMax < 0 || v.CarryOverMax > 366 || v.LeaveEligibilityMonths < 0 || v.LeaveEligibilityMonths > 24 {
		return apperror.Invalid("Carry-over maksimal 366 hari dan masa tunggu cuti 0–24 bulan")
	}
	start, e1 := time.Parse("15:04", v.StartTime)
	end, e2 := time.Parse("15:04", v.EndTime)
	if e1 != nil || e2 != nil || !end.After(start) {
		return apperror.Invalid("Jam kerja harus berada pada hari yang sama dan jam selesai setelah jam mulai")
	}
	return s.repo.SaveCalendar(ctx, u.CompanyID, u.ID, v)
}

func (s *LeaveService) Holidays(ctx context.Context, companyID int64) ([]model.Holiday, error) {
	return s.repo.Holidays(ctx, companyID)
}

func (s *LeaveService) SaveHoliday(ctx context.Context, u *model.User, v model.Holiday) error {
	if _, err := time.Parse("2006-01-02", v.Date); err != nil {
		return apperror.Invalid("Tanggal libur tidak valid")
	}
	v.Name = strings.TrimSpace(v.Name)
	if len(v.Name) < 2 || len(v.Name) > 120 {
		return apperror.Invalid("Nama hari libur harus 2–120 karakter")
	}
	return s.repo.SaveHoliday(ctx, u.CompanyID, u.ID, v)
}

func (s *LeaveService) DeleteHoliday(ctx context.Context, u *model.User, id int64) error {
	return s.repo.DeleteHoliday(ctx, u.CompanyID, u.ID, id)
}

func (s *LeaveService) Balances(ctx context.Context, u *model.User, year int) ([]model.Balance, error) {
	if year < 2000 || year > 2200 {
		return nil, apperror.Invalid("Tahun harus 2000–2200")
	}
	return s.repo.Balances(ctx, u.CompanyID, ownScope(u), year)
}

func (s *LeaveService) SetAllowance(ctx context.Context, u *model.User, employeeID int64, year, allowance int) error {
	if year < 2000 || year > 2200 || allowance < 0 || allowance > 366 {
		return apperror.Invalid("Tahun atau kuota tidak valid")
	}
	return s.repo.SetAllowance(ctx, u.CompanyID, u.ID, employeeID, year, allowance)
}

func (s *LeaveService) Leaves(ctx context.Context, u *model.User, q dto.ListQuery) (Page[model.Leave], error) {
	out, f, err := listFilter[model.Leave](q, 0, "pending", "approved", "rejected", "cancelled")
	if err != nil {
		return out, err
	}
	out.Items, out.Total, err = s.repo.Leaves(ctx, u.CompanyID, ownScope(u), f)
	return out, err
}

func ValidateLeave(v dto.Leave) error {
	start, e1 := time.Parse("2006-01-02", v.StartDate)
	end, e2 := time.Parse("2006-01-02", v.EndDate)
	if e1 != nil || e2 != nil || end.Before(start) {
		return apperror.Invalid("Rentang tanggal cuti tidak valid")
	}
	if end.Sub(start) > 365*24*time.Hour {
		return apperror.Invalid("Durasi cuti maksimal 366 hari")
	}
	if v.Kind != "annual" && v.Kind != "sick" && v.Kind != "personal" && v.Kind != "unpaid" {
		return apperror.Invalid("Jenis cuti tidak valid")
	}
	if len(strings.TrimSpace(v.Reason)) < 5 || len(v.Reason) > 1000 {
		return apperror.Invalid("Alasan harus 5–1000 karakter")
	}
	return nil
}

func (s *LeaveService) CreateLeave(ctx context.Context, u *model.User, v dto.Leave) error {
	if u.EmployeeID == nil {
		return apperror.Forbidden("Akses tidak diizinkan")
	}
	if err := ValidateLeave(v); err != nil {
		return err
	}
	return s.repo.CreateLeave(ctx, u.CompanyID, *u.EmployeeID, u.ID, model.LeaveInput{
		Kind: v.Kind, StartDate: v.StartDate, EndDate: v.EndDate, Reason: v.Reason,
	})
}

func (s *LeaveService) ReviewLeave(ctx context.Context, u *model.User, leaveID int64, status string) error {
	if status != "approved" && status != "rejected" {
		return apperror.Invalid("Keputusan tidak valid")
	}
	return s.repo.ReviewLeave(ctx, u.CompanyID, leaveID, u.ID, status)
}

func (s *LeaveService) CancelLeave(ctx context.Context, u *model.User, leaveID int64) error {
	if u.EmployeeID == nil {
		return apperror.Forbidden("Akses tidak diizinkan")
	}
	return s.repo.CancelLeave(ctx, u.CompanyID, *u.EmployeeID, u.ID, leaveID)
}
