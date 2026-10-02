package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/irvanmhndra/hris-api/config"
	"github.com/irvanmhndra/hris-api/internal/app"
	"github.com/irvanmhndra/hris-api/internal/repository/postgres"
	"github.com/irvanmhndra/hris-api/internal/service"
	"github.com/irvanmhndra/hris-api/migrations"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func TestHRISWorkflow(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL for PostgreSQL integration test")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	schema := fmt.Sprintf("hris_test_%d", time.Now().UnixNano())
	if _, err = db.Exec(`CREATE SCHEMA ` + schema); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = db.Exec(`DROP SCHEMA ` + schema + ` CASCADE`) }()
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	isolated, err := sqlx.Connect("postgres", dsn+sep+"search_path="+schema)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = isolated.Close() }()
	// Apply migrations through the same golang-migrate runner production uses.
	if err = migrations.Up(context.Background(), isolated.DB); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err = migrations.Up(context.Background(), isolated.DB); err != nil {
		t.Fatalf("second migrate run must be a no-op: %v", err)
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("TestPassword123!"), bcrypt.MinCost)
	_, err = isolated.Exec(`INSERT INTO companies(id,name,slug) VALUES(1,'Alpha','alpha'),(2,'Beta','beta');
 INSERT INTO departments(id,company_id,name) VALUES(1,1,'Engineering'),(2,2,'People');
 INSERT INTO employees(id,company_id,code,name,email,department_id,position,joined_on) VALUES(1,1,'A1','Employee A','staff@alpha.test',1,'Engineer','2025-01-01'),(2,2,'B1','Employee B','staff@beta.test',2,'HR','2025-01-01');
 SELECT setval('employees_id_seq',2); SELECT setval('departments_id_seq',2); SELECT setval('companies_id_seq',2);`)
	if err != nil {
		t.Fatal(err)
	}
	for _, u := range []struct {
		id, c       int
		employee    any
		email, role string
	}{{1, 1, nil, "admin@alpha.test", "admin"}, {2, 1, 1, "staff@alpha.test", "employee"}, {3, 2, nil, "admin@beta.test", "admin"}, {4, 2, 2, "staff@beta.test", "employee"}} {
		_, err = isolated.Exec(`INSERT INTO users(id,company_id,employee_id,name,email,password_hash,role) VALUES($1,$2,$3,'Test User',$4,$5,$6)`, u.id, u.c, u.employee, u.email, string(hash), u.role)
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err = isolated.Exec(`SELECT setval('users_id_seq',4)`); err != nil {
		t.Fatal(err)
	}
	auth := postgres.NewAuthRepository(isolated)
	e := app.NewServer(isolated, config.Config{SignupEnabled: true, UploadDir: t.TempDir()})
	request := func(method, path, token string, body any, want int) map[string]any {
		t.Helper()
		b, _ := json.Marshal(body)
		r := httptest.NewRequest(method, "/api/v1"+path, bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		e.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("%s %s: want %d, got %d: %s", method, path, want, w.Code, w.Body.String())
		}
		var v map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	session := func(id int64, token string) {
		if err := auth.CreateSession(context.Background(), id, service.HashToken(token)); err != nil {
			t.Fatal(err)
		}
	}
	session(1, "admin")
	session(2, "staff")
	session(3, "other")
	session(4, "otherstaff")
	request("POST", "/auth/login", "", map[string]string{"company": "alpha", "email": "admin@alpha.test", "password": "wrong"}, 401)
	login := request("POST", "/auth/login", "", map[string]string{"company": "alpha", "email": "admin@alpha.test", "password": "TestPassword123!"}, 200)
	token := login["data"].(map[string]any)["token"].(string)
	request("GET", "/auth/me", token, nil, 200)
	request("GET", "/employees", "", nil, 401)
	request("GET", "/employees", "staff", nil, 403)
	rows := request("GET", "/employees", "admin", nil, 200)["data"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["name"] != "Employee A" {
		t.Fatal("tenant isolation failed")
	}
	employee := map[string]any{"name": "New Person", "email": "new@alpha.test", "code": "A2", "department_id": 2, "position": "Engineer", "status": "active", "joined_on": "2026-01-01", "password": "NewPassword123!"}
	request("POST", "/employees", "admin", employee, 422)
	employee["department_id"] = 1
	created := request("POST", "/employees", "admin", employee, 200)["data"].(map[string]any)["id"].(float64)
	employee["email"] = "changed@alpha.test"
	employee["password"] = ""
	request("PUT", fmt.Sprintf("/employees/%.0f", created), "admin", employee, 200)
	var accounts int
	if err = isolated.Get(&accounts, `SELECT count(*) FROM users WHERE employee_id=$1`, created); err != nil {
		t.Fatal(err)
	}
	if accounts != 1 {
		t.Fatal("email change duplicated accounts")
	}
	employee["email"] = "staff@alpha.test"
	request("PUT", fmt.Sprintf("/employees/%.0f", created), "admin", employee, 409)
	request("PUT", "/employees/2", "admin", employee, 404)
	request("POST", "/attendance/in", "admin", nil, 403)
	request("POST", "/attendance/out", "staff", nil, 404)
	request("POST", "/attendance/in", "staff", nil, 200)
	request("POST", "/attendance/in", "staff", nil, 409)
	request("POST", "/attendance/out", "staff", nil, 200)
	request("POST", "/attendance/out", "staff", nil, 404)
	if len(request("GET", "/attendances", "other", nil, 200)["data"].([]any)) != 0 {
		t.Fatal("attendance leaked across companies")
	}
	leave := map[string]string{"kind": "annual", "start_date": "2026-11-10", "end_date": "2026-11-09", "reason": "Family gathering"}
	request("POST", "/leaves", "staff", leave, 422)
	leave["end_date"] = "2026-11-12"
	request("POST", "/leaves", "staff", leave, 200)
	request("POST", "/leaves", "staff", leave, 409)
	if len(request("GET", "/leaves", "otherstaff", nil, 200)["data"].([]any)) != 0 {
		t.Fatal("leave leaked across companies")
	}
	request("PATCH", "/leaves/1", "other", map[string]string{"status": "approved"}, 404)
	request("PATCH", "/leaves/1", "staff", map[string]string{"status": "approved"}, 403)
	request("PATCH", "/leaves/1", "admin", map[string]string{"status": "approved"}, 200)
	request("PATCH", "/leaves/1", "admin", map[string]string{"status": "rejected"}, 404)

	// Leave balances preserve reservations, ignore holidays, and restore cancelled requests.
	balances := request("GET", "/leave-balances?year=2026", "staff", nil, 200)["data"].([]any)
	if len(balances) != 1 || balances[0].(map[string]any)["used"] != float64(3) {
		t.Fatal("approved leave not charged")
	}
	// Simulate migrated leave without an allocation: changing defaults must preserve its entitlement.
	if _, err = isolated.Exec(`DELETE FROM leave_allocations WHERE employee_id=1 AND year=2026`); err != nil {
		t.Fatal(err)
	}
	calendar := map[string]any{"workdays": []int{1, 2, 3, 4, 5}, "annual_allowance": 8, "start_time": "09:00", "end_time": "18:00"}
	request("PUT", "/calendar", "staff", calendar, 403)
	request("PUT", "/calendar", "admin", calendar, 200)
	preserved := request("GET", "/leave-balances?year=2026", "staff", nil, 200)["data"].([]any)[0].(map[string]any)
	if preserved["allowance"] != float64(12) || preserved["used"] != float64(3) {
		t.Fatalf("calendar change altered existing entitlement: %v", preserved)
	}
	future := request("GET", "/leave-balances?year=2028", "staff", nil, 200)["data"].([]any)[0].(map[string]any)
	if future["allowance"] != float64(8) {
		t.Fatal("new allocation did not use updated calendar default")
	}
	request("POST", "/holidays", "staff", map[string]string{"date": "2026-11-16", "name": "Company holiday"}, 403)
	request("POST", "/holidays", "admin", map[string]string{"date": "2026-11-16", "name": "Company holiday"}, 200)
	request("POST", "/leaves", "staff", map[string]string{"kind": "annual", "start_date": "2026-11-13", "end_date": "2026-11-16", "reason": "Holiday test"}, 200)
	leaveRows := request("GET", "/leaves", "staff", nil, 200)["data"].([]any)
	pending := leaveRows[0].(map[string]any)
	if pending["days"] != float64(1) {
		t.Fatalf("expected one working day: %v", pending)
	}
	request("PUT", "/leave-balances/1", "admin", map[string]int{"year": 2026, "allowance": 3}, 409)
	request("POST", fmt.Sprintf("/leaves/%.0f/cancel", pending["id"]), "staff", nil, 200)
	request("PUT", "/leave-balances/1", "admin", map[string]int{"year": 2026, "allowance": 3}, 200)
	request("POST", "/leaves", "staff", map[string]string{"kind": "annual", "start_date": "2026-11-17", "end_date": "2026-11-17", "reason": "Insufficient balance"}, 409)
	request("POST", "/leaves", "staff", map[string]string{"kind": "annual", "start_date": "2026-11-14", "end_date": "2026-11-15", "reason": "Weekend only"}, 422)
	request("PUT", "/leave-balances/1", "admin", map[string]int{"year": 2027, "allowance": 0}, 200)
	request("POST", "/leaves", "staff", map[string]string{"kind": "annual", "start_date": "2026-12-31", "end_date": "2027-01-04", "reason": "Cross-year balance"}, 409)

	// Monthly accrual counts entitlement up to the leave month; carry-over brings unused days forward.
	calendar["leave_accrual"] = "monthly"
	calendar["carry_over_max"] = 5
	request("PUT", "/calendar", "admin", calendar, 200)
	request("POST", "/leaves", "staff", map[string]string{"kind": "annual", "start_date": "2028-02-01", "end_date": "2028-02-02", "reason": "Accrual not enough"}, 409)
	request("POST", "/leaves", "staff", map[string]string{"kind": "annual", "start_date": "2028-02-01", "end_date": "2028-02-01", "reason": "Accrued one day"}, 200)
	next := request("GET", "/leave-balances?year=2029", "staff", nil, 200)["data"].([]any)[0].(map[string]any)
	if next["carried_over"] != float64(5) || next["accrued"] != float64(8) {
		t.Fatalf("carry-over/accrual wrong: %v", next)
	}
	for _, l := range request("GET", "/leaves", "staff", nil, 200)["data"].([]any) {
		if row := l.(map[string]any); row["start_date"] == "2028-02-01" {
			request("POST", fmt.Sprintf("/leaves/%.0f/cancel", row["id"]), "staff", nil, 200)
		}
	}
	calendar["leave_accrual"] = "weekly"
	request("PUT", "/calendar", "admin", calendar, 422)
	calendar["leave_accrual"] = "annual"
	calendar["carry_over_max"] = 0
	request("PUT", "/calendar", "admin", calendar, 200)

	// Two-step approval: the direct manager forwards to HR or rejects; HR decides last.
	staffEmployee := map[string]any{"name": "Employee A", "email": "staff@alpha.test", "code": "A1", "department_id": 1, "position": "Engineer", "status": "active", "joined_on": "2025-01-01", "manager_id": created}
	request("PUT", "/employees/1", "admin", staffEmployee, 200)
	employee["email"] = "changed@alpha.test"
	employee["manager_id"] = 1
	request("PUT", fmt.Sprintf("/employees/%.0f", created), "admin", employee, 422)
	delete(employee, "manager_id")
	var managerUser int64
	if err = isolated.Get(&managerUser, `SELECT id FROM users WHERE employee_id=$1`, created); err != nil {
		t.Fatal(err)
	}
	session(managerUser, "manager")
	if request("GET", "/auth/me", "manager", nil, 200)["data"].(map[string]any)["is_manager"] != true {
		t.Fatal("manager flag missing")
	}
	// Unpaid leave skips the annual balance (fully used) and reduces December pay.
	request("POST", "/leaves", "staff", map[string]string{"kind": "unpaid", "start_date": "2026-12-02", "end_date": "2026-12-03", "reason": "Personal matters"}, 200)
	if len(request("GET", "/team/approvals", "staff", nil, 200)["data"].([]any)) != 0 {
		t.Fatal("non-manager sees team approvals")
	}
	team := request("GET", "/team/approvals", "manager", nil, 200)["data"].([]any)
	if len(team) != 1 || team[0].(map[string]any)["kind"] != "unpaid" {
		t.Fatalf("manager queue wrong: %v", team)
	}
	unpaidID := team[0].(map[string]any)["id"].(float64)
	reviewPath := fmt.Sprintf("/team/approvals/leave/%.0f", unpaidID)
	request("POST", reviewPath, "manager", map[string]string{"action": "reject"}, 422)
	request("POST", reviewPath, "other", map[string]string{"action": "approve"}, 403)
	request("POST", reviewPath, "manager", map[string]string{"action": "approve"}, 200)
	request("POST", reviewPath, "manager", map[string]string{"action": "approve"}, 404)
	for _, l := range request("GET", "/leaves", "admin", nil, 200)["data"].([]any) {
		if row := l.(map[string]any); row["id"] == unpaidID && (row["stage"] != "hr" || row["status"] != "pending") {
			t.Fatalf("manager approval must forward to HR: %v", row)
		}
	}
	request("PATCH", fmt.Sprintf("/leaves/%.0f", unpaidID), "admin", map[string]string{"status": "approved"}, 200)

	// Self-service profiles cannot mutate identity or another tenant.
	request("PUT", "/profile", "staff", map[string]any{"phone": "0812345678", "name": "Injected Name", "employee_id": 2}, 200)
	profile := request("GET", "/profile", "staff", nil, 200)["data"].(map[string]any)
	if profile["name"] != "Employee A" || profile["phone"] != "0812345678" {
		t.Fatal("profile identity boundary failed")
	}
	request("GET", "/employees/2/profile", "admin", nil, 404)

	itemID := func(v map[string]any) int64 { return int64(v["data"].(map[string]any)["id"].(float64)) }
	actionPath := func(module string, id int64) string { return fmt.Sprintf("/hr/%s/%d/action", module, id) }
	announcement := map[string]any{"title": "Company update", "description": "Welcome everyone", "status": "draft", "data": map[string]any{}}
	request("POST", "/hr/announcements", "staff", announcement, 403)
	news := itemID(request("POST", "/hr/announcements", "admin", announcement, 200))
	if len(request("GET", "/hr/announcements", "staff", nil, 200)["data"].([]any)) != 0 {
		t.Fatal("draft announcement leaked")
	}
	announcement["status"] = "published"
	announcement["version"] = 1
	request("PUT", fmt.Sprintf("/hr/announcements/%d", news), "admin", announcement, 200)
	if len(request("GET", "/hr/announcements", "staff", nil, 200)["data"].([]any)) != 1 {
		t.Fatal("published announcement missing")
	}
	if len(request("GET", "/hr/announcements", "otherstaff", nil, 200)["data"].([]any)) != 0 {
		t.Fatal("announcement tenant leak")
	}
	request("POST", "/hr/documents", "admin", map[string]any{"title": "Unsafe URL", "status": "published", "data": map[string]string{"url": "javascript:alert(1)"}}, 422)
	onboarding := itemID(request("POST", "/hr/onboarding", "admin", map[string]any{"title": "Read handbook", "employee_id": 1, "status": "todo"}, 200))
	request("PATCH", actionPath("onboarding", onboarding), "otherstaff", map[string]any{"action": "complete", "version": 1}, 404)
	request("PATCH", actionPath("onboarding", onboarding), "staff", map[string]any{"action": "start", "version": 1}, 200)
	request("PATCH", actionPath("onboarding", onboarding), "staff", map[string]any{"action": "complete", "version": 1}, 409)
	request("PATCH", actionPath("onboarding", onboarding), "staff", map[string]any{"action": "complete", "version": 2}, 200)
	goal := itemID(request("POST", "/hr/goals", "admin", map[string]any{"title": "Release HRIS", "employee_id": 1, "status": "active", "data": map[string]any{"target": "Deliver five features", "progress": 0}}, 200))
	request("PATCH", actionPath("goals", goal), "staff", map[string]any{"action": "progress", "progress": 101, "version": 1}, 422)
	request("PATCH", actionPath("goals", goal), "staff", map[string]any{"action": "progress", "progress": 100, "version": 1}, 200)
	request("POST", "/hr/assets", "admin", map[string]any{"title": "Laptop", "employee_id": 2, "status": "assigned", "data": map[string]any{"code": "LAP-01"}}, 404)
	request("POST", "/hr/assets", "admin", map[string]any{"title": "Laptop", "employee_id": 1, "status": "assigned", "data": map[string]any{"code": "LAP-01"}}, 200)
	request("POST", "/hr/assets", "admin", map[string]any{"title": "Laptop duplicate", "status": "available", "data": map[string]any{"code": "LAP-01"}}, 409)
	request("GET", "/hr/recruitment", "staff", nil, 403)
	candidate := itemID(request("POST", "/hr/recruitment", "admin", map[string]any{"title": "Candidate Test", "status": "applied", "data": map[string]any{"email": "candidate@example.test", "position": "Engineer"}}, 200))
	overtime := map[string]any{"title": "Release support", "description": "Support production release", "employee_id": 2, "data": map[string]any{"start_at": "2026-11-10T18:00:00+07:00", "end_at": "2026-11-10T20:00:00+07:00"}}
	ot := itemID(request("POST", "/hr/overtime", "staff", overtime, 200))
	request("POST", "/hr/overtime", "staff", overtime, 409)
	request("PATCH", actionPath("overtime", ot), "staff", map[string]any{"action": "approve", "version": 1}, 403)
	request("PATCH", actionPath("overtime", ot), "admin", map[string]any{"action": "approve", "version": 1}, 200)
	request("PATCH", actionPath("overtime", ot), "admin", map[string]any{"action": "approve", "version": 1}, 409)
	correction := itemID(request("POST", "/hr/corrections", "staff", map[string]any{"title": "Forgot attendance", "description": "Forgot to check in yesterday", "data": map[string]any{"date": "2020-01-06", "check_in": "09:00", "check_out": "18:00"}}, 200))
	request("PATCH", actionPath("corrections", correction), "admin", map[string]any{"action": "approve", "version": 1}, 200)
	var corrected int
	if err := isolated.Get(&corrected, `SELECT count(*) FROM attendances WHERE company_id=1 AND employee_id=1 AND date='2020-01-06' AND check_out IS NOT NULL`); err != nil || corrected != 1 {
		t.Fatal("approved correction not applied")
	}

	// Shifts, per-date schedules, geofenced check-in, and the monthly summary.
	request("POST", "/shifts", "admin", map[string]any{"name": "Pagi", "start_time": "25:00", "end_time": "17:00"}, 422)
	request("POST", "/shifts", "staff", map[string]any{"name": "Pagi", "start_time": "08:00", "end_time": "17:00"}, 403)
	pagi := itemID(request("POST", "/shifts", "admin", map[string]any{"name": "Pagi", "start_time": "08:00", "end_time": "17:00", "grace_minutes": 10}, 200))
	request("PUT", "/schedule", "admin", map[string]any{"employee_ids": []int{2}, "from": "2026-03-07", "to": "2026-03-07", "shift_id": pagi}, 404)
	request("PUT", "/schedule", "admin", map[string]any{"employee_ids": []int{1}, "from": "2026-03-07", "to": "2026-03-07", "shift_id": pagi}, 200)
	week := request("GET", "/schedule?from=2026-03-06&to=2026-03-08", "admin", nil, 200)["data"].([]any)
	for _, row := range week {
		r := row.(map[string]any)
		if r["employee_id"] != float64(1) {
			continue
		}
		days := r["days"].([]any)
		fri, sat, sun := days[0].(map[string]any), days[1].(map[string]any), days[2].(map[string]any)
		if fri["shift_name"] != "Jam kantor" || sat["shift_name"] != "Pagi" || sat["assigned"] != true || sun["off"] != true {
			t.Fatalf("schedule resolution wrong: %v", days)
		}
	}
	request("POST", "/attendance-locations", "admin", map[string]any{"name": "HQ", "latitude": -6.2, "longitude": 106.8, "radius_m": 5}, 422)
	request("POST", "/attendance-locations", "admin", map[string]any{"name": "HQ", "latitude": -6.2, "longitude": 106.8, "radius_m": 100}, 200)
	calendar["require_location"] = true
	request("PUT", "/calendar", "admin", calendar, 200)
	request("POST", "/attendance/in", "staff", nil, 422)
	request("POST", "/attendance/in", "staff", map[string]float64{"latitude": -6.3, "longitude": 106.8}, 422)
	request("POST", "/attendance/in", "staff", map[string]float64{"latitude": -6.2003, "longitude": 106.8}, 409) // inside; already checked in today
	calendar["require_location"] = false
	request("PUT", "/calendar", "admin", calendar, 200)
	if request("GET", "/attendance/today", "staff", nil, 200)["data"].(map[string]any)["locations"] != float64(1) {
		t.Fatal("today view must report locations")
	}
	summary := request("GET", "/attendance-summary?month="+time.Now().Format("2006-01"), "admin", nil, 200)["data"].([]any)
	if len(summary) != 2 || summary[0].(map[string]any)["present_days"] != float64(1) {
		t.Fatalf("attendance summary wrong: %v", summary)
	}

	// Payroll calculation, snapshots, locking, per-employee confidentiality, and manual payment lifecycle.
	request("GET", "/salaries", "staff", nil, 403)
	request("POST", "/payroll", "admin", map[string]string{"period": "2026-11"}, 409)
	salary := map[string]any{
		"basic_salary": 10000000, "ptkp_status": "TK/0", "tax_method": "gross",
		"bpjs_kesehatan": true, "bpjs_ketenagakerjaan": true, "bpjs_pensiun": true, "overtime_eligible": true,
		"note": "Verified", "components": []map[string]any{
			{"kind": "allowance", "name": "Tunjangan jabatan", "amount": 1000000, "fixed": true, "taxable": true},
			{"kind": "deduction", "name": "Koperasi", "amount": 500000},
		},
	}
	request("PUT", "/salaries/2", "admin", salary, 404)
	request("PUT", "/salaries/1", "admin", salary, 200)
	manual := map[string]any{"basic_salary": 10000000, "ptkp_status": "TK/0", "tax_method": "none", "components": salary["components"]}
	request("PUT", fmt.Sprintf("/salaries/%.0f", created), "admin", manual, 200)
	run := itemID(request("POST", "/payroll", "admin", map[string]string{"period": "2026-11"}, 200))
	request("POST", "/payroll", "admin", map[string]string{"period": "2026-11"}, 409)
	request("POST", "/payroll", "admin", map[string]string{"period": "2026-12"}, 409) // November still draft
	request("PUT", "/payroll/settings", "admin", map[string]any{"jkk_rate": 25, "jp_wage_cap": 10547400, "kes_wage_cap": 12000000}, 422)
	request("PUT", "/payroll/settings", "admin", map[string]any{"jkk_rate": 54, "jp_wage_cap": 10547400, "kes_wage_cap": 12000000}, 200)
	if request("GET", "/payroll/settings", "admin", nil, 200)["data"].(map[string]any)["jkk_rate"] != float64(54) {
		t.Fatal("payroll settings not saved")
	}
	if len(request("GET", "/payslips", "staff", nil, 200)["data"].([]any)) != 0 {
		t.Fatal("draft payslip leaked")
	}
	payrollRows := request("GET", fmt.Sprintf("/payroll/%d/slips", run), "admin", nil, 200)["data"].([]any)
	if len(payrollRows) != 2 {
		t.Fatal("payroll employee snapshot incomplete")
	}
	slip := payrollRows[0].(map[string]any)
	codes := map[string]float64{}
	var inputs []map[string]any
	for _, l := range slip["lines"].([]any) {
		line := l.(map[string]any)
		codes[line["code"].(string)] += line["amount"].(float64)
		if c := line["code"]; c == "BASIC" || c == "ALLOWANCE" || c == "OVERTIME" || c == "DEDUCTION" {
			inputs = append(inputs, line)
		}
	}
	// Rp11jt fixed wage; 2h weekday overtime = 3.5h × 11jt/173; BPJS on 11jt
	// (JP capped); TER A 4% on gross incl. employer Kes/JKK/JKM premiums.
	if codes["OVERTIME"] != 222543 || codes["BPJS_KES_EE"] != 110000 || codes["JP_EE"] != 105474 ||
		codes["JKK"] != 26400 || slip["pph21"] != float64(468877) || slip["net"] != float64(9818192) {
		t.Fatalf("wrong payroll calculation: %v net %v pph21 %v", codes, slip["net"], slip["pph21"])
	}
	if payrollRows[1].(map[string]any)["net"] != float64(10500000) {
		t.Fatal("manual-tax slip must not add BPJS or PPh 21")
	}
	adjust := map[string]any{"version": 1, "note": "Bonus", "lines": append(inputs,
		map[string]any{"kind": "earning", "code": "ADJUSTMENT", "name": "Bonus proyek", "amount": 1000000, "taxable": true})}
	request("PUT", fmt.Sprintf("/payroll/slips/%.0f", slip["id"]), "admin", map[string]any{"version": 1, "lines": []map[string]any{{"kind": "deduction", "code": "PPH21", "name": "PPh", "amount": 1}}}, 422)
	request("PUT", fmt.Sprintf("/payroll/slips/%.0f", slip["id"]), "admin", adjust, 200)
	request("PUT", fmt.Sprintf("/payroll/slips/%.0f", slip["id"]), "admin", adjust, 409)
	request("PATCH", fmt.Sprintf("/payroll/%d/action", run), "other", map[string]string{"action": "finalize"}, 404)
	request("PATCH", fmt.Sprintf("/payroll/%d/action", run), "admin", map[string]string{"action": "finalize"}, 200)
	adjust["version"] = 2
	request("PUT", fmt.Sprintf("/payroll/slips/%.0f", slip["id"]), "admin", adjust, 409)
	ownSlips := request("GET", "/payslips", "staff", nil, 200)["data"].([]any)
	if len(ownSlips) != 1 || ownSlips[0].(map[string]any)["employee_id"] != float64(1) || ownSlips[0].(map[string]any)["pph21"].(float64) <= 468877 {
		t.Fatal("payslip ownership or bonus recalculation failed")
	}
	if len(request("GET", "/payslips", "otherstaff", nil, 200)["data"].([]any)) != 0 {
		t.Fatal("payslip tenant leak")
	}
	request("PATCH", fmt.Sprintf("/payroll/%d/action", run), "admin", map[string]string{"action": "paid"}, 422)
	request("PATCH", fmt.Sprintf("/payroll/%d/action", run), "admin", map[string]string{"action": "paid", "reference": "BANK-BATCH-001"}, 200)
	request("PATCH", fmt.Sprintf("/payroll/%d/action", run), "admin", map[string]string{"action": "void"}, 409)
	// December settles the year with Pasal 17; paid overtime is not paid twice.
	december := itemID(request("POST", "/payroll", "admin", map[string]any{"period": "2026-12", "thr_date": "2026-12-25"}, 200))
	decSlip := request("GET", fmt.Sprintf("/payroll/%d/slips", december), "admin", nil, 200)["data"].([]any)[0].(map[string]any)
	for _, l := range decSlip["lines"].([]any) {
		if l.(map[string]any)["code"] == "OVERTIME" {
			t.Fatal("overtime paid twice")
		}
	}
	if decSlip["final_period"] != true || decSlip["pph21"] == float64(0) || decSlip["unpaid_leave_days"] != float64(2) ||
		decSlip["worked_days"].(float64) != decSlip["period_days"].(float64)-2 {
		t.Fatalf("December must use the annual calculation: %v", decSlip)
	}
	request("PATCH", fmt.Sprintf("/payroll/%d/action", december), "admin", map[string]string{"action": "void"}, 200)

	// A correction run adds the difference to a locked period, including PPh 21.
	request("POST", fmt.Sprintf("/payroll/%d/correction", december), "admin", nil, 409) // void, not finalized
	fixRun := itemID(request("POST", fmt.Sprintf("/payroll/%d/correction", run), "admin", nil, 200))
	request("POST", fmt.Sprintf("/payroll/%d/correction", run), "admin", nil, 409)    // one open correction
	request("POST", "/payroll", "admin", map[string]string{"period": "2026-12"}, 409) // correction still draft
	fix := request("GET", fmt.Sprintf("/payroll/%d/slips", fixRun), "admin", nil, 200)["data"].([]any)[0].(map[string]any)
	if fix["net"] != float64(0) || fix["pph21"] != float64(0) || fix["run_kind"] != "correction" {
		t.Fatalf("empty correction must change nothing: %v", fix)
	}
	request("PUT", fmt.Sprintf("/payroll/slips/%.0f", fix["id"]), "admin", map[string]any{"version": 1, "lines": []map[string]any{
		{"kind": "earning", "code": "ADJUSTMENT", "name": "Kekurangan bonus", "amount": 1000000, "taxable": true}}}, 200)
	fix = request("GET", fmt.Sprintf("/payroll/%d/slips", fixRun), "admin", nil, 200)["data"].([]any)[0].(map[string]any)
	if pph := fix["pph21"].(float64); pph <= 0 || fix["net"] != 1000000-pph {
		t.Fatalf("correction must withhold the PPh 21 difference: %v", fix)
	}
	request("PATCH", fmt.Sprintf("/payroll/%d/action", fixRun), "admin", map[string]string{"action": "finalize"}, 200)
	if n := len(request("GET", "/payslips", "staff", nil, 200)["data"].([]any)); n != 2 {
		t.Fatalf("employee must see the correction slip: %d", n)
	}
	request("PUT", "/payroll/settings", "admin", map[string]any{"jkk_rate": 24, "jp_wage_cap": 1, "kes_wage_cap": 1, "late_deduction": "per_occurrence"}, 422)
	request("PUT", "/salaries/1", "admin", map[string]any{"basic_salary": 1, "ptkp_status": "TK/0", "tax_method": "gross", "nik": "123"}, 422)
	calendar["carry_over_expiry_months"] = 13
	request("PUT", "/calendar", "admin", calendar, 422)
	calendar["carry_over_expiry_months"] = 0
	request("GET", "/audit-logs", "staff", nil, 403)
	if len(request("GET", "/audit-logs", "admin", nil, 200)["data"].([]any)) < 15 {
		t.Fatal("audit trail incomplete")
	}
	pagination := func(v map[string]any) map[string]any {
		t.Helper()
		meta, ok := v["meta"].(map[string]any)
		if !ok {
			t.Fatalf("missing meta: %v", v)
		}
		return meta["pagination"].(map[string]any)
	}
	auditPage := request("GET", "/audit-logs?page=2&per_page=5", "admin", nil, 200)
	if p := pagination(auditPage); len(auditPage["data"].([]any)) != 5 || p["current_page"] != float64(2) || p["total_records"].(float64) < 15 {
		t.Fatalf("audit pagination wrong: %v", p)
	}
	request("GET", "/audit-logs?page=-1", "admin", nil, 422)

	// Departments and dashboard are tenant-scoped and admin-only.
	request("GET", "/departments", "staff", nil, 403)
	request("POST", "/departments", "admin", map[string]string{"name": "F"}, 422)
	request("POST", "/departments", "admin", map[string]string{"name": "Finance"}, 200)
	if len(request("GET", "/departments", "admin", nil, 200)["data"].([]any)) != 2 {
		t.Fatal("department list wrong")
	}
	if len(request("GET", "/departments", "other", nil, 200)["data"].([]any)) != 1 {
		t.Fatal("department tenant leak")
	}
	request("GET", "/dashboard", "staff", nil, 403)
	dash := request("GET", "/dashboard", "admin", nil, 200)["data"].(map[string]any)
	if dash["employees"] != float64(2) || dash["departments"] != float64(2) || dash["pending"] != float64(0) {
		t.Fatalf("dashboard counts wrong: %v", dash)
	}

	// Server-side employee pagination, search, and filters.
	empPage := request("GET", "/employees?page=1&per_page=1", "admin", nil, 200)
	if p := pagination(empPage); len(empPage["data"].([]any)) != 1 || p["total_records"] != float64(2) || p["total_pages"] != float64(2) {
		t.Fatalf("employee pagination wrong: %v", p)
	}
	found := request("GET", "/employees?page=1&search=PERSON", "admin", nil, 200)["data"].([]any)
	if len(found) != 1 || found[0].(map[string]any)["name"] != "New Person" {
		t.Fatalf("employee search wrong: %v", found)
	}
	if len(request("GET", "/employees?page=1&search=engineering", "admin", nil, 200)["data"].([]any)) != 2 {
		t.Fatal("employee search by department failed")
	}
	if len(request("GET", "/employees?page=1&search=%25", "admin", nil, 200)["data"].([]any)) != 0 {
		t.Fatal("LIKE wildcard not escaped")
	}
	if len(request("GET", "/employees?page=1&status=inactive", "admin", nil, 200)["data"].([]any)) != 0 {
		t.Fatal("status filter failed")
	}
	if len(request("GET", "/employees?page=1&department_id=2", "admin", nil, 200)["data"].([]any)) != 0 {
		t.Fatal("department filter crossed tenants")
	}
	request("GET", "/employees?page=1&status=bogus", "admin", nil, 422)
	request("GET", "/employees?page=abc", "admin", nil, 422)

	// Every error carries the envelope with a machine-readable code.
	for _, c := range []struct {
		method, path, token string
		status              int
		code                string
	}{
		{"GET", "/employees", "", 401, "UNAUTHORIZED"},
		{"GET", "/employees", "staff", 403, "FORBIDDEN"},
		{"GET", "/no-such-route", "admin", 404, "NOT_FOUND"},
		{"GET", "/employees?page=1&status=bogus", "admin", 422, "VALIDATION_ERROR"},
		{"POST", "/payroll", "admin", 409, "CONFLICT"},
	} {
		v := request(c.method, c.path, c.token, map[string]string{"period": "2026-11"}, c.status)
		if v["success"] != false || v["error_code"] != c.code || v["message"] == "" {
			t.Fatalf("%s %s: bad error envelope %v", c.method, c.path, v)
		}
	}
	hw := httptest.NewRecorder()
	e.ServeHTTP(hw, httptest.NewRequest("GET", "/health", nil))
	if hw.Code != 200 || !strings.Contains(hw.Body.String(), `"database":"connected"`) || hw.Header().Get("X-Request-Id") == "" {
		t.Fatalf("health: %d %s", hw.Code, hw.Body.String())
	}

	// Platform: list pagination and filters.
	paged := request("GET", "/leaves?page=1&per_page=1", "admin", nil, 200)
	if len(paged["data"].([]any)) != 1 || paged["meta"].(map[string]any)["pagination"].(map[string]any)["total_records"].(float64) < 2 {
		t.Fatalf("leave pagination: %v", paged["meta"])
	}
	for _, l := range request("GET", "/leaves?status=approved", "admin", nil, 200)["data"].([]any) {
		if l.(map[string]any)["status"] != "approved" {
			t.Fatal("leave status filter")
		}
	}
	request("GET", "/leaves?status=bogus", "admin", nil, 422)
	request("GET", "/hr/overtime?status=bogus", "admin", nil, 422)
	if request("GET", "/attendances?page=1&date=2020-01-06", "admin", nil, 200)["meta"].(map[string]any)["pagination"].(map[string]any)["total_records"] != float64(1) {
		t.Fatal("attendance date filter")
	}
	request("GET", "/attendances?page=1&date=yesterday", "admin", nil, 422)

	// Candidate → employee: only offer/hired candidates, once, atomically with the account.
	hire := map[string]any{"name": "Candidate Test", "email": "candidate@example.test", "code": "A9", "department_id": 1, "position": "Engineer", "joined_on": "2026-12-01", "password": "CandidatePass123!"}
	request("POST", fmt.Sprintf("/hr/recruitment/%d/convert", candidate), "admin", hire, 409)
	request("PUT", fmt.Sprintf("/hr/recruitment/%d", candidate), "admin", map[string]any{"title": "Candidate Test", "status": "offer", "version": 1, "data": map[string]any{"email": "candidate@example.test", "position": "Engineer"}}, 200)
	request("POST", fmt.Sprintf("/hr/recruitment/%d/convert", candidate), "other", hire, 404)
	request("POST", fmt.Sprintf("/hr/recruitment/%d/convert", candidate), "admin", hire, 200)
	request("POST", fmt.Sprintf("/hr/recruitment/%d/convert", candidate), "admin", hire, 409)
	for _, c := range request("GET", "/hr/recruitment", "admin", nil, 200)["data"].([]any) {
		if row := c.(map[string]any); row["id"] == float64(candidate) && (row["status"] != "hired" || row["employee_name"] != "Candidate Test") {
			t.Fatalf("candidate not linked: %v", row)
		}
	}

	// Uploads: type checked against content, employees read only published attachments.
	upload := func(token, name string, body []byte, want int) map[string]any {
		t.Helper()
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		fw, _ := mw.CreateFormFile("file", name)
		_, _ = fw.Write(body)
		_ = mw.Close()
		r := httptest.NewRequest("POST", "/api/v1/files", &buf)
		r.Header.Set("Content-Type", mw.FormDataContentType())
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		e.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("upload %s: want %d, got %d: %s", name, want, w.Code, w.Body.String())
		}
		var v map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &v)
		return v
	}
	pdf := []byte("%PDF-1.4\n1 0 obj << >> endobj\n%%EOF\n")
	upload("staff", "policy.pdf", pdf, 403)
	upload("admin", "policy.pdf", []byte("\x89PNG\r\n\x1a\n not a pdf"), 422)
	upload("admin", "policy.exe", pdf, 422)
	fileID := itemID(upload("admin", "../Kebijakan Cuti.pdf", pdf, 200))
	download := func(token string, want int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/files/%d", fileID), nil)
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		e.ServeHTTP(w, r)
		if w.Code != want {
			t.Fatalf("download as %s: want %d, got %d", token, want, w.Code)
		}
		return w
	}
	download("admin", 200)
	download("staff", 404) // not attached to a published document yet
	download("other", 404)
	request("POST", "/hr/documents", "admin", map[string]any{"title": "Both", "status": "published", "data": map[string]any{"file_id": fileID, "url": "https://example.com"}}, 422)
	request("POST", "/hr/documents", "admin", map[string]any{"title": "Missing", "status": "published", "data": map[string]any{"file_id": 999999}}, 422)
	request("POST", "/hr/documents", "admin", map[string]any{"title": "Leave policy", "status": "published", "data": map[string]any{"file_id": fileID}}, 200)
	got := download("staff", 200)
	if !strings.Contains(got.Header().Get("Content-Disposition"), `filename="Kebijakan Cuti.pdf"`) || got.Body.String() != string(pdf) {
		t.Fatalf("download headers/body wrong: %v", got.Header())
	}

	// Company sign-up creates a tenant with defaults and signs its admin in.
	if request("GET", "/auth/config", "", nil, 200)["data"].(map[string]any)["signup_enabled"] != true {
		t.Fatal("signup flag missing")
	}
	signup := map[string]string{"company_name": "Gamma Corp", "company_slug": "Gamma Co", "name": "Gamma Admin", "email": "admin@gamma.test", "password": "GammaPassword123!"}
	request("POST", "/auth/register", "", signup, 422)
	signup["company_slug"] = "alpha"
	request("POST", "/auth/register", "", signup, 409)
	signup["company_slug"] = "gamma-co"
	gamma := request("POST", "/auth/register", "", signup, 200)["data"].(map[string]any)["token"].(string)
	if me := request("GET", "/auth/me", gamma, nil, 200)["data"].(map[string]any); me["company_name"] != "Gamma Corp" || me["role"] != "admin" {
		t.Fatalf("new tenant admin wrong: %v", me)
	}
	if depts := request("GET", "/departments", gamma, nil, 200)["data"].([]any); len(depts) != 1 {
		t.Fatalf("new tenant must start with one department: %v", depts)
	}
	if len(request("GET", "/employees", gamma, nil, 200)["data"].([]any)) != 0 {
		t.Fatal("new tenant sees other tenants' employees")
	}

	// Password reset: same answer for unknown accounts, single-use token, sessions ended.
	request("POST", "/auth/forgot-password", "", map[string]string{"company": "alpha", "email": "nobody@alpha.test"}, 200)
	request("POST", "/auth/forgot-password", "", map[string]string{"company": "alpha", "email": "staff@alpha.test"}, 200)
	var resets int
	if err = isolated.Get(&resets, `SELECT count(*) FROM password_resets WHERE user_id=2 AND used_at IS NULL`); err != nil || resets != 1 {
		t.Fatalf("reset token not stored: %d %v", resets, err)
	}
	resetToken := strings.Repeat("ab", 32)
	if _, err = isolated.Exec(`UPDATE password_resets SET token_hash=$1 WHERE user_id=2`, service.HashToken(resetToken)); err != nil {
		t.Fatal(err)
	}
	request("POST", "/auth/reset-password", "", map[string]string{"token": resetToken, "password": "short"}, 422)
	request("POST", "/auth/reset-password", "", map[string]string{"token": resetToken, "password": "BrandNewPass123!"}, 200)
	request("POST", "/auth/reset-password", "", map[string]string{"token": resetToken, "password": "BrandNewPass123!"}, 422)
	request("GET", "/auth/me", "staff", nil, 401)
	session(2, "staff")

	request("POST", "/auth/logout", token, nil, 200)
	request("GET", "/auth/me", token, nil, 401)
	if _, err = isolated.Exec(`UPDATE employees SET status='inactive' WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	request("GET", "/auth/me", "staff", nil, 401)
}
