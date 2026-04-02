//go:build darwin

package slides

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// KeynoteConverter uses macOS Keynote (via osascript) to convert presentations to PNGs.
// Handles .key and .pptx/.ppt files. Exports slides directly as PNG images.
type KeynoteConverter struct{}

func (c *KeynoteConverter) CanConvert(ext string) bool {
	switch ext {
	case ".key", ".pptx", ".ppt", ".ppsx", ".pps":
		return true
	}
	return false
}

func (c *KeynoteConverter) IsAvailable() bool {
	_, err := os.Stat("/Applications/Keynote.app")
	return err == nil
}

func (c *KeynoteConverter) ConvertToPNGs(inputFile, outputDir string) ([]string, error) {
	absInput, err := filepath.Abs(inputFile)
	if err != nil {
		return nil, fmt.Errorf("resolving input path: %w", err)
	}
	absOutput, err := filepath.Abs(outputDir)
	if err != nil {
		return nil, fmt.Errorf("resolving output path: %w", err)
	}

	// Keynote creates the export folder itself — remove it first so it doesn't merge with existing files.
	os.RemoveAll(absOutput)

	// Activate Keynote first so it's fully running before we try to open files.
	// This avoids "Connection is invalid" (-609) errors on cold start.
	activateCmd := exec.Command("osascript", "-e", `tell application "Keynote" to activate`)
	activateCmd.Run() // ignore errors — it may already be running
	// Give Keynote a moment to be ready
	exec.Command("osascript", "-e", `delay 1`).Run()

	script := fmt.Sprintf(`
tell application "Keynote"
	set theDoc to open POSIX file %q
	set theOutput to POSIX file %q
	export theDoc to theOutput as slide images with properties {image format:PNG, skipped slides:false}
	close theDoc saving no
end tell`, absInput, absOutput)

	cmd := exec.Command("osascript", "-e", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		if strings.Contains(string(out), "-609") || strings.Contains(string(out), "Connection is invalid") {
			return nil, fmt.Errorf("keynote export: Keynote automation permission required\n"+
				"Go to System Settings → Privacy & Security → Automation\n"+
				"and allow pco2olp to control Keynote, then try again.\n"+
				"(original error: %s)", strings.TrimSpace(string(out)))
		}
		return nil, fmt.Errorf("keynote export: %w\n%s", err, out)
	}

	pngs, err := findPNGs(absOutput)
	if err != nil {
		return nil, err
	}
	sortSlidesByNumber(pngs)
	return pngs, nil
}

func findPNGs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var pngs []string
	for _, e := range entries {
		if e.IsDir() {
			// Keynote sometimes creates a subdirectory — search inside it too
			sub, _ := findPNGs(filepath.Join(dir, e.Name()))
			pngs = append(pngs, sub...)
		} else if strings.ToLower(filepath.Ext(e.Name())) == ".png" {
			pngs = append(pngs, filepath.Join(dir, e.Name()))
		}
	}
	return pngs, nil
}
