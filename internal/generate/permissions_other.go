//go:build !darwin

package generate

// requestPermissions is a no-op on non-macOS platforms.
func requestPermissions() {}
