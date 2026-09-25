package service

import (
	"context"

	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
)

type AuditService struct{ repo repository.AuditRepository }

func NewAuditService(repo repository.AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

// legacyAuditWindow is what the unpaginated endpoint has always returned.
const legacyAuditWindow = 250

type AuditPage struct {
	Items   []model.AuditLog
	Total   int
	Page    int
	PerPage int
	Paged   bool
}

// AuditLogs pages the log newest-first. Page 0 keeps the original response:
// the latest 250 entries, unpaginated.
func (s *AuditService) AuditLogs(ctx context.Context, companyID int64, page, perPage int) (AuditPage, error) {
	out := AuditPage{Paged: page > 0}
	limit, offset := legacyAuditWindow, 0
	if out.Paged {
		limit, offset, out.Page, out.PerPage = pageWindow(page, perPage)
	}
	var err error
	out.Items, out.Total, err = s.repo.AuditLogs(ctx, companyID, limit, offset)
	return out, err
}
