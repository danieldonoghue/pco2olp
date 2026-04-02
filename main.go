package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/danield/pco2olp/internal/auth"
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
	date := flag.String("date", "", "Plan date (YYYY-MM-DD)")
	output := flag.String("output", "service.osz", "Output .osz file path")
	listServiceTypes := flag.Bool("list-service-types", false, "List available service types")
	listPlans := flag.Bool("list-plans", false, "List plans for a service type")
	dryRun := flag.Bool("dry-run", false, "Show what would be generated without creating the file")
	includeMedia := flag.Bool("include-media", false, "Include media attachments")
	debug := flag.Bool("debug", false, "Enable debug logging")
	showVersion := flag.Bool("version", false, "Show version information")
	cleanCache := flag.Bool("clean-cache", false, "Clean the media cache")
	cacheInfo := flag.Bool("cache-info", false, "Show cache information")

	flag.Parse()

	// Suppress unused variable warnings for Phase 2 flags
	_ = includeMedia
	_ = cleanCache
	_ = cacheInfo

	if *showVersion {
		fmt.Printf("pco2olp %s (commit: %s, built: %s)\n", version, commit, buildTime)
		os.Exit(0)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	switch {
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
		if err := runListPlans(ctx, *serviceType, *debug); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case *dryRun:
		if *serviceType == "" || *date == "" {
			fmt.Fprintf(os.Stderr, "Error: --service-type and --date are required with --dry-run\n")
			os.Exit(1)
		}
		if err := runDryRun(ctx, *serviceType, *date, *debug); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		if *serviceType == "" || *date == "" {
			fmt.Fprintf(os.Stderr, "Error: --service-type and --date are required\n")
			fmt.Fprintf(os.Stderr, "Usage: pco2olp --service-type <name|id> --date <YYYY-MM-DD> [--output <file.osz>]\n")
			fmt.Fprintf(os.Stderr, "\nOther commands:\n")
			fmt.Fprintf(os.Stderr, "  pco2olp --list-service-types\n")
			fmt.Fprintf(os.Stderr, "  pco2olp --service-type <name|id> --list-plans\n")
			fmt.Fprintf(os.Stderr, "  pco2olp --service-type <name|id> --date <YYYY-MM-DD> --dry-run\n")
			os.Exit(1)
		}
		if err := runGenerate(ctx, *serviceType, *date, *output, *debug); err != nil {
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

func runListPlans(ctx context.Context, serviceType string, debug bool) error {
	client, err := authenticate(ctx, debug)
	if err != nil {
		return err
	}

	st, err := resolveServiceType(ctx, client, serviceType)
	if err != nil {
		return err
	}

	plans, err := client.ListPlans(ctx, st.ID, pco.ListPlansOpts{})
	if err != nil {
		return fmt.Errorf("fetching plans: %w", err)
	}

	if len(plans) == 0 {
		fmt.Printf("No plans found for %q\n", st.Name)
		return nil
	}

	fmt.Printf("Plans for %q:\n\n", st.Name)
	fmt.Printf("%-10s  %-12s  %-30s  %s\n", "ID", "Date", "Title", "Series")
	fmt.Printf("%-10s  %-12s  %-30s  %s\n", "---", "----", "-----", "------")
	for _, p := range plans {
		series := p.SeriesTitle
		if series == "" {
			series = "-"
		}
		dateStr := p.SortDate.Format("2006-01-02")
		fmt.Printf("%-10s  %-12s  %-30s  %s\n", p.ID, dateStr, p.Title, series)
	}
	return nil
}

func runDryRun(ctx context.Context, serviceType, dateStr string, debug bool) error {
	client, err := authenticate(ctx, debug)
	if err != nil {
		return err
	}

	st, err := resolveServiceType(ctx, client, serviceType)
	if err != nil {
		return err
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return fmt.Errorf("invalid date format %q (expected YYYY-MM-DD): %w", dateStr, err)
	}

	plan, err := client.FindPlanByDate(ctx, st.ID, date)
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
		extra := ""
		if item.Song != nil {
			extra = fmt.Sprintf(" (%s)", item.Song.Author)
		}
		fmt.Printf("  %2d. [%-7s]  %s%s\n", i+1, item.ItemType, item.Title, extra)
	}
	return nil
}

func runGenerate(ctx context.Context, serviceType, dateStr, output string, debug bool) error {
	client, err := authenticate(ctx, debug)
	if err != nil {
		return err
	}

	st, err := resolveServiceType(ctx, client, serviceType)
	if err != nil {
		return err
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return fmt.Errorf("invalid date format %q (expected YYYY-MM-DD): %w", dateStr, err)
	}

	fmt.Printf("Fetching plan for %q on %s...\n", st.Name, dateStr)

	plan, err := client.FindPlanByDate(ctx, st.ID, date)
	if err != nil {
		return err
	}

	items, err := client.GetPlanItems(ctx, st.ID, plan.ID)
	if err != nil {
		return fmt.Errorf("fetching plan items: %w", err)
	}

	fmt.Printf("Found plan: %s (%d items)\n", plan.Title, len(items))

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

	serviceFile := convert.PlanToServiceFile(items)

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
