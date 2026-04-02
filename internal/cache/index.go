package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// IndexEntry represents a single cached file in the index.
type IndexEntry struct {
	AttachmentID string    `json:"attachment_id"`
	PCOUpdatedAt time.Time `json:"pco_updated_at"`
	Filename     string    `json:"filename"`
	ContentType  string    `json:"content_type"`
	FileSize     int64     `json:"file_size"`
	SHA256       string    `json:"sha256"`
	DownloadedAt time.Time `json:"downloaded_at"`
}

// Index is the on-disk cache index.
type Index struct {
	Version int                    `json:"version"`
	Entries map[string]*IndexEntry `json:"entries"`
}

// LoadIndex reads the index from disk. Returns an empty index if the file doesn't exist.
func LoadIndex(path string) (*Index, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Index{Version: 1, Entries: make(map[string]*IndexEntry)}, nil
		}
		return nil, fmt.Errorf("reading cache index: %w", err)
	}

	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parsing cache index: %w", err)
	}
	if idx.Entries == nil {
		idx.Entries = make(map[string]*IndexEntry)
	}
	return &idx, nil
}

// Save writes the index to disk atomically (write temp + rename).
func (idx *Index) Save(path string) error {
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache index: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("writing cache index: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("saving cache index: %w", err)
	}
	return nil
}

// Lookup checks if an attachment is cached and still fresh.
// Returns the entry if cached and updatedAt matches, nil otherwise.
func (idx *Index) Lookup(attachmentID string, updatedAt time.Time) *IndexEntry {
	key := "att_" + attachmentID
	entry, ok := idx.Entries[key]
	if !ok {
		return nil
	}
	if !entry.PCOUpdatedAt.Equal(updatedAt) {
		return nil
	}
	return entry
}

// Set adds or updates an entry in the index.
func (idx *Index) Set(attachmentID string, entry *IndexEntry) {
	key := "att_" + attachmentID
	idx.Entries[key] = entry
}
