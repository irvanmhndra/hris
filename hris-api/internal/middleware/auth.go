package middleware

import (
	"context"
	"strings"

	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/irvanmhndra/hris-api/pkg/httputil"
	"github.com/labstack/echo/v5"
)

// Authenticator resolves a bearer token to the authenticated user.
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*model.User, error)
}

const userKey = "user"

// User returns the authenticated user set by Auth. Only call it on routes
// behind Auth.
func User(c *echo.Context) *model.User {
	u, _ := c.Get(userKey).(*model.User)
	return u
}

// BearerToken extracts the token from "Authorization: Bearer <token>".
func BearerToken(c *echo.Context) (string, bool) {
	return strings.CutPrefix(c.Request().Header.Get("Authorization"), "Bearer ")
}

func Auth(a Authenticator) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			token, ok := BearerToken(c)
			if !ok {
				return httputil.Error(c, apperror.Unauthorized("Silakan masuk terlebih dahulu"))
			}
			u, err := a.Authenticate(c.Request().Context(), token)
			if err != nil {
				return httputil.Error(c, err)
			}
			c.Set(userKey, u)
			return next(c)
		}
	}
}

// Role restricts a route group to one role. An employee account must also be
// linked to an employee record.
func Role(role string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			u := User(c)
			if u == nil || u.Role != role || (role == model.RoleEmployee && u.EmployeeID == nil) {
				return httputil.Error(c, apperror.Forbidden("Akses tidak diizinkan"))
			}
			return next(c)
		}
	}
}
