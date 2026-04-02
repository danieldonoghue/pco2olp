//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// requestPermissions proactively triggers macOS Automation permission dialogs
// for apps we control via AppleScript. Calling this before conversions begin
// ensures the user can approve access before the real work starts, avoiding
// mid-run "Connection is invalid" (-609) failures.
func requestPermissions() {
	if _, err := os.Stat("/Applications/Keynote.app"); err != nil {
		return // Keynote not installed, nothing to do
	}

	// Send a trivial Apple Event to Keynote to surface the permission dialog.
	// `running` is a lightweight property check that doesn't import/open files.
	cmd := exec.Command("osascript", "-e", `tell application "Keynote" to running`)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := string(out)
		if strings.Contains(msg, "not allowed") || strings.Contains(msg, "1743") || strings.Contains(msg, "-1743") {
			fmt.Fprintf(os.Stderr, "Warning: pco2olp is not allowed to control Keynote.\n"+
				"Go to System Settings → Privacy & Security → Automation\n"+
				"and enable Keynote under pco2olp, then run again.\n\n")
		}
	}
}
