package service

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/payroll"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/internal/schedule"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
)

type AttendanceService struct {
	repo repository.AttendanceRepository
	now  func() time.Time
}

func NewAttendanceService(repo repository.AttendanceRepository) *AttendanceService {
	return &AttendanceService{repo: repo, now: time.Now}
}

// Attendances returns the latest records: company-wide for admins, own for employees.
func (s *AttendanceService) Attendances(ctx context.Context, u *model.User) ([]model.Attendance, error) {
	return s.repo.Attendances(ctx, u.CompanyID, ownScope(u))
}

// Today is an employee's schedule for today plus the check-in rules.
type Today struct {
	Schedule        schedule.Day `json:"schedule"`
	RequireLocation bool         `json:"require_location"`
	Locations       int          `json:"locations"`
}

// resolver builds schedules from a loaded source.
type resolver struct {
	cal         payroll.Calendar
	office      schedule.Shift
	shifts      map[int64]*schedule.Shift
	assignments map[string]*schedule.Assignment // employee|date
}

func newResolver(src model.ScheduleSource) resolver {
	r := resolver{
		cal:         payroll.NewCalendar(src.Calendar.Workdays, src.Holidays),
		office:      schedule.Shift{Name: "Jam kantor", Start: src.Calendar.StartTime, End: src.Calendar.EndTime},
		shifts:      map[int64]*schedule.Shift{},
		assignments: map[string]*schedule.Assignment{},
	}
	for _, s := range src.Shifts {
		r.shifts[s.ID] = &schedule.Shift{ID: s.ID, Name: s.Name, Start: s.StartTime, End: s.EndTime, Grace: s.GraceMinutes}
	}
	for _, a := range src.Assignments {
		as := &schedule.Assignment{}
		if a.ShiftID != nil {
			as.Shift = r.shifts[*a.ShiftID]
		}
		r.assignments[fmt.Sprint(a.EmployeeID, "|", a.Date)] = as
	}
	return r
}

func (r resolver) day(e model.ScheduleEmployee, date time.Time) schedule.Day {
	var def *schedule.Shift
	if e.ShiftID != nil {
		def = r.shifts[*e.ShiftID]
	}
	return schedule.Resolve(date, r.cal, r.assignments[fmt.Sprint(e.ID, "|", date.Format(time.DateOnly))], def, r.office)
}

// wibDate is the WIB calendar date of t as a UTC midnight.
func wibDate(t time.Time) time.Time {
	w := t.In(schedule.WIB)
	return time.Date(w.Year(), w.Month(), w.Day(), 0, 0, 0, 0, time.UTC)
}

func (s *AttendanceService) Today(ctx context.Context, u *model.User) (Today, error) {
	if u.EmployeeID == nil {
		return Today{}, apperror.Forbidden("Akses tidak diizinkan")
	}
	date := wibDate(s.now())
	d := date.Format(time.DateOnly)
	src, err := s.repo.ScheduleSource(ctx, u.CompanyID, u.EmployeeID, d, d, false)
	if err != nil {
		return Today{}, err
	}
	if len(src.Employees) == 0 {
		return Today{}, apperror.NotFound("Data karyawan tidak ditemukan")
	}
	return Today{
		Schedule:        newResolver(src).day(src.Employees[0], date),
		RequireLocation: src.Calendar.RequireLocation,
		Locations:       len(src.Locations),
	}, nil
}

// Clock checks in or out. Check-in snapshots today's schedule and lateness.
// When the company has attendance locations the position is measured
// against the nearest one, and rejected outside it if location is required.
func (s *AttendanceService) Clock(ctx context.Context, u *model.User, action string, pos dto.Clock) error {
	if action != "in" && action != "out" {
		return apperror.Invalid("Tindakan tidak valid")
	}
	if u.EmployeeID == nil {
		return apperror.Forbidden("Akses tidak diizinkan")
	}
	now := s.now()
	date := wibDate(now)
	d := date.Format(time.DateOnly)
	src, err := s.repo.ScheduleSource(ctx, u.CompanyID, u.EmployeeID, d, d, false)
	if err != nil {
		return err
	}
	if len(src.Employees) == 0 {
		return apperror.NotFound("Data karyawan tidak ditemukan")
	}
	in := model.ClockInput{Out: action == "out", At: now, Date: d}
	if err = checkPosition(src, pos, &in); err != nil {
		return err
	}
	if !in.Out {
		day := newResolver(src).day(src.Employees[0], date)
		in.ShiftName = day.ShiftName
		if !day.Off {
			in.ScheduledStart, in.ScheduledEnd = &day.Start, &day.End
			in.GraceMinutes = day.Grace
			in.LateMinutes = schedule.LateMinutes(day, now)
		} else {
			in.ShiftName = "Libur"
		}
	}
	return s.repo.Clock(ctx, u.CompanyID, *u.EmployeeID, in)
}

func checkPosition(src model.ScheduleSource, pos dto.Clock, in *model.ClockInput) error {
	if len(src.Locations) == 0 {
		return nil
	}
	if pos.Latitude == nil || pos.Longitude == nil ||
		math.Abs(*pos.Latitude) > 90 || math.Abs(*pos.Longitude) > 180 {
		if src.Calendar.RequireLocation {
			return apperror.Invalid("Aktifkan izin lokasi untuk absensi")
		}
		return nil
	}
	nearest, radius := math.MaxFloat64, 0
	for _, l := range src.Locations {
		if m := schedule.DistanceMeters(*pos.Latitude, *pos.Longitude, l.Latitude, l.Longitude); m < nearest {
			nearest, radius = m, l.RadiusM
		}
	}
	distance := int(math.Round(nearest))
	if src.Calendar.RequireLocation && distance > radius {
		return apperror.Invalid(fmt.Sprintf("Anda berada %d m dari lokasi absensi terdekat (maksimal %d m)", distance, radius))
	}
	in.Latitude, in.Longitude, in.DistanceM = pos.Latitude, pos.Longitude, &distance
	return nil
}

// EmployeeSchedule is one employee's resolved days.
type EmployeeSchedule struct {
	EmployeeID int64          `json:"employee_id"`
	Name       string         `json:"name"`
	ShiftID    *int64         `json:"shift_id"`
	Days       []schedule.Day `json:"days"`
}

func parseRange(from, to string, maxDays int) (time.Time, time.Time, error) {
	a, e1 := time.Parse(time.DateOnly, from)
	b, e2 := time.Parse(time.DateOnly, to)
	if e1 != nil || e2 != nil || b.Before(a) || b.Sub(a) > time.Duration(maxDays-1)*24*time.Hour {
		return a, b, apperror.Invalid(fmt.Sprintf("Rentang tanggal harus valid dan maksimal %d hari", maxDays))
	}
	return a, b, nil
}

func (s *AttendanceService) Schedule(ctx context.Context, u *model.User, from, to string) ([]EmployeeSchedule, error) {
	a, b, err := parseRange(from, to, 42)
	if err != nil {
		return nil, err
	}
	src, err := s.repo.ScheduleSource(ctx, u.CompanyID, nil, from, to, false)
	if err != nil {
		return nil, err
	}
	res := newResolver(src)
	v := make([]EmployeeSchedule, 0, len(src.Employees))
	for _, e := range src.Employees {
		es := EmployeeSchedule{EmployeeID: e.ID, Name: e.Name, ShiftID: e.ShiftID}
		for d := a; !d.After(b); d = d.AddDate(0, 0, 1) {
			es.Days = append(es.Days, res.day(e, d))
		}
		v = append(v, es)
	}
	return v, nil
}

// Summary totals a month per employee: scheduled days up to today, presence,
// lateness, early leave, approved leave, and unexplained absence.
func (s *AttendanceService) Summary(ctx context.Context, u *model.User, month string) ([]model.AttendanceSummary, error) {
	start, err := time.Parse("2006-01", month)
	if err != nil {
		return nil, apperror.Invalid("Bulan harus YYYY-MM")
	}
	end := start.AddDate(0, 1, -1)
	if today := wibDate(s.now()); today.Before(end) {
		end = today
	}
	from, to := start.Format(time.DateOnly), end.Format(time.DateOnly)
	if end.Before(start) {
		return []model.AttendanceSummary{}, nil
	}
	src, err := s.repo.ScheduleSource(ctx, u.CompanyID, nil, from, to, true)
	if err != nil {
		return nil, err
	}
	res := newResolver(src)
	present := map[string]model.Attendance{}
	for _, a := range src.Attendances {
		present[fmt.Sprint(a.EmployeeID, "|", a.Date)] = a
	}
	leave := map[string]bool{}
	for _, l := range src.LeaveDays {
		leave[fmt.Sprint(l.EmployeeID, "|", l.Date)] = true
	}
	v := make([]model.AttendanceSummary, 0, len(src.Employees))
	for _, e := range src.Employees {
		sum := model.AttendanceSummary{EmployeeID: e.ID, Name: e.Name}
		joined, _ := time.Parse(time.DateOnly, e.JoinedOn)
		last := end
		if e.LeftOn != nil {
			if l, err := time.Parse(time.DateOnly, *e.LeftOn); err == nil && l.Before(last) {
				last = l
			}
		}
		for d := start; !d.After(last); d = d.AddDate(0, 0, 1) {
			if d.Before(joined) {
				continue
			}
			key := fmt.Sprint(e.ID, "|", d.Format(time.DateOnly))
			a, ok := present[key]
			if ok {
				sum.PresentDays++
				if a.LateMinutes > 0 {
					sum.LateDays++
					sum.LateMinutes += a.LateMinutes
				}
				if a.EarlyLeaveMinutes > 0 {
					sum.EarlyLeaveDays++
					sum.EarlyLeaveMinutes += a.EarlyLeaveMinutes
				}
			}
			if res.day(e, d).Off {
				continue
			}
			sum.ScheduledDays++
			switch {
			case leave[key]:
				sum.LeaveDays++
			case !ok:
				sum.AbsentDays++
			}
		}
		v = append(v, sum)
	}
	return v, nil
}

var hhmm = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

func (s *AttendanceService) Shifts(ctx context.Context, companyID int64) ([]model.Shift, error) {
	return s.repo.Shifts(ctx, companyID)
}

func (s *AttendanceService) SaveShift(ctx context.Context, u *model.User, id int64, v model.Shift) (int64, error) {
	v.Name = strings.TrimSpace(v.Name)
	if v.Name == "" || len(v.Name) > 80 || !hhmm.MatchString(v.StartTime) || !hhmm.MatchString(v.EndTime) ||
		v.StartTime == v.EndTime || v.GraceMinutes < 0 || v.GraceMinutes > 240 {
		return 0, apperror.Invalid("Shift wajib bernama, jam HH:MM (mulai ≠ selesai), toleransi 0–240 menit")
	}
	if id == 0 {
		v.Active = true
	}
	return s.repo.SaveShift(ctx, u.CompanyID, u.ID, id, v)
}

func (s *AttendanceService) SetAssignments(ctx context.Context, u *model.User, v dto.ShiftAssignments) error {
	if _, _, err := parseRange(v.From, v.To, 92); err != nil {
		return err
	}
	slices.Sort(v.EmployeeIDs)
	v.EmployeeIDs = slices.Compact(v.EmployeeIDs)
	if len(v.EmployeeIDs) == 0 || len(v.EmployeeIDs) > 500 || v.EmployeeIDs[0] <= 0 {
		return apperror.Invalid("Pilih 1–500 karyawan")
	}
	if v.Clear {
		v.ShiftID = nil
	}
	return s.repo.SetAssignments(ctx, u.CompanyID, u.ID, v.EmployeeIDs, v.From, v.To, v.ShiftID, v.Clear)
}

func (s *AttendanceService) Locations(ctx context.Context, companyID int64) ([]model.AttendanceLocation, error) {
	return s.repo.Locations(ctx, companyID)
}

func (s *AttendanceService) SaveLocation(ctx context.Context, u *model.User, id int64, v model.AttendanceLocation) (int64, error) {
	v.Name = strings.TrimSpace(v.Name)
	if v.Name == "" || len(v.Name) > 80 || math.Abs(v.Latitude) > 90 || math.Abs(v.Longitude) > 180 ||
		v.RadiusM < 10 || v.RadiusM > 5000 {
		return 0, apperror.Invalid("Lokasi wajib bernama, koordinat valid, dan radius 10–5000 m")
	}
	return s.repo.SaveLocation(ctx, u.CompanyID, u.ID, id, v)
}

func (s *AttendanceService) DeleteLocation(ctx context.Context, u *model.User, id int64) error {
	return s.repo.DeleteLocation(ctx, u.CompanyID, u.ID, id)
}
