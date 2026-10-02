package model

import "time"

type Shift struct {
	ID           int64  `db:"id" json:"id"`
	Name         string `db:"name" json:"name"`
	StartTime    string `db:"start_time" json:"start_time"`
	EndTime      string `db:"end_time" json:"end_time"`
	GraceMinutes int    `db:"grace_minutes" json:"grace_minutes"`
	Active       bool   `db:"active" json:"active"`
}

type AttendanceLocation struct {
	ID        int64   `db:"id" json:"id"`
	Name      string  `db:"name" json:"name"`
	Latitude  float64 `db:"latitude" json:"latitude"`
	Longitude float64 `db:"longitude" json:"longitude"`
	RadiusM   int     `db:"radius_m" json:"radius_m"`
}

// ShiftAssignment pins one date of an employee to a shift, or to a day off
// when ShiftID is nil.
type ShiftAssignment struct {
	EmployeeID int64  `db:"employee_id"`
	Date       string `db:"date"`
	ShiftID    *int64 `db:"shift_id"`
}

type ScheduleEmployee struct {
	ID       int64   `db:"id"`
	Name     string  `db:"name"`
	ShiftID  *int64  `db:"shift_id"`
	JoinedOn string  `db:"joined_on"`
	LeftOn   *string `db:"left_on"`
}

// ScheduleSource is everything needed to resolve schedules in a date range.
type ScheduleSource struct {
	Calendar    WorkCalendar
	Holidays    []string
	Shifts      []Shift
	Employees   []ScheduleEmployee
	Assignments []ShiftAssignment
	Locations   []AttendanceLocation
	// Attendances and LeaveDays are only loaded for summaries.
	Attendances []Attendance
	LeaveDays   []ShiftAssignment // EmployeeID + Date of approved leave
}

// ClockInput records a check-in (with its schedule snapshot) or check-out.
type ClockInput struct {
	Out            bool
	At             time.Time
	Date           string
	ShiftName      string
	ScheduledStart *time.Time
	ScheduledEnd   *time.Time
	GraceMinutes   int
	LateMinutes    int
	Latitude       *float64
	Longitude      *float64
	DistanceM      *int
}

type AttendanceSummary struct {
	EmployeeID        int64  `json:"employee_id"`
	Name              string `json:"name"`
	ScheduledDays     int    `json:"scheduled_days"`
	PresentDays       int    `json:"present_days"`
	LateDays          int    `json:"late_days"`
	LateMinutes       int    `json:"late_minutes"`
	EarlyLeaveDays    int    `json:"early_leave_days"`
	EarlyLeaveMinutes int    `json:"early_leave_minutes"`
	LeaveDays         int    `json:"leave_days"`
	AbsentDays        int    `json:"absent_days"`
}
