package cache

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEnsureCachedDownloadsOnMiss(t *testing.T) {
	dir := t.TempDir()
	c, err := newCacheAt(dir)
	if err != nil {
		t.Fatalf("creating cache: %v", err)
	}

	downloaded := false
	mf, err := c.EnsureCached("att1", time.Now(), "test.mp4", "video/mp4", 11, false,
		func(w io.Writer) (int64, error) {
			downloaded = true
			n, err := w.Write([]byte("video-bytes"))
			return int64(n), err
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !downloaded {
		t.Error("expected downloader to be called")
	}
	if mf.OriginalFilename != "test.mp4" {
		t.Errorf("expected filename test.mp4, got %s", mf.OriginalFilename)
	}
	if mf.Extension != ".mp4" {
		t.Errorf("expected extension .mp4, got %s", mf.Extension)
	}
	if mf.SHA256 == "" {
		t.Error("expected non-empty SHA256")
	}

	// File should exist on disk
	if _, err := os.Stat(mf.LocalPath); err != nil {
		t.Errorf("cached file should exist: %v", err)
	}
}

func TestEnsureCachedHit(t *testing.T) {
	dir := t.TempDir()
	c, err := newCacheAt(dir)
	if err != nil {
		t.Fatalf("creating cache: %v", err)
	}

	ts := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
	downloadCount := 0

	downloader := func(w io.Writer) (int64, error) {
		downloadCount++
		n, err := w.Write([]byte("content"))
		return int64(n), err
	}

	// First call — cache miss
	_, err = c.EnsureCached("att2", ts, "photo.jpg", "image/jpeg", 7, false, downloader)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}
	if downloadCount != 1 {
		t.Fatalf("expected 1 download, got %d", downloadCount)
	}

	// Second call — cache hit
	mf, err := c.EnsureCached("att2", ts, "photo.jpg", "image/jpeg", 7, false, downloader)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}
	if downloadCount != 1 {
		t.Errorf("expected no additional download, got %d total", downloadCount)
	}
	if mf.SHA256 == "" {
		t.Error("expected non-empty SHA256 on cache hit")
	}
}

func TestEnsureCachedForceDownload(t *testing.T) {
	dir := t.TempDir()
	c, err := newCacheAt(dir)
	if err != nil {
		t.Fatalf("creating cache: %v", err)
	}

	ts := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
	downloadCount := 0

	downloader := func(w io.Writer) (int64, error) {
		downloadCount++
		n, err := w.Write([]byte("content"))
		return int64(n), err
	}

	// First download
	_, err = c.EnsureCached("att3", ts, "file.pdf", "application/pdf", 7, false, downloader)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	// Force download — should call downloader again
	_, err = c.EnsureCached("att3", ts, "file.pdf", "application/pdf", 7, true, downloader)
	if err != nil {
		t.Fatalf("force download error: %v", err)
	}
	if downloadCount != 2 {
		t.Errorf("expected 2 downloads with force, got %d", downloadCount)
	}
}

func TestEnsureCachedStale(t *testing.T) {
	dir := t.TempDir()
	c, err := newCacheAt(dir)
	if err != nil {
		t.Fatalf("creating cache: %v", err)
	}

	oldTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newTime := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	downloadCount := 0

	downloader := func(w io.Writer) (int64, error) {
		downloadCount++
		n, err := w.Write([]byte("content"))
		return int64(n), err
	}

	// Cache with old timestamp
	_, err = c.EnsureCached("att4", oldTime, "doc.pdf", "application/pdf", 7, false, downloader)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	// Lookup with new timestamp — stale, should re-download
	_, err = c.EnsureCached("att4", newTime, "doc.pdf", "application/pdf", 7, false, downloader)
	if err != nil {
		t.Fatalf("stale call error: %v", err)
	}
	if downloadCount != 2 {
		t.Errorf("expected 2 downloads for stale entry, got %d", downloadCount)
	}
}

func TestClean(t *testing.T) {
	dir := t.TempDir()
	c, err := newCacheAt(dir)
	if err != nil {
		t.Fatalf("creating cache: %v", err)
	}

	// Create a file
	os.WriteFile(filepath.Join(dir, "abc123.mp4"), []byte("data"), 0644)

	if err := c.Clean(); err != nil {
		t.Fatalf("clean error: %v", err)
	}

	entries, _ := os.ReadDir(dir)
	if len(entries) != 0 {
		t.Errorf("expected empty dir after clean, got %d entries", len(entries))
	}
}

func TestInfo(t *testing.T) {
	dir := t.TempDir()
	c, err := newCacheAt(dir)
	if err != nil {
		t.Fatalf("creating cache: %v", err)
	}

	// Write some test files
	os.WriteFile(filepath.Join(dir, "abc.mp4"), []byte("12345"), 0644)
	os.WriteFile(filepath.Join(dir, "def.jpg"), []byte("67890"), 0644)
	os.WriteFile(filepath.Join(dir, "index.json"), []byte("{}"), 0644)

	d, count, size, err := c.Info()
	if err != nil {
		t.Fatalf("info error: %v", err)
	}
	if d != dir {
		t.Errorf("expected dir %s, got %s", dir, d)
	}
	if count != 2 {
		t.Errorf("expected 2 files (excluding index.json), got %d", count)
	}
	if size != 10 {
		t.Errorf("expected total size 10, got %d", size)
	}
}
