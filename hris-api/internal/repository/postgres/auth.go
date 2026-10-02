package postgres

import (
	"context"

	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/jmoiron/sqlx"
)

type authRepository struct{ db *sqlx.DB }

func NewAuthRepository(db *sqlx.DB) repository.AuthRepository {
	return &authRepository{db: db}
}

const userFields = `u.id, u.company_id, u.employee_id, u.name, u.email, u.role,
	u.password_hash, c.name AS company_name,
	EXISTS (SELECT 1 FROM employees r WHERE r.company_id = u.company_id AND r.manager_id = u.employee_id
	        AND r.status = 'active') AS is_manager`

// A user may log in (and keep a session) only while their linked employee
// record, if any, is active.
func (r *authRepository) LoginUser(ctx context.Context, companySlug, email string) (*model.User, error) {
	var u model.User
	err := r.db.GetContext(ctx, &u, `
		SELECT `+userFields+`
		FROM users u
		JOIN companies c ON c.id = u.company_id
		LEFT JOIN employees e ON e.id = u.employee_id
		WHERE c.slug = $1 AND u.email = $2
		  AND (u.employee_id IS NULL OR e.status = 'active')`,
		companySlug, email)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *authRepository) CreateSession(ctx context.Context, userID int64, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_sessions (token_hash, user_id, expires_at)
		VALUES ($1, $2, now() + interval '12 hours')`,
		tokenHash, userID)
	return err
}

func (r *authRepository) SessionUser(ctx context.Context, tokenHash string) (*model.User, error) {
	var u model.User
	err := r.db.GetContext(ctx, &u, `
		SELECT `+userFields+`
		FROM user_sessions s
		JOIN users u ON u.id = s.user_id
		JOIN companies c ON c.id = u.company_id
		LEFT JOIN employees e ON e.id = u.employee_id
		WHERE s.token_hash = $1 AND s.expires_at > now()
		  AND (u.employee_id IS NULL OR e.status = 'active')`,
		tokenHash)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *authRepository) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE token_hash = $1`, tokenHash)
	return err
}

// CreatePasswordReset stores a one-hour reset token for an active user and
// revokes their earlier unused tokens. sql.ErrNoRows means no such user.
func (r *authRepository) CreatePasswordReset(ctx context.Context, companySlug, email, tokenHash string) (*model.User, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)
	var u model.User
	if err = tx.GetContext(ctx, &u, `
		SELECT `+userFields+`
		FROM users u
		JOIN companies c ON c.id = u.company_id
		LEFT JOIN employees e ON e.id = u.employee_id
		WHERE c.slug = $1 AND u.email = $2
		  AND (u.employee_id IS NULL OR e.status = 'active')`,
		companySlug, email); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM password_resets WHERE user_id = $1 AND used_at IS NULL`, u.ID); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO password_resets (token_hash, user_id, expires_at) VALUES ($1, $2, now() + interval '1 hour')`,
		tokenHash, u.ID); err != nil {
		return nil, err
	}
	return &u, tx.Commit()
}

// ResetPassword spends a valid token once, sets the new password, and ends
// every session of the user. sql.ErrNoRows means the token is invalid.
func (r *authRepository) ResetPassword(ctx context.Context, tokenHash, passwordHash string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollback(tx)
	var userID, companyID int64
	if err = tx.QueryRowxContext(ctx, `
		SELECT p.user_id, u.company_id
		FROM password_resets p
		JOIN users u ON u.id = p.user_id
		LEFT JOIN employees e ON e.id = u.employee_id
		WHERE p.token_hash = $1 AND p.used_at IS NULL AND p.expires_at > now()
		  AND (u.employee_id IS NULL OR e.status = 'active')
		FOR UPDATE OF p`,
		tokenHash).Scan(&userID, &companyID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE password_resets SET used_at = now() WHERE token_hash = $1`, tokenHash); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1`, userID, passwordHash); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM user_sessions WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if err = audit(ctx, tx, companyID, userID, "reset_password", "user", userID, "Reset password mandiri"); err != nil {
		return err
	}
	return tx.Commit()
}

// Register creates a company with default settings, a general department,
// and its first admin. A taken slug or email surfaces as a unique violation.
func (r *authRepository) Register(ctx context.Context, v model.RegisterInput) (*model.User, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer rollback(tx)
	var companyID, userID int64
	if err = tx.GetContext(ctx, &companyID, `
		INSERT INTO companies (name, slug) VALUES ($1, $2) RETURNING id`, v.CompanyName, v.CompanySlug); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `
		INSERT INTO work_calendars (company_id) VALUES ($1);
		`, companyID); err != nil {
		return nil, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO departments (company_id, name) VALUES ($1, 'Umum')`, companyID); err != nil {
		return nil, err
	}
	if err = tx.GetContext(ctx, &userID, `
		INSERT INTO users (company_id, name, email, password_hash, role) VALUES ($1, $2, $3, $4, 'admin') RETURNING id`,
		companyID, v.Name, v.Email, v.PasswordHash); err != nil {
		return nil, err
	}
	if err = audit(ctx, tx, companyID, userID, "create", "company", companyID, "Pendaftaran perusahaan "+v.CompanyName); err != nil {
		return nil, err
	}
	var u model.User
	if err = tx.GetContext(ctx, &u, `
		SELECT `+userFields+` FROM users u JOIN companies c ON c.id = u.company_id WHERE u.id = $1`, userID); err != nil {
		return nil, err
	}
	return &u, tx.Commit()
}
