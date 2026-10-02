package service

import (
	"context"
	"strings"

	"github.com/irvanmhndra/hris-api/internal/dto"
	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
)

// ApprovalService is the manager stage of two-step approvals: a manager
// forwards a direct report's request to HR or rejects it. HR (admins) can
// still decide a request at any stage.
type ApprovalService struct{ repo repository.ApprovalRepository }

func NewApprovalService(repo repository.ApprovalRepository) *ApprovalService {
	return &ApprovalService{repo: repo}
}

func managerOf(u *model.User) (int64, error) {
	if u.Role != model.RoleEmployee || u.EmployeeID == nil {
		return 0, apperror.Forbidden("Hanya atasan karyawan yang dapat memproses persetujuan tim")
	}
	return *u.EmployeeID, nil
}

func (s *ApprovalService) TeamApprovals(ctx context.Context, u *model.User) ([]model.TeamRequest, error) {
	managerID, err := managerOf(u)
	if err != nil {
		return nil, err
	}
	return s.repo.TeamApprovals(ctx, u.CompanyID, managerID)
}

func (s *ApprovalService) Review(ctx context.Context, u *model.User, kind string, id int64, v dto.TeamReview) error {
	managerID, err := managerOf(u)
	if err != nil {
		return err
	}
	if v.Action != "approve" && v.Action != "reject" {
		return apperror.Invalid("Tindakan harus approve atau reject")
	}
	v.Note = strings.TrimSpace(v.Note)
	if len(v.Note) > 1000 || (v.Action == "reject" && len(v.Note) < 5) {
		return apperror.Invalid("Alasan penolakan minimal 5 karakter (maks. 1000)")
	}
	approve := v.Action == "approve"
	switch kind {
	case "leave":
		return s.repo.ManagerReviewLeave(ctx, u.CompanyID, managerID, u.ID, id, approve, v.Note)
	case "overtime", "corrections":
		if v.Version < 1 {
			return apperror.Invalid("Versi data wajib disertakan")
		}
		return s.repo.ManagerReviewHRItem(ctx, u.CompanyID, managerID, u.ID, kind, id, approve, v.Note, v.Version)
	}
	return apperror.NotFound("Jenis pengajuan tidak dikenal")
}
