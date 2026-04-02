package pco

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"golang.org/x/oauth2"
)

const (
	baseURL   = "https://api.planningcenteronline.com"
	userAgent = "pco2olp/1.0 (https://github.com/danieldonoghue/pco2olp)"
)

// Client is the Planning Center API client.
type Client struct {
	httpClient *http.Client
	debug      *log.Logger
	retryAfter time.Time
}

// NewClient creates a new PCO API client using the given token source.
// The oauth2 transport handles Authorization headers and token refresh.
func NewClient(tokenSource oauth2.TokenSource, debug bool) *Client {
	var logger *log.Logger
	if debug {
		logger = log.New(os.Stderr, "[pco] ", log.LstdFlags)
	}
	return &Client{
		httpClient: oauth2.NewClient(context.Background(), tokenSource),
		debug:      logger,
	}
}

func (c *Client) logf(format string, args ...any) {
	if c.debug != nil {
		c.debug.Printf(format, args...)
	}
}

// get performs an authenticated GET request to the PCO API.
func (c *Client) get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	reqURL := baseURL + path
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}
	return c.doGet(ctx, reqURL)
}

// getFullURL performs a GET on a full URL (used for pagination links).
func (c *Client) getFullURL(ctx context.Context, fullURL string) ([]byte, error) {
	return c.doGet(ctx, fullURL)
}

func (c *Client) doGet(ctx context.Context, reqURL string) ([]byte, error) {
	for attempt := 0; attempt < 3; attempt++ {
		if wait := time.Until(c.retryAfter); wait > 0 {
			c.logf("rate limited, waiting %v", wait)
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)

		c.logf("GET %s", reqURL)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("API request failed: %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}

		c.logf("  -> %d (%d bytes)", resp.StatusCode, len(body))

		switch {
		case resp.StatusCode == http.StatusOK:
			return body, nil

		case resp.StatusCode == http.StatusTooManyRequests:
			retryAfter := resp.Header.Get("Retry-After")
			secs, _ := strconv.Atoi(retryAfter)
			if secs == 0 {
				secs = 5
			}
			c.retryAfter = time.Now().Add(time.Duration(secs) * time.Second)
			c.logf("  rate limited, retry after %ds", secs)
			continue

		case resp.StatusCode == http.StatusUnauthorized:
			return nil, fmt.Errorf("unauthorized (HTTP 401). Try re-authenticating by deleting your token file")

		default:
			var apiErr struct {
				Errors []struct {
					Title  string `json:"title"`
					Detail string `json:"detail"`
				} `json:"errors"`
			}
			if json.Unmarshal(body, &apiErr) == nil && len(apiErr.Errors) > 0 {
				return nil, fmt.Errorf("API error (HTTP %d): %s - %s",
					resp.StatusCode, apiErr.Errors[0].Title, apiErr.Errors[0].Detail)
			}
			return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
		}
	}

	return nil, fmt.Errorf("API request failed after 3 retries (rate limited)")
}

// post performs an authenticated POST request to the PCO API.
func (c *Client) post(ctx context.Context, path string) ([]byte, error) {
	reqURL := baseURL + path
	return c.doPost(ctx, reqURL)
}

func (c *Client) doPost(ctx context.Context, reqURL string) ([]byte, error) {
	for attempt := 0; attempt < 3; attempt++ {
		if wait := time.Until(c.retryAfter); wait > 0 {
			c.logf("rate limited, waiting %v", wait)
			select {
			case <-time.After(wait):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", reqURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)

		c.logf("POST %s", reqURL)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("API request failed: %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response: %w", err)
		}

		c.logf("  -> %d (%d bytes)", resp.StatusCode, len(body))

		switch {
		case resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated:
			return body, nil

		case resp.StatusCode == http.StatusTooManyRequests:
			retryAfter := resp.Header.Get("Retry-After")
			secs, _ := strconv.Atoi(retryAfter)
			if secs == 0 {
				secs = 5
			}
			c.retryAfter = time.Now().Add(time.Duration(secs) * time.Second)
			c.logf("  rate limited, retry after %ds", secs)
			continue

		case resp.StatusCode == http.StatusUnauthorized:
			return nil, fmt.Errorf("unauthorized (HTTP 401). Try re-authenticating by deleting your token file")

		default:
			var apiErr struct {
				Errors []struct {
					Title  string `json:"title"`
					Detail string `json:"detail"`
				} `json:"errors"`
			}
			if json.Unmarshal(body, &apiErr) == nil && len(apiErr.Errors) > 0 {
				return nil, fmt.Errorf("API error (HTTP %d): %s - %s",
					resp.StatusCode, apiErr.Errors[0].Title, apiErr.Errors[0].Detail)
			}
			return nil, fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(body))
		}
	}

	return nil, fmt.Errorf("API POST failed after 3 retries (rate limited)")
}

// DownloadFile streams a file from the given URL to w.
// Uses a plain HTTP client (no OAuth headers) since download URLs are typically
// pre-signed CDN/S3 URLs that reject extra Authorization headers.
func (c *Client) DownloadFile(ctx context.Context, fileURL string, w io.Writer) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fileURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logf("download failed: HTTP %d, body: %s", resp.StatusCode, string(body))
		return 0, fmt.Errorf("download failed (HTTP %d)", resp.StatusCode)
	}

	n, err := io.Copy(w, resp.Body)
	if err != nil {
		return n, fmt.Errorf("downloading file: %w", err)
	}
	return n, nil
}
