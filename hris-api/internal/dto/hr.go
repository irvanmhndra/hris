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
