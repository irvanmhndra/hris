package model

import (
	"encoding/json"
	"github.com/lib/pq"
	"time"
)

type WorkCalendar struct {
	Workdays        pq.Int64Array `db:"workdays" json:"workdays"`
	AnnualAllowance int           `db:"annual_allowance" json:"annual_allowance"`
	StartTime       string        `db:"start_time" json:"start_time"`
	EndTime         string        `db:"end_time" json:"end_time"`
}
type Holiday struct {
	ID   int64  `db:"id" json:"id"`
	Date string `db:"date" json:"date"`
	Name string `db:"name" json:"name"`
}
type Balance struct {
	EmployeeID int64  `db:"employee_id" json:"employee_id"`
	Name       string `db:"name" json:"name"`
	Year       int    `db:"year" json:"year"`
	Allowance  int    `db:"allowance" json:"allowance"`
	Used       int    `db:"used" json:"used"`
	Reserved   int    `db:"reserved" json:"reserved"`
	Available  int    `db:"available" json:"available"`
}
type Profile struct {
	EmployeeID        int64  `db:"employee_id" json:"employee_id"`
	Name              string `db:"name" json:"name"`
	Email             string `db:"email" json:"email"`
	Code              string `db:"code" json:"code"`
	Department        string `db:"department" json:"department"`
	Position          string `db:"position" json:"position"`
	JoinedOn          string `db:"joined_on" json:"joined_on"`
	Phone             string `db:"phone" json:"phone"`
	Address           string `db:"address" json:"address"`
	EmergencyName     string `db:"emergency_name" json:"emergency_name"`
	EmergencyPhone    string `db:"emergency_phone" json:"emergency_phone"`
	EmergencyRelation string `db:"emergency_relation" json:"emergency_relation"`
}
type HRItem struct {
	ID           int64           `db:"id" json:"id"`
	Module       string          `db:"module" json:"module"`
	EmployeeID   *int64          `db:"employee_id" json:"employee_id"`
	EmployeeName string          `db:"employee_name" json:"employee_name"`
	Title        string          `db:"title" json:"title"`
	Description  string          `db:"description" json:"description"`
	Status       string          `db:"status" json:"status"`
	DueDate      string          `db:"due_date" json:"due_date"`
	Data         json.RawMessage `db:"data" json:"data"`
	Version      int             `db:"version" json:"version"`
	ReviewNote   string          `db:"review_note" json:"review_note"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
}
type AuditLog struct {
	ID         int64     `db:"id" json:"id"`
	Actor      string    `db:"actor" json:"actor"`
	Action     string    `db:"action" json:"action"`
	Resource   string    `db:"resource" json:"resource"`
	ResourceID int64     `db:"resource_id" json:"resource_id"`
	Summary    string    `db:"summary" json:"summary"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

// HRData is the typed shape of hr_items.data (JSONB). Each module uses a
// subset of fields; decoding into it keeps clients from injecting metadata.
type HRData struct {
	URL      string `json:"url,omitempty"`
	Category string `json:"category,omitempty"`
	Code     string `json:"code,omitempty"`
	Serial   string `json:"serial,omitempty"`
	Target   string `json:"target,omitempty"`
	Progress int    `json:"progress,omitempty"`
	Email    string `json:"email,omitempty"`
	Position string `json:"position,omitempty"`
	StartAt  string `json:"start_at,omitempty"`
	EndAt    string `json:"end_at,omitempty"`
	Date     string `json:"date,omitempty"`
	CheckIn  string `json:"check_in,omitempty"`
	CheckOut string `json:"check_out,omitempty"`
	Original string `json:"original,omitempty"`
}
