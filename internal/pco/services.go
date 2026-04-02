package pco

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

// JSON:API response structures
type jsonAPIResponse struct {
	Data     json.RawMessage   `json:"data"`
	Included []jsonAPIResource `json:"included,omitempty"`
	Links    jsonAPILinks      `json:"links,omitempty"`
}

type jsonAPIResource struct {
	ID            string                `json:"id"`
	Type          string                `json:"type"`
	Attributes    json.RawMessage       `json:"attributes"`
	Relationships map[string]jsonAPIRel `json:"relationships,omitempty"`
}

type jsonAPIRel struct {
	Data json.RawMessage `json:"data"` // can be object or array
}

type jsonAPILinks struct {
	Next string `json:"next,omitempty"`
}

type jsonAPIRelData struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// ListServiceTypes returns all service types.
func (c *Client) ListServiceTypes(ctx context.Context) ([]ServiceType, error) {
	body, err := c.get(ctx, "/services/v2/service_types", nil)
	if err != nil {
		return nil, err
	}

	var resp jsonAPIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing service types: %w", err)
	}

	var resources []jsonAPIResource
	if err := json.Unmarshal(resp.Data, &resources); err != nil {
		return nil, fmt.Errorf("parsing service types data: %w", err)
	}

	var types []ServiceType
	for _, r := range resources {
		var attrs struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
			continue
		}
		types = append(types, ServiceType{ID: r.ID, Name: attrs.Name})
	}
	return types, nil
}

// ListPlans returns plans for a service type, with optional date filtering.
func (c *Client) ListPlans(ctx context.Context, serviceTypeID string, opts ListPlansOpts) ([]Plan, error) {
	path := fmt.Sprintf("/services/v2/service_types/%s/plans", serviceTypeID)
	params := url.Values{
		"order": {"sort_date"},
	}

	if opts.AfterDate != nil {
		params.Set("filter", "after")
		params.Set("after", opts.AfterDate.Format("2006-01-02"))
	}
	if opts.BeforeDate != nil {
		params.Set("filter", "before")
		params.Set("before", opts.BeforeDate.Format("2006-01-02"))
	}

	var allPlans []Plan
	currentPath := path
	isFullURL := false

	for {
		var body []byte
		var err error
		if isFullURL {
			body, err = c.getFullURL(ctx, currentPath)
		} else {
			body, err = c.get(ctx, currentPath, params)
		}
		if err != nil {
			return nil, err
		}

		var resp jsonAPIResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parsing plans: %w", err)
		}

		var resources []jsonAPIResource
		if err := json.Unmarshal(resp.Data, &resources); err != nil {
			return nil, fmt.Errorf("parsing plans data: %w", err)
		}

		for _, r := range resources {
			plan, err := parsePlan(r)
			if err != nil {
				c.logf("skipping plan %s: %v", r.ID, err)
				continue
			}
			allPlans = append(allPlans, plan)
		}

		if resp.Links.Next == "" {
			break
		}
		currentPath = resp.Links.Next
		isFullURL = true
		params = nil
	}

	return allPlans, nil
}

// FindPlanByDate finds the plan closest to the given date.
func (c *Client) FindPlanByDate(ctx context.Context, serviceTypeID string, date time.Time) (*Plan, error) {
	// Search a window around the target date
	before := date.AddDate(0, 0, -7)
	after := date.AddDate(0, 0, 7)
	plans, err := c.ListPlans(ctx, serviceTypeID, ListPlansOpts{
		AfterDate:  &before,
		BeforeDate: &after,
	})
	if err != nil {
		return nil, err
	}

	if len(plans) == 0 {
		// Try broader search to show available dates
		allPlans, _ := c.ListPlans(ctx, serviceTypeID, ListPlansOpts{})
		if len(allPlans) > 0 {
			var dates []string
			for _, p := range allPlans {
				dates = append(dates, p.SortDate.Format("2006-01-02"))
			}
			if len(dates) > 10 {
				dates = dates[:10]
			}
			return nil, fmt.Errorf("no plan found near %s. Available dates: %s",
				date.Format("2006-01-02"), strings.Join(dates, ", "))
		}
		return nil, fmt.Errorf("no plans found for this service type")
	}

	// Find exact or closest match
	var best *Plan
	bestDiff := time.Duration(1<<63 - 1)
	for i := range plans {
		diff := plans[i].SortDate.Sub(date)
		if diff < 0 {
			diff = -diff
		}
		if diff < bestDiff {
			bestDiff = diff
			best = &plans[i]
		}
	}

	if bestDiff > 24*time.Hour {
		fmt.Printf("Note: no exact match for %s, using closest plan on %s\n",
			date.Format("2006-01-02"), best.SortDate.Format("2006-01-02"))
	}

	return best, nil
}

// GetPlan fetches a single plan by ID.
func (c *Client) GetPlan(ctx context.Context, serviceTypeID, planID string) (*Plan, error) {
	path := fmt.Sprintf("/services/v2/service_types/%s/plans/%s", serviceTypeID, planID)
	body, err := c.get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data jsonAPIResource `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing plan: %w", err)
	}

	plan, err := parsePlan(resp.Data)
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

// GetPlanItems returns all items for a plan, including song/arrangement details.
func (c *Client) GetPlanItems(ctx context.Context, serviceTypeID, planID string) ([]Item, error) {
	path := fmt.Sprintf("/services/v2/service_types/%s/plans/%s/items", serviceTypeID, planID)
	params := url.Values{
		"include":  {"song,arrangement,item_notes"},
		"per_page": {"100"},
	}

	var allItems []Item
	currentPath := path
	isFullURL := false
	var allIncluded []jsonAPIResource

	for {
		var body []byte
		var err error
		if isFullURL {
			body, err = c.getFullURL(ctx, currentPath)
		} else {
			body, err = c.get(ctx, currentPath, params)
		}
		if err != nil {
			return nil, err
		}

		var resp jsonAPIResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("parsing items: %w", err)
		}

		var resources []jsonAPIResource
		if err := json.Unmarshal(resp.Data, &resources); err != nil {
			return nil, fmt.Errorf("parsing items data: %w", err)
		}

		allIncluded = append(allIncluded, resp.Included...)

		for i, r := range resources {
			item, err := parseItem(r, resp.Included)
			if err != nil {
				c.logf("skipping item %s: %v", r.ID, err)
				continue
			}
			item.Sequence = len(allItems) + i
			allItems = append(allItems, item)
		}

		if resp.Links.Next == "" {
			break
		}
		currentPath = resp.Links.Next
		isFullURL = true
		params = nil
	}

	// Sort by sequence
	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].Sequence < allItems[j].Sequence
	})

	return allItems, nil
}

// GetArrangement fetches arrangement details including lyrics.
func (c *Client) GetArrangement(ctx context.Context, songID, arrangementID string) (*Arrangement, error) {
	path := fmt.Sprintf("/services/v2/songs/%s/arrangements/%s", songID, arrangementID)
	body, err := c.get(ctx, path, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data jsonAPIResource `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parsing arrangement: %w", err)
	}

	return parseArrangement(resp.Data)
}

func parsePlan(r jsonAPIResource) (Plan, error) {
	var attrs struct {
		Title       string `json:"title"`
		SortDate    string `json:"sort_date"`
		Dates       string `json:"dates"`
		SeriesTitle string `json:"series_title"`
	}
	if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
		return Plan{}, err
	}

	sortDate, err := time.Parse("2006-01-02T15:04:05Z", attrs.SortDate)
	if err != nil {
		// Try alternative format
		sortDate, err = time.Parse(time.RFC3339, attrs.SortDate)
		if err != nil {
			return Plan{}, fmt.Errorf("parsing sort_date %q: %w", attrs.SortDate, err)
		}
	}

	return Plan{
		ID:          r.ID,
		Title:       attrs.Title,
		SortDate:    sortDate,
		Dates:       attrs.Dates,
		SeriesTitle: attrs.SeriesTitle,
	}, nil
}

func parseItem(r jsonAPIResource, included []jsonAPIResource) (Item, error) {
	var attrs struct {
		Title           string `json:"title"`
		ItemType        string `json:"item_type"`
		Description     string `json:"description"`
		HTMLDetails     string `json:"html_details"`
		Length          int    `json:"length"`
		ServicePosition string `json:"service_position"`
		Sequence        int    `json:"sequence"`
	}
	if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
		return Item{}, err
	}

	item := Item{
		ID:              r.ID,
		Title:           attrs.Title,
		ItemType:        attrs.ItemType,
		Sequence:        attrs.Sequence,
		Description:     attrs.Description,
		HTMLDetails:     attrs.HTMLDetails,
		Length:          attrs.Length,
		ServicePosition: attrs.ServicePosition,
	}

	// Resolve included song and arrangement
	if r.Relationships != nil {
		if songRel, ok := r.Relationships["song"]; ok {
			var relData jsonAPIRelData
			if json.Unmarshal(songRel.Data, &relData) == nil && relData.ID != "" {
				for _, inc := range included {
					if inc.Type == "Song" && inc.ID == relData.ID {
						song, err := parseSong(inc)
						if err == nil {
							item.Song = &song
						}
						break
					}
				}
			}
		}
		if arrRel, ok := r.Relationships["arrangement"]; ok {
			var relData jsonAPIRelData
			if json.Unmarshal(arrRel.Data, &relData) == nil && relData.ID != "" {
				for _, inc := range included {
					if inc.Type == "Arrangement" && inc.ID == relData.ID {
						arr, err := parseArrangement(inc)
						if err == nil {
							item.Arrangement = arr
						}
						break
					}
				}
			}
		}
		if notesRel, ok := r.Relationships["item_notes"]; ok {
			var noteRefs []jsonAPIRelData
			if json.Unmarshal(notesRel.Data, &noteRefs) == nil {
				for _, noteRef := range noteRefs {
					for _, inc := range included {
						if inc.Type == "ItemNote" && inc.ID == noteRef.ID {
							note, err := parseItemNote(inc)
							if err == nil {
								item.Notes = append(item.Notes, note)
							}
							break
						}
					}
				}
			}
		}
	}

	return item, nil
}

func parseSong(r jsonAPIResource) (Song, error) {
	var attrs struct {
		Title     string `json:"title"`
		Author    string `json:"author"`
		CCLI      string `json:"ccli_number"`
		Copyright string `json:"copyright"`
	}
	if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
		return Song{}, err
	}
	return Song{
		ID:        r.ID,
		Title:     attrs.Title,
		Author:    attrs.Author,
		CCLI:      attrs.CCLI,
		Copyright: attrs.Copyright,
	}, nil
}

func parseArrangement(r jsonAPIResource) (*Arrangement, error) {
	var attrs struct {
		Name          string   `json:"name"`
		Lyrics        string   `json:"lyrics"`
		ChordChart    string   `json:"chord_chart"`
		Length        int      `json:"length"`
		Sequence      []string `json:"sequence"`
		SequenceShort []string `json:"sequence_short"`
	}
	if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
		return nil, err
	}
	return &Arrangement{
		ID:            r.ID,
		Name:          attrs.Name,
		Lyrics:        attrs.Lyrics,
		ChordChart:    attrs.ChordChart,
		Length:        attrs.Length,
		Sequence:      attrs.Sequence,
		SequenceShort: attrs.SequenceShort,
	}, nil
}

func parseItemNote(r jsonAPIResource) (ItemNote, error) {
	var attrs struct {
		CategoryName string `json:"category_name"`
		Content      string `json:"content"`
	}
	if err := json.Unmarshal(r.Attributes, &attrs); err != nil {
		return ItemNote{}, err
	}
	return ItemNote{
		ID:       r.ID,
		Category: attrs.CategoryName,
		Content:  attrs.Content,
	}, nil
}
