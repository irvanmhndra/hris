package httputil

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"

	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/labstack/echo/v5"
	"github.com/lib/pq"
)

type Pagination struct {
	CurrentPage  int `json:"current_page"`
	PerPage      int `json:"per_page"`
	TotalRecords int `json:"total_records"`
	TotalPages   int `json:"total_pages"`
}

func NewPagination(page, perPage, total int) Pagination {
	pages := 0
	if perPage > 0 {
		pages = (total + perPage - 1) / perPage
	}
	return Pagination{CurrentPage: page, PerPage: perPage, TotalRecords: total, TotalPages: pages}
}

func Success(c *echo.Context, data any) error {
	return c.JSON(http.StatusOK, map[string]any{"success": true, "data": data})
}

func SuccessWithPagination(c *echo.Context, data any, p Pagination) error {
	return c.JSON(http.StatusOK, map[string]any{
		"success": true,
		"data":    data,
		"meta":    map[string]any{"pagination": p},
	})
}

func Error(c *echo.Context, err error) error {
	e := classify(err)
	if e.Status >= http.StatusInternalServerError {
		slog.ErrorContext(c.Request().Context(), "request failed",
			"error", err,
			"request_id", c.Response().Header().Get(echo.HeaderXRequestID))
	}
	return c.JSON(e.Status, map[string]any{
		"success":    false,
		"error_code": e.Code,
		"message":    e.Message,
	})
}

// classify maps domain, SQL, and PostgreSQL errors to a client-facing error.
// Anything unrecognised becomes a generic 500 so internals never leak.
func classify(err error) *apperror.Error {
	if e, ok := errors.AsType[*apperror.Error](err); ok {
		return e
	}
	if errors.Is(err, sql.ErrNoRows) {
		return apperror.NotFound("Data tidak ditemukan atau tindakan sudah diproses")
	}
	if p, ok := errors.AsType[*pq.Error](err); ok {
		switch p.Code {
		case "23505": // unique_violation
			switch p.Constraint {
			case "payroll_active_period":
				return apperror.Conflict("Payroll aktif untuk periode ini sudah ada")
			case "asset_code_unique":
				return apperror.Conflict("Kode aset sudah digunakan")
			}
			return apperror.Conflict("Data dengan identitas atau periode yang sama sudah ada")
		case "40001", "40P01": // serialization_failure, deadlock_detected
			return apperror.Conflict("Data sedang diperbarui; muat ulang dan coba kembali")
		case "23514": // check_violation
			return apperror.Invalid("Data melanggar aturan modul")
		case "23503": // foreign_key_violation
			return apperror.Invalid("Referensi data tidak valid")
		}
	}
	return apperror.Internal()
}
