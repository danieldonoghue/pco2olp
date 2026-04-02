package openlp

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
)

// OpenLyrics XML types following the OpenLyrics 0.9 spec.
type openLyricsSong struct {
	XMLName    xml.Name     `xml:"song"`
	Xmlns      string       `xml:"xmlns,attr"`
	Version    string       `xml:"version,attr"`
	CreatedIn  string       `xml:"createdIn,attr"`
	Properties olProperties `xml:"properties"`
	Lyrics     olLyrics     `xml:"lyrics"`
}

type olProperties struct {
	Titles     olTitles   `xml:"titles"`
	Authors    *olAuthors `xml:"authors,omitempty"`
	Copyright  string     `xml:"copyright,omitempty"`
	CCLINo     string     `xml:"ccliNo,omitempty"`
	VerseOrder string     `xml:"verseOrder,omitempty"`
}

type olTitles struct {
	Title []olTitle `xml:"title"`
}

type olTitle struct {
	Value string `xml:",chardata"`
}

type olAuthors struct {
	Author []olAuthor `xml:"author"`
}

type olAuthor struct {
	Value string `xml:",chardata"`
}

type olLyrics struct {
	Verse []olVerse `xml:"verse"`
}

type olVerse struct {
	Name  string    `xml:"name,attr"`
	Lines []olLines `xml:"lines"`
}

type olLines struct {
	Content string `xml:",innerxml"`
}

// VerseContent represents a parsed verse with its tag and lines.
type VerseContent struct {
	Tag   string   // OpenLyrics tag, e.g. "v1", "c1", "b1"
	Lines []string // text lines of the verse
}

// GenerateOpenLyrics produces a complete OpenLyrics XML string.
func GenerateOpenLyrics(title, author, copyright, ccli string, verses []VerseContent, verseOrder []string) (string, error) {
	song := openLyricsSong{
		Xmlns:     "http://openlyrics.info/namespace/2009/song",
		Version:   "0.9",
		CreatedIn: "pco2olp",
	}

	song.Properties.Titles.Title = []olTitle{{Value: title}}

	if author != "" {
		song.Properties.Authors = &olAuthors{
			Author: []olAuthor{{Value: author}},
		}
	}
	if copyright != "" {
		song.Properties.Copyright = copyright
	}
	if ccli != "" {
		song.Properties.CCLINo = ccli
	}
	if len(verseOrder) > 0 {
		song.Properties.VerseOrder = strings.Join(verseOrder, " ")
	}

	for _, v := range verses {
		// Escape text content and join with <br/>
		var escapedLines []string
		for _, line := range v.Lines {
			var buf bytes.Buffer
			xml.EscapeText(&buf, []byte(line))
			escapedLines = append(escapedLines, buf.String())
		}
		content := strings.Join(escapedLines, "<br/>")

		song.Lyrics.Verse = append(song.Lyrics.Verse, olVerse{
			Name:  v.Tag,
			Lines: []olLines{{Content: content}},
		})
	}

	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(song); err != nil {
		return "", fmt.Errorf("encoding OpenLyrics XML: %w", err)
	}

	// Fix self-closing br tags
	result := strings.ReplaceAll(buf.String(), "<br></br>", "<br/>")
	return result, nil
}

// Section label patterns for PCO lyrics parsing.
var sectionPattern = regexp.MustCompile(`(?i)^(VERSE|CHORUS|BRIDGE|PRE-?CHORUS|TAG|INTRO|OUTRO|ENDING|INTERLUDE|MISC|VAMP|REFRAIN|TURN|INSTRUMENTAL)\s*(\d*)$`)

// ParsePCOLyrics parses Planning Center lyrics text into structured verses.
// PCO format has section labels like "VERSE 1", "CHORUS", etc. followed by lyrics text,
// separated by blank lines.
func ParsePCOLyrics(raw string) (verses []VerseContent, verseOrder []string) {
	if raw == "" {
		return nil, nil
	}

	lines := strings.Split(raw, "\n")

	var currentTag string
	var currentLines []string
	tagCounts := make(map[string]int)

	flush := func() {
		if currentTag == "" || len(currentLines) == 0 {
			return
		}
		// Trim trailing empty lines
		for len(currentLines) > 0 && strings.TrimSpace(currentLines[len(currentLines)-1]) == "" {
			currentLines = currentLines[:len(currentLines)-1]
		}
		if len(currentLines) > 0 {
			verses = append(verses, VerseContent{Tag: currentTag, Lines: currentLines})
			verseOrder = append(verseOrder, currentTag)
		}
		currentLines = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			if currentTag != "" && len(currentLines) > 0 {
				// Blank line within a section — could be separator or end of section
				// We'll accumulate and decide at next label
			}
			continue
		}

		if match := sectionPattern.FindStringSubmatch(trimmed); match != nil {
			// Found a section label - flush previous
			flush()

			label := strings.ToUpper(match[1])
			num := match[2]

			tag := mapSectionToTag(label, num, tagCounts)
			currentTag = tag
			continue
		}

		if currentTag == "" {
			// Text before any section label — treat as verse 1
			currentTag = "v1"
			tagCounts["v"] = 1
		}

		currentLines = append(currentLines, trimmed)
	}

	flush()
	return verses, verseOrder
}

func mapSectionToTag(label, num string, counts map[string]int) string {
	var prefix string
	switch label {
	case "VERSE":
		prefix = "v"
	case "CHORUS", "REFRAIN":
		prefix = "c"
	case "BRIDGE":
		prefix = "b"
	case "PRECHORUS", "PRE-CHORUS":
		prefix = "p"
	case "INTRO":
		prefix = "i"
	case "OUTRO", "ENDING":
		prefix = "e"
	default:
		prefix = "o"
	}

	if num != "" {
		tag := prefix + num
		n := 0
		fmt.Sscanf(num, "%d", &n)
		if n > counts[prefix] {
			counts[prefix] = n
		}
		return tag
	}

	counts[prefix]++
	return fmt.Sprintf("%s%d", prefix, counts[prefix])
}

// VerseTagToUpper converts an OpenLyrics verse tag to the uppercase format used in OpenLP slide data.
// e.g., "v1" -> "V1", "c1" -> "C1"
func VerseTagToUpper(tag string) string {
	return strings.ToUpper(tag)
}
