package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/irvanmhndra/hris-api/pkg/mailer"
	"golang.org/x/crypto/bcrypt"
)

// AuthOptions configures self-service account features.
type AuthOptions struct {
	SignupEnabled bool
	AdminURL      string
	EmployeeURL   string
	Mailer        mailer.Mailer
}

type AuthService struct {
	repo repository.AuthRepository
	opts AuthOptions
}

func NewAuthService(repo repository.AuthRepository, opts AuthOptions) *AuthService {
	return &AuthService{repo: repo, opts: opts}
}

// HashToken is the only form of a session token the database ever stores, so
// a leaked sessions table cannot be replayed.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// dummyHash lets an unknown email cost the same bcrypt work as a real one, so
// response timing does not reveal which accounts exist.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("hris-timing-equaliser"), bcrypt.DefaultCost)

var errBadCredentials = apperror.Unauthorized("Perusahaan, email, atau password salah")

func (s *AuthService) Login(ctx context.Context, v dto.Login) (string, *model.User, error) {
	u, err := s.repo.LoginUser(ctx, strings.TrimSpace(v.Company), strings.ToLower(strings.TrimSpace(v.Email)))
	if errors.Is(err, sql.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(v.Password))
		return "", nil, errBadCredentials
	}
	if err != nil {
		return "", nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(v.Password)) != nil {
		return "", nil, errBadCredentials
	}
	return s.startSession(ctx, u)
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *AuthService) startSession(ctx context.Context, u *model.User) (string, *model.User, error) {
	token, err := newToken()
	if err != nil {
		return "", nil, err
	}
	if err = s.repo.CreateSession(ctx, u.ID, HashToken(token)); err != nil {
		return "", nil, err
	}
	return token, u, nil
}

// Config tells the login page which self-service features are on.
func (s *AuthService) Config() map[string]bool {
	return map[string]bool{"signup_enabled": s.opts.SignupEnabled}
}

// RequestPasswordReset emails a one-hour reset link. It answers the same way
// whether or not the account exists, so it cannot be used to probe emails.
func (s *AuthService) RequestPasswordReset(ctx context.Context, v dto.ForgotPassword) error {
	company, email := strings.TrimSpace(v.Company), strings.ToLower(strings.TrimSpace(v.Email))
	if company == "" || email == "" || len(email) > 254 {
		return apperror.Invalid("Kode perusahaan dan email wajib diisi")
	}
	token, err := newToken()
	if err != nil {
		return err
	}
	u, err := s.repo.CreatePasswordReset(ctx, company, email, HashToken(token))
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	base := s.opts.EmployeeURL
	if u.Role == model.RoleAdmin {
		base = s.opts.AdminURL
	}
	link := base + "/reset-password?token=" + token
	body := "Halo " + u.Name + ",\n\nKami menerima permintaan reset password akun " + u.CompanyName +
		". Buka tautan berikut dalam 1 jam untuk membuat password baru:\n\n" + link +
		"\n\nAbaikan email ini bila Anda tidak memintanya; password Anda tidak berubah."
	// Send in the background so response time does not reveal the account.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.opts.Mailer.Send(ctx, u.Email, "Reset password People HRIS", body); err != nil {
			slog.Error("password reset email failed", "user_id", u.ID, "error", err)
		}
	}()
	return nil
}

func validPassword(p string) bool { return len(p) >= 12 && len(p) <= 72 }

func (s *AuthService) ResetPassword(ctx context.Context, v dto.ResetPassword) error {
	if !validPassword(v.Password) {
		return apperror.Invalid("Password harus 12–72 karakter")
	}
	if len(v.Token) != 64 {
		return apperror.Invalid("Tautan reset tidak valid atau kedaluwarsa")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(v.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	err = s.repo.ResetPassword(ctx, HashToken(v.Token), string(h))
	if errors.Is(err, sql.ErrNoRows) {
		return apperror.Invalid("Tautan reset tidak valid atau kedaluwarsa")
	}
	return err
}

var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$`)

// Register creates a company and signs its first admin in.
func (s *AuthService) Register(ctx context.Context, v dto.Register) (string, *model.User, error) {
	if !s.opts.SignupEnabled {
		return "", nil, apperror.Forbidden("Pendaftaran perusahaan baru sedang ditutup")
	}
	v.CompanyName, v.Name = strings.TrimSpace(v.CompanyName), strings.TrimSpace(v.Name)
	v.CompanySlug = strings.ToLower(strings.TrimSpace(v.CompanySlug))
	v.Email = strings.ToLower(strings.TrimSpace(v.Email))
	if len(v.CompanyName) < 2 || len(v.CompanyName) > 120 || len(v.Name) < 2 || len(v.Name) > 120 {
		return "", nil, apperror.Invalid("Nama perusahaan dan nama admin harus 2–120 karakter")
	}
	if !slugPattern.MatchString(v.CompanySlug) {
		return "", nil, apperror.Invalid("Kode perusahaan 3–40 karakter: huruf kecil, angka, atau tanda hubung")
	}
	if a, err := mail.ParseAddress(v.Email); err != nil || a.Address != v.Email {
		return "", nil, apperror.Invalid("Email tidak valid")
	}
	if !validPassword(v.Password) {
		return "", nil, apperror.Invalid("Password harus 12–72 karakter")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(v.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", nil, err
	}
	u, err := s.repo.Register(ctx, model.RegisterInput{
		CompanyName: v.CompanyName, CompanySlug: v.CompanySlug, Name: v.Name, Email: v.Email, PasswordHash: string(h),
	})
	if err != nil {
		return "", nil, err
	}
	return s.startSession(ctx, u)
}

// Authenticate resolves a bearer token to its user. An unknown or expired
// token is 401; a database failure is returned as-is (500) rather than
// silently logging everyone out.
func (s *AuthService) Authenticate(ctx context.Context, token string) (*model.User, error) {
	u, err := s.repo.SessionUser(ctx, HashToken(token))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperror.Unauthorized("Sesi telah berakhir")
	}
	return u, err
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	return s.repo.DeleteSession(ctx, HashToken(token))
}
