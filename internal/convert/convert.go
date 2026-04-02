package convert

import (
	"fmt"
	"strings"

	"github.com/danield/pco2olp/internal/cache"
	"github.com/danield/pco2olp/internal/openlp"
	"github.com/danield/pco2olp/internal/pco"
)

// PlanToServiceFile converts PCO plan items into an OpenLP ServiceFile.
// itemMedia maps PCO item IDs to downloaded media files.
// planMedia contains plan-level attachments to append at the end.
func PlanToServiceFile(items []pco.Item, itemMedia map[string]*cache.MediaFile, planMedia []*cache.MediaFile) *openlp.ServiceFile {
	sf := &openlp.ServiceFile{}

	for _, item := range items {
		var mf *cache.MediaFile
		if itemMedia != nil {
			mf = itemMedia[item.ID]
		}
		si := itemToServiceItem(item, mf)
		if si != nil {
			sf.Items = append(sf.Items, *si)
			if mf != nil {
				storedName := mf.SHA256 + mf.Extension
				sf.MediaFiles = append(sf.MediaFiles, openlp.EmbeddedFile{
					StoredName: storedName,
					LocalPath:  mf.LocalPath,
				})
			}
		}
	}

	// Append plan-level attachments as service items
	for _, mf := range planMedia {
		si := attachmentToServiceItem(mf)
		if si != nil {
			sf.Items = append(sf.Items, *si)
			storedName := mf.SHA256 + mf.Extension
			sf.MediaFiles = append(sf.MediaFiles, openlp.EmbeddedFile{
				StoredName: storedName,
				LocalPath:  mf.LocalPath,
			})
		}
	}

	return sf
}

func itemToServiceItem(item pco.Item, mf *cache.MediaFile) *openlp.ServiceItem {
	switch item.ItemType {
	case "song":
		return songToServiceItem(item)
	case "header":
		return headerToServiceItem(item)
	case "media":
		return mediaToServiceItem(item, mf)
	default:
		// "item" and other types -> custom slide
		return customToServiceItem(item)
	}
}

func songToServiceItem(item pco.Item) *openlp.ServiceItem {
	title := item.Title
	author := ""
	copyright := ""
	ccli := ""

	if item.Song != nil {
		title = item.Song.Title
		author = item.Song.Author
		copyright = item.Song.Copyright
		ccli = item.Song.CCLI
	}

	var lyricsText string
	var sequence []string
	if item.Arrangement != nil {
		lyricsText = item.Arrangement.Lyrics
		sequence = item.Arrangement.Sequence
	}

	// Parse lyrics into verses
	verses, verseOrder := openlp.ParsePCOLyrics(lyricsText)

	// If arrangement has a sequence, use it for verse order
	if len(sequence) > 0 {
		verseOrder = mapPCOSequence(sequence)
	}

	// Generate OpenLyrics XML
	xmlStr := ""
	if len(verses) > 0 {
		var err error
		xmlStr, err = openlp.GenerateOpenLyrics(title, author, copyright, ccli, verses, verseOrder)
		if err != nil {
			// Fall back to empty XML on error
			xmlStr = ""
		}
	}

	// Build slides
	var slides []openlp.SlideData
	for _, v := range verses {
		text := strings.Join(v.Lines, "\n")
		slideTitle := text
		if len(slideTitle) > 30 {
			slideTitle = slideTitle[:30]
		}
		slides = append(slides, openlp.SlideData{
			Title:    slideTitle,
			RawSlide: text,
			VerseTag: openlp.VerseTagToUpper(v.Tag),
		})
	}

	// If no lyrics at all, create a single blank slide with the title
	if len(slides) == 0 {
		slides = []openlp.SlideData{{
			Title:    title,
			RawSlide: title,
			VerseTag: "V1",
		}}
	}

	// Collect notes
	notes := collectNotes(item)

	si := openlp.NewSongItem(title, author, copyright, ccli, xmlStr, slides)
	si.Header.Notes = notes
	return &si
}

func headerToServiceItem(item pco.Item) *openlp.ServiceItem {
	slides := []openlp.SlideData{{
		Title:    item.Title,
		RawSlide: item.Title,
		VerseTag: "1",
	}}

	si := openlp.NewCustomItem(item.Title, "", slides)
	return &si
}

func customToServiceItem(item pco.Item) *openlp.ServiceItem {
	content := item.Title
	if item.Description != "" {
		content = item.Description
	}

	slides := []openlp.SlideData{{
		Title:    truncate(content, 30),
		RawSlide: content,
		VerseTag: "1",
	}}

	notes := collectNotes(item)
	si := openlp.NewCustomItem(item.Title, notes, slides)
	return &si
}

func mediaToServiceItem(item pco.Item, mf *cache.MediaFile) *openlp.ServiceItem {
	if mf == nil {
		// No media file downloaded — render as custom placeholder
		slides := []openlp.SlideData{{
			Title:    truncate(item.Title, 30),
			RawSlide: item.Title,
			VerseTag: "1",
		}}
		si := openlp.NewCustomItem(item.Title, "[Media placeholder]", slides)
		return &si
	}

	storedName := mf.SHA256 + mf.Extension
	hash := mf.SHA256
	return newMediaServiceItem(item.Title, mf.PCOMediaType, mf.ContentType, &hash, &storedName)
}

// attachmentToServiceItem creates a service item for a plan-level attachment.
func attachmentToServiceItem(mf *cache.MediaFile) *openlp.ServiceItem {
	title := mf.OriginalFilename
	storedName := mf.SHA256 + mf.Extension
	hash := mf.SHA256
	return newMediaServiceItem(title, mf.PCOMediaType, mf.ContentType, &hash, &storedName)
}

// newMediaServiceItem creates the correct OpenLP service item based on media type.
func newMediaServiceItem(title, pcoMediaType, contentType string, hash, storedName *string) *openlp.ServiceItem {
	// Determine OpenLP item type from PCO media_type or content type
	switch {
	case isImageType(pcoMediaType, contentType):
		si := openlp.NewImageItem(title, hash, storedName)
		return &si
	case isPresentationType(pcoMediaType, contentType):
		si := openlp.NewPresentationItem(title, hash, storedName)
		return &si
	default:
		si := openlp.NewMediaItem(title, hash, storedName)
		return &si
	}
}

func isImageType(pcoMediaType, contentType string) bool {
	switch pcoMediaType {
	case "image", "background_image":
		return true
	}
	return strings.HasPrefix(contentType, "image/")
}

func isPresentationType(pcoMediaType, contentType string) bool {
	switch pcoMediaType {
	case "powerpoint", "document", "curriculum":
		return true
	}
	switch {
	case strings.Contains(contentType, "presentation"),
		strings.Contains(contentType, "powerpoint"),
		contentType == "application/pdf":
		return true
	}
	return false
}

func collectNotes(item pco.Item) string {
	if len(item.Notes) == 0 {
		return ""
	}
	var parts []string
	for _, n := range item.Notes {
		if n.Content != "" {
			if n.Category != "" {
				parts = append(parts, fmt.Sprintf("[%s] %s", n.Category, n.Content))
			} else {
				parts = append(parts, n.Content)
			}
		}
	}
	return strings.Join(parts, "\n")
}

// mapPCOSequence converts PCO arrangement sequence labels to OpenLyrics tags.
func mapPCOSequence(seq []string) []string {
	var result []string
	counts := make(map[string]int)

	for _, label := range seq {
		upper := strings.ToUpper(strings.TrimSpace(label))
		var prefix string
		switch {
		case strings.HasPrefix(upper, "VERSE"), upper == "V":
			prefix = "v"
		case strings.HasPrefix(upper, "CHORUS"), upper == "C", strings.HasPrefix(upper, "REFRAIN"):
			prefix = "c"
		case strings.HasPrefix(upper, "BRIDGE"), upper == "B":
			prefix = "b"
		case strings.HasPrefix(upper, "PRE"), upper == "P":
			prefix = "p"
		case strings.HasPrefix(upper, "INTRO"):
			prefix = "i"
		case strings.HasPrefix(upper, "OUTRO"), strings.HasPrefix(upper, "END"):
			prefix = "e"
		default:
			prefix = "o"
		}

		// Extract number if present
		num := extractNumber(upper)
		if num == "" {
			counts[prefix]++
			num = fmt.Sprintf("%d", counts[prefix])
		}
		result = append(result, prefix+num)
	}

	return result
}

func extractNumber(s string) string {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] >= '0' && s[i] <= '9' {
			start := i
			for start > 0 && s[start-1] >= '0' && s[start-1] <= '9' {
				start--
			}
			return s[start : i+1]
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
