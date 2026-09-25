package model

import "time"

type User struct {
	ID           int64  `db:"id" json:"id"`
	CompanyID    int64  `db:"company_id" json:"company_id"`
	EmployeeID   *int64 `db:"employee_id" json:"employee_id"`
	Name         string `db:"name" json:"name"`
	Email        string `db:"email" json:"email"`
	Role         string `db:"role" json:"role"`
	PasswordHash string `db:"password_hash" json:"-"`
	CompanyName  string `db:"company_name" json:"company_name"`
}
type Department struct {
	ID    int64  `db:"id" json:"id"`
	Name  string `db:"name" json:"name"`
	Count int    `db:"count" json:"count"`
}
type Employee struct {
	ID           int64  `db:"id" json:"id"`
	CompanyID    int64  `db:"company_id" json:"company_id"`
	Code         string `db:"code" json:"code"`
	Name         string `db:"name" json:"name"`
	Email        string `db:"email" json:"email"`
	DepartmentID int64  `db:"department_id" json:"department_id"`
	Department   string `db:"department" json:"department"`
	Position     string `db:"position" json:"position"`
	Status       string `db:"status" json:"status"`
	JoinedOn     string `db:"joined_on" json:"joined_on"`
}
type Attendance struct {
	ID         int64      `db:"id" json:"id"`
	EmployeeID int64      `db:"employee_id" json:"employee_id"`
	Name       string     `db:"name" json:"name"`
	Date       string     `db:"date" json:"date"`
	CheckIn    time.Time  `db:"check_in" json:"check_in"`
	CheckOut   *time.Time `db:"check_out" json:"check_out"`
}
type Leave struct {
	Calculation string `db:"calculation" json:"calculation"`
	ID          int64  `db:"id" json:"id"`
	EmployeeID  int64  `db:"employee_id" json:"employee_id"`
	Name        string `db:"name" json:"name"`
	Kind        string `db:"kind" json:"kind"`
	StartDate   string `db:"start_date" json:"start_date"`
	EndDate     string `db:"end_date" json:"end_date"`
	Reason      string `db:"reason" json:"reason"`
	Status      string `db:"status" json:"status"`
	Days        int    `db:"days" json:"days"`
}
type Dashboard struct {
	Employees   int `db:"employees" json:"employees"`
	Present     int `db:"present" json:"present"`
	Pending     int `db:"pending" json:"pending"`
	Departments int `db:"departments" json:"departments"`
}
