package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/jmoiron/sqlx"
)

const itemFields = `h.id,h.module,h.employee_id,COALESCE(e.name,'') employee_name,h.title,h.description,h.status,COALESCE(h.due_date::text,'') due_date,h.data,h.version,h.review_note,h.created_at`

func scope(u *model.User, module string) *int64 {
	if u.Role == "employee" && module != "announcements" && module != "documents" {
		return u.EmployeeID
	}
	return nil
}
func (r *Repository) HRItems(ctx context.Context, u *model.User, module string) ([]model.HRItem, error) {
	v := []model.HRItem{}
	err := r.DB.SelectContext(ctx, &v, `SELECT `+itemFields+` FROM hr_items h LEFT JOIN employees e ON e.id=h.employee_id AND e.company_id=h.company_id WHERE h.company_id=$1 AND h.module=$2 AND ($3::bigint IS NULL OR h.employee_id=$3) AND ($4 OR h.module NOT IN ('announcements','documents') OR h.status='published') ORDER BY h.created_at DESC,h.id DESC`, u.CompanyID, module, scope(u, module), u.Role == "admin")
	return v, err
}
func attendanceFingerprint(ctx context.Context, tx *sqlx.Tx, c, e int64, date string) (string, error) {
	var v string
	err := tx.GetContext(ctx, &v, `SELECT check_in::text||'|'||COALESCE(check_out::text,'') FROM attendances WHERE company_id=$1 AND employee_id=$2 AND date=$3`, c, e, date)
	if err == sql.ErrNoRows {
		return "missing", nil
	}
	return v, err
}
func (r *Repository) SaveHRItem(ctx context.Context, u *model.User, module string, id int64, v dto.HRItem) (int64, error) {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if v.EmployeeID != nil {
		if err = lockEmployee(ctx, tx, u.CompanyID, *v.EmployeeID); err != nil {
			return 0, err
		}
	}
	if module == "overtime" || module == "corrections" {
		var data dto.HRData
		json.Unmarshal(v.Data, &data)
		if module == "overtime" {
			var overlap bool
			err = tx.GetContext(ctx, &overlap, `SELECT EXISTS(SELECT 1 FROM hr_items WHERE company_id=$1 AND employee_id=$2 AND module='overtime' AND status IN ('pending','approved') AND (data->>'start_at')::timestamptz<$4::timestamptz AND (data->>'end_at')::timestamptz>$3::timestamptz)`, u.CompanyID, v.EmployeeID, data.StartAt, data.EndAt)
			if err != nil {
				return 0, err
			}
			if overlap {
				return 0, &apperror.Error{409, "Waktu lembur bertumpang tindih"}
			}
		}
		if module == "corrections" {
			var exists bool
			err = tx.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM hr_items WHERE company_id=$1 AND employee_id=$2 AND module='corrections' AND status='pending' AND data->>'date'=$3)`, u.CompanyID, v.EmployeeID, data.Date)
			if err != nil {
				return 0, err
			}
			if exists {
				return 0, &apperror.Error{409, "Koreksi tanggal ini masih menunggu persetujuan"}
			}
			data.Original, err = attendanceFingerprint(ctx, tx, u.CompanyID, *v.EmployeeID, data.Date)
			if err != nil {
				return 0, err
			}
			v.Data, _ = json.Marshal(data)
		}
	}
	action := "create"
	if id == 0 {
		err = tx.QueryRowxContext(ctx, `INSERT INTO hr_items(company_id,module,employee_id,title,description,status,due_date,data,created_by) VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,'')::date,$8,$9) RETURNING id`, u.CompanyID, module, v.EmployeeID, v.Title, v.Description, v.Status, v.DueDate, string(v.Data), u.ID).Scan(&id)
	} else {
		action = "update"
		res, e := tx.ExecContext(ctx, `UPDATE hr_items SET employee_id=$4,title=$5,description=$6,status=$7,due_date=NULLIF($8,'')::date,data=$9,version=version+1,updated_at=now() WHERE company_id=$1 AND module=$2 AND id=$3 AND version=$10`, u.CompanyID, module, id, v.EmployeeID, v.Title, v.Description, v.Status, v.DueDate, string(v.Data), v.Version)
		err = e
		if err == nil {
			n, _ := res.RowsAffected()
			if n == 0 {
				return 0, &apperror.Error{409, "Data sudah berubah atau tidak ditemukan. Muat ulang terlebih dahulu."}
			}
		}
	}
	if err != nil {
		return 0, err
	}
	if err = audit(ctx, tx, u.CompanyID, u.ID, action, module, id, v.Title); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}
func (r *Repository) ActHRItem(ctx context.Context, u *model.User, module string, id int64, v dto.HRAction) error {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var item model.HRItem
	// Employee lock serializes corrections with check-in/out and other requests.
	var employee int64
	if err = tx.GetContext(ctx, &employee, `SELECT employee_id FROM hr_items WHERE company_id=$1 AND module=$2 AND id=$3 AND ($4::bigint IS NULL OR employee_id=$4)`, u.CompanyID, module, id, scope(u, module)); err != nil {
		return err
	}
	if err = lockEmployee(ctx, tx, u.CompanyID, employee); err != nil {
		return err
	}
	if err = tx.GetContext(ctx, &item, `SELECT `+itemFields+` FROM hr_items h LEFT JOIN employees e ON e.id=h.employee_id WHERE h.company_id=$1 AND h.module=$2 AND h.id=$3 FOR UPDATE OF h`, u.CompanyID, module, id); err != nil {
		return err
	}
	if item.Version != v.Version {
		return &apperror.Error{409, "Data sudah berubah. Muat ulang terlebih dahulu."}
	}
	var data dto.HRData
	json.Unmarshal(item.Data, &data)
	status := item.Status
	switch module {
	case "overtime", "corrections":
		if item.Status != "pending" {
			return &apperror.Error{409, "Pengajuan sudah diproses"}
		}
		status = map[string]string{"approve": "approved", "reject": "rejected", "cancel": "cancelled"}[v.Action]
		if module == "corrections" && v.Action == "approve" {
			current, e := attendanceFingerprint(ctx, tx, u.CompanyID, employee, data.Date)
			if e != nil {
				return e
			}
			if current != data.Original {
				return &apperror.Error{409, "Absensi berubah sejak pengajuan; tolak dan ajukan koreksi baru"}
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO attendances(company_id,employee_id,date,check_in,check_out) VALUES($1,$2,$3,$4,$5) ON CONFLICT(company_id,employee_id,date) DO UPDATE SET check_in=$4,check_out=$5`, u.CompanyID, employee, data.Date, data.Date+"T"+data.CheckIn+":00+07:00", data.Date+"T"+data.CheckOut+":00+07:00")
			if err != nil {
				return err
			}
		}
	case "onboarding":
		if item.Status == "done" {
			return &apperror.Error{409, "Tugas sudah selesai"}
		}
		status = "in_progress"
		if v.Action == "complete" {
			status = "done"
		}
	case "goals":
		data.Progress = v.Progress
		status = "active"
		if v.Progress == 100 {
			status = "done"
		}
	}
	body, _ := json.Marshal(data)
	_, err = tx.ExecContext(ctx, `UPDATE hr_items SET status=$4,data=$5,version=version+1,updated_at=now(),reviewed_by=$6,review_note=$7 WHERE company_id=$1 AND module=$2 AND id=$3`, u.CompanyID, module, id, status, string(body), u.ID, v.Note)
	if err != nil {
		return err
	}
	if err = audit(ctx, tx, u.CompanyID, u.ID, v.Action, module, id, item.Title); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) Profile(ctx context.Context, c, e int64) (model.Profile, error) {
	var v model.Profile
	err := r.DB.GetContext(ctx, &v, `SELECT e.id employee_id,e.name,e.email,e.code,d.name department,e.position,e.joined_on::text,COALESCE(p.phone,'') phone,COALESCE(p.address,'') address,COALESCE(p.emergency_name,'') emergency_name,COALESCE(p.emergency_phone,'') emergency_phone,COALESCE(p.emergency_relation,'') emergency_relation FROM employees e JOIN departments d ON d.id=e.department_id LEFT JOIN employee_profiles p ON p.company_id=e.company_id AND p.employee_id=e.id WHERE e.company_id=$1 AND e.id=$2`, c, e)
	return v, err
}
func (r *Repository) SaveProfile(ctx context.Context, u *model.User, e int64, v model.Profile) error {
	tx, err := r.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = lockEmployee(ctx, tx, u.CompanyID, e); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO employee_profiles(company_id,employee_id,phone,address,emergency_name,emergency_phone,emergency_relation) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(company_id,employee_id) DO UPDATE SET phone=$3,address=$4,emergency_name=$5,emergency_phone=$6,emergency_relation=$7`, u.CompanyID, e, v.Phone, v.Address, v.EmergencyName, v.EmergencyPhone, v.EmergencyRelation)
	if err != nil {
		return err
	}
	if err = audit(ctx, tx, u.CompanyID, u.ID, "update", "profile", e, "Memperbarui data kontak karyawan"); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *Repository) AuditLogs(ctx context.Context, c int64) ([]model.AuditLog, error) {
	v := []model.AuditLog{}
	err := r.DB.SelectContext(ctx, &v, `SELECT a.id,u.name actor,a.action,a.resource,a.resource_id,a.summary,a.created_at FROM audit_logs a JOIN users u ON u.id=a.actor_id WHERE a.company_id=$1 ORDER BY a.created_at DESC,a.id DESC LIMIT 250`, c)
	return v, err
}
