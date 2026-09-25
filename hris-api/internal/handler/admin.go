package handler

import (
	"github.com/irvanmhndra/hris-api/internal/middleware"
	"github.com/irvanmhndra/hris-api/internal/service"
	"github.com/irvanmhndra/hris-api/pkg/httputil"
	"github.com/labstack/echo/v5"
)

type DashboardHandler struct{ svc *service.DashboardService }

func NewDashboardHandler(svc *service.DashboardService) *DashboardHandler {
	return &DashboardHandler{svc: svc}
}

func (h *DashboardHandler) Dashboard(c *echo.Context) error {
	v, err := h.svc.Dashboard(c.Request().Context(), middleware.User(c).CompanyID)
	return respond(c, v, err)
}

type AuditHandler struct{ svc *service.AuditService }

func NewAuditHandler(svc *service.AuditService) *AuditHandler { return &AuditHandler{svc: svc} }

// AuditLogs: ?page=N paginates; without it the latest 250 entries are returned.
func (h *AuditHandler) AuditLogs(c *echo.Context) error {
	page, err := queryInt(c, "page", "Halaman tidak valid")
	if err != nil {
		return httputil.Error(c, err)
	}
	perPage, err := queryInt(c, "per_page", "Jumlah per halaman tidak valid")
	if err != nil {
		return httputil.Error(c, err)
	}
	out, err := h.svc.AuditLogs(c.Request().Context(), middleware.User(c).CompanyID, page, perPage)
	if err != nil {
		return httputil.Error(c, err)
	}
	if !out.Paged {
		return httputil.Success(c, out.Items)
	}
	return httputil.SuccessWithPagination(c, out.Items, httputil.NewPagination(out.Page, out.PerPage, out.Total))
}
