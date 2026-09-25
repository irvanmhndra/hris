package handler

import (
	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/middleware"
	"github.com/irvanmhndra/hris-api/internal/service"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/irvanmhndra/hris-api/pkg/httputil"
	"github.com/labstack/echo/v5"
	"strconv"
	"strings"
)

type Handler struct{ Service *service.Service }

func result(c *echo.Context, v any, e error) error {
	if e != nil {
		return httputil.Error(c, e)
	}
	return httputil.Success(c, v)
}
func bind(c *echo.Context, v any) error {
	if err := c.Bind(v); err != nil {
		return apperror.Invalid("Format permintaan tidak valid")
	}
	return nil
}
func (h *Handler) Login(c *echo.Context) error {
	var v dto.Login
	if err := bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	token, u, err := h.Service.Login(c.Request().Context(), v)
	return result(c, map[string]any{"token": token, "user": u}, err)
}
func (h *Handler) Me(c *echo.Context) error { return httputil.Success(c, middleware.User(c)) }
func (h *Handler) Logout(c *echo.Context) error {
	return result(c, nil, h.Service.DeleteSession(c.Request().Context(), service.HashToken(strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer "))))
}
func (h *Handler) Departments(c *echo.Context) error {
	v, e := h.Service.Departments(c.Request().Context(), middleware.User(c).CompanyID)
	return result(c, v, e)
}
func (h *Handler) CreateDepartment(c *echo.Context) error {
	var v struct {
		Name string `json:"name"`
	}
	if e := bind(c, &v); e != nil {
		return httputil.Error(c, e)
	}
	v.Name = strings.TrimSpace(v.Name)
	if len(v.Name) < 2 || len(v.Name) > 100 {
		return httputil.Error(c, apperror.Invalid("Nama departemen harus 2–100 karakter"))
	}
	return result(c, nil, h.Service.CreateDepartment(c.Request().Context(), middleware.User(c).CompanyID, v.Name))
}
func (h *Handler) Employees(c *echo.Context) error {
	v, e := h.Service.Employees(c.Request().Context(), middleware.User(c).CompanyID)
	return result(c, v, e)
}
func (h *Handler) SaveEmployee(c *echo.Context) error {
	var v dto.Employee
	if e := bind(c, &v); e != nil {
		return httputil.Error(c, e)
	}
	var id int64
	if raw := c.Param("id"); raw != "" {
		var err error
		id, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			return httputil.Error(c, apperror.Invalid("ID tidak valid"))
		}
	}
	newID, e := h.Service.SaveEmployee(c.Request().Context(), middleware.User(c).CompanyID, id, v)
	return result(c, map[string]int64{"id": newID}, e)
}
func (h *Handler) Dashboard(c *echo.Context) error {
	v, e := h.Service.Dashboard(c.Request().Context(), middleware.User(c).CompanyID)
	return result(c, v, e)
}
func (h *Handler) Attendances(c *echo.Context) error {
	u := middleware.User(c)
	var id *int64
	if u.Role == "employee" {
		id = u.EmployeeID
	}
	v, e := h.Service.Attendances(c.Request().Context(), u.CompanyID, id)
	return result(c, v, e)
}
func (h *Handler) Clock(c *echo.Context) error {
	u := middleware.User(c)
	return result(c, nil, h.Service.Clock(c.Request().Context(), u.CompanyID, *u.EmployeeID, c.Param("action") == "out"))
}
func (h *Handler) Leaves(c *echo.Context) error {
	u := middleware.User(c)
	var id *int64
	if u.Role == "employee" {
		id = u.EmployeeID
	}
	v, e := h.Service.Leaves(c.Request().Context(), u.CompanyID, id)
	return result(c, v, e)
}
func (h *Handler) CreateLeave(c *echo.Context) error {
	var v dto.Leave
	if e := bind(c, &v); e != nil {
		return httputil.Error(c, e)
	}
	return result(c, nil, h.Service.CreateLeave(c.Request().Context(), middleware.User(c), v))
}
func (h *Handler) ReviewLeave(c *echo.Context) error {
	var v struct {
		Status string `json:"status"`
	}
	if e := bind(c, &v); e != nil {
		return httputil.Error(c, e)
	}
	if v.Status != "approved" && v.Status != "rejected" {
		return httputil.Error(c, apperror.Invalid("Keputusan tidak valid"))
	}
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id <= 0 {
		return httputil.Error(c, apperror.Invalid("ID tidak valid"))
	}
	u := middleware.User(c)
	return result(c, nil, h.Service.ReviewLeave(c.Request().Context(), u.CompanyID, id, u.ID, v.Status))
}
