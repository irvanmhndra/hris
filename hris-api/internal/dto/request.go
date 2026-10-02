package dto

type Login struct {
	Company  string `json:"company"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// ListQuery pages a list (?page=&per_page=) and filters it by status or
// date. Without page the list keeps its original unpaginated response.
type ListQuery struct {
	Page    int
	PerPage int
	Status  string
	Date    string
}

type ForgotPassword struct {
	Company string `json:"company"`
	Email   string `json:"email"`
}

type ResetPassword struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type Register struct {
	CompanyName string `json:"company_name"`
	CompanySlug string `json:"company_slug"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

type Employee struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	DepartmentID int64  `json:"department_id"`
	Position     string `json:"position"`
	Status       string `json:"status"`
	JoinedOn     string `json:"joined_on"`
	LeftOn       string `json:"left_on"`
	ManagerID    *int64 `json:"manager_id"`
	ShiftID      *int64 `json:"shift_id"`
	Password     string `json:"password"`
}

// TeamReview is a manager's decision on a direct report's request.
type TeamReview struct {
	Action  string `json:"action"`
	Note    string `json:"note"`
	Version int    `json:"version"`
}

// Clock carries the browser's position for geofenced check-in/out.
type Clock struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

// ShiftAssignments sets (or clears) a date range of shifts for employees.
// A nil ShiftID with Clear false records days off.
type ShiftAssignments struct {
	EmployeeIDs []int64 `json:"employee_ids"`
	From        string  `json:"from"`
	To          string  `json:"to"`
	ShiftID     *int64  `json:"shift_id"`
	Clear       bool    `json:"clear"`
}

// EmployeeQuery filters the employee list. Page 0 returns every match
// (unpaginated), which the CSV export and employee pickers rely on.
type EmployeeQuery struct {
	Search       string
	Status       string
	DepartmentID int64
	Page         int
	PerPage      int
}

type Leave struct {
	Kind      string `json:"kind"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	Reason    string `json:"reason"`
}
