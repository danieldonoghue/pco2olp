//go:build gui

package main

import (
	"os"

	"github.com/danieldonoghue/pco2olp/internal/gui"
)

func runGUI() {
	gui.Run(version, docsVersion, orgName)
	os.Exit(0)
}
