package httputil

import (
	"database/sql"
	"errors"

	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/labstack/echo/v5"
	"github.com/lib/pq"
	"log/slog"
)

func Success(c *echo.Context, data any) error {
	return c.JSON(200, map[string]any{"success": true, "data": data})
}
func Error(c *echo.Context, err error) error {
	status, message := 500, "Terjadi kesalahan server"
	var e *apperror.Error
	var p *pq.Error
	switch {
	case errors.As(err, &e):
		status, message = e.Status, e.Message
	case errors.Is(err, sql.ErrNoRows):
		status, message = 404, "Data tidak ditemukan atau tindakan sudah diproses"
	case errors.As(err, &p):
		if p.Code == "23505" {
			status, message = 409, "Data dengan identitas atau periode yang sama sudah ada"
			if p.Constraint == "payroll_active_period" {
				message = "Payroll aktif untuk periode ini sudah ada"
			}
			if p.Constraint == "asset_code_unique" {
				message = "Kode aset sudah digunakan"
			}
		} else if p.Code == "40001" || p.Code == "40P01" {
			status, message = 409, "Data sedang diperbarui; muat ulang dan coba kembali"
		} else if p.Code == "23514" {
			status, message = 422, "Data melanggar aturan modul"
		} else if p.Code == "23503" {
			status, message = 422, "Referensi data tidak valid"
		}
	}
	if status == 500 {
		slog.Error("request failed", "error", err)
	}
	return c.JSON(status, map[string]any{"success": false, "message": message})
}
