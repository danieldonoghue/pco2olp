package convert

import (
	"fmt"
	"strings"

	"github.com/danield/pco2olp/internal/openlp"
	"github.com/danield/pco2olp/internal/pco"
)

// PlanToServiceFile converts PCO plan items into an OpenLP ServiceFile.
func PlanToServiceFile(items []pco.Item) *openlp.ServiceFile {
	sf := &openlp.ServiceFile{}

	for _, item := range items {
		si := itemToServiceItem(item)
		if si != nil {
			sf.Items = append(sf.Items, *si)
		}
	}

	return sf
}

func itemToServiceItem(item pco.Item) *openlp.ServiceItem {
	switch item.ItemType {
	case "song":
		return songToServiceItem(item)
	case "header":
		return headerToServiceItem(item)
	case "media":
		return mediaToServiceItem(item)
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

func mediaToServiceItem(item pco.Item) *openlp.ServiceItem {
	// Without actual media files (Phase 2), render as a custom placeholder
	slides := []openlp.SlideData{{
		Title:    truncate(item.Title, 30),
		RawSlide: item.Title,
		VerseTag: "1",
	}}
	si := openlp.NewCustomItem(item.Title, "[Media placeholder]", slides)
	return &si
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
