package handler

import (
	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/middleware"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/service"
	"github.com/irvanmhndra/hris-api/pkg/httputil"
	"github.com/labstack/echo/v5"
)

// HRItemHandler serves the lightweight workflows under /hr/:module.
type HRItemHandler struct{ svc *service.HRItemService }

func NewHRItemHandler(svc *service.HRItemService) *HRItemHandler {
	return &HRItemHandler{svc: svc}
}

func (h *HRItemHandler) HRItems(c *echo.Context) error {
	return listed(c, func(q dto.ListQuery) (service.Page[model.HRItem], error) {
		return h.svc.HRItems(c.Request().Context(), middleware.User(c), c.Param("module"), q)
	})
}

func (h *HRItemHandler) SaveHRItem(c *echo.Context) error {
	id, err := optionalPathID(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	var v dto.HRItem
	if err = bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	id, err = h.svc.SaveHRItem(c.Request().Context(), middleware.User(c), c.Param("module"), id, v)
	return respond(c, map[string]int64{"id": id}, err)
}

func (h *HRItemHandler) ActHRItem(c *echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	var v dto.HRAction
	if err = bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	return respond(c, nil, h.svc.ActHRItem(c.Request().Context(), middleware.User(c), c.Param("module"), id, v))
}
