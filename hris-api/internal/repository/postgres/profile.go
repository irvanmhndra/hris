package postgres

import (
	"context"

	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/jmoiron/sqlx"
)

type profileRepository struct{ db *sqlx.DB }

func NewProfileRepository(db *sqlx.DB) repository.ProfileRepository {
	return &profileRepository{db: db}
}

func (r *profileRepository) Profile(ctx context.Context, companyID, employeeID int64) (model.Profile, error) {
	var v model.Profile
	err := r.db.GetContext(ctx, &v, `
		SELECT e.id employee_id, e.name, e.email, e.code, d.name department, e.position,
		       e.joined_on::text,
		       COALESCE(p.phone, '') phone,
		       COALESCE(p.address, '') address,
		       COALESCE(p.emergency_name, '') emergency_name,
		       COALESCE(p.emergency_phone, '') emergency_phone,
		       COALESCE(p.emergency_relation, '') emergency_relation
		FROM employees e
		JOIN departments d ON d.id = e.department_id
		LEFT JOIN employee_profiles p ON p.company_id = e.company_id AND p.employee_id = e.id
		WHERE e.company_id = $1 AND e.id = $2`,
		companyID, employeeID)
	return v, err
}

func (r *profileRepository) SaveProfile(ctx context.Context, companyID, actorID, employeeID int64, v model.Profile) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	if err = lockEmployee(ctx, tx, companyID, employeeID); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO employee_profiles
		       (company_id, employee_id, phone, address, emergency_name, emergency_phone, emergency_relation)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (company_id, employee_id) DO UPDATE
		SET phone = $3, address = $4, emergency_name = $5, emergency_phone = $6, emergency_relation = $7`,
		companyID, employeeID, v.Phone, v.Address, v.EmergencyName, v.EmergencyPhone, v.EmergencyRelation)
	if err != nil {
		return err
	}
	if err = audit(ctx, tx, companyID, actorID, "update", "profile", employeeID,
		"Memperbarui data kontak karyawan"); err != nil {
		return err
	}
	return tx.Commit()
}
