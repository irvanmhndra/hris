// Package service holds validation, authorization, and business rules. Handlers
// pass the authenticated user; services derive the tenant and the data scope
// from it, never from request input.
package service

import (
	"time"

	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
)

// ownScope limits reads to the caller's own records when they are an
// employee; admins see the whole company (nil).
func ownScope(u *model.User) *int64 {
	if u.Role == model.RoleEmployee {
		return u.EmployeeID
	}
	return nil
}

// pageWindow turns a 1-based page into LIMIT/OFFSET, clamping the page size.
func pageWindow(page, perPage int) (limit, offset, p, pp int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	return perPage, (page - 1) * perPage, page, perPage
}

// Page is one page of a list, or the whole list when Paged is false.
type Page[T any] struct {
	Items   []T
	Total   int
	Page    int
	PerPage int
	Paged   bool
}

// listFilter validates a list query. Unpaged requests keep the original
// response, capped at unpagedLimit rows (0 = all).
func listFilter[T any](q dto.ListQuery, unpagedLimit int, statuses ...string) (Page[T], model.ListFilter, error) {
	out := Page[T]{Paged: q.Page > 0}
	f := model.ListFilter{Status: q.Status, Date: q.Date, Limit: unpagedLimit}
	if out.Paged {
		f.Limit, f.Offset, out.Page, out.PerPage = pageWindow(q.Page, q.PerPage)
	}
	if f.Status != "" {
		valid := false
		for _, s := range statuses {
			valid = valid || s == f.Status
		}
		if !valid {
			return out, f, apperror.Invalid("Filter status tidak valid")
		}
	}
	if f.Date != "" {
		if _, err := time.Parse(time.DateOnly, f.Date); err != nil {
			return out, f, apperror.Invalid("Filter tanggal harus YYYY-MM-DD")
		}
	}
	return out, f, nil
}
