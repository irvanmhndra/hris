package dto

import "encoding/json"

type HRItem struct {
	EmployeeID  *int64          `json:"employee_id"`
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Status      string          `json:"status"`
	DueDate     string          `json:"due_date"`
	Data        json.RawMessage `json:"data"`
	Version     int             `json:"version"`
}
type HRAction struct {
	Action   string `json:"action"`
	Note     string `json:"note"`
	Progress int    `json:"progress"`
	Version  int    `json:"version"`
}
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
