package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
)

func roundTrip(t *testing.T, s Storage) {
	t.Helper()
	ctx := context.Background()
	key := strings.Repeat("ab", 16)
	if _, err := s.Put(ctx, "../etc/passwd", bytes.NewReader(nil)); err == nil {
		t.Fatal("unsafe key accepted")
	}
	n, err := s.Put(ctx, key, strings.NewReader("hello"))
	if err != nil || n != 5 {
		t.Fatalf("put: %d %v", n, err)
	}
	rc, err := s.Open(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(rc)
	_ = rc.Close()
	if string(b) != "hello" {
		t.Fatalf("read %q", b)
	}
	if err = s.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Open(ctx, key); err == nil {
		t.Fatal("deleted object still readable")
	}
}

func TestLocal(t *testing.T) { roundTrip(t, Local{Dir: t.TempDir()}) }

// TestS3 runs against an S3-compatible server, e.g.
// S3_TEST_ENDPOINT=127.0.0.1:9100 S3_TEST_BUCKET=hris with MinIO.
func TestS3(t *testing.T) {
	endpoint := os.Getenv("S3_TEST_ENDPOINT")
	if endpoint == "" {
		t.Skip("set S3_TEST_ENDPOINT (and S3_TEST_BUCKET/KEY/SECRET) to test S3 storage")
	}
	s, err := NewS3(S3Config{
		Endpoint: endpoint, Bucket: os.Getenv("S3_TEST_BUCKET"),
		AccessKeyID: os.Getenv("S3_TEST_KEY"), SecretAccessKey: os.Getenv("S3_TEST_SECRET"), Region: "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	roundTrip(t, s)
}
