package generate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/danieldonoghue/pco2olp/internal/auth"
	"github.com/danieldonoghue/pco2olp/internal/cache"
	"github.com/danieldonoghue/pco2olp/internal/convert"
	"github.com/danieldonoghue/pco2olp/internal/pco"
	"github.com/danieldonoghue/pco2olp/internal/slides"
)

// ProgressFunc is called during generation to report status messages.
// It must be safe to call from multiple goroutines.
type ProgressFunc func(message string)

// Config holds options for a generate run.
type Config struct {
	ServiceTypeID string
	PlanID        string
	OutputPath    string
	NoHeaders     bool
	NoCache       bool
	Debug         bool
}

// Authenticate creates an authenticated PCO API client.
func Authenticate(ctx context.Context, debug bool) (*pco.Client, error) {
	tokenStore, err := auth.NewTokenStore()
	if err != nil {
		return nil, fmt.Errorf("initializing token store: %w", err)
	}
	authenticator := auth.NewAuthenticator(tokenStore)
	ts, err := authenticator.TokenSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	// Eagerly probe the token so an expired refresh token triggers the browser
	// flow now (during startup) rather than mid-operation.
	if _, err := ts.Token(); err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	return pco.NewClient(ts, debug), nil
}

// ResolveServiceType finds a service type by name or ID.
func ResolveServiceType(ctx context.Context, client *pco.Client, input string) (*pco.ServiceType, error) {
	types, err := client.ListServiceTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching service types: %w", err)
	}

	for i := range types {
		if types[i].ID == input {
			return &types[i], nil
		}
	}

	inputLower := strings.ToLower(input)
	var matches []pco.ServiceType
	for _, t := range types {
		if strings.ToLower(t.Name) == inputLower {
			return &t, nil
		}
		if strings.Contains(strings.ToLower(t.Name), inputLower) {
			matches = append(matches, t)
		}
	}

	if len(matches) == 1 {
		return &matches[0], nil
	}

	if len(matches) > 1 {
		names := make([]string, len(matches))
		for i, m := range matches {
			names[i] = fmt.Sprintf("%s (ID: %s)", m.Name, m.ID)
		}
		return nil, fmt.Errorf("ambiguous service type %q — matches: %s", input, strings.Join(names, ", "))
	}

	return nil, fmt.Errorf("service type %q not found", input)
}

// ResolvePlan finds a plan by ID or YYYY-MM-DD date string.
func ResolvePlan(ctx context.Context, client *pco.Client, serviceTypeID, planRef string) (*pco.Plan, error) {
	if date, err := time.Parse("2006-01-02", planRef); err == nil {
		return client.FindPlanByDate(ctx, serviceTypeID, date)
	}
	return client.GetPlan(ctx, serviceTypeID, planRef)
}

// Run executes the full generation pipeline.
// cfg.ServiceTypeID and cfg.PlanID must already be resolved (numeric IDs, not names/dates).
// progress may be nil.
func Run(ctx context.Context, client *pco.Client, cfg Config, progress ProgressFunc) (string, error) {
	report := func(msg string) {
		if progress != nil {
			progress(msg)
		}
	}

	report(fmt.Sprintf("Fetching plan %s...", cfg.PlanID))
	requestPermissions()

	plan, err := client.GetPlan(ctx, cfg.ServiceTypeID, cfg.PlanID)
	if err != nil {
		return "", fmt.Errorf("fetching plan: %w", err)
	}

	items, err := client.GetPlanItems(ctx, cfg.ServiceTypeID, plan.ID)
	if err != nil {
		return "", fmt.Errorf("fetching plan items: %w", err)
	}

	report(fmt.Sprintf("Found plan: %s (%d items)", plan.Title, len(items)))

	output := cfg.OutputPath
	if output == "" {
		output = DefaultOutputPath(plan)
	}

	if cfg.NoHeaders {
		filtered := items[:0]
		for _, item := range items {
			if item.ItemType != "header" {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	mediaCache, err := cache.NewCache()
	if err != nil {
		return "", fmt.Errorf("initializing media cache: %w", err)
	}

	var wg sync.WaitGroup
	var itemMedia map[string]*cache.MediaFile
	var planMedia []*cache.MediaFile

	wg.Add(1)
	go func() {
		defer wg.Done()
		enrichLyrics(ctx, client, items)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		itemMedia = downloadItemMedia(ctx, client, cfg.ServiceTypeID, plan.ID, items, cfg.NoCache, mediaCache, report)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		planMedia = downloadPlanAttachments(ctx, client, cfg.ServiceTypeID, plan.ID, cfg.NoCache, mediaCache, report)
	}()

	wg.Wait()

	slidesTempDir, err := os.MkdirTemp("", "pco2olp-slides-*")
	if err != nil {
		return "", fmt.Errorf("creating slides temp dir: %w", err)
	}
	defer os.RemoveAll(slidesTempDir)

	convertSlides(itemMedia, planMedia, slidesTempDir, report)

	serviceFile := convert.PlanToServiceFile(items, itemMedia, planMedia)

	if err := serviceFile.WriteOSZ(output); err != nil {
		return "", fmt.Errorf("writing %s: %w", output, err)
	}

	fi, _ := os.Stat(output)
	size := "unknown size"
	if fi != nil {
		size = FormatSize(fi.Size())
	}
	report(fmt.Sprintf("Written: %s (%s)", output, size))
	return output, nil
}

// DefaultOutputPath returns the default output filename for a plan.
func DefaultOutputPath(plan *pco.Plan) string {
	slug := SanitizeFilename(plan.Title)
	datePrefix := plan.SortDate.Format("2006-01-02")
	if slug != "" {
		return datePrefix + "-" + slug + ".osz"
	}
	return datePrefix + ".osz"
}

// FormatSize formats a byte count as a human-readable string.
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMG"[exp])
}

// SanitizeFilename removes characters that are problematic in filenames.
func SanitizeFilename(s string) string {
	replacer := strings.NewReplacer(
		"/", "-", "\\", "-", ":", "-", "*", "", "?", "",
		"\"", "", "<", "", ">", "", "|", "",
	)
	s = replacer.Replace(s)
	s = strings.TrimSpace(s)
	return strings.Join(strings.Fields(s), " ")
}

func enrichLyrics(ctx context.Context, client *pco.Client, items []pco.Item) {
	for i := range items {
		if items[i].ItemType == "song" && items[i].Arrangement != nil && items[i].Arrangement.Lyrics == "" {
			arr, err := client.GetArrangement(ctx, items[i].Song.ID, items[i].Arrangement.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not fetch lyrics for %q: %v\n", items[i].Title, err)
				continue
			}
			items[i].Arrangement = arr
		}
	}
}

func downloadItemMedia(ctx context.Context, client *pco.Client, serviceTypeID, planID string, items []pco.Item, noCache bool, mediaCache *cache.Cache, report ProgressFunc) map[string]*cache.MediaFile {
	var mediaItems []pco.Item
	for _, item := range items {
		if item.ItemType == "media" {
			mediaItems = append(mediaItems, item)
		}
	}
	if len(mediaItems) == 0 {
		return nil
	}

	var mu sync.Mutex
	mediaMap := make(map[string]*cache.MediaFile)
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)

	for _, item := range mediaItems {
		wg.Add(1)
		go func(item pco.Item) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			mf := downloadSingleItemMedia(ctx, client, serviceTypeID, planID, item, noCache, mediaCache, report)
			if mf != nil {
				mu.Lock()
				mediaMap[item.ID] = mf
				mu.Unlock()
			}
		}(item)
	}
	wg.Wait()
	return mediaMap
}

func downloadSingleItemMedia(ctx context.Context, client *pco.Client, serviceTypeID, planID string, item pco.Item, noCache bool, mediaCache *cache.Cache, report ProgressFunc) *cache.MediaFile {
	mediaList, err := client.GetItemMedia(ctx, serviceTypeID, planID, item.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not fetch media for %q: %v\n", item.Title, err)
		return nil
	}

	var pcoMediaType string
	if len(mediaList) > 0 {
		pcoMediaType = mediaList[0].MediaType
	}

	attachments, err := client.GetItemAttachments(ctx, serviceTypeID, planID, item.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not fetch attachments for %q: %v\n", item.Title, err)
		return nil
	}
	if len(attachments) == 0 {
		return nil
	}

	att := attachments[0]
	report(fmt.Sprintf("Fetching %s...", att.Filename))

	mf, err := mediaCache.EnsureCached(att.ID, att.UpdatedAt, att.Filename, att.ContentType, att.FileSize, noCache, func(w io.Writer) (int64, error) {
		downloadURL, err := client.OpenAttachment(ctx, att.ID)
		if err != nil {
			return 0, err
		}
		return client.DownloadFile(ctx, downloadURL, w)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not download %q: %v\n", att.Filename, err)
		return nil
	}

	if mf.CacheHit {
		report(fmt.Sprintf("[cached] %s", att.Filename))
	} else {
		report(fmt.Sprintf("[downloaded] %s", att.Filename))
	}

	mf.PCOMediaType = pcoMediaType
	return mf
}

func downloadPlanAttachments(ctx context.Context, client *pco.Client, serviceTypeID, planID string, noCache bool, mediaCache *cache.Cache, report ProgressFunc) []*cache.MediaFile {
	attachments, err := client.GetPlanAttachments(ctx, serviceTypeID, planID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not fetch plan attachments: %v\n", err)
		return nil
	}
	if len(attachments) == 0 {
		return nil
	}

	type indexedResult struct {
		index int
		mf    *cache.MediaFile
	}

	results := make(chan indexedResult, len(attachments))
	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup

	for i, att := range attachments {
		if att.FileSize == 0 || filepath.Ext(att.Filename) == "" {
			report(fmt.Sprintf("Skipping %s (URL/link, not a file)", att.Filename))
			continue
		}
		wg.Add(1)
		go func(i int, att pco.Attachment) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			report(fmt.Sprintf("Fetching %s (plan attachment)...", att.Filename))

			mf, err := mediaCache.EnsureCached(att.ID, att.UpdatedAt, att.Filename, att.ContentType, att.FileSize, noCache, func(w io.Writer) (int64, error) {
				downloadURL, err := client.OpenAttachment(ctx, att.ID)
				if err != nil {
					return 0, err
				}
				return client.DownloadFile(ctx, downloadURL, w)
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not download plan attachment %q: %v\n", att.Filename, err)
				return
			}

			if mf.CacheHit {
				report(fmt.Sprintf("[cached] %s (plan attachment)", att.Filename))
			} else {
				report(fmt.Sprintf("[downloaded] %s (plan attachment)", att.Filename))
			}
			results <- indexedResult{index: i, mf: mf}
		}(i, att)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	collected := make([]*cache.MediaFile, len(attachments))
	for r := range results {
		collected[r.index] = r.mf
	}

	var planMedia []*cache.MediaFile
	for _, mf := range collected {
		if mf != nil {
			planMedia = append(planMedia, mf)
		}
	}
	return planMedia
}

func convertSlides(itemMedia map[string]*cache.MediaFile, planMedia []*cache.MediaFile, tempDir string, report ProgressFunc) {
	var allMedia []*cache.MediaFile
	for _, mf := range itemMedia {
		if mf != nil && isPresentationFile(mf) {
			allMedia = append(allMedia, mf)
		}
	}
	for _, mf := range planMedia {
		if mf != nil && isPresentationFile(mf) {
			allMedia = append(allMedia, mf)
		}
	}
	if len(allMedia) == 0 {
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 2)

	for _, mf := range allMedia {
		wg.Add(1)
		go func(mf *cache.MediaFile) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			slideDir := filepath.Join(tempDir, mf.SHA256)
			if err := os.MkdirAll(slideDir, 0700); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not create slide dir for %q: %v\n", mf.OriginalFilename, err)
				return
			}
			report(fmt.Sprintf("Converting %s to slides...", mf.OriginalFilename))
			pngs, err := slides.ConvertToPNGs(mf.LocalPath, slideDir)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not convert %q to slides: %v\n", mf.OriginalFilename, err)
				return
			}
			if len(pngs) == 0 {
				fmt.Fprintf(os.Stderr, "  No slide converter available for %s — using media item fallback\n", mf.Extension)
				return
			}
			converted := make([]cache.Slide, 0, len(pngs))
			for _, path := range pngs {
				hash, err := sha256File(path)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: could not hash slide %q: %v\n", path, err)
					continue
				}
				converted = append(converted, cache.Slide{LocalPath: path, SHA256: hash})
			}
			mf.Slides = converted
			report(fmt.Sprintf("Converted %s (%d slides)", mf.OriginalFilename, len(mf.Slides)))
		}(mf)
	}
	wg.Wait()
}

func isPresentationFile(mf *cache.MediaFile) bool {
	switch mf.PCOMediaType {
	case "video", "audio", "image", "background_image":
		return false
	}
	return slides.IsPresentationType(mf.Extension)
}

func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
