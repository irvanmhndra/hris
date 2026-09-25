package router

import (
	"github.com/irvanmhndra/hris-api/internal/handler"
	"github.com/irvanmhndra/hris-api/internal/middleware"
	"github.com/irvanmhndra/hris-api/internal/service"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/irvanmhndra/hris-api/pkg/httputil"
	"github.com/labstack/echo/v5"
	em "github.com/labstack/echo/v5/middleware"
)

func New(s *service.Service) *echo.Echo {
	e := echo.New()
	e.Use(em.Recover())
	e.Use(em.BodyLimit(1 << 20))
	e.GET("/health", func(c *echo.Context) error { return httputil.Success(c, map[string]string{"status": "ok"}) })
	h := &handler.Handler{Service: s}
	api := e.Group("/api/v1")
	api.POST("/auth/login", h.Login, em.RateLimiter(em.NewRateLimiterMemoryStore(2)))
	auth := api.Group("", middleware.Auth(s))
	auth.GET("/auth/me", h.Me)
	auth.POST("/auth/logout", h.Logout)
	auth.GET("/attendances", h.Attendances)
	auth.GET("/leaves", h.Leaves)
	auth.GET("/calendar", h.WorkCalendar)
	auth.GET("/holidays", h.Holidays)
	auth.GET("/leave-balances", h.Balances)
	auth.GET("/hr/:module", h.HRItems)
	auth.POST("/hr/:module", h.SaveHRItem)
	auth.PUT("/hr/:module/:id", h.SaveHRItem)
	auth.PATCH("/hr/:module/:id/action", h.ActHRItem)
	admin := auth.Group("", middleware.Role("admin"))
	admin.GET("/dashboard", h.Dashboard)
	admin.PUT("/calendar", h.SaveCalendar)
	admin.POST("/holidays", h.SaveHoliday)
	admin.DELETE("/holidays/:id", h.DeleteHoliday)
	admin.PUT("/leave-balances/:id", h.SetAllowance)
	admin.GET("/employees/:id/profile", h.Profile)
	admin.PUT("/employees/:id/profile", h.SaveProfile)
	admin.GET("/audit-logs", h.AuditLogs)
	admin.GET("/salaries", h.Salaries)
	admin.PUT("/salaries/:id", h.SaveSalary)
	admin.GET("/payroll", h.PayrollRuns)
	admin.POST("/payroll", h.CreatePayroll)
	admin.GET("/payroll/:id/slips", h.Payslips)
	admin.PUT("/payroll/slips/:id", h.SavePayslip)
	admin.PATCH("/payroll/:id/action", h.PayrollAction)
	admin.GET("/departments", h.Departments)
	admin.POST("/departments", h.CreateDepartment)
	admin.GET("/employees", h.Employees)
	admin.POST("/employees", h.SaveEmployee)
	admin.PUT("/employees/:id", h.SaveEmployee)
	admin.PATCH("/leaves/:id", h.ReviewLeave)
	staff := auth.Group("", middleware.Role("employee"))
	staff.POST("/leaves", h.CreateLeave)
	staff.POST("/leaves/:id/cancel", h.CancelLeave)
	staff.GET("/profile", h.Profile)
	staff.PUT("/profile", h.SaveProfile)
	staff.GET("/payslips", h.Payslips)
	staff.POST("/attendance/:action", func(c *echo.Context) error {
		if c.Param("action") != "in" && c.Param("action") != "out" {
			return httputil.Error(c, apperror.Invalid("Tindakan tidak valid"))
		}
		return h.Clock(c)
	})
	return e
}
