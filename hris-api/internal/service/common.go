// Package service holds validation, authorization, and business rules. Handlers
// pass the authenticated user; services derive the tenant and the data scope
// from it, never from request input.
package service

import "github.com/irvanmhndra/hris-api/internal/model"

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
