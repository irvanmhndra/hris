package postgres

import (
	"context"

	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/jmoiron/sqlx"
)

type fileRepository struct{ db *sqlx.DB }

func NewFileRepository(db *sqlx.DB) repository.FileRepository {
	return &fileRepository{db: db}
}

func (r *fileRepository) CreateFile(ctx context.Context, companyID, actorID int64, f model.File) (model.File, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return f, err
	}
	defer rollback(tx)
	if err = tx.QueryRowxContext(ctx, `
		INSERT INTO files (company_id, name, content_type, size_bytes, storage_key, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at`,
		companyID, f.Name, f.ContentType, f.SizeBytes, f.StorageKey, actorID).Scan(&f.ID, &f.CreatedAt); err != nil {
		return f, err
	}
	if err = audit(ctx, tx, companyID, actorID, "upload", "file", f.ID, f.Name); err != nil {
		return f, err
	}
	return f, tx.Commit()
}

// File returns a company's file. With employeeOnly, the file must be attached
// to a published announcement or document, the only files employees may read.
func (r *fileRepository) File(ctx context.Context, companyID, id int64, employeeOnly bool) (model.File, error) {
	var f model.File
	err := r.db.GetContext(ctx, &f, `
		SELECT id, name, content_type, size_bytes, storage_key, created_at
		FROM files f
		WHERE f.company_id = $1 AND f.id = $2
		  AND (NOT $3 OR EXISTS (
		        SELECT 1 FROM hr_items h
		        WHERE h.company_id = f.company_id AND h.module IN ('announcements', 'documents')
		          AND h.status = 'published' AND h.data->>'file_id' = f.id::text))`,
		companyID, id, employeeOnly)
	return f, err
}
