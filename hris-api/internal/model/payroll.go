package model

import "time"

// PayrollSettings are company-level BPJS parameters. Rates are hundredths of
// a percent (24 = 0.24%).
type PayrollSettings struct {
	JKKRate    int64 `db:"jkk_rate" json:"jkk_rate"`
	JPWageCap  int64 `db:"jp_wage_cap" json:"jp_wage_cap"`
	KesWageCap int64 `db:"kes_wage_cap" json:"kes_wage_cap"`
}

// SalaryComponent is a recurring allowance or deduction. Fixed allowances
// (tunjangan tetap) count towards the BPJS, overtime, and THR wage base.
type SalaryComponent struct {
	EmployeeID int64  `db:"employee_id" json:"-"`
	Kind       string `db:"kind" json:"kind"`
	Name       string `db:"name" json:"name"`
	Amount     int64  `db:"amount" json:"amount"`
	Fixed      bool   `db:"fixed" json:"fixed"`
	Taxable    bool   `db:"taxable" json:"taxable"`
}

type Salary struct {
	EmployeeID          int64             `db:"employee_id" json:"employee_id"`
	Name                string            `db:"name" json:"name"`
	Code                string            `db:"code" json:"code"`
	Configured          bool              `db:"configured" json:"configured"`
	BasicSalary         int64             `db:"basic_salary" json:"basic_salary"`
	PTKPStatus          string            `db:"ptkp_status" json:"ptkp_status"`
	TaxMethod           string            `db:"tax_method" json:"tax_method"`
	BPJSKesehatan       bool              `db:"bpjs_kesehatan" json:"bpjs_kesehatan"`
	BPJSKetenagakerjaan bool              `db:"bpjs_ketenagakerjaan" json:"bpjs_ketenagakerjaan"`
	BPJSPensiun         bool              `db:"bpjs_pensiun" json:"bpjs_pensiun"`
	OvertimeEligible    bool              `db:"overtime_eligible" json:"overtime_eligible"`
	Note                string            `db:"note" json:"note"`
	Components          []SalaryComponent `db:"-" json:"components"`
}

type PayrollRun struct {
	ID               int64     `db:"id" json:"id"`
	Period           string    `db:"period" json:"period"`
	Status           string    `db:"status" json:"status"`
	Employees        int       `db:"employees" json:"employees"`
	Total            int64     `db:"total" json:"total"`
	Tax              int64     `db:"tax" json:"tax"`
	EmployerCost     int64     `db:"employer_cost" json:"employer_cost"`
	THRDate          *string   `db:"thr_date" json:"thr_date"`
	PaymentReference string    `db:"payment_reference" json:"payment_reference"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}

type PayrollLine struct {
	EntryID int64  `db:"entry_id" json:"-"`
	Kind    string `db:"kind" json:"kind"`
	Code    string `db:"code" json:"code"`
	Name    string `db:"name" json:"name"`
	Amount  int64  `db:"amount" json:"amount"`
	Taxable bool   `db:"taxable" json:"taxable"`
	Fixed   bool   `db:"fixed" json:"fixed"`
}

// Payslip totals: allowance is every earning except basic salary; deduction
// is every deduction, including BPJS and PPh 21.
type Payslip struct {
	ID           int64         `db:"id" json:"id"`
	RunID        int64         `db:"run_id" json:"run_id"`
	EmployeeID   int64         `db:"employee_id" json:"employee_id"`
	EmployeeName string        `db:"employee_name" json:"employee_name"`
	EmployeeCode string        `db:"employee_code" json:"employee_code"`
	Position     string        `db:"position" json:"position"`
	Period       string        `db:"period" json:"period"`
	Status       string        `db:"status" json:"status"`
	BasicSalary  int64         `db:"basic_salary" json:"basic_salary"`
	Allowance    int64         `db:"allowance" json:"allowance"`
	Deduction    int64         `db:"deduction" json:"deduction"`
	Net          int64         `db:"net" json:"net"`
	PTKPStatus   string        `db:"ptkp_status" json:"ptkp_status"`
	TaxMethod    string        `db:"tax_method" json:"tax_method"`
	FinalPeriod  bool          `db:"final_period" json:"final_period"`
	WorkedDays   int           `db:"worked_days" json:"worked_days"`
	PeriodDays   int           `db:"period_days" json:"period_days"`
	UnpaidDays   int           `db:"unpaid_leave_days" json:"unpaid_leave_days"`
	TaxableGross int64         `db:"taxable_gross" json:"taxable_gross"`
	PPh21        int64         `db:"pph21" json:"pph21"`
	EmployerCost int64         `db:"employer_cost" json:"employer_cost"`
	Note         string        `db:"note" json:"note"`
	Version      int           `db:"version" json:"version"`
	Lines        []PayrollLine `db:"-" json:"lines"`
}

// YearToDate sums an employee's locked (finalized or paid), automatically
// taxed slips from earlier periods of the same year.
type YearToDate struct {
	Gross   int64 `db:"gross"`
	Pension int64 `db:"pension"`
	Tax     int64 `db:"tax"`
	Months  int   `db:"months"`
}

// OvertimeClaim is an approved overtime request not yet paid in a payroll.
type OvertimeClaim struct {
	ID      int64  `db:"id"`
	StartAt string `db:"start_at"`
	EndAt   string `db:"end_at"`
}

// PayrollEmployee is one employee included in a new run, with everything the
// calculation needs.
type PayrollEmployee struct {
	ID       int64
	Name     string
	Code     string
	Position string
	JoinedOn string
	LeftOn   *string
	Salary   Salary
	Overtime []OvertimeClaim
	YTD      YearToDate
	// UnpaidLeave lists approved unpaid-leave dates in the period.
	UnpaidLeave []string
}

// PayrollSource is read in one repeatable-read snapshot when a run is created.
type PayrollSource struct {
	Settings  PayrollSettings
	Workdays  []int64
	Holidays  []string
	Employees []PayrollEmployee
}

// PayslipContext is what a draft slip is recalculated with: the snapshot taken
// when the run was created plus the employee's year-to-date tax history.
type PayslipContext struct {
	Settings            PayrollSettings
	PTKPStatus          string `db:"ptkp_status"`
	TaxMethod           string `db:"tax_method"`
	BPJSKesehatan       bool   `db:"bpjs_kesehatan"`
	BPJSKetenagakerjaan bool   `db:"bpjs_ketenagakerjaan"`
	BPJSPensiun         bool   `db:"bpjs_pensiun"`
	FinalPeriod         bool   `db:"final_period"`
	YTD                 YearToDate
}

// PayrollEntryDraft is a calculated slip ready to be stored.
type PayrollEntryDraft struct {
	EmployeeID          int64
	Name                string
	Code                string
	Position            string
	PTKPStatus          string
	TaxMethod           string
	BPJSKesehatan       bool
	BPJSKetenagakerjaan bool
	BPJSPensiun         bool
	FinalPeriod         bool
	WorkedDays          int
	PeriodDays          int
	UnpaidLeaveDays     int
	BasicSalary         int64
	Allowance           int64
	Deduction           int64
	TaxableGross        int64
	Pension             int64
	PPh21               int64
	EmployerCost        int64
	Note                string
	Lines               []PayrollLine
	OvertimeIDs         []int64
}
