//go:build gui

package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/danieldonoghue/pco2olp/resources"
)

// Run starts the Fyne GUI application. Blocks until the window is closed.
// docsVersion, if non-empty, overrides version for the Help → User Guide URL
// (used in org builds where version is the org release tag, not the upstream pco2olp tag).
func Run(version, docsVersion, orgName string) {
	a := app.NewWithID("io.github.danieldonoghue.pco2olp")

	icon := fyne.NewStaticResource("app-icon.svg", resources.AppIcon)
	a.SetIcon(icon)
	app.SetMetadata(fyne.AppMetadata{
		ID:      "io.github.danieldonoghue.pco2olp",
		Name:    "pco2olp",
		Version: version,
		Icon:    icon,
	})

	title := "pco2olp"
	if orgName != "" {
		title = fmt.Sprintf("pco2olp — %s", orgName)
	}

	w := newMainWindow(a, title, version, docsVersion, orgName)
	w.Resize(fyne.NewSize(660, 580))
	w.SetMaster()
	w.ShowAndRun()
}
