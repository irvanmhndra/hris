package handler

import (
	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/middleware"
	"github.com/irvanmhndra/hris-api/internal/service"
	"github.com/irvanmhndra/hris-api/pkg/httputil"
	"github.com/labstack/echo/v5"
)

type AuthHandler struct{ svc *service.AuthService }

func NewAuthHandler(svc *service.AuthService) *AuthHandler { return &AuthHandler{svc: svc} }

func (h *AuthHandler) Login(c *echo.Context) error {
	var v dto.Login
	if err := bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	token, u, err := h.svc.Login(c.Request().Context(), v)
	return respond(c, map[string]any{"token": token, "user": u}, err)
}

func (h *AuthHandler) Me(c *echo.Context) error {
	return httputil.Success(c, middleware.User(c))
}

func (h *AuthHandler) Logout(c *echo.Context) error {
	token, _ := middleware.BearerToken(c)
	return respond(c, nil, h.svc.Logout(c.Request().Context(), token))
}
