package middleware

import (
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/service"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/irvanmhndra/hris-api/pkg/httputil"
	"github.com/labstack/echo/v5"
	"strings"
)

func User(c *echo.Context) *model.User { return c.Get("user").(*model.User) }
func Auth(s *service.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			h := c.Request().Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				return httputil.Error(c, &apperror.Error{Status: 401, Message: "Silakan masuk terlebih dahulu"})
			}
			u, err := s.SessionUser(c.Request().Context(), service.HashToken(strings.TrimPrefix(h, "Bearer ")))
			if err != nil {
				return httputil.Error(c, &apperror.Error{Status: 401, Message: "Sesi telah berakhir"})
			}
			c.Set("user", u)
			return next(c)
		}
	}
}
func Role(role string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			u := User(c)
			if u.Role != role || (role == "employee" && u.EmployeeID == nil) {
				return httputil.Error(c, &apperror.Error{Status: 403, Message: "Akses tidak diizinkan"})
			}
			return next(c)
		}
	}
}
