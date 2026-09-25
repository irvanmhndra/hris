package handler

import (
	"github.com/irvanmhndra/hris-api/internal/middleware"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/service"
	"github.com/irvanmhndra/hris-api/pkg/httputil"
	"github.com/labstack/echo/v5"
)

// ProfileHandler serves both GET/PUT /employees/:id/profile (admin) and
// /profile (the employee's own); the service enforces which one applies.
type ProfileHandler struct{ svc *service.ProfileService }

func NewProfileHandler(svc *service.ProfileService) *ProfileHandler {
	return &ProfileHandler{svc: svc}
}

func (h *ProfileHandler) Profile(c *echo.Context) error {
	id, err := optionalPathID(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	v, err := h.svc.Profile(c.Request().Context(), middleware.User(c), id)
	return respond(c, v, err)
}

func (h *ProfileHandler) SaveProfile(c *echo.Context) error {
	id, err := optionalPathID(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	var v model.Profile
	if err = bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	return respond(c, nil, h.svc.SaveProfile(c.Request().Context(), middleware.User(c), id, v))
}
