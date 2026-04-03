package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/danieldonoghue/pco2olp/internal/cache"
	"github.com/danieldonoghue/pco2olp/internal/generate"
	"github.com/danieldonoghue/pco2olp/internal/pco"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
	orgName   = "" // set at build time via -X main.orgName=... for org-specific builds
)

func main() {
	// No args: launch GUI (if compiled with -tags gui) or show usage.
	if len(os.Args) == 1 {
		runGUI()
		// Reached only in the !gui build — fall through to flag.Parse() which
		// will hit the default case and print usage.
	}

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

func runListServiceTypes(ctx context.Context, debug bool) error {
	client, err := generate.Authenticate(ctx, debug)
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
	client, err := generate.Authenticate(ctx, debug)
	if err != nil {
		return err
	}

	st, err := generate.ResolveServiceType(ctx, client, serviceType)
	if err != nil {
		return err
	}

	opts := pco.ListPlansOpts{}
	if !all {
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

	today := time.Now().Truncate(24 * time.Hour)
	nextIdx := -1
	for i, p := range plans {
		if !p.SortDate.Before(today) {
			nextIdx = i
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

func runDryRun(ctx context.Context, serviceType, planRef string, noHeaders, debug bool) error {
	client, err := generate.Authenticate(ctx, debug)
	if err != nil {
		return err
	}

	st, err := generate.ResolveServiceType(ctx, client, serviceType)
	if err != nil {
		return err
	}

	plan, err := generate.ResolvePlan(ctx, client, st.ID, planRef)
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
				fmt.Printf("  - %s (%s, %s)\n", att.Filename, att.ContentType, generate.FormatSize(att.FileSize))
			}
		}
	}
	return nil
}

func runGenerate(ctx context.Context, serviceType, planRef, output string, noHeaders, noCache, debug bool) error {
	client, err := generate.Authenticate(ctx, debug)
	if err != nil {
		return err
	}

	st, err := generate.ResolveServiceType(ctx, client, serviceType)
	if err != nil {
		return err
	}

	plan, err := generate.ResolvePlan(ctx, client, st.ID, planRef)
	if err != nil {
		return err
	}

	cfg := generate.Config{
		ServiceTypeID: st.ID,
		PlanID:        plan.ID,
		OutputPath:    output,
		NoHeaders:     noHeaders,
		NoCache:       noCache,
		Debug:         debug,
	}

	_, err = generate.Run(ctx, client, cfg, func(msg string) {
		fmt.Println(msg)
	})
	return err
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
	fmt.Printf("Total size:      %s\n", generate.FormatSize(size))
	return nil
}
