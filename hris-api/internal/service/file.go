package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/irvanmhndra/hris-api/internal/model"
	"github.com/irvanmhndra/hris-api/internal/repository"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/irvanmhndra/hris-api/pkg/storage"
)

// MaxFileBytes caps one upload at 10 MB.
const MaxFileBytes = 10 << 20

// allowedTypes maps permitted extensions to their content type and the type
// the content must sniff as (Office files are ZIP containers).
var allowedTypes = map[string]struct{ contentType, sniff string }{
	".pdf":  {"application/pdf", "application/pdf"},
	".png":  {"image/png", "image/png"},
	".jpg":  {"image/jpeg", "image/jpeg"},
	".jpeg": {"image/jpeg", "image/jpeg"},
	".webp": {"image/webp", "image/webp"},
	".docx": {"application/vnd.openxmlformats-officedocument.wordprocessingml.document", "application/zip"},
	".xlsx": {"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/zip"},
}

type FileService struct {
	repo  repository.FileRepository
	store storage.Storage
}

func NewFileService(repo repository.FileRepository, store storage.Storage) *FileService {
	return &FileService{repo: repo, store: store}
}

// Upload stores a policy document or attachment for the admin's company.
// Type is decided by extension and verified against the content.
func (s *FileService) Upload(ctx context.Context, u *model.User, name string, size int64, r io.Reader) (model.File, error) {
	name = strings.TrimSpace(filepath.Base(strings.ReplaceAll(name, `\`, "/")))
	if name == "" || name == "." || len(name) > 200 || !utf8.ValidString(name) {
		return model.File{}, apperror.Invalid("Nama file tidak valid (maks. 200 karakter)")
	}
	if size <= 0 || size > MaxFileBytes {
		return model.File{}, apperror.Invalid("Ukuran file maksimal 10 MB")
	}
	kind, ok := allowedTypes[strings.ToLower(filepath.Ext(name))]
	if !ok {
		return model.File{}, apperror.Invalid("Format yang didukung: PDF, PNG, JPG, WEBP, DOCX, XLSX")
	}
	head := make([]byte, 512)
	n, err := io.ReadFull(r, head)
	if err != nil && err != io.ErrUnexpectedEOF {
		return model.File{}, err
	}
	if http.DetectContentType(head[:n]) != kind.sniff {
		return model.File{}, apperror.Invalid("Isi file tidak sesuai dengan ekstensinya")
	}
	key := make([]byte, 16)
	if _, err = rand.Read(key); err != nil {
		return model.File{}, err
	}
	f := model.File{Name: name, ContentType: kind.contentType, StorageKey: hex.EncodeToString(key)}
	written, err := s.store.Put(ctx, f.StorageKey, io.LimitReader(io.MultiReader(bytes.NewReader(head[:n]), r), MaxFileBytes+1))
	if err != nil {
		return model.File{}, err
	}
	if written > MaxFileBytes {
		_ = s.store.Delete(ctx, f.StorageKey)
		return model.File{}, apperror.Invalid("Ukuran file maksimal 10 MB")
	}
	f.SizeBytes = written
	saved, err := s.repo.CreateFile(ctx, u.CompanyID, u.ID, f)
	if err != nil {
		if derr := s.store.Delete(ctx, f.StorageKey); derr != nil {
			slog.Error("orphaned upload", "key", f.StorageKey, "error", derr)
		}
		return model.File{}, err
	}
	return saved, nil
}

// Open returns a file for download. Employees may only read files attached
// to published announcements or documents.
func (s *FileService) Open(ctx context.Context, u *model.User, id int64) (model.File, io.ReadCloser, error) {
	f, err := s.repo.File(ctx, u.CompanyID, id, u.Role != model.RoleAdmin)
	if err != nil {
		return f, nil, err
	}
	rc, err := s.store.Open(ctx, f.StorageKey)
	return f, rc, err
}
