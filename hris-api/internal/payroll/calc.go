package payroll

// Line kinds. Employer lines are company costs shown on the slip for
// transparency; they never change the employee's net pay.
const (
	Earning   = "earning"
	Deduction = "deduction"
	Employer  = "employer"
)

// Line codes. Input codes come from salary setup or HR adjustments; statutory
// codes are always recomputed and cannot be submitted.
const (
	CodeBasic        = "BASIC"
	CodeAllowance    = "ALLOWANCE"
	CodeDeduction    = "DEDUCTION"
	CodeOvertime     = "OVERTIME"
	CodeTHR          = "THR"
	CodeAdjustment   = "ADJUSTMENT"
	CodeTaxAllowance = "TAX_ALLOWANCE"
	CodeTaxRefund    = "PPH21_REFUND"
	CodePPh21        = "PPH21"
	CodeKesEmployee  = "BPJS_KES_EE"
	CodeJHTEmployee  = "JHT_EE"
	CodeJPEmployee   = "JP_EE"
	CodeKesEmployer  = "BPJS_KES_ER"
	CodeJHTEmployer  = "JHT_ER"
	CodeJPEmployer   = "JP_ER"
	CodeJKK          = "JKK"
	CodeJKM          = "JKM"
)

// InputCodes are the line codes HR may submit when adjusting a draft slip.
var InputCodes = map[string]string{
	CodeBasic: Earning, CodeAllowance: Earning, CodeOvertime: Earning, CodeTHR: Earning,
	CodeAdjustment: "", CodeDeduction: Deduction,
}

// Tax methods. "none" leaves PPh 21 to HR (used for slips created before
// automatic calculation existed).
const (
	TaxGross   = "gross"    // employee bears the tax
	TaxGrossUp = "gross_up" // company pays a tax allowance equal to the tax
	TaxNone    = "none"
)

// JKKRates are the five work-accident risk classes (PP 44/2015).
var JKKRates = []int64{24, 54, 89, 127, 174}

// Statutory contribution rates in hundredths of a percent.
const (
	kesEmployerRate = 400
	kesEmployeeRate = 100
	jhtEmployerRate = 370
	jhtEmployeeRate = 200
	jpEmployerRate  = 200
	jpEmployeeRate  = 100
	jkmRate         = 30
)

// Settings are company-level parameters snapshotted on each payroll run.
type Settings struct {
	JKKRate    int64 // one of JKKRates
	JPWageCap  int64 // maximum wage for Jaminan Pensiun
	KesWageCap int64 // maximum wage for BPJS Kesehatan
}

// DefaultSettings match the published caps as of 2025; HR updates them when
// BPJS announces new ones.
var DefaultSettings = Settings{JKKRate: 24, JPWageCap: 10_547_400, KesWageCap: 12_000_000}

// Enrollment says which BPJS programmes apply to the employee.
type Enrollment struct {
	Kesehatan       bool // BPJS Kesehatan
	Ketenagakerjaan bool // JHT, JKK, JKM
	Pensiun         bool // JP
}

type Line struct {
	Kind    string `json:"kind"`
	Code    string `json:"code"`
	Name    string `json:"name"`
	Amount  int64  `json:"amount"`
	Taxable bool   `json:"taxable"` // earning counts towards PPh 21 gross
	Fixed   bool   `json:"fixed"`   // earning is fixed wage (BPJS, overtime, THR base)
}

type Input struct {
	Lines      []Line // earnings and deductions that are not statutory
	PTKP       string
	TaxMethod  string
	Enrollment Enrollment
	Settings   Settings
	// Final marks the employee's last tax period of the year (December, or
	// the month they leave): PPh 21 uses the annual Pasal 17 calculation.
	Final bool
	YTD   YearToDate
}

type Result struct {
	Lines        []Line
	Earnings     int64 // sum of earning lines
	Deductions   int64 // sum of deduction lines
	Net          int64
	TaxableGross int64 // PPh 21 gross, including employer benefit premiums
	Pension      int64 // employee JHT + JP, deductible in the annual calculation
	Tax          int64 // PPh 21 of this period; negative is a refund
	EmployerCost int64 // employer contributions on top of earnings
}

func pct(amount, rate int64) int64 { return (amount*rate + rateScale/2) / rateScale }

// Compute adds BPJS and PPh 21 lines to the input lines and totals the slip.
// Inputs are not mutated.
func Compute(in Input) Result {
	lines := make([]Line, 0, len(in.Lines)+12)
	var fixedWage, taxable int64
	for _, l := range in.Lines {
		lines = append(lines, l)
		if l.Kind == Earning {
			if l.Fixed {
				fixedWage += l.Amount
			}
			if l.Taxable {
				taxable += l.Amount
			}
		}
	}

	var employerLines, employeeLines []Line
	var benefit, pension int64 // taxable employer premiums; deductible employee pension
	add := func(dst *[]Line, kind, code, name string, amount int64) int64 {
		if amount > 0 {
			*dst = append(*dst, Line{Kind: kind, Code: code, Name: name, Amount: amount})
		}
		return amount
	}
	if in.Enrollment.Kesehatan {
		base := min(fixedWage, in.Settings.KesWageCap)
		benefit += add(&employerLines, Employer, CodeKesEmployer, "BPJS Kesehatan (perusahaan 4%)", pct(base, kesEmployerRate))
		add(&employeeLines, Deduction, CodeKesEmployee, "BPJS Kesehatan (1%)", pct(base, kesEmployeeRate))
	}
	if in.Enrollment.Ketenagakerjaan {
		add(&employerLines, Employer, CodeJHTEmployer, "JHT (perusahaan 3,7%)", pct(fixedWage, jhtEmployerRate))
		benefit += add(&employerLines, Employer, CodeJKK, "JKK (perusahaan)", pct(fixedWage, in.Settings.JKKRate))
		benefit += add(&employerLines, Employer, CodeJKM, "JKM (perusahaan 0,3%)", pct(fixedWage, jkmRate))
		pension += add(&employeeLines, Deduction, CodeJHTEmployee, "JHT (2%)", pct(fixedWage, jhtEmployeeRate))
	}
	if in.Enrollment.Pensiun {
		base := min(fixedWage, in.Settings.JPWageCap)
		add(&employerLines, Employer, CodeJPEmployer, "Jaminan Pensiun (perusahaan 2%)", pct(base, jpEmployerRate))
		pension += add(&employeeLines, Deduction, CodeJPEmployee, "Jaminan Pensiun (1%)", pct(base, jpEmployeeRate))
	}

	taxFor := func(extra int64) int64 {
		gross := taxable + extra + benefit
		if in.Final {
			return AnnualTax(in.PTKP, in.YTD, gross, pension)
		}
		return TERTax(in.PTKP, gross)
	}
	var tax, allowance int64
	if in.TaxMethod != TaxNone {
		if in.TaxMethod == TaxGrossUp {
			// Tax is non-decreasing in the allowance and grows by at most 35%
			// of it, so iterating from zero converges to the smallest
			// allowance that covers its own tax.
			for range 200 {
				next := max(taxFor(allowance), 0)
				if next <= allowance {
					break
				}
				allowance = next
			}
			add(&lines, Earning, CodeTaxAllowance, "Tunjangan PPh 21", allowance)
			if allowance > 0 {
				lines[len(lines)-1].Taxable = true
			}
		}
		tax = taxFor(allowance)
		if tax > 0 {
			add(&employeeLines, Deduction, CodePPh21, "PPh 21", tax)
		} else if tax < 0 {
			add(&lines, Earning, CodeTaxRefund, "Pengembalian kelebihan PPh 21", -tax)
		}
	}

	lines = append(lines, employeeLines...)
	lines = append(lines, employerLines...)
	r := Result{Lines: lines, Tax: tax, Pension: pension, TaxableGross: taxable + allowance + benefit}
	if in.TaxMethod == TaxNone {
		r.TaxableGross, r.Tax = 0, 0
	}
	for _, l := range lines {
		switch l.Kind {
		case Earning:
			r.Earnings += l.Amount
		case Deduction:
			r.Deductions += l.Amount
		case Employer:
			r.EmployerCost += l.Amount
		}
	}
	r.Net = r.Earnings - r.Deductions
	return r
}
