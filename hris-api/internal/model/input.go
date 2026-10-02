package model

import "encoding/json"

// Repository inputs. Services validate/normalise request DTOs and map them onto
// these types, so the persistence layer never depends on HTTP request shapes or
// on the authenticated principal.

type EmployeeInput struct {
	Code         string
	Name         string
	Email        string
	DepartmentID int64
	Position     string
	Status       string
	JoinedOn     string
	// LeftOn is the last working day of an inactive employee; empty keeps the
	// stored date (or today on deactivation).
	LeftOn string
	// PasswordHash is empty when the password is unchanged.
	PasswordHash string
}

// EmployeeFilter narrows and pages the employee list. Limit 0 means "all".
type EmployeeFilter struct {
	Search       string
	Status       string
	DepartmentID int64
	Limit        int
	Offset       int
}

type LeaveInput struct {
	Kind      string
	StartDate string
	EndDate   string
	Reason    string
}

type HRItemInput struct {
	EmployeeID  *int64
	Title       string
	Description string
	Status      string
	DueDate     string
	Data        json.RawMessage
	Version     int
}

type HRActionInput struct {
	Action   string
	Note     string
	Progress int
	Version  int
}

type SalaryInput struct {
	BasicSalary         int64
	PTKPStatus          string
	TaxMethod           string
	BPJSKesehatan       bool
	BPJSKetenagakerjaan bool
	BPJSPensiun         bool
	OvertimeEligible    bool
	Note                string
	Components          []SalaryComponent
}

// HRItemScope restricts which hr_items a caller may see or act on.
type HRItemScope struct {
	// EmployeeID limits rows to one employee's items (nil = company-wide).
	EmployeeID *int64
	// IncludeUnpublished shows draft/archived announcements and documents.
	IncludeUnpublished bool
}
