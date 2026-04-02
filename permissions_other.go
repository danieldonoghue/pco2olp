//go:build !darwin

package main

// requestPermissions is a no-op on non-macOS platforms.
func requestPermissions() {}
