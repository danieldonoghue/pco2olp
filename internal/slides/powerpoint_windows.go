//go:build windows

package slides

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// PowerPointConverter uses Microsoft PowerPoint (via PowerShell COM) to convert
// presentations to 1920×1080 PNG images, one per slide.
type PowerPointConverter struct{}

func (c *PowerPointConverter) CanConvert(ext string) bool {
	switch ext {
	case ".pptx", ".ppt", ".ppsx", ".pps", ".pptm":
		return true
	}
	return false
}

func (c *PowerPointConverter) IsAvailable() bool {
	script := `try { $p = New-Object -ComObject PowerPoint.Application; $p.Quit(); exit 0 } catch { exit 1 }`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	return cmd.Run() == nil
}

func (c *PowerPointConverter) ConvertToPNGs(inputFile, outputDir string) ([]string, error) {
	absInput, err := filepath.Abs(inputFile)
	if err != nil {
		return nil, fmt.Errorf("resolving input path: %w", err)
	}
	absOutput, err := filepath.Abs(outputDir)
	if err != nil {
		return nil, fmt.Errorf("resolving output path: %w", err)
	}

	// PowerShell script: open presentation, export each slide as 1920×1080 PNG, close
	script := fmt.Sprintf(`
$ErrorActionPreference = 'Stop'
$ppt = New-Object -ComObject PowerPoint.Application
$ppt.Visible = [Microsoft.Office.Core.MsoTriState]::msoFalse
try {
	$pres = $ppt.Presentations.Open('%s', [Microsoft.Office.Core.MsoTriState]::msoTrue, [Microsoft.Office.Core.MsoTriState]::msoFalse, [Microsoft.Office.Core.MsoTriState]::msoFalse)
	$count = $pres.Slides.Count
	for ($i = 1; $i -le $count; $i++) {
		$out = '%s\slide-' + $i.ToString('000') + '.png'
		$pres.Slides[$i].Export($out, 'PNG', 1920, 1080)
	}
	$pres.Close()
} finally {
	$ppt.Quit()
}`,
		strings.ReplaceAll(absInput, `'`, `''`),
		strings.ReplaceAll(absOutput, `'`, `''`),
	)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("powerpoint export: %w\n%s", err, out)
	}

	paths, err := filepath.Glob(filepath.Join(absOutput, "slide-*.png"))
	if err != nil {
		return nil, err
	}
	sortSlidesByNumber(paths)
	return paths, nil
}
