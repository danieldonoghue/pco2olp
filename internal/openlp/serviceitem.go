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
	SongCapabilities         = []int{CapCanEdit, CapCanPreview, CapCanLoop, CapOnLoadUpdate, CapAddIfNewItem, CapCanSoftBreak, CapHasMetaData}
	CustomCapabilities       = []int{CapCanEdit, CapCanPreview, CapCanLoop, CapCanSoftBreak, CapOnLoadUpdate, CapCanWordSplit}
	MediaCapabilities        = []int{CapCanAutoStartForLive, CapCanEditTitle, CapRequiresMedia}
	ImageCapabilities        = []int{CapCanPreview, CapCanEditTitle, CapHasThumbnails, CapProvidesOwnTheme}
	PresentationCapabilities = []int{CapCanEditTitle, CapProvidesOwnDisplay, CapHasDisplayTitle, CapHasNotes, CapHasThumbnails}
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
// Text items (songs/custom) use Title, RawSlide, VerseTag.
// Media/Image/Presentation items use Title, Image, Path, DisplayTitle, Notes, FileHash.
type SlideData struct {
	Title        string `json:"title"`
	RawSlide     string `json:"raw_slide"`
	VerseTag     string `json:"verseTag"`
	Image        string `json:"image"`
	Path         string `json:"path"`
	DisplayTitle string `json:"display_title"`
	Notes        string `json:"notes"`
	FileHash     string `json:"file_hash"`
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
// Credits appear in the footer and are used by OpenLP to identify unique content.
func NewCustomItem(title, credits, notes string, slides []SlideData) ServiceItem {
	footer := title
	if credits != "" {
		footer = title + " " + credits
	}
	return ServiceItem{
		Header: ItemHeader{
			Name:            "custom",
			Plugin:          "custom",
			Theme:           nil,
			Title:           title,
			Footer:          []string{footer},
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

// NewMediaItem creates a media service item (video/audio).
func NewMediaItem(title string, sha256Hash, storedFilename *string) ServiceItem {
	processor := "qt6"
	theme := -1
	slideTitle := title
	if storedFilename != nil {
		slideTitle = *storedFilename
	}
	return ServiceItem{
		Header: ItemHeader{
			Name:            "media",
			Plugin:          "media",
			Theme:           theme,
			Title:           title,
			Footer:          []string{},
			Type:            TypeCommand,
			Capabilities:    MediaCapabilities,
			Search:          title,
			Data:            "",
			Processor:       &processor,
			BackgroundAudio: []string{},
			Metadata:        []any{},
			SHA256FileHash:  sha256Hash,
			StoredFilename:  storedFilename,
		},
		Data: []SlideData{{
			Title: slideTitle,
			Image: "clapperboard",
		}},
	}
}

// NewImageItem creates an image service item.
func NewImageItem(title string, sha256Hash, storedFilename *string) ServiceItem {
	var fileHash string
	slideTitle := title
	if sha256Hash != nil {
		fileHash = *sha256Hash
	}
	if storedFilename != nil {
		slideTitle = *storedFilename
	}
	return ServiceItem{
		Header: ItemHeader{
			Name:            "images",
			Plugin:          "images",
			Theme:           nil,
			Title:           title,
			Footer:          []string{},
			Type:            TypeImage,
			Capabilities:    ImageCapabilities,
			Search:          title,
			Data:            "",
			BackgroundAudio: []string{},
			Metadata:        []any{},
			SHA256FileHash:  sha256Hash,
			StoredFilename:  storedFilename,
		},
		Data: []SlideData{{
			Title:    slideTitle,
			FileHash: fileHash,
		}},
	}
}

// NewImageGroupItem creates an image service item from multiple slide PNGs.
// Each SlideRef provides the SHA256 hash and stored filename for one slide.
// This is used when a presentation is pre-converted to PNG images.
func NewImageGroupItem(title string, slides []SlideRef) ServiceItem {
	var data []SlideData
	for _, s := range slides {
		data = append(data, SlideData{
			Title:    s.StoredName,
			FileHash: s.SHA256,
		})
	}
	return ServiceItem{
		Header: ItemHeader{
			Name:            "images",
			Plugin:          "images",
			Theme:           nil,
			Title:           title,
			Footer:          []string{},
			Type:            TypeImage,
			Capabilities:    ImageCapabilities,
			Search:          title,
			Data:            "",
			BackgroundAudio: []string{},
			Metadata:        []any{},
		},
		Data: data,
	}
}

// SlideRef is a reference to a single slide PNG embedded in the .osz archive.
type SlideRef struct {
	SHA256     string
	StoredName string // "{sha256}.png"
}

// NewPresentationItem creates a presentation service item (PowerPoint, Keynote, PDF, etc.).
// The processor should match an OpenLP presentation controller name (e.g. "Impress", "Powerpoint", "Keynote", "Pdf").
// Image is set to "thumbnail.png" to avoid OpenLP bugs with "clapperboard" (QIcon crash) and "" (relative path crash).
func NewPresentationItem(title string, sha256Hash, storedFilename *string, processor string) ServiceItem {
	slideTitle := title
	if storedFilename != nil {
		slideTitle = *storedFilename
	}
	return ServiceItem{
		Header: ItemHeader{
			Name:            "presentations",
			Plugin:          "presentations",
			Theme:           nil,
			Title:           title,
			Footer:          []string{},
			Type:            TypeCommand,
			Capabilities:    PresentationCapabilities,
			Search:          title,
			Data:            "",
			Processor:       &processor,
			BackgroundAudio: []string{},
			Metadata:        []any{},
			SHA256FileHash:  sha256Hash,
			StoredFilename:  storedFilename,
		},
		Data: []SlideData{{
			Title: slideTitle,
			Image: "thumbnail.png",
		}},
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
