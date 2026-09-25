package service

import (
	"context"
	"encoding/json"
	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"net/mail"
	"net/url"
	"strings"
	"time"
)

var modules = map[string][]string{
	"overtime": {"pending", "approved", "rejected", "cancelled"}, "corrections": {"pending", "approved", "rejected", "cancelled"},
	"announcements": {"draft", "published", "archived"}, "documents": {"draft", "published", "archived"},
	"onboarding": {"todo", "in_progress", "done"}, "assets": {"available", "assigned", "maintenance", "retired"}, "goals": {"active", "done"},
	"recruitment": {"applied", "screening", "interview", "offer", "hired", "rejected"},
}

func moduleAccess(u *model.User, module string) error {
	if _, ok := modules[module]; !ok {
		return &apperror.Error{404, "Modul tidak ditemukan"}
	}
	if u.Role != "admin" && (module == "recruitment" || u.EmployeeID == nil) {
		return &apperror.Error{403, "Akses tidak diizinkan"}
	}
	return nil
}
func (s *Service) HRItems(ctx context.Context, u *model.User, module string) ([]model.HRItem, error) {
	if err := moduleAccess(u, module); err != nil {
		return nil, err
	}
	return s.Repo.HRItems(ctx, u, module)
}
func ValidateHRItem(module string, v *dto.HRItem) error {
	v.Title = strings.TrimSpace(v.Title)
	v.Description = strings.TrimSpace(v.Description)
	if len(v.Title) < 2 || len(v.Title) > 160 || len(v.Description) > 5000 {
		return apperror.Invalid("Judul harus 2–160 karakter; deskripsi maksimal 5000 karakter")
	}
	valid := false
	for _, x := range modules[module] {
		if v.Status == x {
			valid = true
		}
	}
	if !valid {
		return apperror.Invalid("Status tidak valid")
	}
	if v.DueDate != "" {
		if _, err := time.Parse("2006-01-02", v.DueDate); err != nil {
			return apperror.Invalid("Tanggal tidak valid")
		}
	}
	var data dto.HRData
	if len(v.Data) == 0 {
		v.Data = json.RawMessage(`{}`)
	}
	if err := json.Unmarshal(v.Data, &data); err != nil {
		return apperror.Invalid("Detail modul tidak valid")
	}
	if len(v.Data) > 12000 {
		return apperror.Invalid("Detail terlalu panjang")
	}
	if module == "documents" {
		link, err := url.Parse(data.URL)
		if err != nil || link.Scheme != "https" || link.Host == "" || link.User != nil {
			return apperror.Invalid("Dokumen harus menggunakan tautan HTTPS tanpa kredensial")
		}
	}
	if module == "assets" {
		data.Code = strings.TrimSpace(data.Code)
		if len(data.Code) < 2 || len(data.Code) > 80 {
			return apperror.Invalid("Kode aset harus 2–80 karakter")
		}
		if (v.Status == "assigned") != (v.EmployeeID != nil) {
			return apperror.Invalid("Aset berstatus dipinjam wajib memiliki karyawan; status lain tidak boleh memiliki peminjam")
		}
	}
	if module == "goals" {
		if data.Target == "" || len(data.Target) > 500 || data.Progress < 0 || data.Progress > 100 {
			return apperror.Invalid("Target wajib diisi dan progres harus 0–100")
		}
		v.Status = "active"
		if data.Progress == 100 {
			v.Status = "done"
		}
	}
	if module == "recruitment" {
		a, err := mail.ParseAddress(data.Email)
		if err != nil || a.Address != data.Email || strings.TrimSpace(data.Position) == "" {
			return apperror.Invalid("Email kandidat dan posisi wajib valid")
		}
	}
	if module == "overtime" {
		start, e1 := time.Parse(time.RFC3339, data.StartAt)
		end, e2 := time.Parse(time.RFC3339, data.EndAt)
		duration := end.Sub(start)
		if e1 != nil || e2 != nil || duration < 15*time.Minute || duration > 16*time.Hour {
			return apperror.Invalid("Lembur harus memiliki waktu valid dengan durasi 15 menit–16 jam")
		}
		data.StartAt = start.UTC().Format(time.RFC3339)
		data.EndAt = end.UTC().Format(time.RFC3339)
	}
	if module == "corrections" {
		in, e1 := time.Parse(time.RFC3339, data.Date+"T"+data.CheckIn+":00+07:00")
		out, e2 := time.Parse(time.RFC3339, data.Date+"T"+data.CheckOut+":00+07:00")
		if e1 != nil || e2 != nil || !out.After(in) || out.After(time.Now()) {
			return apperror.Invalid("Koreksi harus berisi jam masuk/pulang di hari yang sama, urut, dan tidak di masa depan")
		}
		data.Original = ""
	}
	if (module == "overtime" || module == "corrections") && len(v.Description) < 5 {
		return apperror.Invalid("Alasan minimal 5 karakter")
	}
	// Decode into a typed structure before persistence; clients cannot inject workflow metadata.
	v.Data, _ = json.Marshal(data)
	return nil
}
func (s *Service) SaveHRItem(ctx context.Context, u *model.User, module string, id int64, v dto.HRItem) (int64, error) {
	if err := moduleAccess(u, module); err != nil {
		return 0, err
	}
	request := module == "overtime" || module == "corrections"
	if request {
		if u.Role != "employee" || id != 0 {
			return 0, &apperror.Error{403, "Pengajuan hanya dapat dibuat karyawan; gunakan tindakan persetujuan untuk memprosesnya"}
		}
		v.EmployeeID = u.EmployeeID
		v.Status = "pending"
	} else if u.Role != "admin" {
		return 0, &apperror.Error{403, "Hanya admin dapat mengelola data ini"}
	}
	if module == "announcements" || module == "documents" || module == "recruitment" {
		v.EmployeeID = nil
	}
	if (request || module == "onboarding" || module == "goals") && (v.EmployeeID == nil || *v.EmployeeID <= 0) {
		return 0, apperror.Invalid("Karyawan wajib dipilih")
	}
	if id != 0 && v.Version < 1 {
		return 0, apperror.Invalid("Versi data wajib disertakan")
	}
	if err := ValidateHRItem(module, &v); err != nil {
		return 0, err
	}
	return s.Repo.SaveHRItem(ctx, u, module, id, v)
}
func (s *Service) ActHRItem(ctx context.Context, u *model.User, module string, id int64, v dto.HRAction) error {
	if err := moduleAccess(u, module); err != nil {
		return err
	}
	if v.Version < 1 || len(v.Note) > 1000 {
		return apperror.Invalid("Versi atau catatan tidak valid")
	}
	allowed := false
	if module == "overtime" || module == "corrections" {
		allowed = (u.Role == "admin" && (v.Action == "approve" || v.Action == "reject")) || (u.Role == "employee" && v.Action == "cancel")
		if v.Action == "reject" && len(strings.TrimSpace(v.Note)) < 5 {
			return apperror.Invalid("Alasan penolakan minimal 5 karakter")
		}
	}
	if module == "onboarding" {
		allowed = v.Action == "start" || v.Action == "complete"
	}
	if module == "goals" {
		allowed = v.Action == "progress"
		if v.Progress < 0 || v.Progress > 100 {
			return apperror.Invalid("Progres harus 0–100")
		}
	}
	if !allowed {
		return &apperror.Error{403, "Tindakan tidak diizinkan"}
	}
	return s.Repo.ActHRItem(ctx, u, module, id, v)
}
func (s *Service) WorkCalendar(ctx context.Context, c int64) (model.WorkCalendar, error) {
	return s.Repo.WorkCalendar(ctx, c)
}
func (s *Service) SaveCalendar(ctx context.Context, u *model.User, v model.WorkCalendar) error {
	if len(v.Workdays) < 1 || len(v.Workdays) > 7 || v.AnnualAllowance < 0 || v.AnnualAllowance > 366 {
		return apperror.Invalid("Hari kerja atau kuota cuti tidak valid")
	}
	seen := map[int64]bool{}
	for _, d := range v.Workdays {
		if d < 0 || d > 6 || seen[d] {
			return apperror.Invalid("Hari kerja harus unik (0–6)")
		}
		seen[d] = true
	}
	start, e1 := time.Parse("15:04", v.StartTime)
	end, e2 := time.Parse("15:04", v.EndTime)
	if e1 != nil || e2 != nil || !end.After(start) {
		return apperror.Invalid("Jam kerja harus berada pada hari yang sama dan jam selesai setelah jam mulai")
	}
	return s.Repo.SaveCalendar(ctx, u, v)
}
func (s *Service) Holidays(ctx context.Context, c int64) ([]model.Holiday, error) {
	return s.Repo.Holidays(ctx, c)
}
func (s *Service) SaveHoliday(ctx context.Context, u *model.User, v model.Holiday) error {
	if _, err := time.Parse("2006-01-02", v.Date); err != nil {
		return apperror.Invalid("Tanggal libur tidak valid")
	}
	v.Name = strings.TrimSpace(v.Name)
	if len(v.Name) < 2 || len(v.Name) > 120 {
		return apperror.Invalid("Nama hari libur harus 2–120 karakter")
	}
	return s.Repo.SaveHoliday(ctx, u, v)
}
func (s *Service) DeleteHoliday(ctx context.Context, u *model.User, id int64) error {
	return s.Repo.DeleteHoliday(ctx, u, id)
}
func (s *Service) Balances(ctx context.Context, u *model.User, year int) ([]model.Balance, error) {
	if year < 2000 || year > 2200 {
		return nil, apperror.Invalid("Tahun harus 2000–2200")
	}
	var e *int64
	if u.Role == "employee" {
		e = u.EmployeeID
	}
	return s.Repo.Balances(ctx, u.CompanyID, e, year)
}
func (s *Service) SetAllowance(ctx context.Context, u *model.User, e int64, year, allowance int) error {
	if year < 2000 || year > 2200 || allowance < 0 || allowance > 366 {
		return apperror.Invalid("Tahun atau kuota tidak valid")
	}
	return s.Repo.SetAllowance(ctx, u, e, year, allowance)
}
func (s *Service) CancelLeave(ctx context.Context, u *model.User, id int64) error {
	return s.Repo.CancelLeave(ctx, u, id)
}
func (s *Service) Profile(ctx context.Context, u *model.User, e int64) (model.Profile, error) {
	return s.Repo.Profile(ctx, u.CompanyID, e)
}
func (s *Service) SaveProfile(ctx context.Context, u *model.User, e int64, v model.Profile) error {
	if len(v.Phone) > 30 || len(v.Address) > 500 || len(v.EmergencyName) > 120 || len(v.EmergencyPhone) > 30 || len(v.EmergencyRelation) > 80 {
		return apperror.Invalid("Data kontak terlalu panjang")
	}
	return s.Repo.SaveProfile(ctx, u, e, v)
}
func (s *Service) AuditLogs(ctx context.Context, c int64) ([]model.AuditLog, error) {
	return s.Repo.AuditLogs(ctx, c)
}
