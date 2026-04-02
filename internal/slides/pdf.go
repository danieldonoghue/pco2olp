package slides

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// PDFConverter converts PDF files to PNGs using pdftoppm (from poppler).
// Install: brew install poppler  /  apt install poppler-utils  /  choco install poppler
type PDFConverter struct{}

func (c *PDFConverter) CanConvert(ext string) bool { return ext == ".pdf" }

func (c *PDFConverter) IsAvailable() bool {
	_, err := exec.LookPath("pdftoppm")
	return err == nil
}

// ConvertToPNGs converts a PDF to 1920-wide PNGs, one per page.
// For standard 16:9 presentations (10"×5.625") this yields 1920×1080 at 192 DPI.
func (c *PDFConverter) ConvertToPNGs(inputFile, outputDir string) ([]string, error) {
	prefix := filepath.Join(outputDir, "slide")
	cmd := exec.Command("pdftoppm",
		"-scale-to-x", "1920",
		"-scale-to-y", "-1", // maintain aspect ratio
		"-png",
		inputFile,
		prefix,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("pdftoppm: %w\n%s", err, out)
	}
	paths, err := filepath.Glob(filepath.Join(outputDir, "slide-*.png"))
	if err != nil {
		return nil, err
	}
	sortSlidesByNumber(paths)
	return paths, nil
}
