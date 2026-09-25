// Package handler binds HTTP requests to services and writes the response
// envelope. Handlers hold no business rules.
package handler

import (
	"strconv"

	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/irvanmhndra/hris-api/pkg/httputil"
	"github.com/labstack/echo/v5"
)

// respond writes data on success or the classified error.
func respond(c *echo.Context, data any, err error) error {
	if err != nil {
		return httputil.Error(c, err)
	}
	return httputil.Success(c, data)
}

func bind(c *echo.Context, v any) error {
	if err := c.Bind(v); err != nil {
		return apperror.Invalid("Format permintaan tidak valid")
	}
	return nil
}

// pathID parses a required positive :id route parameter.
func pathID(c *echo.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, apperror.Invalid("ID tidak valid")
	}
	return id, nil
}

// optionalPathID returns 0 when the route has no :id (e.g. create or "self").
func optionalPathID(c *echo.Context) (int64, error) {
	if c.Param("id") == "" {
		return 0, nil
	}
	return pathID(c)
}

// queryInt parses an optional non-negative integer query parameter (0 if absent).
func queryInt(c *echo.Context, name, message string) (int, error) {
	raw := c.QueryParam(name)
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0, apperror.Invalid(message)
	}
	return n, nil
}
