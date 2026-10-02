// Package schedule resolves who works when: per-date shift assignments,
// an employee's default shift on company workdays, or company office hours.
// It also measures lateness, early leave, and distance to an attendance
// location. Times are Asia/Jakarta (WIB).
package schedule

import (
	"math"
	"time"
)

// WIB is Asia/Jakarta without depending on the host's tz database.
var WIB = time.FixedZone("WIB", 7*3600)

type Shift struct {
	ID    int64
	Name  string
	Start string // HH:MM
	End   string // HH:MM; at or before Start means the shift ends next day
	Grace int    // minutes of tolerated lateness
}

// Assignment pins a date to a shift, or to a day off when Shift is nil.
type Assignment struct {
	Shift *Shift
}

// Day is an employee's resolved schedule for one date.
type Day struct {
	Date      string    `json:"date"`
	Off       bool      `json:"off"`
	ShiftID   *int64    `json:"shift_id"`
	ShiftName string    `json:"shift_name"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Grace     int       `json:"grace_minutes"`
	Assigned  bool      `json:"assigned"` // from a per-date assignment
}

// Calendar answers whether a date is a company workday.
type Calendar interface {
	IsWorkday(d time.Time) bool
}

// Resolve applies, in order: a per-date assignment, then (on company
// workdays only) the employee's default shift or the company office hours.
// date must be a UTC midnight.
func Resolve(date time.Time, cal Calendar, assignment *Assignment, defaultShift *Shift, office Shift) Day {
	d := Day{Date: date.Format(time.DateOnly)}
	var s *Shift
	switch {
	case assignment != nil:
		d.Assigned = true
		s = assignment.Shift
	case !cal.IsWorkday(date):
	case defaultShift != nil:
		s = defaultShift
	default:
		s = &office
	}
	if s == nil {
		d.Off = true
		return d
	}
	d.ShiftName, d.Grace = s.Name, s.Grace
	if s.ID != 0 {
		id := s.ID
		d.ShiftID = &id
	}
	d.Start, d.End = Window(date, s.Start, s.End)
	return d
}

// Window turns a date and HH:MM times into instants; an end at or before the
// start rolls to the next day (overnight shift).
func Window(date time.Time, start, end string) (time.Time, time.Time) {
	at := func(hhmm string) time.Time {
		t, _ := time.Parse("15:04", hhmm)
		return time.Date(date.Year(), date.Month(), date.Day(), t.Hour(), t.Minute(), 0, 0, WIB)
	}
	s, e := at(start), at(end)
	if !e.After(s) {
		e = e.AddDate(0, 0, 1)
	}
	return s, e
}

// LateMinutes is how late a check-in is beyond the grace period.
func LateMinutes(d Day, checkIn time.Time) int {
	if d.Off {
		return 0
	}
	late := checkIn.Sub(d.Start.Add(time.Duration(d.Grace) * time.Minute))
	return max(int(late/time.Minute), 0)
}

// EarlyMinutes is how early a check-out is before the scheduled end.
func EarlyMinutes(d Day, checkOut time.Time) int {
	if d.Off {
		return 0
	}
	return max(int(d.End.Sub(checkOut)/time.Minute), 0)
}

// DistanceMeters is the great-circle (haversine) distance.
func DistanceMeters(lat1, lng1, lat2, lng2 float64) float64 {
	const r = 6_371_000.0
	rad := math.Pi / 180
	dLat, dLng := (lat2-lat1)*rad, (lng2-lng1)*rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * r * math.Asin(math.Sqrt(a))
}
