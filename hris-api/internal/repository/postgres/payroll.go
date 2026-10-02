package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type payrollRepository struct{ db *sqlx.DB }

func NewPayrollRepository(db *sqlx.DB) repository.PayrollRepository {
	return &payrollRepository{db: db}
}

// settings returns the company's payroll settings, or the defaults when the
// company never saved any.
func settings(ctx context.Context, q sqlx.QueryerContext, companyID int64) (model.PayrollSettings, error) {
	v := model.PayrollSettings{JKKRate: 24, JPWageCap: 10_547_400, KesWageCap: 12_000_000}
	err := sqlx.GetContext(ctx, q, &v, `
		SELECT jkk_rate, jp_wage_cap, kes_wage_cap FROM payroll_settings WHERE company_id = $1`,
		companyID)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return v, err
}

func (r *payrollRepository) Settings(ctx context.Context, companyID int64) (model.PayrollSettings, error) {
	return settings(ctx, r.db, companyID)
}

func (r *payrollRepository) SaveSettings(ctx context.Context, companyID, actorID int64, v model.PayrollSettings) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO payroll_settings (company_id, jkk_rate, jp_wage_cap, kes_wage_cap)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (company_id) DO UPDATE SET jkk_rate = $2, jp_wage_cap = $3, kes_wage_cap = $4`,
		companyID, v.JKKRate, v.JPWageCap, v.KesWageCap)
	if err != nil {
		return err
	}
	if err = audit(ctx, tx, companyID, actorID, "update", "payroll_settings", companyID, "Memperbarui pengaturan BPJS payroll"); err != nil {
		return err
	}
	return tx.Commit()
}

const salaryColumns = `
	e.id employee_id, e.name, e.code, (s.employee_id IS NOT NULL) configured,
	COALESCE(s.basic_salary, 0) basic_salary,
	COALESCE(s.ptkp_status, 'TK/0') ptkp_status,
	COALESCE(s.tax_method, 'gross') tax_method,
	COALESCE(s.bpjs_kesehatan, true) bpjs_kesehatan,
	COALESCE(s.bpjs_ketenagakerjaan, true) bpjs_ketenagakerjaan,
	COALESCE(s.bpjs_pensiun, true) bpjs_pensiun,
	COALESCE(s.overtime_eligible, true) overtime_eligible,
	COALESCE(s.note, '') note`

// attachComponents loads the salary components of the listed employees.
func attachComponents(ctx context.Context, q sqlx.QueryerContext, companyID int64, salaries []model.Salary) error {
	ids := make([]int64, len(salaries))
	byID := make(map[int64]*model.Salary, len(salaries))
	for i := range salaries {
		ids[i] = salaries[i].EmployeeID
		salaries[i].Components = []model.SalaryComponent{}
		byID[ids[i]] = &salaries[i]
	}
	var comps []model.SalaryComponent
	err := sqlx.SelectContext(ctx, q, &comps, `
		SELECT employee_id, kind, name, amount, fixed, taxable
		FROM salary_components
		WHERE company_id = $1 AND employee_id = ANY($2)
		ORDER BY employee_id, position, id`,
		companyID, pq.Array(ids))
	for _, c := range comps {
		byID[c.EmployeeID].Components = append(byID[c.EmployeeID].Components, c)
	}
	return err
}

func (r *payrollRepository) Salaries(ctx context.Context, companyID int64) ([]model.Salary, error) {
	v := []model.Salary{}
	err := r.db.SelectContext(ctx, &v, `
		SELECT `+salaryColumns+`
		FROM employees e
		LEFT JOIN salary_profiles s ON s.company_id = e.company_id AND s.employee_id = e.id
		WHERE e.company_id = $1 AND e.status = 'active'
		ORDER BY e.name`,
		companyID)
	if err != nil {
		return nil, err
	}
	return v, attachComponents(ctx, r.db, companyID, v)
}

func (r *payrollRepository) SaveSalary(ctx context.Context, companyID, actorID, employeeID int64, v model.SalaryInput) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if err = lockEmployee(ctx, tx, companyID, employeeID); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO salary_profiles (company_id, employee_id, basic_salary, note, ptkp_status, tax_method,
		                             bpjs_kesehatan, bpjs_ketenagakerjaan, bpjs_pensiun, overtime_eligible)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (company_id, employee_id) DO UPDATE
		SET basic_salary = $3, note = $4, ptkp_status = $5, tax_method = $6,
		    bpjs_kesehatan = $7, bpjs_ketenagakerjaan = $8, bpjs_pensiun = $9, overtime_eligible = $10`,
		companyID, employeeID, v.BasicSalary, v.Note, v.PTKPStatus, v.TaxMethod,
		v.BPJSKesehatan, v.BPJSKetenagakerjaan, v.BPJSPensiun, v.OvertimeEligible)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `
		DELETE FROM salary_components WHERE company_id = $1 AND employee_id = $2`,
		companyID, employeeID); err != nil {
		return err
	}
	for i, c := range v.Components {
		if _, err = tx.ExecContext(ctx, `
			INSERT INTO salary_components (company_id, employee_id, kind, name, amount, fixed, taxable, position)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			companyID, employeeID, c.Kind, c.Name, c.Amount, c.Fixed, c.Taxable, i); err != nil {
			return err
		}
	}
	if err = audit(ctx, tx, companyID, actorID, "update", "salary_profile", employeeID, "Memperbarui komponen gaji"); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *payrollRepository) PayrollRuns(ctx context.Context, companyID int64) ([]model.PayrollRun, error) {
	v := []model.PayrollRun{}
	err := r.db.SelectContext(ctx, &v, `
		SELECT r.id, to_char(r.period, 'YYYY-MM') period, r.status, count(e.id) employees,
		       COALESCE(sum(e.basic_salary + e.allowance - e.deduction), 0) total,
		       COALESCE(sum(e.pph21), 0) tax, COALESCE(sum(e.employer_cost), 0) employer_cost,
		       r.thr_date::text thr_date, r.payment_reference, r.created_at
		FROM payroll_runs r
		LEFT JOIN payroll_entries e ON e.run_id = r.id AND e.company_id = r.company_id
		WHERE r.company_id = $1
		GROUP BY r.id
		ORDER BY r.period DESC, r.id DESC`,
		companyID)
	return v, err
}

// ytdSQL sums locked, automatically taxed slips of the same year before the
// given period. $1 company, $2 employee ids, $3 period start.
const ytdSQL = `
	SELECT e.employee_id, sum(e.taxable_gross) gross, sum(e.pension_deduction) pension,
	       sum(e.pph21) tax, count(*) months
	FROM payroll_entries e
	JOIN payroll_runs r ON r.company_id = e.company_id AND r.id = e.run_id
	WHERE e.company_id = $1 AND e.employee_id = ANY($2)
	  AND r.status IN ('finalized', 'paid') AND e.tax_method <> 'none'
	  AND r.period >= date_trunc('year', $3::date) AND r.period < $3::date
	GROUP BY e.employee_id`

func yearToDate(ctx context.Context, q sqlx.QueryerContext, companyID int64, employeeIDs []int64, periodStart string) (map[int64]model.YearToDate, error) {
	var rows []struct {
		EmployeeID int64 `db:"employee_id"`
		model.YearToDate
	}
	if err := sqlx.SelectContext(ctx, q, &rows, ytdSQL, companyID, pq.Array(employeeIDs), periodStart); err != nil {
		return nil, err
	}
	v := make(map[int64]model.YearToDate, len(rows))
	for _, r := range rows {
		v[r.EmployeeID] = r.YearToDate
	}
	return v, nil
}

// CreatePayroll reads everything a run is calculated from in one repeatable
// read snapshot, lets build calculate the slips, and stores them. Approved
// overtime is linked to the slip that pays it, so it is paid only once.
func (r *payrollRepository) CreatePayroll(ctx context.Context, companyID, actorID int64, periodStart string, thrDate *string,
	build func(model.PayrollSource) ([]model.PayrollEntryDraft, error)) (int64, error) {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return 0, err
	}
	defer rollback(tx)

	// Year-to-date tax needs every earlier period locked, and a new period
	// must not slip in before one whose tax was already settled.
	var earlierDraft, laterRun bool
	if err = tx.QueryRowxContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM payroll_runs WHERE company_id = $1 AND status = 'draft' AND period < $2::date),
		       EXISTS (SELECT 1 FROM payroll_runs WHERE company_id = $1 AND status <> 'void' AND period > $2::date
		               AND date_trunc('year', period) = date_trunc('year', $2::date))`,
		companyID, periodStart).Scan(&earlierDraft, &laterRun); err != nil {
		return 0, err
	}
	if earlierDraft {
		return 0, apperror.Conflict("Finalisasi atau batalkan draft payroll periode sebelumnya terlebih dahulu")
	}
	if laterRun {
		return 0, apperror.Conflict("Sudah ada payroll untuk periode setelahnya pada tahun yang sama")
	}

	var src model.PayrollSource
	if src.Settings, err = settings(ctx, tx, companyID); err != nil {
		return 0, err
	}
	src.Workdays = []int64{1, 2, 3, 4, 5}
	var workdays pq.Int64Array
	err = tx.GetContext(ctx, &workdays, `SELECT workdays FROM work_calendars WHERE company_id = $1`, companyID)
	if err == nil {
		src.Workdays = workdays
	} else if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	if err = tx.SelectContext(ctx, &src.Holidays, `
		SELECT date::text FROM holidays WHERE company_id = $1`, companyID); err != nil {
		return 0, err
	}

	// Employees who worked at least part of the period: active ones that
	// joined before it ended, plus leavers whose last day falls in or after it.
	var people []struct {
		model.Salary
		Position string  `db:"position"`
		JoinedOn string  `db:"joined_on"`
		LeftOn   *string `db:"left_on"`
	}
	if err = tx.SelectContext(ctx, &people, `
		SELECT `+salaryColumns+`, e.position, e.joined_on::text joined_on, e.left_on::text left_on
		FROM employees e
		LEFT JOIN salary_profiles s ON s.company_id = e.company_id AND s.employee_id = e.id
		WHERE e.company_id = $1 AND e.joined_on < ($2::date + interval '1 month')
		  AND (e.status = 'active' OR e.left_on >= $2::date)
		ORDER BY e.name, e.id`,
		companyID, periodStart); err != nil {
		return 0, err
	}
	if len(people) == 0 {
		return 0, apperror.Conflict("Tidak ada karyawan aktif pada periode tersebut")
	}
	salaries := make([]model.Salary, len(people))
	ids := make([]int64, len(people))
	for i, p := range people {
		if !p.Configured {
			return 0, apperror.Conflict("Lengkapi komponen gaji seluruh karyawan aktif pada periode tersebut sebelum membuat payroll")
		}
		salaries[i], ids[i] = p.Salary, p.EmployeeID
	}
	if err = attachComponents(ctx, tx, companyID, salaries); err != nil {
		return 0, err
	}
	ytd, err := yearToDate(ctx, tx, companyID, ids, periodStart)
	if err != nil {
		return 0, err
	}
	var claims []struct {
		EmployeeID int64 `db:"employee_id"`
		model.OvertimeClaim
	}
	if err = tx.SelectContext(ctx, &claims, `
		SELECT h.employee_id, h.id, h.data->>'start_at' start_at, h.data->>'end_at' end_at
		FROM hr_items h
		WHERE h.company_id = $1 AND h.module = 'overtime' AND h.status = 'approved'
		  AND h.employee_id = ANY($2)
		  AND ((h.data->>'start_at')::timestamptz AT TIME ZONE 'Asia/Jakarta')::date < ($3::date + interval '1 month')
		  AND NOT EXISTS (SELECT 1 FROM payroll_overtime p WHERE p.company_id = h.company_id AND p.hr_item_id = h.id)
		ORDER BY h.id`,
		companyID, pq.Array(ids), periodStart); err != nil {
		return 0, err
	}
	overtime := map[int64][]model.OvertimeClaim{}
	for _, c := range claims {
		overtime[c.EmployeeID] = append(overtime[c.EmployeeID], c.OvertimeClaim)
	}
	for i, p := range people {
		src.Employees = append(src.Employees, model.PayrollEmployee{
			ID: p.EmployeeID, Name: p.Name, Code: p.Code, Position: p.Position,
			JoinedOn: p.JoinedOn, LeftOn: p.LeftOn, Salary: salaries[i],
			Overtime: overtime[p.EmployeeID], YTD: ytd[p.EmployeeID],
		})
	}

	drafts, err := build(src)
	if err != nil {
		return 0, err
	}
	var id int64
	if err = tx.QueryRowxContext(ctx, `
		INSERT INTO payroll_runs (company_id, period, created_by, jkk_rate, jp_wage_cap, kes_wage_cap, thr_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		companyID, periodStart, actorID, src.Settings.JKKRate, src.Settings.JPWageCap, src.Settings.KesWageCap, thrDate,
	).Scan(&id); err != nil {
		return 0, err
	}
	for _, d := range drafts {
		var entryID int64
		if err = tx.QueryRowxContext(ctx, `
			INSERT INTO payroll_entries
			       (company_id, run_id, employee_id, employee_name, employee_code, position, note,
			        ptkp_status, tax_method, bpjs_kesehatan, bpjs_ketenagakerjaan, bpjs_pensiun,
			        final_period, worked_days, period_days, basic_salary, allowance, deduction,
			        taxable_gross, pension_deduction, pph21, employer_cost)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
			RETURNING id`,
			companyID, id, d.EmployeeID, d.Name, d.Code, d.Position, d.Note,
			d.PTKPStatus, d.TaxMethod, d.BPJSKesehatan, d.BPJSKetenagakerjaan, d.BPJSPensiun,
			d.FinalPeriod, d.WorkedDays, d.PeriodDays, d.BasicSalary, d.Allowance, d.Deduction,
			d.TaxableGross, d.Pension, d.PPh21, d.EmployerCost,
		).Scan(&entryID); err != nil {
			return 0, err
		}
		if err = storeLines(ctx, tx, companyID, entryID, d.Lines); err != nil {
			return 0, err
		}
		for _, itemID := range d.OvertimeIDs {
			if _, err = tx.ExecContext(ctx, `
				INSERT INTO payroll_overtime (company_id, hr_item_id, entry_id) VALUES ($1, $2, $3)`,
				companyID, itemID, entryID); err != nil {
				return 0, err
			}
		}
	}
	if err = audit(ctx, tx, companyID, actorID, "create", "payroll", id, "Membuat draft payroll "+periodStart); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

// storeLines replaces a slip's lines.
func storeLines(ctx context.Context, tx *sqlx.Tx, companyID, entryID int64, lines []model.PayrollLine) error {
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM payroll_lines WHERE company_id = $1 AND entry_id = $2`, companyID, entryID); err != nil {
		return err
	}
	for i, l := range lines {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO payroll_lines (company_id, entry_id, kind, code, name, amount, taxable, fixed, position)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			companyID, entryID, l.Kind, l.Code, l.Name, l.Amount, l.Taxable, l.Fixed, i); err != nil {
			return err
		}
	}
	return nil
}

func (r *payrollRepository) Payslips(ctx context.Context, companyID int64, employeeID *int64, runID int64) ([]model.Payslip, error) {
	v := []model.Payslip{}
	err := r.db.SelectContext(ctx, &v, `
		SELECT e.id, e.run_id, e.employee_id, e.employee_name, e.employee_code, e.position,
		       to_char(r.period, 'YYYY-MM') period, r.status, e.basic_salary, e.allowance, e.deduction,
		       (e.basic_salary + e.allowance - e.deduction) net, e.ptkp_status, e.tax_method,
		       e.final_period, e.worked_days, e.period_days, e.taxable_gross, e.pph21, e.employer_cost,
		       e.note, e.version
		FROM payroll_entries e
		JOIN payroll_runs r ON r.id = e.run_id AND r.company_id = e.company_id
		WHERE e.company_id = $1
		  AND ($2::bigint IS NULL OR (e.employee_id = $2 AND r.status IN ('finalized', 'paid')))
		  AND ($3::bigint = 0 OR e.run_id = $3)
		ORDER BY r.period DESC, e.employee_name`,
		companyID, employeeID, runID)
	if err != nil || len(v) == 0 {
		return v, err
	}
	ids := make([]int64, len(v))
	byID := make(map[int64]*model.Payslip, len(v))
	for i := range v {
		ids[i] = v[i].ID
		v[i].Lines = []model.PayrollLine{}
		byID[ids[i]] = &v[i]
	}
	var lines []model.PayrollLine
	if err = r.db.SelectContext(ctx, &lines, `
		SELECT entry_id, kind, code, name, amount, taxable, fixed
		FROM payroll_lines
		WHERE company_id = $1 AND entry_id = ANY($2)
		ORDER BY entry_id, position`,
		companyID, pq.Array(ids)); err != nil {
		return nil, err
	}
	for _, l := range lines {
		byID[l.EntryID].Lines = append(byID[l.EntryID].Lines, l)
	}
	return v, nil
}

// SavePayslip recalculates one slip of a draft run. The run row is locked so a
// concurrent finalisation cannot interleave, and version guards stale edits.
func (r *payrollRepository) SavePayslip(ctx context.Context, companyID, actorID, entryID int64, version int, note string,
	build func(model.PayslipContext) (model.PayrollEntryDraft, error)) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	var entry struct {
		model.PayslipContext
		RunID      int64  `db:"run_id"`
		EmployeeID int64  `db:"employee_id"`
		Version    int    `db:"version"`
		Status     string `db:"status"`
		Period     string `db:"period"`
	}
	if err = tx.GetContext(ctx, &entry, `
		SELECT run_id, employee_id, version, ptkp_status, tax_method, bpjs_kesehatan,
		       bpjs_ketenagakerjaan, bpjs_pensiun, final_period
		FROM payroll_entries WHERE company_id = $1 AND id = $2`,
		companyID, entryID); err != nil {
		return err
	}
	if err = tx.QueryRowxContext(ctx, `
		SELECT status, period::text, jkk_rate, jp_wage_cap, kes_wage_cap
		FROM payroll_runs WHERE company_id = $1 AND id = $2 FOR UPDATE`,
		companyID, entry.RunID).Scan(&entry.Status, &entry.Period,
		&entry.Settings.JKKRate, &entry.Settings.JPWageCap, &entry.Settings.KesWageCap); err != nil {
		return err
	}
	if entry.Status != "draft" {
		return apperror.Conflict("Payroll yang sudah difinalisasi tidak dapat diubah")
	}
	if entry.Version != version {
		return apperror.Conflict("Slip sudah berubah; muat ulang terlebih dahulu")
	}
	ytd, err := yearToDate(ctx, tx, companyID, []int64{entry.EmployeeID}, entry.Period)
	if err != nil {
		return err
	}
	entry.YTD = ytd[entry.EmployeeID]
	d, err := build(entry.PayslipContext)
	if err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE payroll_entries
		SET note = $3, basic_salary = $5, allowance = $6, deduction = $7, taxable_gross = $8,
		    pension_deduction = $9, pph21 = $10, employer_cost = $11, version = version + 1
		WHERE company_id = $1 AND id = $2 AND version = $4`,
		companyID, entryID, note, version, d.BasicSalary, d.Allowance, d.Deduction, d.TaxableGross,
		d.Pension, d.PPh21, d.EmployerCost)
	if err != nil {
		return err
	}
	if rowsAffected(res) == 0 {
		return apperror.Conflict("Slip sudah berubah; muat ulang terlebih dahulu")
	}
	if err = storeLines(ctx, tx, companyID, entryID, d.Lines); err != nil {
		return err
	}
	if err = audit(ctx, tx, companyID, actorID, "update", "payslip", entryID, "Penyesuaian komponen slip draft"); err != nil {
		return err
	}
	return tx.Commit()
}

// PayrollAction moves a run through draft → finalized → paid (or draft → void).
// Voiding releases the run's overtime so the next run pays it.
func (r *payrollRepository) PayrollAction(ctx context.Context, companyID, actorID, runID int64, action, reference string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	var status string
	if err = tx.GetContext(ctx, &status, `
		SELECT status FROM payroll_runs WHERE company_id = $1 AND id = $2 FOR UPDATE`,
		companyID, runID); err != nil {
		return err
	}
	var next string
	switch {
	case action == "finalize" && status == "draft":
		next = "finalized"
	case action == "paid" && status == "finalized":
		next = "paid"
	case action == "void" && status == "draft":
		next = "void"
	default:
		return apperror.Conflict("Perubahan status payroll tidak diizinkan")
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE payroll_runs
		SET status = $3,
		    finalized_at = CASE WHEN $3 = 'finalized' THEN now() ELSE finalized_at END,
		    paid_at = CASE WHEN $3 = 'paid' THEN now() ELSE paid_at END,
		    payment_reference = CASE WHEN $3 = 'paid' THEN $4 ELSE payment_reference END
		WHERE company_id = $1 AND id = $2`,
		companyID, runID, next, reference)
	if err != nil {
		return err
	}
	if next == "void" {
		if _, err = tx.ExecContext(ctx, `
			DELETE FROM payroll_overtime p
			USING payroll_entries e
			WHERE p.company_id = $1 AND e.company_id = $1 AND e.run_id = $2 AND p.entry_id = e.id`,
			companyID, runID); err != nil {
			return err
		}
	}
	if err = audit(ctx, tx, companyID, actorID, action, "payroll", runID, "Status payroll: "+next); err != nil {
		return err
	}
	return tx.Commit()
}
