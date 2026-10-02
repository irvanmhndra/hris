package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
)

type fakeAttendance struct {
	repository.AttendanceRepository
	src model.ScheduleSource
	got model.ClockInput
}

func (f *fakeAttendance) ScheduleSource(context.Context, int64, *int64, string, string, bool) (model.ScheduleSource, error) {
	return f.src, nil
}

func (f *fakeAttendance) Clock(_ context.Context, _, _ int64, v model.ClockInput) error {
	f.got = v
	return nil
}

func TestClockSnapshotsShiftAndGeofence(t *testing.T) {
	shift := int64(3)
	repo := &fakeAttendance{src: model.ScheduleSource{
		Calendar:  model.WorkCalendar{Workdays: []int64{1, 2, 3, 4, 5}, StartTime: "09:00", EndTime: "18:00", RequireLocation: true},
		Shifts:    []model.Shift{{ID: 3, Name: "Malam", StartTime: "22:00", EndTime: "06:00", GraceMinutes: 5}},
		Employees: []model.ScheduleEmployee{{ID: 1, Name: "A", ShiftID: &shift, JoinedOn: "2025-01-01"}},
		Locations: []model.AttendanceLocation{{Name: "HQ", Latitude: -6.2, Longitude: 106.8, RadiusM: 100}},
	}}
	svc := NewAttendanceService(repo)
	// Monday 22:20 WIB.
	svc.now = func() time.Time { return time.Date(2026, 3, 2, 15, 20, 0, 0, time.UTC) }
	employee := int64(1)
	u := &model.User{CompanyID: 1, EmployeeID: &employee, Role: model.RoleEmployee}

	if err := svc.Clock(context.Background(), u, "in", dto.Clock{}); err == nil {
		t.Fatal("missing position must be rejected when location is required")
	}
	far := dto.Clock{Latitude: fptr(-6.21), Longitude: fptr(106.8)}
	if err := svc.Clock(context.Background(), u, "in", far); err == nil || !strings.Contains(err.Error(), "maksimal 100 m") {
		t.Fatalf("outside geofence accepted: %v", err)
	}
	near := dto.Clock{Latitude: fptr(-6.2003), Longitude: fptr(106.8)}
	if err := svc.Clock(context.Background(), u, "in", near); err != nil {
		t.Fatal(err)
	}
	g := repo.got
	if g.ShiftName != "Malam" || g.LateMinutes != 15 || g.ScheduledEnd.Sub(*g.ScheduledStart) != 8*time.Hour ||
		g.Date != "2026-03-02" || *g.DistanceM < 30 || *g.DistanceM > 40 {
		t.Fatalf("wrong snapshot: %+v distance %d", g, *g.DistanceM)
	}
}

func fptr(v float64) *float64 { return &v }
