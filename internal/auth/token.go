package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

// TokenStore manages token persistence in the platform config directory.
type TokenStore struct {
	path string
}

// NewTokenStore creates a token store in the platform-appropriate config directory.
func NewTokenStore() (*TokenStore, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("determining config directory: %w", err)
	}
	dir = filepath.Join(dir, "pco2olp")
	return &TokenStore{path: filepath.Join(dir, "tokens.json")}, nil
}

// Load reads the stored token from disk. Returns nil, nil if no token file exists.
func (ts *TokenStore) Load() (*oauth2.Token, error) {
	data, err := os.ReadFile(ts.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading token file: %w", err)
	}
	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("parsing token file: %w", err)
	}
	return &tok, nil
}

// Save writes the token to disk with restrictive permissions.
func (ts *TokenStore) Save(tok *oauth2.Token) error {
	dir := filepath.Dir(ts.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ts.path, data, 0600)
}

// Delete removes the stored token file.
func (ts *TokenStore) Delete() error {
	err := os.Remove(ts.path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// Path returns the token file path.
func (ts *TokenStore) Path() string {
	return ts.path
}
