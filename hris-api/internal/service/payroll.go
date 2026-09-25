package service

import (
	"context"
	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"strings"
	"time"
)

func ValidateSalary(v dto.Salary) error {
	if v.BasicSalary < 0 || v.Allowance < 0 || v.Deduction < 0 || v.BasicSalary > 1000000000000 || v.Allowance > 1000000000000 || v.Deduction > 1000000000000 {
		return apperror.Invalid("Nominal harus rupiah bulat, antara 0 dan 1 triliun")
	}
	if v.Deduction > v.BasicSalary+v.Allowance {
		return apperror.Invalid("Potongan tidak boleh melebihi gaji bruto")
	}
	if len(v.Note) > 1000 {
		return apperror.Invalid("Catatan maksimal 1000 karakter")
	}
	return nil
}
func (s *Service) Salaries(ctx context.Context, c int64) ([]model.Salary, error) {
	return s.Repo.Salaries(ctx, c)
}
func (s *Service) SaveSalary(ctx context.Context, u *model.User, e int64, v dto.Salary) error {
	if err := ValidateSalary(v); err != nil {
		return err
	}
	return s.Repo.SaveSalary(ctx, u, e, v)
}
func (s *Service) PayrollRuns(ctx context.Context, c int64) ([]model.PayrollRun, error) {
	return s.Repo.PayrollRuns(ctx, c)
}
func (s *Service) CreatePayroll(ctx context.Context, u *model.User, period string) (int64, error) {
	d, err := time.Parse("2006-01", period)
	if err != nil || d.Year() < 2000 || d.Year() > 2200 {
		return 0, apperror.Invalid("Periode payroll harus YYYY-MM (2000–2200)")
	}
	return s.Repo.CreatePayroll(ctx, u, period+"-01")
}
func (s *Service) Payslips(ctx context.Context, u *model.User, run int64) ([]model.Payslip, error) {
	return s.Repo.Payslips(ctx, u, run)
}
func (s *Service) SavePayslip(ctx context.Context, u *model.User, id int64, v dto.Salary) error {
	if err := ValidateSalary(v); err != nil {
		return err
	}
	if v.Version < 1 {
		return apperror.Invalid("Versi slip wajib disertakan")
	}
	return s.Repo.SavePayslip(ctx, u, id, v)
}
func (s *Service) PayrollAction(ctx context.Context, u *model.User, id int64, v dto.PayrollAction) error {
	if v.Action != "finalize" && v.Action != "paid" && v.Action != "void" {
		return apperror.Invalid("Tindakan payroll tidak valid")
	}
	v.Reference = strings.TrimSpace(v.Reference)
	if len(v.Reference) > 150 || (v.Action == "paid" && len(v.Reference) < 5) {
		return apperror.Invalid("Referensi pembayaran manual wajib 5–150 karakter")
	}
	return s.Repo.PayrollAction(ctx, u, id, v)
}
