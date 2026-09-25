// Package repository declares one persistence contract per domain. Every
// method is tenant-scoped by an explicit companyID; mutating methods take the
// acting user's ID for audit trails. Authorization is decided by services —
// repositories only receive the resulting scope.
//
// Some methods are deliberately a single atomic operation (lock the employee,
// check invariants, write, audit — in one transaction). Splitting those across
// calls would reopen the races the locks exist to prevent.
package repository

import (
	"context"

	"github.com/irvanmhndra/hris-api/internal/model"
)

type AuthRepository interface {
	LoginUser(ctx context.Context, companySlug, email string) (*model.User, error)
	CreateSession(ctx context.Context, userID int64, tokenHash string) error
	SessionUser(ctx context.Context, tokenHash string) (*model.User, error)
	DeleteSession(ctx context.Context, tokenHash string) error
}

type EmployeeRepository interface {
	Departments(ctx context.Context, companyID int64) ([]model.Department, error)
	CreateDepartment(ctx context.Context, companyID int64, name string) error
	Employees(ctx context.Context, companyID int64, f model.EmployeeFilter) ([]model.Employee, int, error)
	// SaveEmployee creates (id == 0) or updates an employee together with the
	// linked portal account, revoking sessions on password reset/deactivation.
	SaveEmployee(ctx context.Context, companyID, id int64, v model.EmployeeInput) (int64, error)
}

type AttendanceRepository interface {
	Attendances(ctx context.Context, companyID int64, employeeID *int64) ([]model.Attendance, error)
	Clock(ctx context.Context, companyID, employeeID int64, out bool) error
}

type LeaveRepository interface {
	WorkCalendar(ctx context.Context, companyID int64) (model.WorkCalendar, error)
	SaveCalendar(ctx context.Context, companyID, actorID int64, v model.WorkCalendar) error
	Holidays(ctx context.Context, companyID int64) ([]model.Holiday, error)
	SaveHoliday(ctx context.Context, companyID, actorID int64, v model.Holiday) error
	DeleteHoliday(ctx context.Context, companyID, actorID, id int64) error
	Balances(ctx context.Context, companyID int64, employeeID *int64, year int) ([]model.Balance, error)
	SetAllowance(ctx context.Context, companyID, actorID, employeeID int64, year, allowance int) error
	Leaves(ctx context.Context, companyID int64, employeeID *int64) ([]model.Leave, error)
	CreateLeave(ctx context.Context, companyID, employeeID, actorID int64, v model.LeaveInput) error
	ReviewLeave(ctx context.Context, companyID, leaveID, reviewerID int64, status string) error
	CancelLeave(ctx context.Context, companyID, employeeID, actorID, leaveID int64) error
}

type ProfileRepository interface {
	Profile(ctx context.Context, companyID, employeeID int64) (model.Profile, error)
	SaveProfile(ctx context.Context, companyID, actorID, employeeID int64, v model.Profile) error
}

type HRItemRepository interface {
	HRItems(ctx context.Context, companyID int64, module string, scope model.HRItemScope) ([]model.HRItem, error)
	SaveHRItem(ctx context.Context, companyID, actorID int64, module string, id int64, v model.HRItemInput) (int64, error)
	ActHRItem(ctx context.Context, companyID, actorID int64, module string, id int64, scope model.HRItemScope, v model.HRActionInput) error
}

type PayrollRepository interface {
	Salaries(ctx context.Context, companyID int64) ([]model.Salary, error)
	SaveSalary(ctx context.Context, companyID, actorID, employeeID int64, v model.SalaryInput) error
	PayrollRuns(ctx context.Context, companyID int64) ([]model.PayrollRun, error)
	CreatePayroll(ctx context.Context, companyID, actorID int64, periodStart string) (int64, error)
	// Payslips lists slips of one run (runID > 0) or all runs. With employeeID
	// set, only that employee's finalized or paid slips are returned.
	Payslips(ctx context.Context, companyID int64, employeeID *int64, runID int64) ([]model.Payslip, error)
	SavePayslip(ctx context.Context, companyID, actorID, entryID int64, v model.SalaryInput) error
	PayrollAction(ctx context.Context, companyID, actorID, runID int64, action, reference string) error
}

type AuditRepository interface {
	AuditLogs(ctx context.Context, companyID int64, limit, offset int) ([]model.AuditLog, int, error)
}

type DashboardRepository interface {
	Dashboard(ctx context.Context, companyID int64) (model.Dashboard, error)
}
