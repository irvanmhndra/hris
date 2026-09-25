package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"golang.org/x/crypto/bcrypt"
	"net/mail"
	"strings"
	"time"
)

type Service struct{ Repo repository.Repository }

func HashToken(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func (s *Service) Login(ctx context.Context, v dto.Login) (string, *model.User, error) {
	u, err := s.Repo.LoginUser(ctx, strings.TrimSpace(v.Company), strings.ToLower(strings.TrimSpace(v.Email)))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", nil, err
	}
	if err != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(v.Password)) != nil {
		return "", nil, &apperror.Error{401, "Perusahaan, email, atau password salah"}
	}
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", nil, err
	}
	token := hex.EncodeToString(b)
	if err = s.Repo.CreateSession(ctx, u.ID, HashToken(token)); err != nil {
		return "", nil, err
	}
	return token, u, nil
}
func ValidateEmployee(v *dto.Employee, create bool) error {
	v.Name = strings.TrimSpace(v.Name)
	v.Code = strings.TrimSpace(v.Code)
	v.Email = strings.ToLower(strings.TrimSpace(v.Email))
	v.Position = strings.TrimSpace(v.Position)
	if v.Name == "" || v.Code == "" || v.Position == "" || v.DepartmentID <= 0 {
		return apperror.Invalid("Nama, NIK, jabatan, dan departemen wajib diisi")
	}
	if len(v.Name) > 120 || len(v.Code) > 40 || len(v.Position) > 120 {
		return apperror.Invalid("Data karyawan terlalu panjang")
	}
	addr, err := mail.ParseAddress(v.Email)
	if err != nil || addr.Address != v.Email {
		return apperror.Invalid("Email tidak valid")
	}
	if _, err = time.Parse("2006-01-02", v.JoinedOn); err != nil {
		return apperror.Invalid("Tanggal bergabung tidak valid")
	}
	if v.Status != "active" && v.Status != "inactive" {
		return apperror.Invalid("Status tidak valid")
	}
	if (create || v.Password != "") && (len(v.Password) < 12 || len(v.Password) > 72) {
		return apperror.Invalid("Password harus 12–72 karakter")
	}
	return nil
}
func (s *Service) SaveEmployee(ctx context.Context, c, id int64, v dto.Employee) (int64, error) {
	if err := ValidateEmployee(&v, id == 0); err != nil {
		return 0, err
	}
	hash := ""
	if v.Password != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(v.Password), bcrypt.DefaultCost)
		if err != nil {
			return 0, err
		}
		hash = string(h)
	}
	return s.Repo.SaveEmployee(ctx, c, id, v, hash)
}
func ValidateLeave(v dto.Leave) error {
	start, e1 := time.Parse("2006-01-02", v.StartDate)
	end, e2 := time.Parse("2006-01-02", v.EndDate)
	if e1 != nil || e2 != nil || end.Before(start) {
		return apperror.Invalid("Rentang tanggal cuti tidak valid")
	}
	if end.Sub(start) > 365*24*time.Hour {
		return apperror.Invalid("Durasi cuti maksimal 366 hari")
	}
	if v.Kind != "annual" && v.Kind != "sick" && v.Kind != "personal" {
		return apperror.Invalid("Jenis cuti tidak valid")
	}
	if len(strings.TrimSpace(v.Reason)) < 5 || len(v.Reason) > 1000 {
		return apperror.Invalid("Alasan harus 5–1000 karakter")
	}
	return nil
}
func (s *Service) CreateLeave(ctx context.Context, u *model.User, v dto.Leave) error {
	if err := ValidateLeave(v); err != nil {
		return err
	}
	return s.Repo.CreateLeave(ctx, u.CompanyID, *u.EmployeeID, v)
}
