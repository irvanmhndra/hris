package service

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/payroll"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
)

type PayrollService struct{ repo repository.PayrollRepository }

func NewPayrollService(repo repository.PayrollRepository) *PayrollService {
	return &PayrollService{repo: repo}
}

// maxAmount caps every rupiah amount at 1 trillion (whole rupiah, no fractions).
const maxAmount = 1_000_000_000_000

// wib is Asia/Jakarta without depending on the host's tz database.
var wib = time.FixedZone("WIB", 7*3600)

func validAmount(v int64) bool { return v >= 0 && v <= maxAmount }

func ValidateSalary(v *dto.Salary) error {
	if !validAmount(v.BasicSalary) {
		return apperror.Invalid("Nominal harus rupiah bulat, antara 0 dan 1 triliun")
	}
	if _, ok := payroll.PTKP(v.PTKPStatus); !ok {
		return apperror.Invalid("Status PTKP tidak valid")
	}
	if v.TaxMethod != payroll.TaxGross && v.TaxMethod != payroll.TaxGrossUp && v.TaxMethod != payroll.TaxNone {
		return apperror.Invalid("Metode pajak tidak valid")
	}
	digits := func(s string, lengths ...int) bool {
		if s == "" {
			return true
		}
		for _, r := range s {
			if r < '0' || r > '9' {
				return false
			}
		}
		return len(lengths) == 0 || slices.Contains(lengths, len(s))
	}
	for _, f := range []*string{&v.NIK, &v.NPWP, &v.BPJSKesehatanNo, &v.BPJSKetenagakerjaNo} {
		*f = strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(*f), ".", ""), "-", "")
	}
	if !digits(v.NIK, 16) || !digits(v.NPWP, 15, 16) || !digits(v.BPJSKesehatanNo) || !digits(v.BPJSKetenagakerjaNo) ||
		len(v.BPJSKesehatanNo) > 20 || len(v.BPJSKetenagakerjaNo) > 20 ||
		(v.BPJSKesehatanNo != "" && len(v.BPJSKesehatanNo) < 8) || (v.BPJSKetenagakerjaNo != "" && len(v.BPJSKetenagakerjaNo) < 8) {
		return apperror.Invalid("NIK 16 digit, NPWP 15/16 digit, nomor BPJS 8–20 digit (boleh dikosongkan)")
	}
	if v.BPJSPensiun && !v.BPJSKetenagakerjaan {
		return apperror.Invalid("Jaminan Pensiun membutuhkan kepesertaan BPJS Ketenagakerjaan")
	}
	if len(v.Note) > 1000 {
		return apperror.Invalid("Catatan maksimal 1000 karakter")
	}
	if len(v.Components) > 30 {
		return apperror.Invalid("Maksimal 30 komponen gaji")
	}
	var earnings, deductions int64 = v.BasicSalary, 0
	for i := range v.Components {
		c := &v.Components[i]
		c.Name = strings.TrimSpace(c.Name)
		if c.Name == "" || len(c.Name) > 80 || !validAmount(c.Amount) {
			return apperror.Invalid("Setiap komponen wajib bernama (maks. 80 karakter) dengan nominal 0–1 triliun")
		}
		switch c.Kind {
		case "allowance":
			earnings += c.Amount
		case "deduction":
			c.Fixed, c.Taxable = false, true
			deductions += c.Amount
		default:
			return apperror.Invalid("Jenis komponen harus tunjangan atau potongan")
		}
	}
	if deductions > earnings {
		return apperror.Invalid("Potongan tidak boleh melebihi gaji bruto")
	}
	return nil
}

func (s *PayrollService) Salaries(ctx context.Context, companyID int64) ([]model.Salary, error) {
	return s.repo.Salaries(ctx, companyID)
}

func (s *PayrollService) SaveSalary(ctx context.Context, u *model.User, employeeID int64, v dto.Salary) error {
	if err := ValidateSalary(&v); err != nil {
		return err
	}
	in := model.SalaryInput{
		BasicSalary: v.BasicSalary, PTKPStatus: v.PTKPStatus, TaxMethod: v.TaxMethod,
		BPJSKesehatan: v.BPJSKesehatan, BPJSKetenagakerjaan: v.BPJSKetenagakerjaan,
		BPJSPensiun: v.BPJSPensiun, OvertimeEligible: v.OvertimeEligible, Note: v.Note,
		NIK: v.NIK, NPWP: v.NPWP, BPJSKesehatanNo: v.BPJSKesehatanNo, BPJSKetenagakerjaNo: v.BPJSKetenagakerjaNo,
	}
	for _, c := range v.Components {
		in.Components = append(in.Components, model.SalaryComponent{
			Kind: c.Kind, Name: c.Name, Amount: c.Amount, Fixed: c.Fixed, Taxable: c.Taxable,
		})
	}
	return s.repo.SaveSalary(ctx, u.CompanyID, u.ID, employeeID, in)
}

func (s *PayrollService) Settings(ctx context.Context, companyID int64) (model.PayrollSettings, error) {
	return s.repo.Settings(ctx, companyID)
}

func (s *PayrollService) SaveSettings(ctx context.Context, u *model.User, v model.PayrollSettings) error {
	if !slices.Contains(payroll.JKKRates, v.JKKRate) {
		return apperror.Invalid("Tarif JKK harus salah satu kelas risiko: 0,24%, 0,54%, 0,89%, 1,27%, atau 1,74%")
	}
	if !validAmount(v.JPWageCap) || !validAmount(v.KesWageCap) {
		return apperror.Invalid("Batas upah BPJS harus 0–1 triliun")
	}
	switch v.LateDeduction {
	case "", "none":
		v.LateDeduction, v.LateDeductionAmount = "none", 0
	case "per_minute":
		v.LateDeductionAmount = 0
	case "per_occurrence":
		if v.LateDeductionAmount < 1 || v.LateDeductionAmount > 1_000_000_000 {
			return apperror.Invalid("Potongan per keterlambatan harus Rp1–Rp1 miliar")
		}
	default:
		return apperror.Invalid("Aturan potongan keterlambatan tidak valid")
	}
	return s.repo.SaveSettings(ctx, u.CompanyID, u.ID, v)
}

func (s *PayrollService) PayrollRuns(ctx context.Context, companyID int64) ([]model.PayrollRun, error) {
	return s.repo.PayrollRuns(ctx, companyID)
}

// CreatePayroll takes a "YYYY-MM" period (and an optional THR date) and
// calculates a draft run.
func (s *PayrollService) CreatePayroll(ctx context.Context, u *model.User, v dto.CreatePayroll) (int64, error) {
	start, err := time.Parse("2006-01", v.Period)
	if err != nil || start.Year() < 2000 || start.Year() > 2200 {
		return 0, apperror.Invalid("Periode payroll harus YYYY-MM (2000–2200)")
	}
	var thr *time.Time
	var thrDate *string
	if v.THRDate != "" {
		d, err := time.Parse(time.DateOnly, v.THRDate)
		if err != nil || d.Year() != start.Year() {
			return 0, apperror.Invalid("Tanggal hari raya untuk THR harus valid dan pada tahun periode payroll")
		}
		thr, thrDate = &d, &v.THRDate
	}
	return s.repo.CreatePayroll(ctx, u.CompanyID, u.ID, start.Format(time.DateOnly), thrDate,
		func(src model.PayrollSource) ([]model.PayrollEntryDraft, error) {
			return BuildPayroll(src, start, thr)
		})
}

func settingsOf(v model.PayrollSettings) payroll.Settings {
	return payroll.Settings{JKKRate: v.JKKRate, JPWageCap: v.JPWageCap, KesWageCap: v.KesWageCap}
}

func ytdOf(v model.YearToDate) payroll.YearToDate {
	return payroll.YearToDate{Gross: v.Gross, Pension: v.Pension, Tax: v.Tax, Months: v.Months}
}

// BuildPayroll calculates every slip of a new run: prorated salary for joiners
// and leavers, approved overtime, THR, BPJS, and PPh 21.
func BuildPayroll(src model.PayrollSource, start time.Time, thrDate *time.Time) ([]model.PayrollEntryDraft, error) {
	end := start.AddDate(0, 1, -1)
	cal := payroll.NewCalendar(src.Workdays, src.Holidays)
	attendance := map[int64]model.AttendanceSummary{}
	if through, err := time.Parse(time.DateOnly, src.Through); err == nil && !through.Before(start) {
		for _, a := range summarize(src.Schedule, start, through) {
			attendance[a.EmployeeID] = a
		}
	}
	periodDays := cal.Workdays(start, end)
	sixDayWeek := len(src.Workdays) >= 6
	drafts := make([]model.PayrollEntryDraft, 0, len(src.Employees))
	for _, emp := range src.Employees {
		joined, err := time.Parse(time.DateOnly, emp.JoinedOn)
		if err != nil {
			return nil, err
		}
		from, to := start, end
		if joined.After(from) {
			from = joined
		}
		final := start.Month() == time.December
		if emp.LeftOn != nil {
			left, err := time.Parse(time.DateOnly, *emp.LeftOn)
			if err != nil {
				return nil, err
			}
			if !left.After(end) {
				to, final = left, true
			}
		}
		// Approved unpaid leave on scheduled working days is not paid.
		unpaid := 0
		for _, day := range emp.UnpaidLeave {
			dt, err := time.Parse(time.DateOnly, day)
			if err != nil {
				return nil, err
			}
			if !dt.Before(from) && !dt.After(to) && cal.IsWorkday(dt) {
				unpaid++
			}
		}
		att := attendance[emp.ID]
		absent := 0
		if src.Settings.DeductAbsence {
			absent = att.AbsentDays
		}
		worked := max(cal.Workdays(from, to)-unpaid-absent, 0)
		sal := emp.Salary

		// Fixed monthly wage (basic + fixed allowances) is the base for
		// overtime and THR; earnings themselves are prorated.
		wage := sal.BasicSalary
		lines := []payroll.Line{{
			Kind: payroll.Earning, Code: payroll.CodeBasic, Name: "Gaji pokok",
			Amount: payroll.Prorate(sal.BasicSalary, worked, periodDays), Taxable: true, Fixed: true,
		}}
		var deductions []payroll.Line
		for _, c := range sal.Components {
			if c.Kind == "deduction" {
				deductions = append(deductions, payroll.Line{
					Kind: payroll.Deduction, Code: payroll.CodeDeduction, Name: c.Name, Amount: c.Amount,
				})
				continue
			}
			if c.Fixed {
				wage += c.Amount
			}
			lines = append(lines, payroll.Line{
				Kind: payroll.Earning, Code: payroll.CodeAllowance, Name: c.Name,
				Amount: payroll.Prorate(c.Amount, worked, periodDays), Taxable: c.Taxable, Fixed: c.Fixed,
			})
		}

		var overtimeIDs []int64
		if sal.OvertimeEligible && len(emp.Overtime) > 0 {
			var pay int64
			var minutes int
			for _, o := range emp.Overtime {
				a, e1 := time.Parse(time.RFC3339, o.StartAt)
				b, e2 := time.Parse(time.RFC3339, o.EndAt)
				if e1 != nil || e2 != nil || !b.After(a) {
					return nil, fmt.Errorf("overtime %d has invalid times", o.ID)
				}
				m := int(b.Sub(a).Minutes())
				day := a.In(wib)
				date := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
				pay += payroll.OvertimePay(wage, m, !cal.IsWorkday(date), sixDayWeek)
				minutes += m
				overtimeIDs = append(overtimeIDs, o.ID)
			}
			lines = append(lines, payroll.Line{
				Kind: payroll.Earning, Code: payroll.CodeOvertime,
				Name:   fmt.Sprintf("Lembur %d sesi (%dj %02dm)", len(emp.Overtime), minutes/60, minutes%60),
				Amount: pay, Taxable: true,
			})
		}
		// Lateness is deducted as a separate line HR can adjust on the draft.
		var late int64
		switch src.Settings.LateDeduction {
		case "per_minute":
			late = (wage*int64(att.LateMinutes) + payroll.HourlyDivisor*30) / (payroll.HourlyDivisor * 60)
		case "per_occurrence":
			late = src.Settings.LateDeductionAmount * int64(att.LateDays)
		}
		if late > 0 {
			deductions = append(deductions, payroll.Line{
				Kind: payroll.Deduction, Code: payroll.CodeLate,
				Name:   fmt.Sprintf("Potongan keterlambatan (%d kali, %d menit)", att.LateDays, att.LateMinutes),
				Amount: late,
			})
		}
		if thrDate != nil {
			if amount := payroll.THR(wage, joined, *thrDate); amount > 0 {
				lines = append(lines, payroll.Line{
					Kind: payroll.Earning, Code: payroll.CodeTHR, Name: "THR Keagamaan", Amount: amount, Taxable: true,
				})
			}
		}

		d := model.PayrollEntryDraft{
			EmployeeID: emp.ID, Name: emp.Name, Code: emp.Code, Position: emp.Position,
			PTKPStatus: sal.PTKPStatus, TaxMethod: sal.TaxMethod, BPJSKesehatan: sal.BPJSKesehatan,
			BPJSKetenagakerjaan: sal.BPJSKetenagakerjaan, BPJSPensiun: sal.BPJSPensiun,
			FinalPeriod: final, WorkedDays: worked, PeriodDays: periodDays, UnpaidLeaveDays: unpaid, Note: sal.Note,
			AbsentDays: absent, LateCount: att.LateDays, LateMinutes: att.LateMinutes, NIK: sal.NIK, NPWP: sal.NPWP,
			OvertimeIDs: overtimeIDs,
		}
		if err := calculate(&d, append(lines, deductions...), settingsOf(src.Settings), ytdOf(emp.YTD), payroll.YearToDate{}); err != nil {
			return nil, err
		}
		drafts = append(drafts, d)
	}
	return drafts, nil
}

// calculate runs the statutory calculation on a slip's input lines and fills
// its totals and stored lines.
func calculate(d *model.PayrollEntryDraft, lines []payroll.Line, s payroll.Settings, ytd, base payroll.YearToDate) error {
	r := payroll.Compute(payroll.Input{
		Lines: lines, PTKP: d.PTKPStatus, TaxMethod: d.TaxMethod, Settings: s, Final: d.FinalPeriod, YTD: ytd, Base: base,
		Enrollment: payroll.Enrollment{
			Kesehatan: d.BPJSKesehatan, Ketenagakerjaan: d.BPJSKetenagakerjaan, Pensiun: d.BPJSPensiun,
		},
	})
	if r.Net < 0 {
		return apperror.Invalid(fmt.Sprintf("Potongan slip %s melebihi pendapatan", d.Name))
	}
	if r.Earnings > maxAmount {
		return apperror.Invalid("Total pendapatan slip melebihi batas 1 triliun")
	}
	d.Lines = d.Lines[:0]
	d.BasicSalary = 0
	for _, l := range r.Lines {
		if l.Code == payroll.CodeBasic {
			d.BasicSalary += l.Amount
		}
		d.Lines = append(d.Lines, model.PayrollLine{
			Kind: l.Kind, Code: l.Code, Name: l.Name, Amount: l.Amount, Taxable: l.Taxable, Fixed: l.Fixed,
		})
	}
	d.Allowance = r.Earnings - d.BasicSalary
	d.Deduction = r.Deductions
	d.TaxableGross, d.Pension, d.PPh21, d.EmployerCost = r.TaxableGross, r.Pension, r.Tax, r.EmployerCost
	return nil
}

// Payslips: admins see every slip (optionally of one run); employees only
// their own finalized or paid slips.
func (s *PayrollService) Payslips(ctx context.Context, u *model.User, runID int64) ([]model.Payslip, error) {
	v, err := s.repo.Payslips(ctx, u.CompanyID, ownScope(u), runID)
	for i := range v {
		if p := &v[i]; p.TaxMethod != payroll.TaxNone && !p.FinalPeriod && p.RunKind == "regular" {
			p.TERRate = payroll.TERRate(p.PTKPStatus, p.TaxableGross)
		}
	}
	return v, err
}

// CreateCorrection opens a correction run for a finalized regular run.
func (s *PayrollService) CreateCorrection(ctx context.Context, u *model.User, runID int64) (int64, error) {
	return s.repo.CreateCorrection(ctx, u.CompanyID, u.ID, runID)
}

// ValidatePayslip normalises HR's adjustment lines. Only non-statutory codes
// are accepted; BPJS and PPh 21 are always recalculated.
func ValidatePayslip(v *dto.Payslip) ([]payroll.Line, error) {
	if v.Version < 1 {
		return nil, apperror.Invalid("Versi slip wajib disertakan")
	}
	if len(v.Note) > 1000 {
		return nil, apperror.Invalid("Catatan maksimal 1000 karakter")
	}
	if len(v.Lines) > 60 {
		return nil, apperror.Invalid("Slip maksimal 60 baris")
	}
	lines := make([]payroll.Line, 0, len(v.Lines))
	for _, l := range v.Lines {
		kind, ok := payroll.InputCodes[l.Code]
		if !ok || (l.Kind != payroll.Earning && l.Kind != payroll.Deduction) || (kind != "" && kind != l.Kind) {
			return nil, apperror.Invalid("Baris slip tidak valid; BPJS dan PPh 21 dihitung otomatis")
		}
		name := strings.TrimSpace(l.Name)
		if name == "" || len(name) > 80 || !validAmount(l.Amount) {
			return nil, apperror.Invalid("Setiap baris wajib bernama (maks. 80 karakter) dengan nominal 0–1 triliun")
		}
		line := payroll.Line{Kind: l.Kind, Code: l.Code, Name: name, Amount: l.Amount}
		if l.Kind == payroll.Earning {
			line.Taxable, line.Fixed = l.Taxable, l.Fixed
		}
		lines = append(lines, line)
	}
	// Earnings first, then deductions, matching calculated slips.
	slices.SortStableFunc(lines, func(a, b payroll.Line) int {
		return strings.Compare(b.Kind, a.Kind)
	})
	return lines, nil
}

func (s *PayrollService) SavePayslip(ctx context.Context, u *model.User, entryID int64, v dto.Payslip) error {
	lines, err := ValidatePayslip(&v)
	if err != nil {
		return err
	}
	return s.repo.SavePayslip(ctx, u.CompanyID, u.ID, entryID, v.Version, v.Note,
		func(c model.PayslipContext) (model.PayrollEntryDraft, error) {
			d := model.PayrollEntryDraft{
				Name: "ini", PTKPStatus: c.PTKPStatus, TaxMethod: c.TaxMethod, BPJSKesehatan: c.BPJSKesehatan,
				BPJSKetenagakerjaan: c.BPJSKetenagakerjaan, BPJSPensiun: c.BPJSPensiun, FinalPeriod: c.FinalPeriod,
			}
			err := calculate(&d, lines, settingsOf(c.Settings), ytdOf(c.YTD), ytdOf(c.Base))
			return d, err
		})
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
