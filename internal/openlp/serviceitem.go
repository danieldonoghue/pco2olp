package openlp

import "encoding/json"

// ItemType constants for OpenLP service items.
const (
	TypeText    = 1
	TypeImage   = 2
	TypeCommand = 3
)

// Item capability constants from OpenLP source.
const (
	CapCanPreview          = 1
	CapCanEdit             = 2
	CapCanMaintain         = 3
	CapRequiresMedia       = 4
	CapCanLoop             = 5
	CapCanAppend           = 6
	CapOnLoadUpdate        = 8
	CapAddIfNewItem        = 9
	CapProvidesOwnDisplay  = 10
	CapCanSoftBreak        = 13
	CapCanWordSplit        = 14
	CapCanAutoStartForLive = 16
	CapCanEditTitle        = 17
	CapHasDisplayTitle     = 19
	CapHasNotes            = 20
	CapHasThumbnails       = 21
	CapHasMetaData         = 22
	CapProvidesOwnTheme    = 26
)

// Common capability sets.
var (
	SongCapabilities   = []int{CapCanEdit, CapCanPreview, CapCanLoop, CapOnLoadUpdate, CapAddIfNewItem, CapCanSoftBreak, CapHasMetaData}
	CustomCapabilities = []int{CapCanEdit, CapCanPreview, CapCanLoop, CapCanSoftBreak, CapOnLoadUpdate, CapCanWordSplit}
	MediaCapabilities  = []int{CapCanAutoStartForLive, CapCanEditTitle, CapRequiresMedia}
)

// ServiceItem represents a single item in an OpenLP service.
type ServiceItem struct {
	Header ItemHeader  `json:"header"`
	Data   []SlideData `json:"data"`
}

// ItemHeader contains all metadata for a service item.
type ItemHeader struct {
	Name               string      `json:"name"`
	Plugin             string      `json:"plugin"`
	Theme              interface{} `json:"theme"`
	Title              string      `json:"title"`
	Footer             []string    `json:"footer"`
	Type               int         `json:"type"`
	Audit              string      `json:"audit"`
	Notes              string      `json:"notes"`
	FromPlugin         bool        `json:"from_plugin"`
	Capabilities       []int       `json:"capabilities"`
	Search             string      `json:"search"`
	Data               interface{} `json:"data"`
	XMLVersion         *string     `json:"xml_version"`
	AutoPlaySlidesOnce bool        `json:"auto_play_slides_once"`
	AutoPlaySlidesLoop bool        `json:"auto_play_slides_loop"`
	TimedSlideInterval int         `json:"timed_slide_interval"`
	StartTime          int         `json:"start_time"`
	EndTime            int         `json:"end_time"`
	MediaLength        int         `json:"media_length"`
	BackgroundAudio    []string    `json:"background_audio"`
	ThemeOverwritten   bool        `json:"theme_overwritten"`
	WillAutoStart      bool        `json:"will_auto_start"`
	Processor          *string     `json:"processor"`
	Metadata           []any       `json:"metadata"`
	SHA256FileHash     *string     `json:"sha256_file_hash"`
	StoredFilename     *string     `json:"stored_filename"`
}

// SlideData represents a single slide within a service item.
type SlideData struct {
	Title    string `json:"title"`
	RawSlide string `json:"raw_slide"`
	VerseTag string `json:"verseTag"`
}

// SongHeaderData is the plugin-specific data for a song item header.
type SongHeaderData struct {
	Title          string `json:"title"`
	AlternateTitle string `json:"alternate_title"`
	Authors        string `json:"authors"`
	CCLINumber     string `json:"ccli_number"`
	Copyright      string `json:"copyright"`
}

// NewSongItem creates a song service item.
func NewSongItem(title, author, copyright, ccli, xmlVersion string, slides []SlideData) ServiceItem {
	footer := []string{title}
	if author != "" {
		footer = append(footer, "Written by: "+author)
	}
	if copyright != "" {
		footer = append(footer, copyright)
	}

	search := title
	if author != "" {
		search += "@" + author
	}

	return ServiceItem{
		Header: ItemHeader{
			Name:         "songs",
			Plugin:       "songs",
			Theme:        nil,
			Title:        title,
			Footer:       footer,
			Type:         TypeText,
			Audit:        "",
			Notes:        "",
			FromPlugin:   false,
			Capabilities: SongCapabilities,
			Search:       search,
			Data: SongHeaderData{
				Title:      title,
				Authors:    author,
				CCLINumber: ccli,
				Copyright:  copyright,
			},
			XMLVersion:      &xmlVersion,
			BackgroundAudio: []string{},
			Metadata:        []any{},
		},
		Data: slides,
	}
}

// NewCustomItem creates a custom slide service item.
func NewCustomItem(title, notes string, slides []SlideData) ServiceItem {
	return ServiceItem{
		Header: ItemHeader{
			Name:            "custom",
			Plugin:          "custom",
			Theme:           nil,
			Title:           title,
			Footer:          []string{},
			Type:            TypeText,
			Audit:           "",
			Notes:           notes,
			FromPlugin:      false,
			Capabilities:    CustomCapabilities,
			Search:          title,
			Data:            "",
			XMLVersion:      nil,
			BackgroundAudio: []string{},
			Metadata:        []any{},
		},
		Data: slides,
	}
}

// NewMediaItem creates a media service item placeholder.
func NewMediaItem(title string) ServiceItem {
	processor := "qt6"
	theme := -1
	return ServiceItem{
		Header: ItemHeader{
			Name:            "media",
			Plugin:          "media",
			Theme:           theme,
			Title:           title,
			Footer:          []string{},
			Type:            TypeCommand,
			Audit:           "",
			Notes:           "",
			FromPlugin:      false,
			Capabilities:    MediaCapabilities,
			Search:          title,
			Data:            "",
			XMLVersion:      nil,
			Processor:       &processor,
			BackgroundAudio: []string{},
			Metadata:        []any{},
		},
		Data: []SlideData{},
	}
}

// oszEntry is used for JSON serialization of the .osj file.
type oszEntry struct {
	OPLPCore    *json.RawMessage    `json:"openlp_core,omitempty"`
	ServiceItem *serviceItemWrapper `json:"serviceitem,omitempty"`
}

type serviceItemWrapper struct {
	Header ItemHeader  `json:"header"`
	Data   []SlideData `json:"data"`
}
