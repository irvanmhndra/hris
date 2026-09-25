package dto

type Salary struct {
	BasicSalary int64  `json:"basic_salary"`
	Allowance   int64  `json:"allowance"`
	Deduction   int64  `json:"deduction"`
	Note        string `json:"note"`
	Version     int    `json:"version"`
}
type PayrollAction struct {
	Action    string `json:"action"`
	Reference string `json:"reference"`
}
