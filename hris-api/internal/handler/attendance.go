package handler

import (
	"github.com/irvanmhndra/hris-api/internal/middleware"
	"github.com/irvanmhndra/hris-api/internal/service"
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

// Clock handles POST /attendance/in and /attendance/out.
func (h *AttendanceHandler) Clock(c *echo.Context) error {
	return respond(c, nil, h.svc.Clock(c.Request().Context(), middleware.User(c), c.Param("action")))
}
