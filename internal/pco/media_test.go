package pco

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func newTestClient(handler http.HandlerFunc) (*Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	client := &Client{
		httpClient: oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test"})),
	}
	return client, server
}

func TestGetItemMedia(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"data": [{
				"id": "111",
				"type": "Media",
				"attributes": {
					"title": "Worship Video",
					"media_type": "video",
					"updated_at": "2024-06-15T10:00:00Z"
				}
			}]
		}`))
	})
	defer server.Close()

	// Override get to use test server
	media, err := client.getItemMediaFromURL(context.Background(), server.URL+"/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(media) != 1 {
		t.Fatalf("expected 1 media, got %d", len(media))
	}
	if media[0].ID != "111" {
		t.Errorf("expected ID 111, got %s", media[0].ID)
	}
	if media[0].MediaType != "video" {
		t.Errorf("expected media_type video, got %s", media[0].MediaType)
	}
	if media[0].Title != "Worship Video" {
		t.Errorf("expected title 'Worship Video', got %s", media[0].Title)
	}
}

func TestGetItemAttachments(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"data": [{
				"id": "222",
				"type": "Attachment",
				"attributes": {
					"filename": "worship-bg.mp4",
					"content_type": "video/mp4",
					"file_size": 52428800,
					"url": "https://example.com/files/worship-bg.mp4",
					"updated_at": "2024-06-15T10:00:00Z"
				}
			}]
		}`))
	})
	defer server.Close()

	atts, err := client.fetchAttachmentsFromURL(context.Background(), server.URL+"/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(atts) != 1 {
		t.Fatalf("expected 1 attachment, got %d", len(atts))
	}
	if atts[0].Filename != "worship-bg.mp4" {
		t.Errorf("expected filename worship-bg.mp4, got %s", atts[0].Filename)
	}
	if atts[0].FileSize != 52428800 {
		t.Errorf("expected file_size 52428800, got %d", atts[0].FileSize)
	}
}

func TestOpenAttachment(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"data": {
				"id": "333",
				"type": "AttachmentActivity",
				"attributes": {
					"attachment_url": "https://cdn.example.com/signed/worship-bg.mp4?token=abc"
				}
			}
		}`))
	})
	defer server.Close()

	url, err := client.openAttachmentFromURL(context.Background(), server.URL+"/test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "https://cdn.example.com/signed/worship-bg.mp4?token=abc"
	if url != expected {
		t.Errorf("expected URL %s, got %s", expected, url)
	}
}

func TestDownloadFile(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("file-content-here"))
	})
	defer server.Close()

	var buf strings.Builder
	n, err := client.DownloadFile(context.Background(), server.URL+"/file.mp4", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n != 17 {
		t.Errorf("expected 17 bytes, got %d", n)
	}
	if buf.String() != "file-content-here" {
		t.Errorf("unexpected content: %s", buf.String())
	}
}

// Test helpers that use full URLs directly (for httptest)

func (c *Client) getItemMediaFromURL(ctx context.Context, fullURL string) ([]Media, error) {
	body, err := c.doGet(ctx, fullURL)
	if err != nil {
		return nil, err
	}
	return c.parseMediaResponse(body)
}

func (c *Client) parseMediaResponse(body []byte) ([]Media, error) {
	var resp jsonAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	var resources []jsonAPIResource
	if err := json.Unmarshal(resp.Data, &resources); err != nil {
		return nil, err
	}
	var media []Media
	for _, r := range resources {
		m, err := parseMedia(r)
		if err != nil {
			continue
		}
		media = append(media, m)
	}
	return media, nil
}

func (c *Client) fetchAttachmentsFromURL(ctx context.Context, fullURL string) ([]Attachment, error) {
	body, err := c.doGet(ctx, fullURL)
	if err != nil {
		return nil, err
	}
	var resp jsonAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	var resources []jsonAPIResource
	if err := json.Unmarshal(resp.Data, &resources); err != nil {
		return nil, err
	}
	var attachments []Attachment
	for _, r := range resources {
		att, err := parseAttachment(r)
		if err != nil {
			continue
		}
		attachments = append(attachments, att)
	}
	return attachments, nil
}

func (c *Client) openAttachmentFromURL(ctx context.Context, fullURL string) (string, error) {
	body, err := c.doPost(ctx, fullURL)
	if err != nil {
		return "", err
	}
	var resp struct {
		Data struct {
			Attributes struct {
				AttachmentURL string `json:"attachment_url"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	return resp.Data.Attributes.AttachmentURL, nil
}

// Need encoding/json import for test helpers
