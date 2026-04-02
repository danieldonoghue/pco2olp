package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/danield/pco2olp/internal/auth"
	"github.com/danield/pco2olp/internal/cache"
	"github.com/danield/pco2olp/internal/convert"
	"github.com/danield/pco2olp/internal/pco"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	serviceType := flag.String("service-type", "", "Service type name or ID")
	planID := flag.String("plan", "", "Plan ID (alternative to --date)")
	date := flag.String("date", "", "Plan date (YYYY-MM-DD)")
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
		if *serviceType == "" || (*date == "" && *planID == "") {
			fmt.Fprintf(os.Stderr, "Error: --service-type and --date (or --plan) are required with --dry-run\n")
			os.Exit(1)
		}
		if err := runDryRun(ctx, *serviceType, *date, *planID, *noHeaders, *debug); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		if *serviceType == "" || (*date == "" && *planID == "") {
			fmt.Fprintf(os.Stderr, "Error: --service-type and --date (or --plan) are required\n")
			fmt.Fprintf(os.Stderr, "Usage: pco2olp --service-type <name|id> --date <YYYY-MM-DD> [--output <file.osz>]\n")
			fmt.Fprintf(os.Stderr, "       pco2olp --service-type <name|id> --plan <id> [--output <file.osz>]\n")
			fmt.Fprintf(os.Stderr, "\nOther commands:\n")
			fmt.Fprintf(os.Stderr, "  pco2olp --list-service-types\n")
			fmt.Fprintf(os.Stderr, "  pco2olp --service-type <name|id> --list-plans\n")
			fmt.Fprintf(os.Stderr, "  pco2olp --service-type <name|id> --date <YYYY-MM-DD> --dry-run\n")
			os.Exit(1)
		}
		if err := runGenerate(ctx, *serviceType, *date, *planID, *output, *noHeaders, *noCache, *debug); err != nil {
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

func resolvePlan(ctx context.Context, client *pco.Client, serviceTypeID, dateStr, planIDStr string) (*pco.Plan, error) {
	if planIDStr != "" {
		return client.GetPlan(ctx, serviceTypeID, planIDStr)
	}
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date format %q (expected YYYY-MM-DD): %w", dateStr, err)
	}
	return client.FindPlanByDate(ctx, serviceTypeID, date)
}

func runDryRun(ctx context.Context, serviceType, dateStr, planIDStr string, noHeaders, debug bool) error {
	client, err := authenticate(ctx, debug)
	if err != nil {
		return err
	}

	st, err := resolveServiceType(ctx, client, serviceType)
	if err != nil {
		return err
	}

	plan, err := resolvePlan(ctx, client, st.ID, dateStr, planIDStr)
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
	return nil
}

func runGenerate(ctx context.Context, serviceType, dateStr, planIDStr, output string, noHeaders, noCache, debug bool) error {
	client, err := authenticate(ctx, debug)
	if err != nil {
		return err
	}

	st, err := resolveServiceType(ctx, client, serviceType)
	if err != nil {
		return err
	}

	if planIDStr != "" {
		fmt.Printf("Fetching plan %s for %q...\n", planIDStr, st.Name)
	} else {
		fmt.Printf("Fetching plan for %q on %s...\n", st.Name, dateStr)
	}

	plan, err := resolvePlan(ctx, client, st.ID, dateStr, planIDStr)
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

	// Enrich items with arrangement lyrics where needed
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

	// Download media
	mediaCache, err := cache.NewCache()
	if err != nil {
		return fmt.Errorf("initializing media cache: %w", err)
	}

	itemMedia := downloadItemMedia(ctx, client, st.ID, plan.ID, items, noCache, mediaCache)
	planMedia := downloadPlanAttachments(ctx, client, st.ID, plan.ID, noCache, mediaCache)

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

func downloadItemMedia(ctx context.Context, client *pco.Client, serviceTypeID, planID string, items []pco.Item, noCache bool, mediaCache *cache.Cache) map[string]*cache.MediaFile {
	mediaMap := make(map[string]*cache.MediaFile)

	for _, item := range items {
		if item.ItemType != "media" {
			continue
		}

		// Get the media resource for this item (provides media_type)
		mediaList, err := client.GetItemMedia(ctx, serviceTypeID, planID, item.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not fetch media for %q: %v\n", item.Title, err)
			continue
		}

		var pcoMediaType string
		if len(mediaList) > 0 {
			pcoMediaType = mediaList[0].MediaType
		}

		// Get attachments (the actual downloadable files)
		attachments, err := client.GetItemAttachments(ctx, serviceTypeID, planID, item.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not fetch attachments for %q: %v\n", item.Title, err)
			continue
		}

		if len(attachments) == 0 {
			continue
		}

		att := attachments[0] // Use the first attachment
		fmt.Printf("  Downloading %s...\n", att.Filename)

		mf, err := mediaCache.EnsureCached(att.ID, att.UpdatedAt, att.Filename, att.ContentType, att.FileSize, noCache, func(w io.Writer) (int64, error) {
			downloadURL, err := client.OpenAttachment(ctx, att.ID)
			if err != nil {
				return 0, err
			}
			return client.DownloadFile(ctx, downloadURL, w)
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not download %q: %v\n", att.Filename, err)
			continue
		}

		mf.PCOMediaType = pcoMediaType
		mediaMap[item.ID] = mf
	}

	return mediaMap
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

	var planMedia []*cache.MediaFile
	for _, att := range attachments {
		fmt.Printf("  Downloading plan attachment %s...\n", att.Filename)

		mf, err := mediaCache.EnsureCached(att.ID, att.UpdatedAt, att.Filename, att.ContentType, att.FileSize, noCache, func(w io.Writer) (int64, error) {
			downloadURL, err := client.OpenAttachment(ctx, att.ID)
			if err != nil {
				return 0, err
			}
			return client.DownloadFile(ctx, downloadURL, w)
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not download plan attachment %q: %v\n", att.Filename, err)
			continue
		}

		planMedia = append(planMedia, mf)
	}

	return planMedia
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
