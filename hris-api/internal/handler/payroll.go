package handler

import (
	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/middleware"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/service"
	"github.com/irvanmhndra/hris-api/pkg/httputil"
	"github.com/labstack/echo/v5"
)

type PayrollHandler struct{ svc *service.PayrollService }

func NewPayrollHandler(svc *service.PayrollService) *PayrollHandler {
	return &PayrollHandler{svc: svc}
}

func (h *PayrollHandler) Salaries(c *echo.Context) error {
	v, err := h.svc.Salaries(c.Request().Context(), middleware.User(c).CompanyID)
	return respond(c, v, err)
}

func (h *PayrollHandler) SaveSalary(c *echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	var v dto.Salary
	if err = bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	return respond(c, nil, h.svc.SaveSalary(c.Request().Context(), middleware.User(c), id, v))
}

func (h *PayrollHandler) PayrollRuns(c *echo.Context) error {
	v, err := h.svc.PayrollRuns(c.Request().Context(), middleware.User(c).CompanyID)
	return respond(c, v, err)
}

func (h *PayrollHandler) CreatePayroll(c *echo.Context) error {
	var v dto.CreatePayroll
	if err := bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	id, err := h.svc.CreatePayroll(c.Request().Context(), middleware.User(c), v)
	return respond(c, map[string]int64{"id": id}, err)
}

// Payslips serves /payroll/:id/slips (admin, one run) and /payslips (own).
func (h *PayrollHandler) Payslips(c *echo.Context) error {
	runID, err := optionalPathID(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	v, err := h.svc.Payslips(c.Request().Context(), middleware.User(c), runID)
	return respond(c, v, err)
}

func (h *PayrollHandler) CreateCorrection(c *echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	newID, err := h.svc.CreateCorrection(c.Request().Context(), middleware.User(c), id)
	return respond(c, map[string]int64{"id": newID}, err)
}

func (h *PayrollHandler) Settings(c *echo.Context) error {
	v, err := h.svc.Settings(c.Request().Context(), middleware.User(c).CompanyID)
	return respond(c, v, err)
}

func (h *PayrollHandler) SaveSettings(c *echo.Context) error {
	var v model.PayrollSettings
	if err := bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	return respond(c, nil, h.svc.SaveSettings(c.Request().Context(), middleware.User(c), v))
}

func (h *PayrollHandler) SavePayslip(c *echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	var v dto.Payslip
	if err = bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	return respond(c, nil, h.svc.SavePayslip(c.Request().Context(), middleware.User(c), id, v))
}

func (h *PayrollHandler) PayrollAction(c *echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	var v dto.PayrollAction
	if err = bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	return respond(c, nil, h.svc.PayrollAction(c.Request().Context(), middleware.User(c), id, v))
}
