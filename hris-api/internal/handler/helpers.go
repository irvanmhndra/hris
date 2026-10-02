// Package handler binds HTTP requests to services and writes the response
// envelope. Handlers hold no business rules.
package handler

import (
	"strconv"

	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/service"
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

// listQuery reads ?page=&per_page=&status=&date=.
func listQuery(c *echo.Context) (dto.ListQuery, error) {
	page, err := queryInt(c, "page", "Halaman tidak valid")
	if err != nil {
		return dto.ListQuery{}, err
	}
	perPage, err := queryInt(c, "per_page", "Jumlah per halaman tidak valid")
	if err != nil {
		return dto.ListQuery{}, err
	}
	return dto.ListQuery{Page: page, PerPage: perPage, Status: c.QueryParam("status"), Date: c.QueryParam("date")}, nil
}

// respondPage sends a paged list with pagination meta, or the plain list.
func respondPage[T any](c *echo.Context, p service.Page[T], err error) error {
	if err != nil {
		return httputil.Error(c, err)
	}
	if !p.Paged {
		return httputil.Success(c, p.Items)
	}
	return httputil.SuccessWithPagination(c, p.Items, httputil.NewPagination(p.Page, p.PerPage, p.Total))
}

// listed wraps a list handler: parse the query, call fn, respond.
func listed[T any](c *echo.Context, fn func(dto.ListQuery) (service.Page[T], error)) error {
	q, err := listQuery(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	p, err := fn(q)
	return respondPage(c, p, err)
}
