package pco

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// GetItemMedia fetches the media resources for a plan item.
func (c *Client) GetItemMedia(ctx context.Context, serviceTypeID, planID, itemID string) ([]Media, error) {
	path := fmt.Sprintf("/services/v2/service_types/%s/plans/%s/items/%s/media", serviceTypeID, planID, itemID)
	body, err := c.get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	var resp jsonAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing media: %w", err)
	}

	var resources []jsonAPIResource
	if err := json.Unmarshal(resp.Data, &resources); err != nil {
		return nil, fmt.Errorf("parsing media data: %w", err)
	}

	var media []Media
	for _, r := range resources {
		m, err := parseMedia(r)
		if err != nil {
			c.logf("skipping media %s: %v", r.ID, err)
			continue
		}
		media = append(media, m)
	}
	return media, nil
}

// GetItemAttachments fetches downloadable attachments for a plan item.
func (c *Client) GetItemAttachments(ctx context.Context, serviceTypeID, planID, itemID string) ([]Attachment, error) {
	path := fmt.Sprintf("/services/v2/service_types/%s/plans/%s/items/%s/attachments", serviceTypeID, planID, itemID)
	return c.fetchAttachments(ctx, path)
}

// GetPlanAttachments fetches attachments for a plan itself.
func (c *Client) GetPlanAttachments(ctx context.Context, serviceTypeID, planID string) ([]Attachment, error) {
	path := fmt.Sprintf("/services/v2/service_types/%s/plans/%s/attachments", serviceTypeID, planID)
	return c.fetchAttachments(ctx, path)
}

func (c *Client) fetchAttachments(ctx context.Context, path string) ([]Attachment, error) {
	params := url.Values{"per_page": {"100"}}
	body, err := c.get(ctx, path, params)
	if err != nil {
		return nil, err
	}

	var resp jsonAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing attachments: %w", err)
	}

	var resources []jsonAPIResource
	if err := json.Unmarshal(resp.Data, &resources); err != nil {
		return nil, fmt.Errorf("parsing attachments data: %w", err)
	}

	var attachments []Attachment
	for _, r := range resources {
		att, err := parseAttachment(r)
		if err != nil {
			c.logf("skipping attachment %s: %v", r.ID, err)
			continue
		}
		attachments = append(attachments, att)
	}
	return attachments, nil
}

// OpenAttachment calls the open action on an attachment to get a temporary download URL.
func (c *Client) OpenAttachment(ctx context.Context, attachmentID string) (string, error) {
	path := fmt.Sprintf("/services/v2/attachments/%s/open", attachmentID)
	body, err := c.post(ctx, path)
	if err != nil {
		return "", fmt.Errorf("opening attachment %s: %w", attachmentID, err)
	}

	var resp struct {
		Data struct {
			Attributes struct {
				AttachmentURL string `json:"attachment_url"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("parsing attachment open response: %w", err)
	}

	if resp.Data.Attributes.AttachmentURL == "" {
		return "", fmt.Errorf("no download URL returned for attachment %s", attachmentID)
	}

	return resp.Data.Attributes.AttachmentURL, nil
}

func parseMedia(r jsonAPIResource) (Media, error) {
	var attrs struct {
		Title     string `json:"title"`
		MediaType string `json:"media_type"`
		UpdatedAt string `json:"updated_at"`
	}
	if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
		return Media{}, err
	}

	updatedAt, _ := time.Parse(time.RFC3339, attrs.UpdatedAt)

	return Media{
		ID:        r.ID,
		Title:     attrs.Title,
		MediaType: attrs.MediaType,
		UpdatedAt: updatedAt,
	}, nil
}

func parseAttachment(r jsonAPIResource) (Attachment, error) {
	var attrs struct {
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
		FileSize    int64  `json:"file_size"`
		URL         string `json:"url"`
		UpdatedAt   string `json:"updated_at"`
	}
	if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
		return Attachment{}, err
	}

	updatedAt, _ := time.Parse(time.RFC3339, attrs.UpdatedAt)

	return Attachment{
		ID:          r.ID,
		Filename:    attrs.Filename,
		ContentType: attrs.ContentType,
		FileSize:    attrs.FileSize,
		URL:         attrs.URL,
		UpdatedAt:   updatedAt,
	}, nil
}
