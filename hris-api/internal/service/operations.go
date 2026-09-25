package service

import (
	"context"
	"github.com/irvanmhndra/hris-api/internal/model"
)

// Read operations preserve the tenant and personal scopes established by authentication.
func (s *Service) SessionUser(ctx context.Context, hash string) (*model.User, error) {
	return s.Repo.SessionUser(ctx, hash)
}
func (s *Service) DeleteSession(ctx context.Context, hash string) error {
	return s.Repo.DeleteSession(ctx, hash)
}
func (s *Service) Departments(ctx context.Context, c int64) ([]model.Department, error) {
	return s.Repo.Departments(ctx, c)
}
func (s *Service) CreateDepartment(ctx context.Context, c int64, name string) error {
	return s.Repo.CreateDepartment(ctx, c, name)
}
func (s *Service) Employees(ctx context.Context, c int64) ([]model.Employee, error) {
	return s.Repo.Employees(ctx, c)
}
func (s *Service) Dashboard(ctx context.Context, c int64) (model.Dashboard, error) {
	return s.Repo.Dashboard(ctx, c)
}
func (s *Service) Attendances(ctx context.Context, c int64, e *int64) ([]model.Attendance, error) {
	return s.Repo.Attendances(ctx, c, e)
}
func (s *Service) Clock(ctx context.Context, c, e int64, out bool) error {
	return s.Repo.Clock(ctx, c, e, out)
}
func (s *Service) Leaves(ctx context.Context, c int64, e *int64) ([]model.Leave, error) {
	return s.Repo.Leaves(ctx, c, e)
}
func (s *Service) ReviewLeave(ctx context.Context, c, id, u int64, status string) error {
	return s.Repo.ReviewLeave(ctx, c, id, u, status)
}
