//go:build gui

package main

import (
	"os"

	"github.com/danieldonoghue/pco2olp/internal/gui"
)

func runGUI() {
	gui.Run(version, orgName)
	os.Exit(0)
}
