package convert

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
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
			sf.MediaFiles = append(sf.MediaFiles, embeddedFiles(mf)...)
		}
	}

	// Append plan-level attachments as service items
	for _, mf := range planMedia {
		si := attachmentToServiceItem(mf)
		if si != nil {
			sf.Items = append(sf.Items, *si)
			sf.MediaFiles = append(sf.MediaFiles, embeddedFiles(mf)...)
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

	si := openlp.NewCustomItem(item.Title, "", "", slides)
	return &si
}

func customToServiceItem(item pco.Item) *openlp.ServiceItem {
	// Prefer HTMLDetails (the "detail" section in PCO) over Description.
	// HTMLDetails contains the actual slide content; Description is a short summary.
	content := ""
	if item.HTMLDetails != "" {
		content = htmlToOpenLP(item.HTMLDetails)
	} else if item.Description != "" {
		content = item.Description
	}

	// Split on [===] for multi-slide items
	var slides []openlp.SlideData
	if content != "" {
		pages := splitSlides(content)
		for i, page := range pages {
			page = strings.TrimSpace(page)
			if page == "" {
				continue
			}
			slides = append(slides, openlp.SlideData{
				Title:    truncate(page, 30),
				RawSlide: page,
				VerseTag: fmt.Sprintf("%d", i+1),
			})
		}
	}

	// Fallback: single slide with the title
	if len(slides) == 0 {
		slides = []openlp.SlideData{{
			Title:    item.Title,
			RawSlide: item.Title,
			VerseTag: "1",
		}}
	}

	notes := collectNotes(item)
	credits := contentFingerprint(slides)
	si := openlp.NewCustomItem(item.Title, credits, notes, slides)
	return &si
}

// splitSlides splits text on [===] slide dividers (OpenLP convention).
// The divider is expected on its own line, but we handle it with or without surrounding newlines.
func splitSlides(text string) []string {
	// OpenLP uses \n[===]\n as the canonical divider
	if strings.Contains(text, "\n[===]\n") {
		return strings.Split(text, "\n[===]\n")
	}
	// Fall back to splitting on [===] with optional surrounding whitespace
	if strings.Contains(text, "[===]") {
		return strings.Split(text, "[===]")
	}
	return []string{text}
}

// htmlToOpenLP converts PCO HTML details to OpenLP formatted text.
// Converts HTML formatting to OpenLP display tags where possible,
// and strips remaining HTML.
func htmlToOpenLP(html string) string {
	// Convert HTML formatting to OpenLP display tags
	// Bold: <b>, <strong> → {st}{/st}
	for _, tag := range []string{"b", "strong"} {
		html = replaceHTMLTag(html, tag, "{st}", "{/st}")
	}
	// Italic: <i>, <em> → {it}{/it}
	for _, tag := range []string{"i", "em"} {
		html = replaceHTMLTag(html, tag, "{it}", "{/it}")
	}
	// Underline: <u> → {u}{/u}
	html = replaceHTMLTag(html, "u", "{u}", "{/u}")

	// Replace <br>, <br/>, <br /> with newlines
	for _, br := range []string{"<br>", "<br/>", "<br />", "<BR>", "<BR/>", "<BR />"} {
		html = strings.ReplaceAll(html, br, "\n")
	}
	// Replace <p> and </p> with newlines
	html = replaceTagInsensitive(html, "<p>", "")
	html = replaceTagInsensitive(html, "</p>", "\n")

	// Strip remaining HTML tags
	var result strings.Builder
	inTag := false
	for _, r := range html {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}

	// Decode common HTML entities
	s := result.String()
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")

	// Clean up excessive blank lines
	lines := strings.Split(s, "\n")
	var cleaned []string
	blankCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			blankCount++
			if blankCount <= 1 {
				cleaned = append(cleaned, "")
			}
		} else {
			blankCount = 0
			cleaned = append(cleaned, trimmed)
		}
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

// replaceHTMLTag replaces <tag>...</tag> with OpenLP formatting tags (case-insensitive).
func replaceHTMLTag(html, tag, openTag, closeTag string) string {
	// Handle both cases
	for _, t := range []string{strings.ToLower(tag), strings.ToUpper(tag)} {
		html = strings.ReplaceAll(html, "<"+t+">", openTag)
		html = strings.ReplaceAll(html, "</"+t+">", closeTag)
	}
	return html
}

// replaceTagInsensitive replaces an HTML tag string case-insensitively.
func replaceTagInsensitive(html, tag, replacement string) string {
	html = strings.ReplaceAll(html, tag, replacement)
	html = strings.ReplaceAll(html, strings.ToUpper(tag), replacement)
	return html
}

func mediaToServiceItem(item pco.Item, mf *cache.MediaFile) *openlp.ServiceItem {
	if mf == nil {
		// No media file downloaded — render as custom placeholder
		slides := []openlp.SlideData{{
			Title:    truncate(item.Title, 30),
			RawSlide: item.Title,
			VerseTag: "1",
		}}
		si := openlp.NewCustomItem(item.Title, "", "[Media placeholder]", slides)
		return &si
	}

	storedName := mf.SHA256 + mf.Extension
	hash := mf.SHA256

	if len(mf.Slides) > 0 {
		si := slidesToImageGroup(item.Title, mf.Slides)
		return &si
	}
	return newMediaServiceItem(item.Title, mf.PCOMediaType, mf.ContentType, &hash, &storedName)
}

// attachmentToServiceItem creates a service item for a plan-level attachment.
func attachmentToServiceItem(mf *cache.MediaFile) *openlp.ServiceItem {
	title := "ATTACHMENT - " + mf.OriginalFilename
	storedName := mf.SHA256 + mf.Extension
	hash := mf.SHA256

	if len(mf.Slides) > 0 {
		si := slidesToImageGroup(title, mf.Slides)
		return &si
	}
	return newMediaServiceItem(title, mf.PCOMediaType, mf.ContentType, &hash, &storedName)
}

// slidesToImageGroup creates an image group service item from converted slide PNGs.
func slidesToImageGroup(title string, slides []cache.Slide) openlp.ServiceItem {
	refs := make([]openlp.SlideRef, len(slides))
	for i, s := range slides {
		refs[i] = openlp.SlideRef{
			SHA256:     s.SHA256,
			StoredName: s.SHA256 + ".png",
		}
	}
	return openlp.NewImageGroupItem(title, refs)
}

// embeddedFiles returns the EmbeddedFile entries for a MediaFile.
// If slides are present, returns one entry per slide PNG; otherwise one entry for the source file.
func embeddedFiles(mf *cache.MediaFile) []openlp.EmbeddedFile {
	if mf == nil {
		return nil
	}
	if len(mf.Slides) > 0 {
		files := make([]openlp.EmbeddedFile, len(mf.Slides))
		for i, s := range mf.Slides {
			files[i] = openlp.EmbeddedFile{
				StoredName: s.SHA256 + ".png",
				LocalPath:  s.LocalPath,
			}
		}
		return files
	}
	return []openlp.EmbeddedFile{{
		StoredName: mf.SHA256 + mf.Extension,
		LocalPath:  mf.LocalPath,
	}}
}

// newMediaServiceItem creates the correct OpenLP service item based on media type.
// PDFs use the presentations plugin (OpenLP renders them internally — no external app needed).
// Keynote and PowerPoint use the media plugin: OpenLP's native controllers have AppleScript
// bugs and macOS App Store sandbox restrictions that prevent loading files from the service
// archive directory.
func newMediaServiceItem(title, pcoMediaType, contentType string, hash, storedName *string) *openlp.ServiceItem {
	switch {
	case isImageType(pcoMediaType, contentType):
		si := openlp.NewImageItem(title, hash, storedName)
		return &si
	case isPDFType(contentType, storedName):
		si := openlp.NewPresentationItem(title, hash, storedName, "Pdf")
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

func isPDFType(contentType string, storedName *string) bool {
	if contentType == "application/pdf" {
		return true
	}
	if storedName != nil && strings.ToLower(filepath.Ext(*storedName)) == ".pdf" {
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

// contentFingerprint returns a short hash of the slide content.
// Used as credits in OpenLP to distinguish items with the same title but different content.
func contentFingerprint(slides []openlp.SlideData) string {
	h := sha256.New()
	for _, s := range slides {
		h.Write([]byte(s.RawSlide))
	}
	return "#" + hex.EncodeToString(h.Sum(nil))[:6]
}
