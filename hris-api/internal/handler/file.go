package handler

import (
	"mime"
	"net/http"

	"github.com/irvanmhndra/hris-api/internal/middleware"
	"github.com/irvanmhndra/hris-api/internal/service"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/irvanmhndra/hris-api/pkg/httputil"
	"github.com/labstack/echo/v5"
)

type FileHandler struct{ svc *service.FileService }

func NewFileHandler(svc *service.FileService) *FileHandler { return &FileHandler{svc: svc} }

// Upload accepts multipart field "file" (max 10 MB).
func (h *FileHandler) Upload(c *echo.Context) error {
	req := c.Request()
	req.Body = http.MaxBytesReader(c.Response(), req.Body, service.MaxFileBytes+1<<20)
	fh, err := c.FormFile("file")
	if err != nil {
		return httputil.Error(c, apperror.Invalid("Pilih file (maks. 10 MB) pada field \"file\""))
	}
	src, err := fh.Open()
	if err != nil {
		return httputil.Error(c, err)
	}
	defer func() { _ = src.Close() }()
	v, err := h.svc.Upload(req.Context(), middleware.User(c), fh.Filename, fh.Size, src)
	return respond(c, v, err)
}

// Download streams a file as an attachment; the stored type is never
// rendered inline, so an uploaded file cannot run in the app's origin.
func (h *FileHandler) Download(c *echo.Context) error {
	id, err := pathID(c)
	if err != nil {
		return httputil.Error(c, err)
	}
	f, rc, err := h.svc.Open(c.Request().Context(), middleware.User(c), id)
	if err != nil {
		return httputil.Error(c, err)
	}
	defer func() { _ = rc.Close() }()
	res := c.Response()
	res.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": f.Name}))
	res.Header().Set("X-Content-Type-Options", "nosniff")
	res.Header().Set("Cache-Control", "private, no-store")
	return c.Stream(http.StatusOK, f.ContentType, rc)
}
