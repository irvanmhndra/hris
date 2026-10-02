package router

import (
	"github.com/irvanmhndra/hris-api/internal/handler"
	"github.com/irvanmhndra/hris-api/internal/middleware"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/labstack/echo/v5"
	em "github.com/labstack/echo/v5/middleware"
)

type Handlers struct {
	Auth       *handler.AuthHandler
	Employee   *handler.EmployeeHandler
	Attendance *handler.AttendanceHandler
	Leave      *handler.LeaveHandler
	Profile    *handler.ProfileHandler
	HRItem     *handler.HRItemHandler
	Payroll    *handler.PayrollHandler
	Approval   *handler.ApprovalHandler
	Dashboard  *handler.DashboardHandler
	Audit      *handler.AuditHandler
}

// Setup registers every /api/v1 route. Identity and tenant always come from
// the session; Role groups gate admin-only and employee-only endpoints.
func Setup(e *echo.Echo, h *Handlers, authn middleware.Authenticator) {
	api := e.Group("/api/v1")
	api.POST("/auth/login", h.Auth.Login, em.RateLimiter(em.NewRateLimiterMemoryStore(2)))

	// Any signed-in user (data is scoped to the caller by the services).
	auth := api.Group("", middleware.Auth(authn))
	auth.GET("/auth/me", h.Auth.Me)
	auth.POST("/auth/logout", h.Auth.Logout)
	auth.GET("/attendances", h.Attendance.Attendances)
	auth.GET("/leaves", h.Leave.Leaves)
	auth.GET("/calendar", h.Leave.WorkCalendar)
	auth.GET("/holidays", h.Leave.Holidays)
	auth.GET("/leave-balances", h.Leave.Balances)
	auth.GET("/hr/:module", h.HRItem.HRItems)
	auth.POST("/hr/:module", h.HRItem.SaveHRItem)
	auth.PUT("/hr/:module/:id", h.HRItem.SaveHRItem)
	auth.PATCH("/hr/:module/:id/action", h.HRItem.ActHRItem)

	admin := auth.Group("", middleware.Role(model.RoleAdmin))
	admin.GET("/dashboard", h.Dashboard.Dashboard)
	admin.PUT("/calendar", h.Leave.SaveCalendar)
	admin.POST("/holidays", h.Leave.SaveHoliday)
	admin.DELETE("/holidays/:id", h.Leave.DeleteHoliday)
	admin.PUT("/leave-balances/:id", h.Leave.SetAllowance)
	admin.GET("/employees/:id/profile", h.Profile.Profile)
	admin.PUT("/employees/:id/profile", h.Profile.SaveProfile)
	admin.GET("/audit-logs", h.Audit.AuditLogs)
	admin.GET("/salaries", h.Payroll.Salaries)
	admin.PUT("/salaries/:id", h.Payroll.SaveSalary)
	admin.GET("/payroll/settings", h.Payroll.Settings)
	admin.PUT("/payroll/settings", h.Payroll.SaveSettings)
	admin.GET("/payroll", h.Payroll.PayrollRuns)
	admin.POST("/payroll", h.Payroll.CreatePayroll)
	admin.GET("/payroll/:id/slips", h.Payroll.Payslips)
	admin.PUT("/payroll/slips/:id", h.Payroll.SavePayslip)
	admin.PATCH("/payroll/:id/action", h.Payroll.PayrollAction)
	admin.GET("/departments", h.Employee.Departments)
	admin.POST("/departments", h.Employee.CreateDepartment)
	admin.GET("/employees", h.Employee.Employees)
	admin.POST("/employees", h.Employee.SaveEmployee)
	admin.PUT("/employees/:id", h.Employee.SaveEmployee)
	admin.PATCH("/leaves/:id", h.Leave.ReviewLeave)
	admin.GET("/shifts", h.Attendance.Shifts)
	admin.POST("/shifts", h.Attendance.SaveShift)
	admin.PUT("/shifts/:id", h.Attendance.SaveShift)
	admin.GET("/schedule", h.Attendance.Schedule)
	admin.PUT("/schedule", h.Attendance.SetAssignments)
	admin.GET("/attendance-summary", h.Attendance.Summary)
	admin.GET("/attendance-locations", h.Attendance.Locations)
	admin.POST("/attendance-locations", h.Attendance.SaveLocation)
	admin.PUT("/attendance-locations/:id", h.Attendance.SaveLocation)
	admin.DELETE("/attendance-locations/:id", h.Attendance.DeleteLocation)

	staff := auth.Group("", middleware.Role(model.RoleEmployee))
	staff.POST("/leaves", h.Leave.CreateLeave)
	staff.POST("/leaves/:id/cancel", h.Leave.CancelLeave)
	staff.GET("/profile", h.Profile.Profile)
	staff.PUT("/profile", h.Profile.SaveProfile)
	staff.GET("/payslips", h.Payroll.Payslips)
	staff.GET("/attendance/today", h.Attendance.Today)
	staff.POST("/attendance/:action", h.Attendance.Clock)
	staff.GET("/team/approvals", h.Approval.TeamApprovals)
	staff.POST("/team/approvals/:type/:id", h.Approval.Review)
}
