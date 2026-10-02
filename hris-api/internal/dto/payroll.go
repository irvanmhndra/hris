package dto

type SalaryComponent struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Amount  int64  `json:"amount"`
	Fixed   bool   `json:"fixed"`
	Taxable bool   `json:"taxable"`
}

type Salary struct {
	BasicSalary         int64             `json:"basic_salary"`
	PTKPStatus          string            `json:"ptkp_status"`
	TaxMethod           string            `json:"tax_method"`
	BPJSKesehatan       bool              `json:"bpjs_kesehatan"`
	BPJSKetenagakerjaan bool              `json:"bpjs_ketenagakerjaan"`
	BPJSPensiun         bool              `json:"bpjs_pensiun"`
	OvertimeEligible    bool              `json:"overtime_eligible"`
	NIK                 string            `json:"nik"`
	NPWP                string            `json:"npwp"`
	BPJSKesehatanNo     string            `json:"bpjs_kesehatan_number"`
	BPJSKetenagakerjaNo string            `json:"bpjs_ketenagakerjaan_number"`
	Note                string            `json:"note"`
	Components          []SalaryComponent `json:"components"`
}

type CreatePayroll struct {
	Period string `json:"period"`
	// THRDate (YYYY-MM-DD) adds THR to every slip, with service length
	// counted up to this date. Empty means no THR in this run.
	THRDate string `json:"thr_date"`
}

type PayslipLine struct {
	Kind    string `json:"kind"`
	Code    string `json:"code"`
	Name    string `json:"name"`
	Amount  int64  `json:"amount"`
	Taxable bool   `json:"taxable"`
	Fixed   bool   `json:"fixed"`
}

// Payslip adjusts a draft slip. Lines are the non-statutory earnings and
// deductions; BPJS and PPh 21 are recalculated from them.
type Payslip struct {
	Lines   []PayslipLine `json:"lines"`
	Note    string        `json:"note"`
	Version int           `json:"version"`
}

type PayrollAction struct {
	Action    string `json:"action"`
	Reference string `json:"reference"`
}
