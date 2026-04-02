package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"golang.org/x/oauth2"
)

const (
	pcoAuthURL  = "https://api.planningcenteronline.com/oauth/authorize"
	pcoTokenURL = "https://api.planningcenteronline.com/oauth/token"
)

// These can be set at build time via -ldflags:
//
//	-X github.com/danield/pco2olp/internal/auth.defaultClientID=...
//	-X github.com/danield/pco2olp/internal/auth.defaultClientSecret=...
var (
	defaultClientID     = ""
	defaultClientSecret = ""
)

// Authenticator manages the OAuth 2.0 flow with Planning Center.
type Authenticator struct {
	oauthConfig *oauth2.Config
	tokenStore  *TokenStore
}

// NewAuthenticator creates an authenticator with the given token store.
func NewAuthenticator(tokenStore *TokenStore) *Authenticator {
	clientID := os.Getenv("PCO_CLIENT_ID")
	if clientID == "" {
		clientID = defaultClientID
	}
	clientSecret := os.Getenv("PCO_CLIENT_SECRET")
	if clientSecret == "" {
		clientSecret = defaultClientSecret
	}
	return &Authenticator{
		oauthConfig: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint: oauth2.Endpoint{
				AuthURL:  pcoAuthURL,
				TokenURL: pcoTokenURL,
			},
			Scopes: []string{"services"},
		},
		tokenStore: tokenStore,
	}
}

// TokenSource returns an oauth2.TokenSource that auto-refreshes tokens.
// If no valid token exists, it initiates the browser-based auth flow.
func (a *Authenticator) TokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	if a.oauthConfig.ClientID == "" {
		return nil, fmt.Errorf("PCO_CLIENT_ID environment variable is not set.\n" +
			"Register an OAuth app at https://api.planningcenteronline.com/oauth/applications\n" +
			"Then set PCO_CLIENT_ID and PCO_CLIENT_SECRET to your application's credentials")
	}

	tok, err := a.tokenStore.Load()
	if err != nil {
		return nil, fmt.Errorf("loading stored token: %w", err)
	}

	if tok == nil || (!tok.Valid() && tok.RefreshToken == "") {
		tok, err = a.authenticate(ctx)
		if err != nil {
			return nil, err
		}
	}

	// Wrap in a ReuseTokenSource that auto-refreshes and persists
	return &persistingTokenSource{
		base:       a.oauthConfig.TokenSource(ctx, tok),
		tokenStore: a.tokenStore,
		lastToken:  tok,
		authFunc:   a.authenticate,
	}, nil
}

// HTTPClient returns an *http.Client that automatically injects auth headers
// and refreshes tokens. This is the primary way other packages should make
// authenticated requests.
func (a *Authenticator) HTTPClient(ctx context.Context) (*http.Client, error) {
	ts, err := a.TokenSource(ctx)
	if err != nil {
		return nil, err
	}
	return oauth2.NewClient(ctx, ts), nil
}

// persistingTokenSource wraps an oauth2.TokenSource and saves new tokens to disk.
type persistingTokenSource struct {
	base       oauth2.TokenSource
	tokenStore *TokenStore
	lastToken  *oauth2.Token
	authFunc   func(ctx context.Context) (*oauth2.Token, error)
}

func (s *persistingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := s.base.Token()
	if err != nil {
		// Refresh may have failed — try full re-auth
		fmt.Fprintf(os.Stderr, "Token refresh failed, re-authenticating...\n")
		tok, err = s.authFunc(context.Background())
		if err != nil {
			return nil, err
		}
	}

	// Persist if the token changed (i.e. was refreshed)
	if tok.AccessToken != s.lastToken.AccessToken {
		_ = s.tokenStore.Save(tok)
		s.lastToken = tok
	}

	return tok, nil
}

func (a *Authenticator) authenticate(ctx context.Context) (*oauth2.Token, error) {
	verifier := oauth2.GenerateVerifier()

	state, err := generateState()
	if err != nil {
		return nil, fmt.Errorf("generating state: %w", err)
	}

	// Start local callback server on a fixed port (must match PCO app redirect URI)
	const callbackPort = 11019
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", callbackPort))
	if err != nil {
		return nil, fmt.Errorf("starting callback server on port %d (is another instance running?): %w", callbackPort, err)
	}
	a.oauthConfig.RedirectURL = fmt.Sprintf("http://localhost:%d/callback", callbackPort)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("state mismatch in OAuth callback")
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			desc := r.URL.Query().Get("error_description")
			errCh <- fmt.Errorf("OAuth error: %s - %s", errMsg, desc)
			fmt.Fprintf(w, "<html><body><h1>Authentication Failed</h1><p>%s</p><p>You can close this tab.</p></body></html>", desc)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no authorization code in callback")
			http.Error(w, "Missing code", http.StatusBadRequest)
			return
		}
		codeCh <- code
		fmt.Fprint(w, `<html><body><h1>Authentication Successful</h1><p>You can close this tab and return to the terminal.</p></body></html>`)
	})

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("callback server error: %w", err)
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
	}()

	// Build authorization URL with PKCE
	authorizationURL := a.oauthConfig.AuthCodeURL(
		state,
		oauth2.S256ChallengeOption(verifier),
	)

	fmt.Fprintf(os.Stderr, "Opening browser for authentication...\n")
	fmt.Fprintf(os.Stderr, "If the browser doesn't open, visit:\n%s\n\n", authorizationURL)

	if err := openBrowser(authorizationURL); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not open browser: %v\n", err)
	}

	// Wait for callback
	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("authentication timed out (5 minutes)")
	}

	// Exchange code for tokens using x/oauth2 (handles PKCE verifier)
	tok, err := a.oauthConfig.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("exchanging authorization code: %w", err)
	}

	if err := a.tokenStore.Save(tok); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not save token: %v\n", err)
	}

	fmt.Fprintf(os.Stderr, "Authentication successful!\n")
	return tok, nil
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b)[:22], nil
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", url).Start()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
