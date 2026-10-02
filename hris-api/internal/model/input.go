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
	LeftOn    string
	ManagerID *int64
	ShiftID   *int64
	// CandidateID converts a recruitment candidate (offer or hired) into
	// this new employee; ActorID is recorded in the audit log.
	CandidateID *int64
	ActorID     int64
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

// ListFilter narrows and pages a list. Limit 0 means no limit.
type ListFilter struct {
	Status string
	Date   string
	Limit  int
	Offset int
}

type RegisterInput struct {
	CompanyName  string
	CompanySlug  string
	Name         string
	Email        string
	PasswordHash string
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
