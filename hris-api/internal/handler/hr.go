package handler

import (
	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/middleware"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/irvanmhndra/hris-api/pkg/httputil"
	"github.com/labstack/echo/v5"
	"strconv"
	"time"
)

func paramID(c *echo.Context) (int64, error) {
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id <= 0 {
		return 0, apperror.Invalid("ID tidak valid")
	}
	return id, nil
}
func (h *Handler) WorkCalendar(c *echo.Context) error {
	v, e := h.Service.WorkCalendar(c.Request().Context(), middleware.User(c).CompanyID)
	return result(c, v, e)
}
func (h *Handler) SaveCalendar(c *echo.Context) error {
	var v model.WorkCalendar
	if e := bind(c, &v); e != nil {
		return httputil.Error(c, e)
	}
	return result(c, nil, h.Service.SaveCalendar(c.Request().Context(), middleware.User(c), v))
}
func (h *Handler) Holidays(c *echo.Context) error {
	v, e := h.Service.Holidays(c.Request().Context(), middleware.User(c).CompanyID)
	return result(c, v, e)
}
func (h *Handler) SaveHoliday(c *echo.Context) error {
	var v model.Holiday
	if e := bind(c, &v); e != nil {
		return httputil.Error(c, e)
	}
	return result(c, nil, h.Service.SaveHoliday(c.Request().Context(), middleware.User(c), v))
}
func (h *Handler) DeleteHoliday(c *echo.Context) error {
	id, e := paramID(c)
	if e != nil {
		return httputil.Error(c, e)
	}
	return result(c, nil, h.Service.DeleteHoliday(c.Request().Context(), middleware.User(c), id))
}
func (h *Handler) Balances(c *echo.Context) error {
	year := time.Now().Year()
	if raw := c.QueryParam("year"); raw != "" {
		var e error
		year, e = strconv.Atoi(raw)
		if e != nil {
			return httputil.Error(c, apperror.Invalid("Tahun tidak valid"))
		}
	}
	v, e := h.Service.Balances(c.Request().Context(), middleware.User(c), year)
	return result(c, v, e)
}
func (h *Handler) SetAllowance(c *echo.Context) error {
	id, e := paramID(c)
	if e != nil {
		return httputil.Error(c, e)
	}
	var v struct {
		Year      int `json:"year"`
		Allowance int `json:"allowance"`
	}
	if e = bind(c, &v); e != nil {
		return httputil.Error(c, e)
	}
	return result(c, nil, h.Service.SetAllowance(c.Request().Context(), middleware.User(c), id, v.Year, v.Allowance))
}
func (h *Handler) CancelLeave(c *echo.Context) error {
	id, e := paramID(c)
	if e != nil {
		return httputil.Error(c, e)
	}
	return result(c, nil, h.Service.CancelLeave(c.Request().Context(), middleware.User(c), id))
}
func profileID(c *echo.Context) (int64, error) {
	u := middleware.User(c)
	if u.Role == "employee" {
		return *u.EmployeeID, nil
	}
	return paramID(c)
}
func (h *Handler) Profile(c *echo.Context) error {
	id, e := profileID(c)
	if e != nil {
		return httputil.Error(c, e)
	}
	v, e := h.Service.Profile(c.Request().Context(), middleware.User(c), id)
	return result(c, v, e)
}
func (h *Handler) SaveProfile(c *echo.Context) error {
	id, e := profileID(c)
	if e != nil {
		return httputil.Error(c, e)
	}
	var v model.Profile
	if e = bind(c, &v); e != nil {
		return httputil.Error(c, e)
	}
	return result(c, nil, h.Service.SaveProfile(c.Request().Context(), middleware.User(c), id, v))
}
func (h *Handler) HRItems(c *echo.Context) error {
	v, e := h.Service.HRItems(c.Request().Context(), middleware.User(c), c.Param("module"))
	return result(c, v, e)
}
func (h *Handler) SaveHRItem(c *echo.Context) error {
	var id int64
	var e error
	if c.Param("id") != "" {
		id, e = paramID(c)
		if e != nil {
			return httputil.Error(c, e)
		}
	}
	var v dto.HRItem
	if e = bind(c, &v); e != nil {
		return httputil.Error(c, e)
	}
	id, e = h.Service.SaveHRItem(c.Request().Context(), middleware.User(c), c.Param("module"), id, v)
	return result(c, map[string]int64{"id": id}, e)
}
func (h *Handler) ActHRItem(c *echo.Context) error {
	id, e := paramID(c)
	if e != nil {
		return httputil.Error(c, e)
	}
	var v dto.HRAction
	if e = bind(c, &v); e != nil {
		return httputil.Error(c, e)
	}
	return result(c, nil, h.Service.ActHRItem(c.Request().Context(), middleware.User(c), c.Param("module"), id, v))
}
func (h *Handler) AuditLogs(c *echo.Context) error {
	v, e := h.Service.AuditLogs(c.Request().Context(), middleware.User(c).CompanyID)
	return result(c, v, e)
}
func (h *Handler) Salaries(c *echo.Context) error {
	v, e := h.Service.Salaries(c.Request().Context(), middleware.User(c).CompanyID)
	return result(c, v, e)
}
func (h *Handler) SaveSalary(c *echo.Context) error {
	id, e := paramID(c)
	if e != nil {
		return httputil.Error(c, e)
	}
	var v dto.Salary
	if e = bind(c, &v); e != nil {
		return httputil.Error(c, e)
	}
	return result(c, nil, h.Service.SaveSalary(c.Request().Context(), middleware.User(c), id, v))
}
func (h *Handler) PayrollRuns(c *echo.Context) error {
	v, e := h.Service.PayrollRuns(c.Request().Context(), middleware.User(c).CompanyID)
	return result(c, v, e)
}
func (h *Handler) CreatePayroll(c *echo.Context) error {
	var v struct {
		Period string `json:"period"`
	}
	if e := bind(c, &v); e != nil {
		return httputil.Error(c, e)
	}
	id, e := h.Service.CreatePayroll(c.Request().Context(), middleware.User(c), v.Period)
	return result(c, map[string]int64{"id": id}, e)
}
func (h *Handler) Payslips(c *echo.Context) error {
	var id int64
	var e error
	if c.Param("id") != "" {
		id, e = paramID(c)
		if e != nil {
			return httputil.Error(c, e)
		}
	}
	v, e := h.Service.Payslips(c.Request().Context(), middleware.User(c), id)
	return result(c, v, e)
}
func (h *Handler) SavePayslip(c *echo.Context) error {
	id, e := paramID(c)
	if e != nil {
		return httputil.Error(c, e)
	}
	var v dto.Salary
	if e = bind(c, &v); e != nil {
		return httputil.Error(c, e)
	}
	return result(c, nil, h.Service.SavePayslip(c.Request().Context(), middleware.User(c), id, v))
}
func (h *Handler) PayrollAction(c *echo.Context) error {
	id, e := paramID(c)
	if e != nil {
		return httputil.Error(c, e)
	}
	var v dto.PayrollAction
	if e = bind(c, &v); e != nil {
		return httputil.Error(c, e)
	}
	return result(c, nil, h.Service.PayrollAction(c.Request().Context(), middleware.User(c), id, v))
}
