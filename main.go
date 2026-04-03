package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
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

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
	orgName   = "" // set at build time via -X main.orgName=... for org-specific builds
)

func main() {
	serviceType := flag.String("service-type", "", "Service type name or ID")
	planID := flag.String("plan", "", "Plan ID or date (YYYY-MM-DD)")
	output := flag.String("output", "", "Output .osz file path (default: <date>-<title>.osz)")
	listServiceTypes := flag.Bool("list-service-types", false, "List available service types")
	listPlans := flag.Bool("list-plans", false, "List plans for a service type")
	dryRun := flag.Bool("dry-run", false, "Show what would be generated without creating the file")
	noHeaders := flag.Bool("no-headers", false, "Exclude header items from the generated service")
	noCache := flag.Bool("no-cache", false, "Bypass media cache and re-download all files")
	debug := flag.Bool("debug", false, "Enable debug logging")
	showVersion := flag.Bool("version", false, "Show version information")
	allPlans := flag.Bool("all", false, "Show all plans (default: recent and upcoming only)")
	cleanCache := flag.Bool("clean-cache", false, "Clean the media cache")
	cacheInfo := flag.Bool("cache-info", false, "Show cache information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: pco2olp [flags]\n\nFlags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n")
		fmt.Fprintf(os.Stderr, "  PCO_CLIENT_ID      OAuth Client ID (required)\n")
		fmt.Fprintf(os.Stderr, "  PCO_CLIENT_SECRET   OAuth Client Secret (required)\n")
		fmt.Fprintf(os.Stderr, "\n  Register an OAuth application at:\n")
		fmt.Fprintf(os.Stderr, "  https://api.planningcenteronline.com/oauth/applications\n")
		fmt.Fprintf(os.Stderr, "  Set the redirect URI to http://localhost:11019/callback\n")
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("pco2olp %s (commit: %s, built: %s)\n", version, commit, buildTime)
		if orgName != "" {
			fmt.Printf("Built for: %s\n", orgName)
		}
		os.Exit(0)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	switch {
	case *cleanCache:
		if err := runCleanCache(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case *cacheInfo:
		if err := runCacheInfo(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case *listServiceTypes:
		if err := runListServiceTypes(ctx, *debug); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case *listPlans:
		if *serviceType == "" {
			fmt.Fprintf(os.Stderr, "Error: --service-type is required with --list-plans\n")
			os.Exit(1)
		}
		if err := runListPlans(ctx, *serviceType, *allPlans, *debug); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case *dryRun:
		if *serviceType == "" || *planID == "" {
			fmt.Fprintf(os.Stderr, "Error: --service-type and --plan are required with --dry-run\n")
			os.Exit(1)
		}
		if err := runDryRun(ctx, *serviceType, *planID, *noHeaders, *debug); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		if *serviceType == "" || *planID == "" {
			fmt.Fprintf(os.Stderr, "Error: --service-type and --plan are required\n")
			fmt.Fprintf(os.Stderr, "Usage: pco2olp --service-type <name|id> --plan <id|YYYY-MM-DD> [--output <file.osz>]\n")
			fmt.Fprintf(os.Stderr, "\nOther commands:\n")
			fmt.Fprintf(os.Stderr, "  pco2olp --list-service-types\n")
			fmt.Fprintf(os.Stderr, "  pco2olp --service-type <name|id> --list-plans\n")
			fmt.Fprintf(os.Stderr, "  pco2olp --service-type <name|id> --plan <id|YYYY-MM-DD> --dry-run\n")
			os.Exit(1)
		}
		if err := runGenerate(ctx, *serviceType, *planID, *output, *noHeaders, *noCache, *debug); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func authenticate(ctx context.Context, debug bool) (*pco.Client, error) {
	tokenStore, err := auth.NewTokenStore()
	if err != nil {
		return nil, fmt.Errorf("initializing token store: %w", err)
	}

	authenticator := auth.NewAuthenticator(tokenStore)
	ts, err := authenticator.TokenSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	return pco.NewClient(ts, debug), nil
}

func runListServiceTypes(ctx context.Context, debug bool) error {
	client, err := authenticate(ctx, debug)
	if err != nil {
		return err
	}

	types, err := client.ListServiceTypes(ctx)
	if err != nil {
		return fmt.Errorf("fetching service types: %w", err)
	}

	fmt.Printf("%-10s  %s\n", "ID", "Name")
	fmt.Printf("%-10s  %s\n", "---", "----")
	for _, t := range types {
		fmt.Printf("%-10s  %s\n", t.ID, t.Name)
	}
	return nil
}

func runListPlans(ctx context.Context, serviceType string, all bool, debug bool) error {
	client, err := authenticate(ctx, debug)
	if err != nil {
		return err
	}

	st, err := resolveServiceType(ctx, client, serviceType)
	if err != nil {
		return err
	}

	opts := pco.ListPlansOpts{}
	if !all {
		// Default: show plans from 4 weeks ago onward
		cutoff := time.Now().AddDate(0, 0, -28)
		opts.AfterDate = &cutoff
	}

	plans, err := client.ListPlans(ctx, st.ID, opts)
	if err != nil {
		return fmt.Errorf("fetching plans: %w", err)
	}

	if len(plans) == 0 {
		if all {
			fmt.Printf("No plans found for %q\n", st.Name)
		} else {
			fmt.Printf("No recent plans found for %q (use --all to show all plans)\n", st.Name)
		}
		return nil
	}

	fmt.Printf("Plans for %q", st.Name)
	if !all {
		fmt.Printf(" (recent & upcoming, use --all for full history)")
	}
	fmt.Printf(":\n\n")
	// Find the next upcoming plan (first plan on or after today)
	today := time.Now().Truncate(24 * time.Hour)
	nextIdx := -1
	for i, p := range plans {
		if !p.SortDate.Before(today) {
			nextIdx = i
			break
		}
	}

	fmt.Printf("     %-10s  %-12s  %-30s  %s\n", "ID", "Date", "Title", "Series")
	fmt.Printf("     %-10s  %-12s  %-30s  %s\n", "---", "----", "-----", "------")
	for i, p := range plans {
		series := p.SeriesTitle
		if series == "" {
			series = "-"
		}
		dateStr := p.SortDate.Format("2006-01-02")
		marker := "    "
		if i == nextIdx {
			marker = " >> "
		}
		fmt.Printf("%s %-10s  %-12s  %-30s  %s\n", marker, p.ID, dateStr, p.Title, series)
	}
	return nil
}

func resolvePlan(ctx context.Context, client *pco.Client, serviceTypeID, planRef string) (*pco.Plan, error) {
	// If it looks like a date (YYYY-MM-DD), find plan by date
	if date, err := time.Parse("2006-01-02", planRef); err == nil {
		return client.FindPlanByDate(ctx, serviceTypeID, date)
	}
	// Otherwise treat as plan ID
	return client.GetPlan(ctx, serviceTypeID, planRef)
}

func runDryRun(ctx context.Context, serviceType, planRef string, noHeaders, debug bool) error {
	client, err := authenticate(ctx, debug)
	if err != nil {
		return err
	}

	st, err := resolveServiceType(ctx, client, serviceType)
	if err != nil {
		return err
	}

	plan, err := resolvePlan(ctx, client, st.ID, planRef)
	if err != nil {
		return err
	}

	items, err := client.GetPlanItems(ctx, st.ID, plan.ID)
	if err != nil {
		return fmt.Errorf("fetching plan items: %w", err)
	}

	fmt.Printf("Plan: %s", plan.Title)
	if plan.SeriesTitle != "" {
		fmt.Printf(" (%s)", plan.SeriesTitle)
	}
	fmt.Printf("\nDate: %s\n\n", plan.SortDate.Format("2006-01-02"))

	fmt.Println("Items:")
	for i, item := range items {
		if noHeaders && item.ItemType == "header" {
			continue
		}
		extra := ""
		if item.Song != nil {
			extra = fmt.Sprintf(" (%s)", item.Song.Author)
		}
		fmt.Printf("  %2d. [%-7s]  %s%s\n", i+1, item.ItemType, item.Title, extra)
	}

	attachments, err := client.GetPlanAttachments(ctx, st.ID, plan.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not fetch plan attachments: %v\n", err)
		return nil
	}
	if len(attachments) > 0 {
		fmt.Println("\nPlan attachments:")
		for _, att := range attachments {
			if att.FileSize == 0 || filepath.Ext(att.Filename) == "" {
				fmt.Printf("  - %s (link, skipped)\n", att.Filename)
			} else {
				fmt.Printf("  - %s (%s, %s)\n", att.Filename, att.ContentType, formatSize(att.FileSize))
			}
		}
	}
	return nil
}

func runGenerate(ctx context.Context, serviceType, planRef, output string, noHeaders, noCache, debug bool) error {
	client, err := authenticate(ctx, debug)
	if err != nil {
		return err
	}

	st, err := resolveServiceType(ctx, client, serviceType)
	if err != nil {
		return err
	}

	fmt.Printf("Fetching plan %s for %q...\n", planRef, st.Name)

	requestPermissions()

	plan, err := resolvePlan(ctx, client, st.ID, planRef)
	if err != nil {
		return err
	}

	items, err := client.GetPlanItems(ctx, st.ID, plan.ID)
	if err != nil {
		return fmt.Errorf("fetching plan items: %w", err)
	}

	fmt.Printf("Found plan: %s (%d items)\n", plan.Title, len(items))

	// Default output filename from plan date and title
	if output == "" {
		slug := sanitizeFilename(plan.Title)
		datePrefix := plan.SortDate.Format("2006-01-02")
		if slug != "" {
			output = datePrefix + "-" + slug + ".osz"
		} else {
			output = datePrefix + ".osz"
		}
	}

	// Filter out headers if requested
	if noHeaders {
		filtered := items[:0]
		for _, item := range items {
			if item.ItemType != "header" {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	// Initialize cache
	mediaCache, err := cache.NewCache()
	if err != nil {
		return fmt.Errorf("initializing media cache: %w", err)
	}

	// Run lyrics enrichment, item media downloads, and plan attachment downloads concurrently
	var wg sync.WaitGroup
	var itemMedia map[string]*cache.MediaFile
	var planMedia []*cache.MediaFile

	// Lyrics enrichment
	wg.Add(1)
	go func() {
		defer wg.Done()
		enrichLyrics(ctx, client, items)
	}()

	// Item media downloads (parallel within)
	wg.Add(1)
	go func() {
		defer wg.Done()
		itemMedia = downloadItemMedia(ctx, client, st.ID, plan.ID, items, noCache, mediaCache)
	}()

	// Plan-level attachment downloads (parallel within)
	wg.Add(1)
	go func() {
		defer wg.Done()
		planMedia = downloadPlanAttachments(ctx, client, st.ID, plan.ID, noCache, mediaCache)
	}()

	wg.Wait()

	// Convert presentation files to slide images if tools are available
	slidesTempDir, err := os.MkdirTemp("", "pco2olp-slides-*")
	if err != nil {
		return fmt.Errorf("creating slides temp dir: %w", err)
	}
	defer os.RemoveAll(slidesTempDir)

	convertSlides(itemMedia, planMedia, slidesTempDir)

	serviceFile := convert.PlanToServiceFile(items, itemMedia, planMedia)

	if err := serviceFile.WriteOSZ(output); err != nil {
		return fmt.Errorf("writing %s: %w", output, err)
	}

	fi, _ := os.Stat(output)
	size := "unknown size"
	if fi != nil {
		size = formatSize(fi.Size())
	}
	fmt.Printf("Written: %s (%s)\n", output, size)
	return nil
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

func downloadItemMedia(ctx context.Context, client *pco.Client, serviceTypeID, planID string, items []pco.Item, noCache bool, mediaCache *cache.Cache) map[string]*cache.MediaFile {
	// Collect media items
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

	// Download up to 4 media items concurrently
	sem := make(chan struct{}, 4)

	for _, item := range mediaItems {
		wg.Add(1)
		go func(item pco.Item) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			mf := downloadSingleItemMedia(ctx, client, serviceTypeID, planID, item, noCache, mediaCache)
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

func downloadSingleItemMedia(ctx context.Context, client *pco.Client, serviceTypeID, planID string, item pco.Item, noCache bool, mediaCache *cache.Cache) *cache.MediaFile {
	// Get the media resource for this item (provides media_type)
	mediaList, err := client.GetItemMedia(ctx, serviceTypeID, planID, item.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not fetch media for %q: %v\n", item.Title, err)
		return nil
	}

	var pcoMediaType string
	if len(mediaList) > 0 {
		pcoMediaType = mediaList[0].MediaType
	}

	// Get attachments (the actual downloadable files)
	attachments, err := client.GetItemAttachments(ctx, serviceTypeID, planID, item.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not fetch attachments for %q: %v\n", item.Title, err)
		return nil
	}

	if len(attachments) == 0 {
		return nil
	}

	att := attachments[0] // Use the first attachment
	fmt.Printf("  Fetching %s...\n", att.Filename)

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
		fmt.Printf("  [cached] %s\n", att.Filename)
	} else {
		fmt.Printf("  [downloaded] %s\n", att.Filename)
	}

	mf.PCOMediaType = pcoMediaType
	return mf
}

func downloadPlanAttachments(ctx context.Context, client *pco.Client, serviceTypeID, planID string, noCache bool, mediaCache *cache.Cache) []*cache.MediaFile {
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
		// Skip URL/link attachments — they have no downloadable file
		if att.FileSize == 0 || filepath.Ext(att.Filename) == "" {
			fmt.Printf("  Skipping %s (plan attachment is a URL/link, not a file)\n", att.Filename)
			continue
		}
		wg.Add(1)
		go func(i int, att pco.Attachment) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fmt.Printf("  Fetching %s (plan attachment)...\n", att.Filename)

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
				fmt.Printf("  [cached] %s (plan attachment)\n", att.Filename)
			} else {
				fmt.Printf("  [downloaded] %s (plan attachment)\n", att.Filename)
			}

			results <- indexedResult{index: i, mf: mf}
		}(i, att)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results, preserving original order
	collected := make([]*cache.MediaFile, len(attachments))
	for r := range results {
		collected[r.index] = r.mf
	}

	// Filter out nils (failed downloads)
	var planMedia []*cache.MediaFile
	for _, mf := range collected {
		if mf != nil {
			planMedia = append(planMedia, mf)
		}
	}
	return planMedia
}

func convertSlides(itemMedia map[string]*cache.MediaFile, planMedia []*cache.MediaFile, tempDir string) {
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
			fmt.Printf("  Converting %s to slides...\n", mf.OriginalFilename)
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
			fmt.Printf("  Converted %s (%d slides)\n", mf.OriginalFilename, len(mf.Slides))
		}(mf)
	}
	wg.Wait()
}

// isPresentationFile returns true if the media file should be converted to slides.
// Respects PCO's media type when present: audio/video/image are never presentations.
// Falls back to file extension for plan attachments (which have no PCO media type).
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

func runCleanCache() error {
	c, err := cache.NewCache()
	if err != nil {
		return err
	}
	if err := c.Clean(); err != nil {
		return fmt.Errorf("cleaning cache: %w", err)
	}
	fmt.Println("Media cache cleaned.")
	return nil
}

func runCacheInfo() error {
	c, err := cache.NewCache()
	if err != nil {
		return err
	}
	dir, count, size, err := c.Info()
	if err != nil {
		return fmt.Errorf("reading cache info: %w", err)
	}
	fmt.Printf("Cache directory: %s\n", dir)
	fmt.Printf("Cached files:    %d\n", count)
	fmt.Printf("Total size:      %s\n", formatSize(size))
	return nil
}

func resolveServiceType(ctx context.Context, client *pco.Client, input string) (*pco.ServiceType, error) {
	types, err := client.ListServiceTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching service types: %w", err)
	}

	// Check for exact ID match
	for i := range types {
		if types[i].ID == input {
			return &types[i], nil
		}
	}

	// Case-insensitive name match
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
		fmt.Fprintf(os.Stderr, "Multiple service types match %q:\n", input)
		for _, m := range matches {
			fmt.Fprintf(os.Stderr, "  - %s (ID: %s)\n", m.Name, m.ID)
		}
		return nil, fmt.Errorf("ambiguous service type %q, please be more specific", input)
	}

	fmt.Fprintf(os.Stderr, "Available service types:\n")
	for _, t := range types {
		fmt.Fprintf(os.Stderr, "  - %s (ID: %s)\n", t.Name, t.ID)
	}
	return nil, fmt.Errorf("service type %q not found", input)
}

func formatSize(bytes int64) string {
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

func sanitizeFilename(s string) string {
	// Replace characters that are problematic in filenames
	replacer := strings.NewReplacer(
		"/", "-", "\\", "-", ":", "-", "*", "", "?", "",
		"\"", "", "<", "", ">", "", "|", "",
	)
	s = replacer.Replace(s)
	s = strings.TrimSpace(s)
	// Collapse multiple spaces/dashes
	parts := strings.Fields(s)
	return strings.Join(parts, " ")
}
