package handler

import (
	"strconv"

	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/middleware"
	"github.com/irvanmhndra/hris-api/internal/service"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/irvanmhndra/hris-api/pkg/httputil"
	"github.com/labstack/echo/v5"
)

type EmployeeHandler struct{ svc *service.EmployeeService }

func NewEmployeeHandler(svc *service.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{svc: svc}
}

func (h *EmployeeHandler) Departments(c *echo.Context) error {
	v, err := h.svc.Departments(c.Request().Context(), middleware.User(c).CompanyID)
	return respond(c, v, err)
}

func (h *EmployeeHandler) CreateDepartment(c *echo.Context) error {
	var v struct {
		Name string `json:"name"`
	}
	if err := bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	return respond(c, nil, h.svc.CreateDepartment(c.Request().Context(), middleware.User(c).CompanyID, v.Name))
}

// Employees lists employees. With ?page=N the result is paginated and
// filtered server-side (?search, ?status, ?department_id, ?per_page);
// without it every matching employee is returned, as before.
func (h *EmployeeHandler) Employees(c *echo.Context) error {
	q := dto.EmployeeQuery{Search: c.QueryParam("search"), Status: c.QueryParam("status")}
	var err error
	if q.Page, err = queryInt(c, "page", "Halaman tidak valid"); err != nil {
		return httputil.Error(c, err)
	}
	if q.PerPage, err = queryInt(c, "per_page", "Jumlah per halaman tidak valid"); err != nil {
		return httputil.Error(c, err)
	}
	if raw := c.QueryParam("department_id"); raw != "" {
		if q.DepartmentID, err = strconv.ParseInt(raw, 10, 64); err != nil || q.DepartmentID < 0 {
			return httputil.Error(c, apperror.Invalid("Departemen tidak valid"))
		}
	}
	page, err := h.svc.Employees(c.Request().Context(), middleware.User(c).CompanyID, q)
	if err != nil {
		return httputil.Error(c, err)
	}
	if !page.Paged {
		return httputil.Success(c, page.Items)
	}
	return httputil.SuccessWithPagination(c, page.Items, httputil.NewPagination(page.Page, page.PerPage, page.Total))
}

func (h *EmployeeHandler) ConvertCandidate(c *echo.Context) error {
	candidateID, err := pathID(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	var v dto.Employee
	if err = bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	id, err := h.svc.ConvertCandidate(c.Request().Context(), middleware.User(c), candidateID, v)
	return respond(c, map[string]int64{"id": id}, err)
}

func (h *EmployeeHandler) SaveEmployee(c *echo.Context) error {
	var v dto.Employee
	if err := bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	id, err := optionalPathID(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	newID, err := h.svc.SaveEmployee(c.Request().Context(), middleware.User(c).CompanyID, id, v)
	return respond(c, map[string]int64{"id": newID}, err)
}
