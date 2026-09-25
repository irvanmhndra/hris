package service

import (
	"context"
	"strings"
	"time"

	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
)

type PayrollService struct{ repo repository.PayrollRepository }

func NewPayrollService(repo repository.PayrollRepository) *PayrollService {
	return &PayrollService{repo: repo}
}

// maxAmount caps every rupiah component at 1 trillion (whole rupiah, no fractions).
const maxAmount = 1_000_000_000_000

func ValidateSalary(v dto.Salary) error {
	if v.BasicSalary < 0 || v.Allowance < 0 || v.Deduction < 0 ||
		v.BasicSalary > maxAmount || v.Allowance > maxAmount || v.Deduction > maxAmount {
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

func salaryInput(v dto.Salary) model.SalaryInput {
	return model.SalaryInput{
		BasicSalary: v.BasicSalary, Allowance: v.Allowance, Deduction: v.Deduction,
		Note: v.Note, Version: v.Version,
	}
}

func (s *PayrollService) Salaries(ctx context.Context, companyID int64) ([]model.Salary, error) {
	return s.repo.Salaries(ctx, companyID)
}

func (s *PayrollService) SaveSalary(ctx context.Context, u *model.User, employeeID int64, v dto.Salary) error {
	if err := ValidateSalary(v); err != nil {
		return err
	}
	return s.repo.SaveSalary(ctx, u.CompanyID, u.ID, employeeID, salaryInput(v))
}

func (s *PayrollService) PayrollRuns(ctx context.Context, companyID int64) ([]model.PayrollRun, error) {
	return s.repo.PayrollRuns(ctx, companyID)
}

// CreatePayroll takes a "YYYY-MM" period and snapshots a draft run.
func (s *PayrollService) CreatePayroll(ctx context.Context, u *model.User, period string) (int64, error) {
	d, err := time.Parse("2006-01", period)
	if err != nil || d.Year() < 2000 || d.Year() > 2200 {
		return 0, apperror.Invalid("Periode payroll harus YYYY-MM (2000–2200)")
	}
	return s.repo.CreatePayroll(ctx, u.CompanyID, u.ID, period+"-01")
}

// Payslips: admins see every slip (optionally of one run); employees only
// their own finalized or paid slips.
func (s *PayrollService) Payslips(ctx context.Context, u *model.User, runID int64) ([]model.Payslip, error) {
	return s.repo.Payslips(ctx, u.CompanyID, ownScope(u), runID)
}

func (s *PayrollService) SavePayslip(ctx context.Context, u *model.User, entryID int64, v dto.Salary) error {
	if err := ValidateSalary(v); err != nil {
		return err
	}
	if v.Version < 1 {
		return apperror.Invalid("Versi slip wajib disertakan")
	}
	return s.repo.SavePayslip(ctx, u.CompanyID, u.ID, entryID, salaryInput(v))
}

func (s *PayrollService) PayrollAction(ctx context.Context, u *model.User, runID int64, v dto.PayrollAction) error {
	if v.Action != "finalize" && v.Action != "paid" && v.Action != "void" {
		return apperror.Invalid("Tindakan payroll tidak valid")
	}
	ref := strings.TrimSpace(v.Reference)
	if len(ref) > 150 || (v.Action == "paid" && len(ref) < 5) {
		return apperror.Invalid("Referensi pembayaran manual wajib 5–150 karakter")
	}
	return s.repo.PayrollAction(ctx, u.CompanyID, u.ID, runID, v.Action, ref)
}
