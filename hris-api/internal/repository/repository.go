package repository

import (
	"context"
	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/model"
)

type Repository interface {
	Salaries(context.Context, int64) ([]model.Salary, error)
	SaveSalary(context.Context, *model.User, int64, dto.Salary) error
	PayrollRuns(context.Context, int64) ([]model.PayrollRun, error)
	CreatePayroll(context.Context, *model.User, string) (int64, error)
	Payslips(context.Context, *model.User, int64) ([]model.Payslip, error)
	SavePayslip(context.Context, *model.User, int64, dto.Salary) error
	PayrollAction(context.Context, *model.User, int64, dto.PayrollAction) error

	WorkCalendar(context.Context, int64) (model.WorkCalendar, error)
	SaveCalendar(context.Context, *model.User, model.WorkCalendar) error
	Holidays(context.Context, int64) ([]model.Holiday, error)
	SaveHoliday(context.Context, *model.User, model.Holiday) error
	DeleteHoliday(context.Context, *model.User, int64) error
	Balances(context.Context, int64, *int64, int) ([]model.Balance, error)
	SetAllowance(context.Context, *model.User, int64, int, int) error
	CancelLeave(context.Context, *model.User, int64) error
	Profile(context.Context, int64, int64) (model.Profile, error)
	SaveProfile(context.Context, *model.User, int64, model.Profile) error
	HRItems(context.Context, *model.User, string) ([]model.HRItem, error)
	SaveHRItem(context.Context, *model.User, string, int64, dto.HRItem) (int64, error)
	ActHRItem(context.Context, *model.User, string, int64, dto.HRAction) error
	AuditLogs(context.Context, int64) ([]model.AuditLog, error)

	LoginUser(context.Context, string, string) (*model.User, error)
	CreateSession(context.Context, int64, string) error
	SessionUser(context.Context, string) (*model.User, error)
	DeleteSession(context.Context, string) error
	Departments(context.Context, int64) ([]model.Department, error)
	CreateDepartment(context.Context, int64, string) error
	Employees(context.Context, int64) ([]model.Employee, error)
	SaveEmployee(context.Context, int64, int64, dto.Employee, string) (int64, error)
	Dashboard(context.Context, int64) (model.Dashboard, error)
	Attendances(context.Context, int64, *int64) ([]model.Attendance, error)
	Clock(context.Context, int64, int64, bool) error
	Leaves(context.Context, int64, *int64) ([]model.Leave, error)
	CreateLeave(context.Context, int64, int64, dto.Leave) error
	ReviewLeave(context.Context, int64, int64, int64, string) error
}
