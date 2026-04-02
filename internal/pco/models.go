package pco

import "time"

// ServiceType represents a PCO service type.
type ServiceType struct {
	ID   string
	Name string
}

// Plan represents a PCO plan (a service for a specific date).
type Plan struct {
	ID          string
	Title       string
	SortDate    time.Time
	Dates       string
	SeriesTitle string
}

// Item represents a single item in a plan's order of service.
type Item struct {
	ID              string
	Title           string
	ItemType        string // "song", "header", "media", "item"
	Sequence        int
	Description     string
	HTMLDetails     string
	Length          int
	ServicePosition string
	Song            *Song
	Arrangement     *Arrangement
	Notes           []ItemNote
}

// Song represents a PCO song.
type Song struct {
	ID        string
	Title     string
	Author    string
	CCLI      string
	Copyright string
}

// Arrangement represents a song arrangement with lyrics.
type Arrangement struct {
	ID            string
	Name          string
	Lyrics        string
	ChordChart    string
	Length        int
	Sequence      []string
	SequenceShort []string
}

// ItemNote represents a note attached to a plan item.
type ItemNote struct {
	ID       string
	Category string
	Content  string
}

// ListPlansOpts configures plan listing.
type ListPlansOpts struct {
	AfterDate  *time.Time
	BeforeDate *time.Time
}
