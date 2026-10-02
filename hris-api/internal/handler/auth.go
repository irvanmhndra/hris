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

func (h *AuthHandler) Config(c *echo.Context) error {
	return httputil.Success(c, h.svc.Config())
}

func (h *AuthHandler) ForgotPassword(c *echo.Context) error {
	var v dto.ForgotPassword
	if err := bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	return respond(c, nil, h.svc.RequestPasswordReset(c.Request().Context(), v))
}

func (h *AuthHandler) ResetPassword(c *echo.Context) error {
	var v dto.ResetPassword
	if err := bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	return respond(c, nil, h.svc.ResetPassword(c.Request().Context(), v))
}

func (h *AuthHandler) Register(c *echo.Context) error {
	var v dto.Register
	if err := bind(c, &v); err != nil {
		return httputil.Error(c, err)
	}
	token, u, err := h.svc.Register(c.Request().Context(), v)
	return respond(c, map[string]any{"token": token, "user": u}, err)
}
