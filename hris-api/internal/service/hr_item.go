package service

import (
	"context"
	"encoding/json"
	"net/mail"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
)

type HRItemService struct{ repo repository.HRItemRepository }

func NewHRItemService(repo repository.HRItemRepository) *HRItemService {
	return &HRItemService{repo: repo}
}

// modules whitelists the lightweight HR workflows stored in hr_items and the
// statuses each allows.
var modules = map[string][]string{
	"overtime":      {"pending", "approved", "rejected", "cancelled"},
	"corrections":   {"pending", "approved", "rejected", "cancelled"},
	"announcements": {"draft", "published", "archived"},
	"documents":     {"draft", "published", "archived"},
	"onboarding":    {"todo", "in_progress", "done"},
	"assets":        {"available", "assigned", "maintenance", "retired"},
	"goals":         {"active", "done"},
	"recruitment":   {"applied", "screening", "interview", "offer", "hired", "rejected"},
}

// isRequest reports modules that employees file and admins decide.
func isRequest(module string) bool { return module == "overtime" || module == "corrections" }

func moduleAccess(u *model.User, module string) error {
	if _, ok := modules[module]; !ok {
		return apperror.NotFound("Modul tidak ditemukan")
	}
	if u.Role != model.RoleAdmin && (module == "recruitment" || u.EmployeeID == nil) {
		return apperror.Forbidden("Akses tidak diizinkan")
	}
	return nil
}

// itemScope: employees see their own items, plus company-wide published
// announcements and documents; admins see everything.
func itemScope(u *model.User, module string) model.HRItemScope {
	s := model.HRItemScope{IncludeUnpublished: u.Role == model.RoleAdmin}
	if u.Role == model.RoleEmployee && module != "announcements" && module != "documents" {
		s.EmployeeID = u.EmployeeID
	}
	return s
}

func (s *HRItemService) HRItems(ctx context.Context, u *model.User, module string) ([]model.HRItem, error) {
	if err := moduleAccess(u, module); err != nil {
		return nil, err
	}
	return s.repo.HRItems(ctx, u.CompanyID, module, itemScope(u, module))
}

// ValidateHRItem normalises and validates an item. Data is decoded into the
// typed HRData and re-encoded, so clients cannot inject workflow metadata.
func ValidateHRItem(module string, v *dto.HRItem) error {
	v.Title = strings.TrimSpace(v.Title)
	v.Description = strings.TrimSpace(v.Description)
	if len(v.Title) < 2 || len(v.Title) > 160 || len(v.Description) > 5000 {
		return apperror.Invalid("Judul harus 2–160 karakter; deskripsi maksimal 5000 karakter")
	}
	if !slices.Contains(modules[module], v.Status) {
		return apperror.Invalid("Status tidak valid")
	}
	if v.DueDate != "" {
		if _, err := time.Parse("2006-01-02", v.DueDate); err != nil {
			return apperror.Invalid("Tanggal tidak valid")
		}
	}
	if len(v.Data) == 0 {
		v.Data = json.RawMessage(`{}`)
	}
	var data model.HRData
	if err := json.Unmarshal(v.Data, &data); err != nil {
		return apperror.Invalid("Detail modul tidak valid")
	}
	if len(v.Data) > 12000 {
		return apperror.Invalid("Detail terlalu panjang")
	}

	switch module {
	case "documents":
		link, err := url.Parse(data.URL)
		if err != nil || link.Scheme != "https" || link.Host == "" || link.User != nil {
			return apperror.Invalid("Dokumen harus menggunakan tautan HTTPS tanpa kredensial")
		}
	case "assets":
		data.Code = strings.TrimSpace(data.Code)
		if len(data.Code) < 2 || len(data.Code) > 80 {
			return apperror.Invalid("Kode aset harus 2–80 karakter")
		}
		if (v.Status == "assigned") != (v.EmployeeID != nil) {
			return apperror.Invalid("Aset berstatus dipinjam wajib memiliki karyawan; status lain tidak boleh memiliki peminjam")
		}
	case "goals":
		if data.Target == "" || len(data.Target) > 500 || data.Progress < 0 || data.Progress > 100 {
			return apperror.Invalid("Target wajib diisi dan progres harus 0–100")
		}
		v.Status = "active"
		if data.Progress == 100 {
			v.Status = "done"
		}
	case "recruitment":
		a, err := mail.ParseAddress(data.Email)
		if err != nil || a.Address != data.Email || strings.TrimSpace(data.Position) == "" {
			return apperror.Invalid("Email kandidat dan posisi wajib valid")
		}
	case "overtime":
		start, e1 := time.Parse(time.RFC3339, data.StartAt)
		end, e2 := time.Parse(time.RFC3339, data.EndAt)
		duration := end.Sub(start)
		if e1 != nil || e2 != nil || duration < 15*time.Minute || duration > 16*time.Hour {
			return apperror.Invalid("Lembur harus memiliki waktu valid dengan durasi 15 menit–16 jam")
		}
		data.StartAt = start.UTC().Format(time.RFC3339)
		data.EndAt = end.UTC().Format(time.RFC3339)
	case "corrections":
		in, e1 := time.Parse(time.RFC3339, data.Date+"T"+data.CheckIn+":00+07:00")
		out, e2 := time.Parse(time.RFC3339, data.Date+"T"+data.CheckOut+":00+07:00")
		if e1 != nil || e2 != nil || !out.After(in) || out.After(time.Now()) {
			return apperror.Invalid("Koreksi harus berisi jam masuk/pulang di hari yang sama, urut, dan tidak di masa depan")
		}
		data.Original = "" // set server-side from the live attendance row
	}
	if isRequest(module) && len(v.Description) < 5 {
		return apperror.Invalid("Alasan minimal 5 karakter")
	}

	encoded, err := json.Marshal(data)
	if err != nil {
		return err
	}
	v.Data = encoded
	return nil
}

func (s *HRItemService) SaveHRItem(ctx context.Context, u *model.User, module string, id int64, v dto.HRItem) (int64, error) {
	if err := moduleAccess(u, module); err != nil {
		return 0, err
	}
	if isRequest(module) {
		if u.Role != model.RoleEmployee || id != 0 {
			return 0, apperror.Forbidden("Pengajuan hanya dapat dibuat karyawan; gunakan tindakan persetujuan untuk memprosesnya")
		}
		v.EmployeeID = u.EmployeeID
		v.Status = "pending"
	} else if u.Role != model.RoleAdmin {
		return 0, apperror.Forbidden("Hanya admin dapat mengelola data ini")
	}
	if module == "announcements" || module == "documents" || module == "recruitment" {
		v.EmployeeID = nil
	}
	if (isRequest(module) || module == "onboarding" || module == "goals") && (v.EmployeeID == nil || *v.EmployeeID <= 0) {
		return 0, apperror.Invalid("Karyawan wajib dipilih")
	}
	if id != 0 && v.Version < 1 {
		return 0, apperror.Invalid("Versi data wajib disertakan")
	}
	if err := ValidateHRItem(module, &v); err != nil {
		return 0, err
	}
	return s.repo.SaveHRItem(ctx, u.CompanyID, u.ID, module, id, model.HRItemInput{
		EmployeeID: v.EmployeeID, Title: v.Title, Description: v.Description, Status: v.Status,
		DueDate: v.DueDate, Data: v.Data, Version: v.Version,
	})
}

func (s *HRItemService) ActHRItem(ctx context.Context, u *model.User, module string, id int64, v dto.HRAction) error {
	if err := moduleAccess(u, module); err != nil {
		return err
	}
	if v.Version < 1 || len(v.Note) > 1000 {
		return apperror.Invalid("Versi atau catatan tidak valid")
	}
	allowed := false
	switch module {
	case "overtime", "corrections":
		allowed = (u.Role == model.RoleAdmin && (v.Action == "approve" || v.Action == "reject")) ||
			(u.Role == model.RoleEmployee && v.Action == "cancel")
		if v.Action == "reject" && len(strings.TrimSpace(v.Note)) < 5 {
			return apperror.Invalid("Alasan penolakan minimal 5 karakter")
		}
	case "onboarding":
		allowed = v.Action == "start" || v.Action == "complete"
	case "goals":
		allowed = v.Action == "progress"
		if v.Progress < 0 || v.Progress > 100 {
			return apperror.Invalid("Progres harus 0–100")
		}
	}
	if !allowed {
		return apperror.Forbidden("Tindakan tidak diizinkan")
	}
	return s.repo.ActHRItem(ctx, u.CompanyID, u.ID, module, id, itemScope(u, module), model.HRActionInput{
		Action: v.Action, Note: v.Note, Progress: v.Progress, Version: v.Version,
	})
}
