package slides

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Converter converts a presentation file into ordered PNG slide images.
type Converter interface {
	// CanConvert returns true if this converter handles the given file extension.
	CanConvert(ext string) bool
	// IsAvailable returns true if the required tool or application is installed.
	IsAvailable() bool
	// ConvertToPNGs converts the input file to PNG images in outputDir.
	// Returns the absolute paths of generated PNG files in slide order.
	ConvertToPNGs(inputFile, outputDir string) ([]string, error)
}

// ConvertToPNGs selects the best available converter for the file and returns
// paths to PNG images, one per slide, in slide order. Returns nil, nil if no
// converter is available — callers should fall back gracefully.
func ConvertToPNGs(inputFile, outputDir string) ([]string, error) {
	ext := strings.ToLower(filepath.Ext(inputFile))
	for _, c := range platformConverters() {
		if c.CanConvert(ext) && c.IsAvailable() {
			return c.ConvertToPNGs(inputFile, outputDir)
		}
	}
	return nil, nil
}

// IsPresentationType reports whether a file extension should be converted to slides.
func IsPresentationType(ext string) bool {
	switch strings.ToLower(ext) {
	case ".pdf", ".pptx", ".ppt", ".ppsx", ".pps", ".key":
		return true
	}
	return false
}

// sortSlidesByNumber sorts a list of PNG paths by the trailing number in their filename.
// Handles names like "slide-001.png", "001.png", "Slide 1.png".
func sortSlidesByNumber(paths []string) {
	sort.Slice(paths, func(i, j int) bool {
		return slideNumber(paths[i]) < slideNumber(paths[j])
	})
}

// slideNumber extracts the last run of digits from a filename, used for natural sort.
func slideNumber(path string) int {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(filepath.Base(path)))
	end := len(name)
	for end > 0 && (name[end-1] < '0' || name[end-1] > '9') {
		end--
	}
	start := end
	for start > 0 && name[start-1] >= '0' && name[start-1] <= '9' {
		start--
	}
	if start == end {
		return 0
	}
	n, _ := strconv.Atoi(name[start:end])
	return n
}
