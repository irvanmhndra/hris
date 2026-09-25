package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct{ repo repository.AuthRepository }

func NewAuthService(repo repository.AuthRepository) *AuthService {
	return &AuthService{repo: repo}
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
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", nil, err
	}
	token := hex.EncodeToString(b)
	if err = s.repo.CreateSession(ctx, u.ID, HashToken(token)); err != nil {
		return "", nil, err
	}
	return token, u, nil
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
