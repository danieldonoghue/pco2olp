//go:build !gui

package main

// runGUI is a no-op when the binary is built without -tags gui.
// main() will fall through to flag.Parse() and show CLI usage.
func runGUI() {}
