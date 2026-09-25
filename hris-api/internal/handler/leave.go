package handler

import (
	"strconv"
	"time"

	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/middleware"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/service"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/irvanmhndra/hris-api/pkg/httputil"
	"github.com/labstack/echo/v5"
)

type LeaveHandler struct{ svc *service.LeaveService }

func NewLeaveHandler(svc *service.LeaveService) *LeaveHandler { return &LeaveHandler{svc: svc} }

func (h *LeaveHandler) WorkCalendar(c *echo.Context) error {
	v, err := h.svc.WorkCalendar(c.Request().Context(), middleware.User(c).CompanyID)
	return respond(c, v, err)
}

func (h *LeaveHandler) SaveCalendar(c *echo.Context) error {
	var v model.WorkCalendar
	if err := bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	return respond(c, nil, h.svc.SaveCalendar(c.Request().Context(), middleware.User(c), v))
}

func (h *LeaveHandler) Holidays(c *echo.Context) error {
	v, err := h.svc.Holidays(c.Request().Context(), middleware.User(c).CompanyID)
	return respond(c, v, err)
}

func (h *LeaveHandler) SaveHoliday(c *echo.Context) error {
	var v model.Holiday
	if err := bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	return respond(c, nil, h.svc.SaveHoliday(c.Request().Context(), middleware.User(c), v))
}

func (h *LeaveHandler) DeleteHoliday(c *echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	return respond(c, nil, h.svc.DeleteHoliday(c.Request().Context(), middleware.User(c), id))
}

// Balances returns annual-leave balances for ?year (default: current year).
func (h *LeaveHandler) Balances(c *echo.Context) error {
	year := time.Now().Year()
	if raw := c.QueryParam("year"); raw != "" {
		var err error
		if year, err = strconv.Atoi(raw); err != nil {
			return httputil.Error(c, apperror.Invalid("Tahun tidak valid"))
		}
	}
	v, err := h.svc.Balances(c.Request().Context(), middleware.User(c), year)
	return respond(c, v, err)
}

func (h *LeaveHandler) SetAllowance(c *echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	var v struct {
		Year      int `json:"year"`
		Allowance int `json:"allowance"`
	}
	if err = bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	return respond(c, nil, h.svc.SetAllowance(c.Request().Context(), middleware.User(c), id, v.Year, v.Allowance))
}

func (h *LeaveHandler) Leaves(c *echo.Context) error {
	v, err := h.svc.Leaves(c.Request().Context(), middleware.User(c))
	return respond(c, v, err)
}

func (h *LeaveHandler) CreateLeave(c *echo.Context) error {
	var v dto.Leave
	if err := bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	return respond(c, nil, h.svc.CreateLeave(c.Request().Context(), middleware.User(c), v))
}

func (h *LeaveHandler) ReviewLeave(c *echo.Context) error {
	var v struct {
		Status string `json:"status"`
	}
	if err := bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	id, err := pathID(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	return respond(c, nil, h.svc.ReviewLeave(c.Request().Context(), middleware.User(c), id, v.Status))
}

func (h *LeaveHandler) CancelLeave(c *echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	return respond(c, nil, h.svc.CancelLeave(c.Request().Context(), middleware.User(c), id))
}
