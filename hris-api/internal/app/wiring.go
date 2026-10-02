package app

import (
	"github.com/irvanmhndra/hris-api/config"
	"github.com/irvanmhndra/hris-api/internal/handler"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/internal/repository/postgres"
	"github.com/irvanmhndra/hris-api/internal/router"
	"github.com/irvanmhndra/hris-api/internal/service"
	"github.com/irvanmhndra/hris-api/pkg/mailer"
	"github.com/irvanmhndra/hris-api/pkg/storage"
	"github.com/jmoiron/sqlx"
)

type repositories struct {
	auth       repository.AuthRepository
	employee   repository.EmployeeRepository
	attendance repository.AttendanceRepository
	leave      repository.LeaveRepository
	profile    repository.ProfileRepository
	hrItem     repository.HRItemRepository
	payroll    repository.PayrollRepository
	audit      repository.AuditRepository
	dashboard  repository.DashboardRepository
	approval   repository.ApprovalRepository
	file       repository.FileRepository
}

func newRepositories(db *sqlx.DB) repositories {
	return repositories{
		auth:       postgres.NewAuthRepository(db),
		employee:   postgres.NewEmployeeRepository(db),
		attendance: postgres.NewAttendanceRepository(db),
		leave:      postgres.NewLeaveRepository(db),
		profile:    postgres.NewProfileRepository(db),
		hrItem:     postgres.NewHRItemRepository(db),
		payroll:    postgres.NewPayrollRepository(db),
		audit:      postgres.NewAuditRepository(db),
		dashboard:  postgres.NewDashboardRepository(db),
		approval:   postgres.NewApprovalRepository(db),
		file:       postgres.NewFileRepository(db),
	}
}

type services struct {
	Auth       *service.AuthService
	Employee   *service.EmployeeService
	Attendance *service.AttendanceService
	Leave      *service.LeaveService
	Profile    *service.ProfileService
	HRItem     *service.HRItemService
	Payroll    *service.PayrollService
	Audit      *service.AuditService
	Dashboard  *service.DashboardService
	Approval   *service.ApprovalService
	File       *service.FileService
}

func newServices(r repositories, cfg config.Config) services {
	return services{
		Auth: service.NewAuthService(r.auth, service.AuthOptions{
			SignupEnabled: cfg.SignupEnabled, AdminURL: cfg.AdminURL, EmployeeURL: cfg.EmployeeURL,
			Mailer: mailer.New(mailer.Config(cfg.SMTP)),
		}),
		Employee:   service.NewEmployeeService(r.employee),
		Attendance: service.NewAttendanceService(r.attendance),
		Leave:      service.NewLeaveService(r.leave),
		Profile:    service.NewProfileService(r.profile),
		HRItem:     service.NewHRItemService(r.hrItem),
		Payroll:    service.NewPayrollService(r.payroll),
		Audit:      service.NewAuditService(r.audit),
		Dashboard:  service.NewDashboardService(r.dashboard),
		Approval:   service.NewApprovalService(r.approval),
		File:       service.NewFileService(r.file, storage.Local{Dir: cfg.UploadDir}),
	}
}

func newHandlers(s services) *router.Handlers {
	return &router.Handlers{
		Auth:       handler.NewAuthHandler(s.Auth),
		Employee:   handler.NewEmployeeHandler(s.Employee),
		Attendance: handler.NewAttendanceHandler(s.Attendance),
		Leave:      handler.NewLeaveHandler(s.Leave),
		Profile:    handler.NewProfileHandler(s.Profile),
		HRItem:     handler.NewHRItemHandler(s.HRItem),
		Payroll:    handler.NewPayrollHandler(s.Payroll),
		Dashboard:  handler.NewDashboardHandler(s.Dashboard),
		Audit:      handler.NewAuditHandler(s.Audit),
		Approval:   handler.NewApprovalHandler(s.Approval),
		File:       handler.NewFileHandler(s.File),
	}
}
