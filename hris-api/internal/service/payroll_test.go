package service

import (
	"testing"
	"time"

	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/payroll"
)

func TestSalaryValidation(t *testing.T) {
	ok := func() dto.Salary {
		return dto.Salary{BasicSalary: 100, PTKPStatus: "TK/0", TaxMethod: "gross", BPJSKetenagakerjaan: true, BPJSPensiun: true,
			Components: []dto.SalaryComponent{{Kind: "allowance", Name: "Makan", Amount: 50}, {Kind: "deduction", Name: "Koperasi", Amount: 150}}}
	}
	bad := []func(*dto.Salary){
		func(v *dto.Salary) { v.BasicSalary = -1 },
		func(v *dto.Salary) { v.BasicSalary = 1_000_000_000_001 },
		func(v *dto.Salary) { v.Components[1].Amount = 151 },
		func(v *dto.Salary) { v.PTKPStatus = "K/4" },
		func(v *dto.Salary) { v.TaxMethod = "net" },
		func(v *dto.Salary) { v.BPJSKetenagakerjaan = false },
		func(v *dto.Salary) { v.Components[0].Name = "  " },
		func(v *dto.Salary) { v.Components[0].Kind = "bonus" },
	}
	for i, mutate := range bad {
		v := ok()
		mutate(&v)
		if ValidateSalary(&v) == nil {
			t.Fatalf("case %d: accepted invalid salary %+v", i, v)
		}
	}
	v := ok()
	v.Components[1].Fixed = true
	if err := ValidateSalary(&v); err != nil || v.Components[1].Fixed {
		t.Fatalf("valid salary rejected or deduction kept fixed: %v", err)
	}
}

func TestPayslipValidationRejectsStatutoryLines(t *testing.T) {
	for _, l := range []dto.PayslipLine{
		{Kind: "deduction", Code: "PPH21", Name: "PPh 21", Amount: 1},
		{Kind: "employer", Code: "ADJUSTMENT", Name: "x", Amount: 1},
		{Kind: "deduction", Code: "BASIC", Name: "Gaji", Amount: 1},
		{Kind: "earning", Code: "ADJUSTMENT", Name: "", Amount: 1},
	} {
		if _, err := ValidatePayslip(&dto.Payslip{Version: 1, Lines: []dto.PayslipLine{l}}); err == nil {
			t.Fatalf("accepted %+v", l)
		}
	}
	lines, err := ValidatePayslip(&dto.Payslip{Version: 1, Lines: []dto.PayslipLine{
		{Kind: "deduction", Code: "ADJUSTMENT", Name: "Kasbon", Amount: 100, Taxable: true},
		{Kind: "earning", Code: "ADJUSTMENT", Name: "Bonus", Amount: 500, Taxable: true},
	}})
	if err != nil || lines[0].Kind != payroll.Earning || lines[1].Taxable {
		t.Fatalf("lines not normalised: %+v %v", lines, err)
	}
}

func TestBuildPayroll(t *testing.T) {
	left := "2026-03-13"
	salary := func(basic int64) model.Salary {
		return model.Salary{BasicSalary: basic, PTKPStatus: "TK/0", TaxMethod: "gross", OvertimeEligible: true,
			Components: []model.SalaryComponent{{Kind: "allowance", Name: "Jabatan", Amount: 1_730_000, Fixed: true, Taxable: true}}}
	}
	src := model.PayrollSource{
		Settings: model.PayrollSettings{JKKRate: 24, JPWageCap: 10_547_400, KesWageCap: 12_000_000},
		Workdays: []int64{1, 2, 3, 4, 5},
		Holidays: []string{"2026-03-20"},
		Employees: []model.PayrollEmployee{
			{ID: 1, Name: "Full", JoinedOn: "2024-01-05", Salary: salary(0), Overtime: []model.OvertimeClaim{
				{ID: 7, StartAt: "2026-03-02T11:00:00Z", EndAt: "2026-03-02T13:00:00Z"}, // Monday 18:00–20:00 WIB
				{ID: 8, StartAt: "2026-03-20T02:00:00Z", EndAt: "2026-03-20T03:00:00Z"}, // holiday
			}, UnpaidLeave: []string{"2026-03-03", "2026-03-20"}}, // a workday and a holiday
			{ID: 2, Name: "Joiner", JoinedOn: "2026-03-16", Salary: salary(8_770_000)},
			{ID: 3, Name: "Leaver", JoinedOn: "2025-11-01", LeftOn: &left, Salary: salary(8_770_000)},
		},
	}
	thr := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	drafts, err := BuildPayroll(src, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), &thr)
	if err != nil {
		t.Fatal(err)
	}
	line := func(d model.PayrollEntryDraft, code string) int64 {
		var v int64
		for _, l := range d.Lines {
			if l.Code == code {
				v += l.Amount
			}
		}
		return v
	}
	full, joiner, leaver := drafts[0], drafts[1], drafts[2]
	// Wage Rp1,730,000 → Rp10,000/hour: workday 2h = 1.5h+2h = 35,000; holiday 1h = 20,000.
	if line(full, payroll.CodeOvertime) != 55_000 || len(full.OvertimeIDs) != 2 {
		t.Fatalf("overtime = %d ids %v", line(full, payroll.CodeOvertime), full.OvertimeIDs)
	}
	if line(full, payroll.CodeTHR) != 1_730_000 || full.PeriodDays != 21 || full.WorkedDays != 20 || full.UnpaidLeaveDays != 1 || full.FinalPeriod {
		t.Fatalf("full-month employee: %+v", full)
	}
	// Joined Mon 16 March: 11 of 21 working days (20 March is a holiday).
	if joiner.WorkedDays != 11 || line(joiner, payroll.CodeBasic) != 4_593_810 || line(joiner, payroll.CodeTHR) != 0 {
		t.Fatalf("joiner: worked %d basic %d thr %d", joiner.WorkedDays, line(joiner, payroll.CodeBasic), line(joiner, payroll.CodeTHR))
	}
	// Leaver: 10 working days, last tax period of the year, THR 4/12 of wage.
	if leaver.WorkedDays != 10 || !leaver.FinalPeriod || line(leaver, payroll.CodeTHR) != 10_500_000*4/12 {
		t.Fatalf("leaver: %+v", leaver)
	}
	for _, d := range drafts {
		if d.BasicSalary+d.Allowance-d.Deduction < 0 || d.PPh21 != line(d, payroll.CodePPh21)-line(d, payroll.CodeTaxRefund) {
			t.Fatalf("inconsistent totals: %+v", d)
		}
	}
}
