package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadIndexNonExistent(t *testing.T) {
	idx, err := LoadIndex(filepath.Join(t.TempDir(), "index.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx.Version != 1 {
		t.Errorf("expected version 1, got %d", idx.Version)
	}
	if len(idx.Entries) != 0 {
		t.Errorf("expected empty entries, got %d", len(idx.Entries))
	}
}

func TestSaveAndLoadRoundtrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.json")
	ts := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)

	idx := &Index{
		Version: 1,
		Entries: map[string]*IndexEntry{
			"att_123": {
				AttachmentID: "123",
				PCOUpdatedAt: ts,
				Filename:     "video.mp4",
				ContentType:  "video/mp4",
				FileSize:     1024,
				SHA256:       "abc123",
				DownloadedAt: ts,
			},
		},
	}

	if err := idx.Save(path); err != nil {
		t.Fatalf("save error: %v", err)
	}

	loaded, err := LoadIndex(path)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	if len(loaded.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(loaded.Entries))
	}

	entry := loaded.Entries["att_123"]
	if entry == nil {
		t.Fatal("expected entry att_123")
	}
	if entry.Filename != "video.mp4" {
		t.Errorf("expected filename video.mp4, got %s", entry.Filename)
	}
	if entry.SHA256 != "abc123" {
		t.Errorf("expected sha256 abc123, got %s", entry.SHA256)
	}
}

func TestLookupHit(t *testing.T) {
	ts := time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC)
	idx := &Index{
		Entries: map[string]*IndexEntry{
			"att_456": {
				AttachmentID: "456",
				PCOUpdatedAt: ts,
				SHA256:       "def789",
			},
		},
	}

	entry := idx.Lookup("456", ts)
	if entry == nil {
		t.Fatal("expected cache hit")
	}
	if entry.SHA256 != "def789" {
		t.Errorf("expected sha256 def789, got %s", entry.SHA256)
	}
}

func TestLookupMiss(t *testing.T) {
	idx := &Index{Entries: make(map[string]*IndexEntry)}

	if idx.Lookup("999", time.Now()) != nil {
		t.Error("expected cache miss for nonexistent ID")
	}
}

func TestLookupStale(t *testing.T) {
	oldTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newTime := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	idx := &Index{
		Entries: map[string]*IndexEntry{
			"att_789": {
				AttachmentID: "789",
				PCOUpdatedAt: oldTime,
			},
		},
	}

	if idx.Lookup("789", newTime) != nil {
		t.Error("expected cache miss for stale entry")
	}
}

func TestSaveAtomicity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "index.json")

	idx := &Index{Version: 1, Entries: make(map[string]*IndexEntry)}
	if err := idx.Save(path); err != nil {
		t.Fatalf("save error: %v", err)
	}

	// Temp file should not linger
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("temp file should not exist after save")
	}

	// Actual file should exist
	if _, err := os.Stat(path); err != nil {
		t.Errorf("index file should exist: %v", err)
	}
}
