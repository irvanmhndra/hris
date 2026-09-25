package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"golang.org/x/crypto/bcrypt"
)

// These tests pin the authorization rules that moved from the repository
// into the services: the fakes record what the service asked the repository
// for, so a scope regression shows up without a database.

func ptr(v int64) *int64 { return &v }

var (
	admin    = &model.User{ID: 1, CompanyID: 10, Role: model.RoleAdmin}
	employee = &model.User{ID: 2, CompanyID: 10, Role: model.RoleEmployee, EmployeeID: ptr(7)}
	orphan   = &model.User{ID: 3, CompanyID: 10, Role: model.RoleEmployee}
)

func wantStatus(t *testing.T, err error, status int) {
	t.Helper()
	e, ok := errors.AsType[*apperror.Error](err)
	if !ok || e.Status != status {
		t.Fatalf("want HTTP %d, got %v", status, err)
	}
}

func scopeID(v *int64) string {
	if v == nil {
		return "company"
	}
	return "employee"
}

type fakeProfiles struct {
	repository.ProfileRepository
	companyID, employeeID int64
}

func (f *fakeProfiles) Profile(_ context.Context, companyID, employeeID int64) (model.Profile, error) {
	f.companyID, f.employeeID = companyID, employeeID
	return model.Profile{}, nil
}

func TestProfileTargetsOwnRecordForEmployees(t *testing.T) {
	repo := &fakeProfiles{}
	svc := NewProfileService(repo)
	ctx := context.Background()

	if _, err := svc.Profile(ctx, employee, 99); err != nil || repo.employeeID != 7 || repo.companyID != 10 {
		t.Fatalf("employee reached employee %d in company %d (%v)", repo.employeeID, repo.companyID, err)
	}
	if _, err := svc.Profile(ctx, admin, 99); err != nil || repo.employeeID != 99 {
		t.Fatalf("admin should address the route employee, got %d (%v)", repo.employeeID, err)
	}
	_, err := svc.Profile(ctx, orphan, 99)
	wantStatus(t, err, http.StatusForbidden)
	_, err = svc.Profile(ctx, admin, 0)
	wantStatus(t, err, http.StatusUnprocessableEntity)
}

type fakeHRItems struct {
	repository.HRItemRepository
	scope model.HRItemScope
	saved model.HRItemInput
	calls int
}

func (f *fakeHRItems) HRItems(_ context.Context, _ int64, _ string, s model.HRItemScope) ([]model.HRItem, error) {
	f.scope, f.calls = s, f.calls+1
	return nil, nil
}

func (f *fakeHRItems) SaveHRItem(_ context.Context, _, _ int64, _ string, _ int64, v model.HRItemInput) (int64, error) {
	f.saved, f.calls = v, f.calls+1
	return 1, nil
}

func (f *fakeHRItems) ActHRItem(_ context.Context, _, _ int64, _ string, _ int64, s model.HRItemScope, _ model.HRActionInput) error {
	f.scope, f.calls = s, f.calls+1
	return nil
}

func TestHRItemReadScope(t *testing.T) {
	ctx := context.Background()
	for _, c := range []struct {
		name        string
		user        *model.User
		module      string
		scope       string
		unpublished bool
	}{
		{"employee own requests", employee, "overtime", "employee", false},
		{"employee company announcements", employee, "announcements", "company", false},
		{"employee company documents", employee, "documents", "company", false},
		{"admin everything", admin, "overtime", "company", true},
	} {
		repo := &fakeHRItems{}
		if _, err := NewHRItemService(repo).HRItems(ctx, c.user, c.module); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if scopeID(repo.scope.EmployeeID) != c.scope || repo.scope.IncludeUnpublished != c.unpublished {
			t.Fatalf("%s: got %+v", c.name, repo.scope)
		}
	}

	repo := &fakeHRItems{}
	svc := NewHRItemService(repo)
	_, err := svc.HRItems(ctx, employee, "recruitment")
	wantStatus(t, err, http.StatusForbidden)
	_, err = svc.HRItems(ctx, orphan, "goals")
	wantStatus(t, err, http.StatusForbidden)
	_, err = svc.HRItems(ctx, admin, "payroll")
	wantStatus(t, err, http.StatusNotFound)
	if repo.calls != 0 {
		t.Fatal("rejected request still reached the repository")
	}
}

func TestHRItemWriteRules(t *testing.T) {
	ctx := context.Background()
	overtime := dto.HRItem{
		Title: "Release", Description: "Production release support", Status: "approved",
		EmployeeID: ptr(99),
		Data:       json.RawMessage(`{"start_at":"2026-11-10T18:00:00+07:00","end_at":"2026-11-10T20:00:00+07:00"}`),
	}

	repo := &fakeHRItems{}
	svc := NewHRItemService(repo)
	if _, err := svc.SaveHRItem(ctx, employee, "overtime", 0, overtime); err != nil {
		t.Fatal(err)
	}
	if repo.saved.EmployeeID == nil || *repo.saved.EmployeeID != 7 || repo.saved.Status != "pending" {
		t.Fatalf("request must be filed for the caller as pending, got %+v", repo.saved)
	}

	_, err := svc.SaveHRItem(ctx, admin, "overtime", 0, overtime)
	wantStatus(t, err, http.StatusForbidden)
	_, err = svc.SaveHRItem(ctx, employee, "overtime", 5, overtime)
	wantStatus(t, err, http.StatusForbidden)
	_, err = svc.SaveHRItem(ctx, employee, "announcements", 0, dto.HRItem{Title: "Hi", Status: "published"})
	wantStatus(t, err, http.StatusForbidden)
	_, err = svc.SaveHRItem(ctx, admin, "announcements", 5, dto.HRItem{Title: "Hi", Status: "published"})
	wantStatus(t, err, http.StatusUnprocessableEntity) // edits need a version

	if _, err = svc.SaveHRItem(ctx, admin, "announcements", 0, dto.HRItem{Title: "Hi", Status: "published", EmployeeID: ptr(7)}); err != nil {
		t.Fatal(err)
	}
	if repo.saved.EmployeeID != nil {
		t.Fatal("company-wide announcement kept an employee")
	}

	for _, c := range []struct {
		user   *model.User
		module string
		action dto.HRAction
		status int
	}{
		{employee, "overtime", dto.HRAction{Action: "approve", Version: 1}, http.StatusForbidden},
		{admin, "overtime", dto.HRAction{Action: "cancel", Version: 1}, http.StatusForbidden},
		{admin, "overtime", dto.HRAction{Action: "reject", Note: "no", Version: 1}, http.StatusUnprocessableEntity},
		{admin, "overtime", dto.HRAction{Action: "approve"}, http.StatusUnprocessableEntity},
		{employee, "goals", dto.HRAction{Action: "progress", Progress: 101, Version: 1}, http.StatusUnprocessableEntity},
		{employee, "assets", dto.HRAction{Action: "start", Version: 1}, http.StatusForbidden},
	} {
		wantStatus(t, svc.ActHRItem(ctx, c.user, c.module, 1, c.action), c.status)
	}

	repo.calls = 0
	if err = svc.ActHRItem(ctx, employee, "onboarding", 1, dto.HRAction{Action: "start", Version: 1}); err != nil {
		t.Fatal(err)
	}
	if repo.calls != 1 || scopeID(repo.scope.EmployeeID) != "employee" {
		t.Fatalf("employee action must be scoped to their own items, got %+v", repo.scope)
	}
}

type fakeLeaves struct {
	repository.LeaveRepository
	scope *int64
}

func (f *fakeLeaves) Leaves(_ context.Context, _ int64, employeeID *int64) ([]model.Leave, error) {
	f.scope = employeeID
	return nil, nil
}

type fakePayroll struct {
	repository.PayrollRepository
	scope *int64
}

func (f *fakePayroll) Payslips(_ context.Context, _ int64, employeeID *int64, _ int64) ([]model.Payslip, error) {
	f.scope = employeeID
	return nil, nil
}

func TestEmployeesOnlyReadOwnLeavesAndPayslips(t *testing.T) {
	ctx := context.Background()
	leaves := &fakeLeaves{}
	payroll := &fakePayroll{}
	for _, c := range []struct {
		user *model.User
		want string
	}{{employee, "employee"}, {admin, "company"}} {
		if _, err := NewLeaveService(leaves).Leaves(ctx, c.user); err != nil || scopeID(leaves.scope) != c.want {
			t.Fatalf("%s leaves scope = %s", c.user.Role, scopeID(leaves.scope))
		}
		if _, err := NewPayrollService(payroll).Payslips(ctx, c.user, 0); err != nil || scopeID(payroll.scope) != c.want {
			t.Fatalf("%s payslip scope = %s", c.user.Role, scopeID(payroll.scope))
		}
	}
	wantStatus(t, NewLeaveService(leaves).CreateLeave(ctx, admin, dto.Leave{}), http.StatusForbidden)
	wantStatus(t, NewLeaveService(leaves).CancelLeave(ctx, admin, 1), http.StatusForbidden)
}

type fakeAuth struct {
	user        *model.User
	lookupErr   error
	sessionHash string
}

func (f *fakeAuth) LoginUser(context.Context, string, string) (*model.User, error) {
	if f.user == nil {
		return nil, sql.ErrNoRows
	}
	return f.user, nil
}

func (f *fakeAuth) CreateSession(_ context.Context, _ int64, hash string) error {
	f.sessionHash = hash
	return nil
}

func (f *fakeAuth) SessionUser(_ context.Context, hash string) (*model.User, error) {
	if f.lookupErr != nil {
		return nil, f.lookupErr
	}
	if hash != f.sessionHash {
		return nil, sql.ErrNoRows
	}
	return f.user, nil
}

func (f *fakeAuth) DeleteSession(context.Context, string) error { return nil }

func TestSessionsStoreOnlyTokenHashes(t *testing.T) {
	ctx := context.Background()
	hash, _ := bcrypt.GenerateFromPassword([]byte("CorrectHorse123"), bcrypt.MinCost)
	repo := &fakeAuth{user: &model.User{ID: 1, PasswordHash: string(hash)}}
	svc := NewAuthService(repo)

	_, _, err := svc.Login(ctx, dto.Login{Company: "alpha", Email: "a@b.test", Password: "wrong"})
	wantStatus(t, err, http.StatusUnauthorized)

	token, _, err := svc.Login(ctx, dto.Login{Company: "alpha", Email: "a@b.test", Password: "CorrectHorse123"})
	if err != nil {
		t.Fatal(err)
	}
	if repo.sessionHash == token || repo.sessionHash != HashToken(token) {
		t.Fatal("session must be stored as the SHA-256 of the token")
	}
	if _, err = svc.Authenticate(ctx, token); err != nil {
		t.Fatal(err)
	}
	_, err = svc.Authenticate(ctx, repo.sessionHash) // a leaked hash is not a credential
	wantStatus(t, err, http.StatusUnauthorized)

	repo.lookupErr = errors.New("connection refused")
	if _, err = svc.Authenticate(ctx, token); !errors.Is(err, repo.lookupErr) {
		t.Fatalf("a database outage must not look like an expired session: %v", err)
	}

	unknown := NewAuthService(&fakeAuth{})
	_, _, err = unknown.Login(ctx, dto.Login{Company: "alpha", Email: "ghost@b.test", Password: "x"})
	wantStatus(t, err, http.StatusUnauthorized)
}
