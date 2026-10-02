package model

import "time"

type File struct {
	ID          int64     `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	ContentType string    `db:"content_type" json:"content_type"`
	SizeBytes   int64     `db:"size_bytes" json:"size_bytes"`
	StorageKey  string    `db:"storage_key" json:"-"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}
