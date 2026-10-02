package postgres

import (
	"context"
	"encoding/json"

	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/jmoiron/sqlx"
)

type hrItemRepository struct{ db *sqlx.DB }

func NewHRItemRepository(db *sqlx.DB) repository.HRItemRepository {
	return &hrItemRepository{db: db}
}

const itemFields = `h.id, h.module, h.employee_id, COALESCE(e.name, '') employee_name, h.title,
	h.description, h.status, COALESCE(h.due_date::text, '') due_date, h.data, h.version,
	h.stage, h.review_note, h.created_at`

func (r *hrItemRepository) HRItems(ctx context.Context, companyID int64, module string, scope model.HRItemScope) ([]model.HRItem, error) {
	v := []model.HRItem{}
	err := r.db.SelectContext(ctx, &v, `
		SELECT `+itemFields+`
		FROM hr_items h
		LEFT JOIN employees e ON e.id = h.employee_id AND e.company_id = h.company_id
		WHERE h.company_id = $1 AND h.module = $2
		  AND ($3::bigint IS NULL OR h.employee_id = $3)
		  AND ($4 OR h.module NOT IN ('announcements', 'documents') OR h.status = 'published')
		ORDER BY h.created_at DESC, h.id DESC`,
		companyID, module, scope.EmployeeID, scope.IncludeUnpublished)
	return v, err
}

// SaveHRItem creates (id == 0) or updates an item. Updates use optimistic
// locking on version; overtime and correction requests are checked for
// conflicts under the employee lock.
func (r *hrItemRepository) SaveHRItem(ctx context.Context, companyID, actorID int64, module string, id int64, v model.HRItemInput) (int64, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer rollback(tx)
	if v.EmployeeID != nil {
		if err = lockEmployee(ctx, tx, companyID, *v.EmployeeID); err != nil {
			return 0, err
		}
	}
	switch module {
	case "overtime":
		var data model.HRData
		if err = json.Unmarshal(v.Data, &data); err != nil {
			return 0, err
		}
		var overlap bool
		err = tx.GetContext(ctx, &overlap, `
			SELECT EXISTS (
				SELECT 1 FROM hr_items
				WHERE company_id = $1 AND employee_id = $2 AND module = 'overtime'
				  AND status IN ('pending', 'approved')
				  AND (data->>'start_at')::timestamptz < $4::timestamptz
				  AND (data->>'end_at')::timestamptz > $3::timestamptz)`,
			companyID, v.EmployeeID, data.StartAt, data.EndAt)
		if err != nil {
			return 0, err
		}
		if overlap {
			return 0, apperror.Conflict("Waktu lembur bertumpang tindih")
		}
	case "corrections":
		var data model.HRData
		if err = json.Unmarshal(v.Data, &data); err != nil {
			return 0, err
		}
		var exists bool
		err = tx.GetContext(ctx, &exists, `
			SELECT EXISTS (
				SELECT 1 FROM hr_items
				WHERE company_id = $1 AND employee_id = $2 AND module = 'corrections'
				  AND status = 'pending' AND data->>'date' = $3)`,
			companyID, v.EmployeeID, data.Date)
		if err != nil {
			return 0, err
		}
		if exists {
			return 0, apperror.Conflict("Koreksi tanggal ini masih menunggu persetujuan")
		}
		// Remember the attendance as it was, so approval can reject a stale correction.
		if data.Original, err = attendanceFingerprint(ctx, tx, companyID, *v.EmployeeID, data.Date); err != nil {
			return 0, err
		}
		if v.Data, err = json.Marshal(data); err != nil {
			return 0, err
		}
	}

	action := "create"
	if id == 0 {
		err = tx.QueryRowxContext(ctx, `
			INSERT INTO hr_items (company_id, module, employee_id, title, description, status, due_date, data, created_by, stage)
			VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::date, $8, $9,
			        CASE WHEN $2 IN ('overtime', 'corrections') THEN `+approvalStage("$1", "$3")+` ELSE 'hr' END)
			RETURNING id`,
			companyID, module, v.EmployeeID, v.Title, v.Description, v.Status, v.DueDate, string(v.Data), actorID).Scan(&id)
		if err != nil {
			return 0, err
		}
	} else {
		action = "update"
		res, err := tx.ExecContext(ctx, `
			UPDATE hr_items
			SET employee_id = $4, title = $5, description = $6, status = $7,
			    due_date = NULLIF($8, '')::date, data = $9, version = version + 1, updated_at = now()
			WHERE company_id = $1 AND module = $2 AND id = $3 AND version = $10`,
			companyID, module, id, v.EmployeeID, v.Title, v.Description, v.Status, v.DueDate, string(v.Data), v.Version)
		if err != nil {
			return 0, err
		}
		if rowsAffected(res) == 0 {
			return 0, apperror.Conflict("Data sudah berubah atau tidak ditemukan. Muat ulang terlebih dahulu.")
		}
	}
	if err = audit(ctx, tx, companyID, actorID, action, module, id, v.Title); err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

// ActHRItem applies a workflow action (approve/reject/cancel, start/complete,
// progress). Approving a correction rewrites the attendance in the same
// transaction, but only if the attendance still matches its fingerprint.
func (r *hrItemRepository) ActHRItem(ctx context.Context, companyID, actorID int64, module string, id int64, scope model.HRItemScope, v model.HRActionInput) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	// The employee lock serialises corrections with check-in/out and other requests.
	var employeeID int64
	if err = tx.GetContext(ctx, &employeeID, `
		SELECT employee_id FROM hr_items
		WHERE company_id = $1 AND module = $2 AND id = $3
		  AND ($4::bigint IS NULL OR employee_id = $4)`,
		companyID, module, id, scope.EmployeeID); err != nil {
		return err
	}
	if err = lockEmployee(ctx, tx, companyID, employeeID); err != nil {
		return err
	}
	var item model.HRItem
	if err = tx.GetContext(ctx, &item, `
		SELECT `+itemFields+`
		FROM hr_items h
		LEFT JOIN employees e ON e.id = h.employee_id
		WHERE h.company_id = $1 AND h.module = $2 AND h.id = $3
		FOR UPDATE OF h`,
		companyID, module, id); err != nil {
		return err
	}
	if item.Version != v.Version {
		return apperror.Conflict("Data sudah berubah. Muat ulang terlebih dahulu.")
	}
	var data model.HRData
	if err = json.Unmarshal(item.Data, &data); err != nil {
		return err
	}

	status := item.Status
	switch module {
	case "overtime", "corrections":
		if item.Status != "pending" {
			return apperror.Conflict("Pengajuan sudah diproses")
		}
		status = map[string]string{"approve": "approved", "reject": "rejected", "cancel": "cancelled"}[v.Action]
		if module == "corrections" && v.Action == "approve" {
			current, err := attendanceFingerprint(ctx, tx, companyID, employeeID, data.Date)
			if err != nil {
				return err
			}
			if current != data.Original {
				return apperror.Conflict("Absensi berubah sejak pengajuan; tolak dan ajukan koreksi baru")
			}
			_, err = tx.ExecContext(ctx, `
				INSERT INTO attendances (company_id, employee_id, date, check_in, check_out)
				VALUES ($1, $2, $3, $4, $5)
				ON CONFLICT (company_id, employee_id, date) DO UPDATE SET check_in = $4, check_out = $5`,
				companyID, employeeID, data.Date,
				data.Date+"T"+data.CheckIn+":00+07:00",
				data.Date+"T"+data.CheckOut+":00+07:00")
			if err != nil {
				return err
			}
		}
	case "onboarding":
		if item.Status == "done" {
			return apperror.Conflict("Tugas sudah selesai")
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

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE hr_items
		SET status = $4, data = $5, version = version + 1, updated_at = now(),
		    reviewed_by = $6, review_note = $7
		WHERE company_id = $1 AND module = $2 AND id = $3`,
		companyID, module, id, status, string(body), actorID, v.Note)
	if err != nil {
		return err
	}
	if err = audit(ctx, tx, companyID, actorID, v.Action, module, id, item.Title); err != nil {
		return err
	}
	return tx.Commit()
}
