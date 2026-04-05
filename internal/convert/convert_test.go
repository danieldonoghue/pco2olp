package convert

import (
	"testing"

	"github.com/danieldonoghue/pco2olp/internal/cache"
	"github.com/danieldonoghue/pco2olp/internal/pco"
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

	sf := PlanToServiceFile(items, nil, nil, nil)

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

	sf := PlanToServiceFile(items, mediaMap, nil, nil)
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
	sf := PlanToServiceFile(items, mediaMap, nil, nil)
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

	sf := PlanToServiceFile(items, nil, nil, nil)
	if len(sf.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(sf.Items))
	}
	if len(sf.Items[0].Data) == 0 {
		t.Error("song with no lyrics should still have at least one slide")
	}
}

func TestCustomItemPrefersHTMLDetails(t *testing.T) {
	items := []pco.Item{
		{
			ID:          "1",
			Title:       "Confession",
			ItemType:    "item",
			Description: "A short description",
			HTMLDetails: "<p>I confess before you, O Holy God</p><p>that I have sinned.</p>",
		},
	}

	sf := PlanToServiceFile(items, nil, nil, nil)
	if len(sf.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(sf.Items))
	}
	slide := sf.Items[0].Data[0].RawSlide
	if slide == "A short description" {
		t.Error("should prefer HTMLDetails over Description")
	}
	if !contains(slide, "I confess before you") {
		t.Errorf("slide should contain detail text, got %q", slide)
	}
}

func TestCustomItemSlidesSplitOnDivider(t *testing.T) {
	items := []pco.Item{
		{
			ID:       "1",
			Title:    "Confession",
			ItemType: "item",
			HTMLDetails: "<p>Slide one text</p>" +
				"[===]" +
				"<p>Slide two text</p>" +
				"[===]" +
				"<p>Slide three text</p>",
		},
	}

	sf := PlanToServiceFile(items, nil, nil, nil)
	if len(sf.Items[0].Data) != 3 {
		t.Fatalf("expected 3 slides, got %d", len(sf.Items[0].Data))
	}
	if !contains(sf.Items[0].Data[0].RawSlide, "Slide one") {
		t.Errorf("slide 1 = %q", sf.Items[0].Data[0].RawSlide)
	}
	if !contains(sf.Items[0].Data[2].RawSlide, "Slide three") {
		t.Errorf("slide 3 = %q", sf.Items[0].Data[2].RawSlide)
	}
}

func TestCustomItemFallsBackToDescription(t *testing.T) {
	items := []pco.Item{
		{
			ID:          "1",
			Title:       "Announcements",
			ItemType:    "item",
			Description: "Remember the potluck",
		},
	}

	sf := PlanToServiceFile(items, nil, nil, nil)
	if sf.Items[0].Data[0].RawSlide != "Remember the potluck" {
		t.Errorf("expected description as body, got %q", sf.Items[0].Data[0].RawSlide)
	}
}

func TestCustomItemFallsBackToTitle(t *testing.T) {
	items := []pco.Item{
		{
			ID:       "1",
			Title:    "Offering",
			ItemType: "item",
		},
	}

	sf := PlanToServiceFile(items, nil, nil, nil)
	if sf.Items[0].Data[0].RawSlide != "Offering" {
		t.Errorf("expected title as fallback, got %q", sf.Items[0].Data[0].RawSlide)
	}
}

func TestHTMLToOpenLP(t *testing.T) {
	input := "<p>Hello <b>world</b></p><br/><p>&amp; goodbye</p>"
	got := htmlToOpenLP(input)
	if !contains(got, "Hello {st}world{/st}") {
		t.Errorf("expected bold converted to OpenLP tags, got %q", got)
	}
	if !contains(got, "& goodbye") {
		t.Errorf("expected '& goodbye', got %q", got)
	}
}

func TestHTMLToOpenLPItalics(t *testing.T) {
	input := "<p>Say <i>Amen</i> and <em>Hallelujah</em></p>"
	got := htmlToOpenLP(input)
	if !contains(got, "{it}Amen{/it}") {
		t.Errorf("expected italic converted, got %q", got)
	}
	if !contains(got, "{it}Hallelujah{/it}") {
		t.Errorf("expected em converted, got %q", got)
	}
}

func TestExternalOverrideForItemMedia(t *testing.T) {
	items := []pco.Item{
		{ID: "1", Title: "Worship Video", ItemType: "media"},
	}
	mediaMap := map[string]*cache.MediaFile{
		"1": {
			SHA256: "abc123", OriginalFilename: "worship.mp4",
			LocalPath: "/tmp/abc123.mp4", Extension: ".mp4",
			ContentType: "video/mp4", PCOMediaType: "video",
		},
	}
	overrides := map[string]string{"abc123": "See worship.mp4 in media folder"}

	sf := PlanToServiceFile(items, mediaMap, nil, overrides)
	if len(sf.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(sf.Items))
	}
	// Should be a custom note, not an embedded media item.
	if sf.Items[0].Header.Plugin != "custom" {
		t.Errorf("expected custom plugin for external note, got %q", sf.Items[0].Header.Plugin)
	}
	if sf.Items[0].Header.Type != 1 {
		t.Errorf("expected type 1 (text), got %d", sf.Items[0].Header.Type)
	}
	if !contains(sf.Items[0].Data[0].RawSlide, "media folder") {
		t.Errorf("expected override note in slide body, got %q", sf.Items[0].Data[0].RawSlide)
	}
	if len(sf.MediaFiles) != 0 {
		t.Error("expected no embedded media files when override is present")
	}
}

func TestExternalOverrideForPlanAttachment(t *testing.T) {
	planMedia := []*cache.MediaFile{
		{
			SHA256: "def456", OriginalFilename: "background.jpg",
			LocalPath: "/tmp/def456.jpg", Extension: ".jpg",
			ContentType: "image/jpeg", PCOMediaType: "background_image",
		},
	}
	overrides := map[string]string{"def456": "See background.jpg in media folder"}

	sf := PlanToServiceFile(nil, nil, planMedia, overrides)
	if len(sf.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(sf.Items))
	}
	if sf.Items[0].Header.Plugin != "custom" {
		t.Errorf("expected custom plugin for external attachment note, got %q", sf.Items[0].Header.Plugin)
	}
	if !contains(sf.Items[0].Header.Title, "background.jpg") {
		t.Errorf("expected attachment filename in title, got %q", sf.Items[0].Header.Title)
	}
	if !contains(sf.Items[0].Data[0].RawSlide, "media folder") {
		t.Errorf("expected override note in slide body, got %q", sf.Items[0].Data[0].RawSlide)
	}
	if len(sf.MediaFiles) != 0 {
		t.Error("expected no embedded media files when override is present")
	}
}


func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
