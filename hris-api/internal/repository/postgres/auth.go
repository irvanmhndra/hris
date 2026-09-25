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
	u.password_hash, c.name AS company_name`

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
