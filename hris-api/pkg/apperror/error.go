// Package apperror defines client-facing errors: an HTTP status, a stable
// machine-readable code (same vocabulary as NexPOS/FNB), and a user-facing
// message in Indonesian.
package apperror

import "net/http"

type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func Invalid(message string) *Error {
	return New(http.StatusUnprocessableEntity, "VALIDATION_ERROR", message)
}

func Unauthorized(message string) *Error {
	return New(http.StatusUnauthorized, "UNAUTHORIZED", message)
}

func Forbidden(message string) *Error {
	return New(http.StatusForbidden, "FORBIDDEN", message)
}

func NotFound(message string) *Error {
	return New(http.StatusNotFound, "NOT_FOUND", message)
}

func Conflict(message string) *Error {
	return New(http.StatusConflict, "CONFLICT", message)
}

func Internal() *Error {
	return New(http.StatusInternalServerError, "INTERNAL_ERROR", "Terjadi kesalahan server")
}

var ErrOverlap = Conflict("Tanggal cuti bertumpang tindih dengan pengajuan sebelumnya")
