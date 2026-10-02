package service

import (
	"context"
	"net/mail"
	"strings"
	"time"

	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"golang.org/x/crypto/bcrypt"
)

type EmployeeService struct{ repo repository.EmployeeRepository }

func NewEmployeeService(repo repository.EmployeeRepository) *EmployeeService {
	return &EmployeeService{repo: repo}
}

func (s *EmployeeService) Departments(ctx context.Context, companyID int64) ([]model.Department, error) {
	return s.repo.Departments(ctx, companyID)
}

func (s *EmployeeService) CreateDepartment(ctx context.Context, companyID int64, name string) error {
	name = strings.TrimSpace(name)
	if len(name) < 2 || len(name) > 100 {
		return apperror.Invalid("Nama departemen harus 2–100 karakter")
	}
	return s.repo.CreateDepartment(ctx, companyID, name)
}

// EmployeePage is one page of the employee list. Paged is false when the
// caller asked for every match (Page 0).
type EmployeePage struct {
	Items   []model.Employee
	Total   int
	Page    int
	PerPage int
	Paged   bool
}

func (s *EmployeeService) Employees(ctx context.Context, companyID int64, q dto.EmployeeQuery) (EmployeePage, error) {
	if q.Status != "" && q.Status != "active" && q.Status != "inactive" {
		return EmployeePage{}, apperror.Invalid("Status tidak valid")
	}
	if len(q.Search) > 120 {
		return EmployeePage{}, apperror.Invalid("Kata kunci terlalu panjang")
	}
	f := model.EmployeeFilter{Search: q.Search, Status: q.Status, DepartmentID: q.DepartmentID}
	var page EmployeePage
	if q.Page > 0 {
		f.Limit, f.Offset, page.Page, page.PerPage = pageWindow(q.Page, q.PerPage)
		page.Paged = true
	}
	var err error
	page.Items, page.Total, err = s.repo.Employees(ctx, companyID, f)
	return page, err
}

func ValidateEmployee(v *dto.Employee, create bool) error {
	v.Name = strings.TrimSpace(v.Name)
	v.Code = strings.TrimSpace(v.Code)
	v.Email = strings.ToLower(strings.TrimSpace(v.Email))
	v.Position = strings.TrimSpace(v.Position)
	if v.Name == "" || v.Code == "" || v.Position == "" || v.DepartmentID <= 0 {
		return apperror.Invalid("Nama, NIK, jabatan, dan departemen wajib diisi")
	}
	if len(v.Name) > 120 || len(v.Code) > 40 || len(v.Position) > 120 {
		return apperror.Invalid("Data karyawan terlalu panjang")
	}
	addr, err := mail.ParseAddress(v.Email)
	if err != nil || addr.Address != v.Email {
		return apperror.Invalid("Email tidak valid")
	}
	joined, err := time.Parse("2006-01-02", v.JoinedOn)
	if err != nil {
		return apperror.Invalid("Tanggal bergabung tidak valid")
	}
	if v.Status != "active" && v.Status != "inactive" {
		return apperror.Invalid("Status tidak valid")
	}
	if v.ManagerID != nil && *v.ManagerID <= 0 {
		v.ManagerID = nil
	}
	if v.ShiftID != nil && *v.ShiftID <= 0 {
		v.ShiftID = nil
	}
	if v.Status == "active" {
		v.LeftOn = ""
	} else if v.LeftOn != "" {
		left, err := time.Parse("2006-01-02", v.LeftOn)
		if err != nil || left.Before(joined) {
			return apperror.Invalid("Tanggal keluar harus valid dan tidak sebelum tanggal bergabung")
		}
	}
	// A new portal account always needs a password; on edit it is optional (reset).
	if (create || v.Password != "") && (len(v.Password) < 12 || len(v.Password) > 72) {
		return apperror.Invalid("Password harus 12–72 karakter")
	}
	return nil
}

// ConvertCandidate creates an employee (and portal account) from a
// recruitment candidate in one transaction and links the candidate to it.
func (s *EmployeeService) ConvertCandidate(ctx context.Context, u *model.User, candidateID int64, v dto.Employee) (int64, error) {
	v.Status = "active"
	if err := ValidateEmployee(&v, true); err != nil {
		return 0, err
	}
	h, err := bcrypt.GenerateFromPassword([]byte(v.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	return s.repo.SaveEmployee(ctx, u.CompanyID, 0, model.EmployeeInput{
		Code: v.Code, Name: v.Name, Email: v.Email, DepartmentID: v.DepartmentID, Position: v.Position,
		Status: v.Status, JoinedOn: v.JoinedOn, ManagerID: v.ManagerID, ShiftID: v.ShiftID,
		PasswordHash: string(h), CandidateID: &candidateID, ActorID: u.ID,
	})
}

func (s *EmployeeService) SaveEmployee(ctx context.Context, companyID, id int64, v dto.Employee) (int64, error) {
	if err := ValidateEmployee(&v, id == 0); err != nil {
		return 0, err
	}
	in := model.EmployeeInput{
		Code: v.Code, Name: v.Name, Email: v.Email, DepartmentID: v.DepartmentID,
		Position: v.Position, Status: v.Status, JoinedOn: v.JoinedOn, LeftOn: v.LeftOn, ManagerID: v.ManagerID, ShiftID: v.ShiftID,
	}
	if v.Password != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(v.Password), bcrypt.DefaultCost)
		if err != nil {
			return 0, err
		}
		in.PasswordHash = string(h)
	}
	return s.repo.SaveEmployee(ctx, companyID, id, in)
}
