package handler

import (
	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/middleware"
	"github.com/irvanmhndra/hris-api/internal/service"
	"github.com/irvanmhndra/hris-api/pkg/httputil"
	"github.com/labstack/echo/v5"
)

type ApprovalHandler struct{ svc *service.ApprovalService }

func NewApprovalHandler(svc *service.ApprovalService) *ApprovalHandler {
	return &ApprovalHandler{svc: svc}
}

func (h *ApprovalHandler) TeamApprovals(c *echo.Context) error {
	v, err := h.svc.TeamApprovals(c.Request().Context(), middleware.User(c))
	return respond(c, v, err)
}

func (h *ApprovalHandler) Review(c *echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	var v dto.TeamReview
	if err = bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	return respond(c, nil, h.svc.Review(c.Request().Context(), middleware.User(c), c.Param("type"), id, v))
}
