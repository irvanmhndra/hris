package dto

type Login struct {
	Company  string `json:"company"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
type Employee struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	DepartmentID int64  `json:"department_id"`
	Position     string `json:"position"`
	Status       string `json:"status"`
	JoinedOn     string `json:"joined_on"`
	Password     string `json:"password"`
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
