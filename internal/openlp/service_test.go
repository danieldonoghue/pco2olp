package openlp

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteOSZ(t *testing.T) {
	sf := &ServiceFile{
		Items: []ServiceItem{
			NewSongItem("Test Song", "Test Author", "(c) 2024", "99999",
				"<xml>test</xml>",
				[]SlideData{
					{Title: "First verse", RawSlide: "First verse\nSecond line", VerseTag: "V1"},
					{Title: "Chorus", RawSlide: "Chorus line", VerseTag: "C1"},
				}),
			NewCustomItem("Announcements", "Some notes", []SlideData{
				{Title: "Announcements", RawSlide: "Welcome everyone!", VerseTag: "1"},
			}),
			NewMediaItem("video.mp4"),
		},
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.osz")

	if err := sf.WriteOSZ(path); err != nil {
		t.Fatalf("WriteOSZ: %v", err)
	}

	// Verify it's a valid ZIP
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("opening zip: %v", err)
	}
	defer r.Close()

	// Should contain service_data.osj
	if len(r.File) != 1 {
		t.Fatalf("expected 1 file in zip, got %d", len(r.File))
	}
	if r.File[0].Name != "service_data.osj" {
		t.Fatalf("expected service_data.osj, got %s", r.File[0].Name)
	}

	// Read and parse the JSON
	rc, err := r.File[0].Open()
	if err != nil {
		t.Fatalf("opening osj: %v", err)
	}
	defer rc.Close()

	var entries []json.RawMessage
	dec := json.NewDecoder(rc)
	if err := dec.Decode(&entries); err != nil {
		t.Fatalf("parsing osj: %v", err)
	}

	// First entry should be openlp_core
	if len(entries) != 4 { // 1 header + 3 items
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	// Verify first entry has openlp_core
	var header map[string]json.RawMessage
	if err := json.Unmarshal(entries[0], &header); err != nil {
		t.Fatalf("parsing header: %v", err)
	}
	if _, ok := header["openlp_core"]; !ok {
		t.Fatal("first entry missing openlp_core")
	}

	// Verify second entry has serviceitem
	var item map[string]json.RawMessage
	if err := json.Unmarshal(entries[1], &item); err != nil {
		t.Fatalf("parsing item: %v", err)
	}
	if _, ok := item["serviceitem"]; !ok {
		t.Fatal("second entry missing serviceitem")
	}

	// Verify file size
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if fi.Size() == 0 {
		t.Fatal("output file is empty")
	}
}
