package model

import "time"

type Salary struct {
	EmployeeID  int64  `db:"employee_id" json:"employee_id"`
	Name        string `db:"name" json:"name"`
	Code        string `db:"code" json:"code"`
	Configured  bool   `db:"configured" json:"configured"`
	BasicSalary int64  `db:"basic_salary" json:"basic_salary"`
	Allowance   int64  `db:"allowance" json:"allowance"`
	Deduction   int64  `db:"deduction" json:"deduction"`
	Note        string `db:"note" json:"note"`
}
type PayrollRun struct {
	ID               int64     `db:"id" json:"id"`
	Period           string    `db:"period" json:"period"`
	Status           string    `db:"status" json:"status"`
	Employees        int       `db:"employees" json:"employees"`
	Total            int64     `db:"total" json:"total"`
	PaymentReference string    `db:"payment_reference" json:"payment_reference"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}
type Payslip struct {
	ID           int64  `db:"id" json:"id"`
	RunID        int64  `db:"run_id" json:"run_id"`
	EmployeeID   int64  `db:"employee_id" json:"employee_id"`
	EmployeeName string `db:"employee_name" json:"employee_name"`
	EmployeeCode string `db:"employee_code" json:"employee_code"`
	Position     string `db:"position" json:"position"`
	Period       string `db:"period" json:"period"`
	Status       string `db:"status" json:"status"`
	BasicSalary  int64  `db:"basic_salary" json:"basic_salary"`
	Allowance    int64  `db:"allowance" json:"allowance"`
	Deduction    int64  `db:"deduction" json:"deduction"`
	Net          int64  `db:"net" json:"net"`
	Note         string `db:"note" json:"note"`
	Version      int    `db:"version" json:"version"`
}
