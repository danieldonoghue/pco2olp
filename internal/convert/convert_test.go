package convert

import (
	"testing"

	"github.com/danield/pco2olp/internal/cache"
	"github.com/danield/pco2olp/internal/pco"
)

func TestPlanToServiceFile(t *testing.T) {
	items := []pco.Item{
		{
			ID:       "1",
			Title:    "Pre-Service",
			ItemType: "header",
			Sequence: 0,
		},
		{
			ID:       "2",
			Title:    "Amazing Grace",
			ItemType: "song",
			Sequence: 1,
			Song: &pco.Song{
				ID:        "100",
				Title:     "Amazing Grace",
				Author:    "John Newton",
				CCLI:      "12345",
				Copyright: "Public Domain",
			},
			Arrangement: &pco.Arrangement{
				ID:   "200",
				Name: "Default",
				Lyrics: `VERSE 1
Amazing grace how sweet the sound
That saved a wretch like me

CHORUS
How sweet the sound`,
			},
		},
		{
			ID:       "3",
			Title:    "Announcements",
			ItemType: "item",
			Sequence: 2,
			Notes: []pco.ItemNote{
				{Content: "Remember the potluck", Category: "Notes"},
			},
		},
		{
			ID:       "4",
			Title:    "sermon.mp4",
			ItemType: "media",
			Sequence: 3,
		},
	}

	sf := PlanToServiceFile(items, nil, nil)

	if len(sf.Items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(sf.Items))
	}

	// Check header -> custom
	if sf.Items[0].Header.Plugin != "custom" {
		t.Errorf("header item plugin = %q, want 'custom'", sf.Items[0].Header.Plugin)
	}

	// Check song
	if sf.Items[1].Header.Plugin != "songs" {
		t.Errorf("song item plugin = %q, want 'songs'", sf.Items[1].Header.Plugin)
	}
	if sf.Items[1].Header.XMLVersion == nil || *sf.Items[1].Header.XMLVersion == "" {
		t.Error("song should have XMLVersion")
	}
	if len(sf.Items[1].Data) != 2 {
		t.Errorf("song should have 2 slides (verse + chorus), got %d", len(sf.Items[1].Data))
	}

	// Check custom/item
	if sf.Items[2].Header.Plugin != "custom" {
		t.Errorf("custom item plugin = %q, want 'custom'", sf.Items[2].Header.Plugin)
	}
	if sf.Items[2].Header.Notes == "" {
		t.Error("custom item should have notes")
	}

	// Check media (rendered as custom placeholder when no media file downloaded)
	if sf.Items[3].Header.Plugin != "custom" {
		t.Errorf("media item plugin = %q, want 'custom'", sf.Items[3].Header.Plugin)
	}
	if sf.Items[3].Header.Type != 1 {
		t.Errorf("media item type = %d, want 1", sf.Items[3].Header.Type)
	}
}

func TestMediaWithDownloadedFile(t *testing.T) {
	items := []pco.Item{
		{
			ID:       "10",
			Title:    "Worship Video",
			ItemType: "media",
		},
	}

	mediaMap := map[string]*cache.MediaFile{
		"10": {
			OriginalFilename: "worship.mp4",
			LocalPath:        "/tmp/abc123.mp4",
			SHA256:           "abc123",
			Extension:        ".mp4",
			ContentType:      "video/mp4",
			PCOMediaType:     "video",
		},
	}

	sf := PlanToServiceFile(items, mediaMap, nil)
	if len(sf.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(sf.Items))
	}
	if sf.Items[0].Header.Plugin != "media" {
		t.Errorf("media item plugin = %q, want 'media'", sf.Items[0].Header.Plugin)
	}
	if sf.Items[0].Header.Type != 3 {
		t.Errorf("media item type = %d, want 3", sf.Items[0].Header.Type)
	}
	if sf.Items[0].Header.SHA256FileHash == nil || *sf.Items[0].Header.SHA256FileHash != "abc123" {
		t.Error("expected sha256 hash abc123")
	}
	if len(sf.MediaFiles) != 1 {
		t.Fatalf("expected 1 media file, got %d", len(sf.MediaFiles))
	}
}

func TestImageMediaType(t *testing.T) {
	items := []pco.Item{
		{ID: "20", Title: "Background", ItemType: "media"},
	}
	mediaMap := map[string]*cache.MediaFile{
		"20": {
			SHA256: "def456", Extension: ".jpg", ContentType: "image/jpeg",
			PCOMediaType: "background_image", LocalPath: "/tmp/def456.jpg",
		},
	}
	sf := PlanToServiceFile(items, mediaMap, nil)
	if sf.Items[0].Header.Plugin != "images" {
		t.Errorf("expected images plugin, got %s", sf.Items[0].Header.Plugin)
	}
	if sf.Items[0].Header.Type != 2 {
		t.Errorf("expected type 2 (image), got %d", sf.Items[0].Header.Type)
	}
}

func TestSongWithNoLyrics(t *testing.T) {
	items := []pco.Item{
		{
			ID:       "1",
			Title:    "Unknown Song",
			ItemType: "song",
			Song:     &pco.Song{Title: "Unknown Song"},
		},
	}

	sf := PlanToServiceFile(items, nil, nil)
	if len(sf.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(sf.Items))
	}
	if len(sf.Items[0].Data) == 0 {
		t.Error("song with no lyrics should still have at least one slide")
	}
}
