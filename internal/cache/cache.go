package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Slide represents a single converted slide PNG, ready to embed in an .osz image group.
type Slide struct {
	LocalPath string // Absolute path to the PNG file (temp dir, valid for this run)
	SHA256    string // Hex-encoded SHA256 hash of the PNG
}

// MediaFile represents a downloaded and cached media file ready for .osz embedding.
type MediaFile struct {
	OriginalFilename string
	LocalPath        string // Absolute path in cache directory
	SHA256           string // Hex-encoded SHA256 hash
	Extension        string // e.g., ".mp4"
	ContentType      string
	FileSize         int64
	PCOMediaType     string // PCO media_type field (e.g., "video", "image")
	CacheHit         bool   // True if served from cache without downloading
	Slides           []Slide // Converted slide PNGs (populated at generation time, not cached)
}

// Cache manages downloaded media files.
type Cache struct {
	mu        sync.Mutex
	dir       string
	index     *Index
	indexPath string
}

// NewCache creates or opens the media cache in the platform cache directory.
func NewCache() (*Cache, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("finding cache directory: %w", err)
	}

	dir := filepath.Join(cacheDir, "pco2olp", "media")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("creating cache directory: %w", err)
	}

	indexPath := filepath.Join(dir, "index.json")
	idx, err := LoadIndex(indexPath)
	if err != nil {
		return nil, err
	}

	return &Cache{
		dir:       dir,
		index:     idx,
		indexPath: indexPath,
	}, nil
}

// newCacheAt creates a cache in a specific directory (for testing).
func newCacheAt(dir string) (*Cache, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	indexPath := filepath.Join(dir, "index.json")
	idx, err := LoadIndex(indexPath)
	if err != nil {
		return nil, err
	}
	return &Cache{dir: dir, index: idx, indexPath: indexPath}, nil
}

// EnsureCached checks the cache and downloads the file if needed.
// The downloader function is called only when a download is necessary.
func (c *Cache) EnsureCached(
	attachmentID string,
	updatedAt time.Time,
	filename string,
	contentType string,
	fileSize int64,
	noCache bool,
	downloader func(w io.Writer) (int64, error),
) (*MediaFile, error) {
	// Check cache (unless force download)
	if !noCache {
		c.mu.Lock()
		entry := c.index.Lookup(attachmentID, updatedAt)
		c.mu.Unlock()
		if entry != nil {
			localPath := filepath.Join(c.dir, entry.SHA256+extFromFilename(entry.Filename))
			if _, err := os.Stat(localPath); err == nil {
				return &MediaFile{
					OriginalFilename: entry.Filename,
					LocalPath:        localPath,
					SHA256:           entry.SHA256,
					Extension:        extFromFilename(entry.Filename),
					ContentType:      entry.ContentType,
					FileSize:         entry.FileSize,
					CacheHit:         true,
				}, nil
			}
			// File missing from disk — re-download
		}
	}

	// Download to temp file, computing SHA256 as we go
	tmpFile, err := os.CreateTemp(c.dir, "download-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath) // clean up temp if we renamed it
	}()

	hasher := sha256.New()
	multi := io.MultiWriter(tmpFile, hasher)

	written, err := downloader(multi)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", filename, err)
	}
	tmpFile.Close()

	hash := hex.EncodeToString(hasher.Sum(nil))
	ext := extFromFilename(filename)
	storedName := hash + ext
	finalPath := filepath.Join(c.dir, storedName)

	// Move to final location
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return nil, fmt.Errorf("storing cached file: %w", err)
	}

	// Update index (thread-safe)
	c.mu.Lock()
	c.index.Set(attachmentID, &IndexEntry{
		AttachmentID: attachmentID,
		PCOUpdatedAt: updatedAt,
		Filename:     filename,
		ContentType:  contentType,
		FileSize:     written,
		SHA256:       hash,
		DownloadedAt: time.Now(),
	})
	if err := c.index.Save(c.indexPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save cache index: %v\n", err)
	}
	c.mu.Unlock()

	return &MediaFile{
		OriginalFilename: filename,
		LocalPath:        finalPath,
		SHA256:           hash,
		Extension:        ext,
		ContentType:      contentType,
		FileSize:         written,
	}, nil
}

// Clean removes all cached files and the index.
func (c *Cache) Clean() error {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading cache directory: %w", err)
	}

	for _, e := range entries {
		os.Remove(filepath.Join(c.dir, e.Name()))
	}

	c.index = &Index{Version: 1, Entries: make(map[string]*IndexEntry)}
	return nil
}

// Info returns cache statistics.
func (c *Cache) Info() (dir string, fileCount int, totalSize int64, err error) {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return c.dir, 0, 0, nil
		}
		return c.dir, 0, 0, err
	}

	for _, e := range entries {
		if e.Name() == "index.json" || e.Name() == "index.json.tmp" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		fileCount++
		totalSize += info.Size()
	}
	return c.dir, fileCount, totalSize, nil
}

func extFromFilename(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return ""
	}
	return strings.ToLower(ext)
}
