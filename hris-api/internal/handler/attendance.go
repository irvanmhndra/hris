package handler

import (
	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/middleware"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/service"
	"github.com/irvanmhndra/hris-api/pkg/httputil"
	"github.com/labstack/echo/v5"
)

type AttendanceHandler struct{ svc *service.AttendanceService }

func NewAttendanceHandler(svc *service.AttendanceService) *AttendanceHandler {
	return &AttendanceHandler{svc: svc}
}

func (h *AttendanceHandler) Attendances(c *echo.Context) error {
	v, err := h.svc.Attendances(c.Request().Context(), middleware.User(c))
	return respond(c, v, err)
}

func (h *AttendanceHandler) Today(c *echo.Context) error {
	v, err := h.svc.Today(c.Request().Context(), middleware.User(c))
	return respond(c, v, err)
}

// Clock handles POST /attendance/in and /attendance/out. The body (position)
// is optional.
func (h *AttendanceHandler) Clock(c *echo.Context) error {
	var v dto.Clock
	if c.Request().ContentLength > 0 {
		if err := bind(c, &v); err != nil {
			return httputil.Error(c, err)
		}
	}
	return respond(c, nil, h.svc.Clock(c.Request().Context(), middleware.User(c), c.Param("action"), v))
}

func (h *AttendanceHandler) Schedule(c *echo.Context) error {
	v, err := h.svc.Schedule(c.Request().Context(), middleware.User(c), c.QueryParam("from"), c.QueryParam("to"))
	return respond(c, v, err)
}

func (h *AttendanceHandler) Summary(c *echo.Context) error {
	v, err := h.svc.Summary(c.Request().Context(), middleware.User(c), c.QueryParam("month"))
	return respond(c, v, err)
}

func (h *AttendanceHandler) Shifts(c *echo.Context) error {
	v, err := h.svc.Shifts(c.Request().Context(), middleware.User(c).CompanyID)
	return respond(c, v, err)
}

func (h *AttendanceHandler) SaveShift(c *echo.Context) error {
	id, err := optionalPathID(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	var v model.Shift
	if err = bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	id, err = h.svc.SaveShift(c.Request().Context(), middleware.User(c), id, v)
	return respond(c, map[string]int64{"id": id}, err)
}

func (h *AttendanceHandler) SetAssignments(c *echo.Context) error {
	var v dto.ShiftAssignments
	if err := bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	return respond(c, nil, h.svc.SetAssignments(c.Request().Context(), middleware.User(c), v))
}

func (h *AttendanceHandler) Locations(c *echo.Context) error {
	v, err := h.svc.Locations(c.Request().Context(), middleware.User(c).CompanyID)
	return respond(c, v, err)
}

func (h *AttendanceHandler) SaveLocation(c *echo.Context) error {
	id, err := optionalPathID(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	var v model.AttendanceLocation
	if err = bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	id, err = h.svc.SaveLocation(c.Request().Context(), middleware.User(c), id, v)
	return respond(c, map[string]int64{"id": id}, err)
}

func (h *AttendanceHandler) DeleteLocation(c *echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	return respond(c, nil, h.svc.DeleteLocation(c.Request().Context(), middleware.User(c), id))
}
