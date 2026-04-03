//go:build gui

package gui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"github.com/danieldonoghue/pco2olp/internal/generate"
	"github.com/danieldonoghue/pco2olp/internal/pco"
)

type mainWindow struct {
	win     fyne.Window
	app     fyne.App
	version string
	orgName string
	client  *pco.Client

	serviceTypes    []pco.ServiceType
	plans           []pco.Plan
	planItems       []pco.Item
	planAttachments []pco.Attachment

	selectedST   *pco.ServiceType
	selectedPlan *pco.Plan

	showAllPlans bool
	noHeaders    bool // true = exclude headers from generated service
	noCache      bool

	// widgets
	stSelect        *widget.Select
	planSelect      *widget.Select
	itemList        *widget.List
	attachmentList  *widget.List
	outputEntry     *widget.Entry
	genButton       *widget.Button
	statusLabel     *widget.Label
}

func newMainWindow(a fyne.App, title, version, orgName string) fyne.Window {
	w := a.NewWindow(title)
	s := &mainWindow{
		win:       w,
		app:       a,
		version:   version,
		orgName:   orgName,
		noHeaders: true, // default: exclude headers
	}
	s.setupMenu()
	w.SetContent(s.build())
	go s.authenticate(context.Background(), nil)
	return w
}

func (s *mainWindow) build() fyne.CanvasObject {
	// ── Service type selector ─────────────────────────────────────────────
	s.stSelect = widget.NewSelect(nil, func(selected string) {
		s.onServiceTypeSelected(selected)
	})
	s.stSelect.PlaceHolder = "Connecting to Planning Center…"
	s.stSelect.Disable()

	// ── Plan selector ─────────────────────────────────────────────────────
	s.planSelect = widget.NewSelect(nil, func(selected string) {
		s.onPlanSelected(selected)
	})
	s.planSelect.PlaceHolder = "Select a service type first"
	s.planSelect.Disable()

	allPlansCheck := widget.NewCheck("Show all", func(v bool) {
		s.showAllPlans = v
		if s.selectedST != nil {
			go s.loadPlans(context.Background())
		}
	})

	planRow := container.NewBorder(nil, nil, nil, allPlansCheck, s.planSelect)

	form := widget.NewForm(
		widget.NewFormItem("Service type", s.stSelect),
		widget.NewFormItem("Plan", planRow),
	)

	// ── Items list ────────────────────────────────────────────────────────
	s.itemList = widget.NewList(
		func() int { return len(s.planItems) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(iconItem),
				widget.NewLabel("Item title placeholder"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(s.planItems) {
				return
			}
			row := obj.(*fyne.Container)
			it := s.planItems[id]
			row.Objects[0].(*widget.Icon).SetResource(itemTypeIcon(it.ItemType))
			row.Objects[1].(*widget.Label).SetText(it.Title)
		},
	)

	itemsLabel := widget.NewLabel("Service items:")
	itemsLabel.TextStyle = fyne.TextStyle{Bold: true}
	itemsSection := container.NewBorder(itemsLabel, nil, nil, nil, s.itemList)

	// ── Plan attachments list ─────────────────────────────────────────────
	s.attachmentList = widget.NewList(
		func() int { return len(s.planAttachments) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(iconAttachment),
				widget.NewLabel("attachment placeholder"),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id >= len(s.planAttachments) {
				return
			}
			row := obj.(*fyne.Container)
			row.Objects[1].(*widget.Label).SetText(s.planAttachments[id].Filename)
		},
	)

	attachmentsLabel := widget.NewLabel("Plan attachments:")
	attachmentsLabel.TextStyle = fyne.TextStyle{Bold: true}
	attachmentsSection := container.NewBorder(attachmentsLabel, nil, nil, nil, s.attachmentList)

	centerSplit := container.NewVSplit(itemsSection, attachmentsSection)
	centerSplit.Offset = 0.7

	// ── Output path ──────────────────────────────────────────────────────
	s.outputEntry = widget.NewEntry()
	s.outputEntry.SetPlaceHolder("Auto-generated from plan date and title")
	outputRow := container.NewBorder(nil, nil, nil,
		widget.NewButton("Browse…", s.browseOutput),
		s.outputEntry,
	)

	// ── Options ──────────────────────────────────────────────────────────
	// noHeaders defaults true, so "Include headers" is unchecked by default.
	headersCheck := widget.NewCheck("Include headers", func(v bool) {
		s.noHeaders = !v
	})
	headersCheck.SetChecked(false)

	noCacheCheck := widget.NewCheck("Re-download media", func(v bool) {
		s.noCache = v
	})

	// ── Generate button ───────────────────────────────────────────────────
	s.genButton = widget.NewButton("Generate Service File", s.runGenerate)
	s.genButton.Importance = widget.HighImportance
	s.genButton.Disable()

	// ── Status label ──────────────────────────────────────────────────────
	s.statusLabel = widget.NewLabel("Connecting to Planning Center…")
	s.statusLabel.Alignment = fyne.TextAlignCenter

	// ── Assemble ─────────────────────────────────────────────────────────
	bottom := container.NewVBox(
		widget.NewSeparator(),
		widget.NewForm(widget.NewFormItem("Output", outputRow)),
		container.NewHBox(headersCheck, noCacheCheck),
		s.genButton,
		s.statusLabel,
	)
	return container.NewBorder(nil, bottom, nil, nil,
		container.NewBorder(form, nil, nil, nil, centerSplit),
	)
}

// ── Menu ──────────────────────────────────────────────────────────────────────

func (s *mainWindow) setupMenu() {
	settingsItem := fyne.NewMenuItem("Settings…", func() {
		showSettingsDialog(s.win, s.app, s.reauthenticate, s.client != nil)
	})
	settingsItem.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyComma,
		Modifier: fyne.KeyModifierSuper,
	}

	appMenu := fyne.NewMenu("", settingsItem)
	s.win.SetMainMenu(fyne.NewMainMenu(appMenu))
}

// ── Auth & data loading ───────────────────────────────────────────────────────

func (s *mainWindow) authenticate(ctx context.Context, onDone func(error)) {
	fyne.Do(func() { s.statusLabel.SetText("Opening browser for authentication…") })

	prefs := s.app.Preferences()
	if id := prefs.String("clientID"); id != "" {
		os.Setenv("PCO_CLIENT_ID", id)
	}
	if secret := prefs.String("clientSecret"); secret != "" {
		os.Setenv("PCO_CLIENT_SECRET", secret)
	}

	client, err := generate.Authenticate(ctx, false)
	if err != nil {
		fyne.Do(func() {
			s.statusLabel.SetText("Authentication failed")
			dialog.ShowError(
				fmt.Errorf("authentication failed: %w\n\nOpen Settings to enter your PCO Client ID and Secret.", err),
				s.win,
			)
		})
		if onDone != nil {
			onDone(err)
		}
		return
	}
	s.client = client
	if onDone != nil {
		onDone(nil)
	}
	fyne.Do(func() { s.statusLabel.SetText("Loading service types…") })
	s.loadServiceTypes(ctx)
}

func (s *mainWindow) reauthenticate(onDone func(error)) {
	s.client = nil
	fyne.Do(func() {
		s.stSelect.Options = nil
		s.stSelect.PlaceHolder = "Connecting to Planning Center…"
		s.stSelect.Disable()
		s.stSelect.Refresh()
		s.planSelect.Options = nil
		s.planSelect.PlaceHolder = "Select a service type first"
		s.planSelect.Disable()
		s.planSelect.Refresh()
		s.planItems = nil
		s.planAttachments = nil
		s.itemList.Refresh()
		s.attachmentList.Refresh()
		s.genButton.Disable()
	})
	go s.authenticate(context.Background(), onDone)
}

func (s *mainWindow) loadServiceTypes(ctx context.Context) {
	types, err := s.client.ListServiceTypes(ctx)
	if err != nil {
		fyne.Do(func() {
			s.statusLabel.SetText("Failed to load service types")
			dialog.ShowError(err, s.win)
		})
		return
	}
	s.serviceTypes = types
	names := make([]string, len(types))
	for i, t := range types {
		names[i] = t.Name
	}
	fyne.Do(func() {
		s.stSelect.Options = names
		s.stSelect.PlaceHolder = "Select a service type"
		s.stSelect.Enable()
		s.stSelect.Refresh()
		s.statusLabel.SetText("Select a service type to continue")
	})
}

func (s *mainWindow) onServiceTypeSelected(name string) {
	s.selectedST = nil
	for i := range s.serviceTypes {
		if s.serviceTypes[i].Name == name {
			s.selectedST = &s.serviceTypes[i]
			break
		}
	}
	if s.selectedST == nil {
		return
	}
	s.plans = nil
	s.planItems = nil
	s.planAttachments = nil
	s.selectedPlan = nil
	s.genButton.Disable()
	s.itemList.Refresh()
	s.attachmentList.Refresh()
	s.planSelect.Options = nil
	s.planSelect.PlaceHolder = "Loading plans…"
	s.planSelect.Disable()
	s.planSelect.Refresh()
	go s.loadPlans(context.Background())
}

// showSpinner creates and shows a small progress dialog on the UI thread,
// returning it so the caller can hide it via fyne.Do when done.
// Must be called from a non-UI goroutine (uses fyne.DoAndWait).
func (s *mainWindow) showSpinner(title, message string) *dialog.CustomDialog {
	var d *dialog.CustomDialog
	fyne.DoAndWait(func() {
		spinner := widget.NewProgressBarInfinite()
		label := widget.NewLabel(message)
		label.Alignment = fyne.TextAlignCenter
		d = dialog.NewCustomWithoutButtons(title, container.NewVBox(spinner, label), s.win)
		d.Resize(fyne.NewSize(320, 100))
		d.Show()
	})
	return d
}

func (s *mainWindow) loadPlans(ctx context.Context) {
	if s.selectedST == nil || s.client == nil {
		return
	}

	dlg := s.showSpinner("Loading Plans", "Fetching plans from Planning Center…")
	defer fyne.Do(func() { dlg.Hide() })

	opts := pco.ListPlansOpts{}
	if !s.showAllPlans {
		t := time.Now().AddDate(0, 0, -28)
		opts.AfterDate = &t
	}
	plans, err := s.client.ListPlans(ctx, s.selectedST.ID, opts)
	if err != nil {
		fyne.Do(func() {
			s.statusLabel.SetText("Failed to load plans")
			dialog.ShowError(err, s.win)
		})
		return
	}

	// Descending sort — next upcoming service appears at top.
	sort.Slice(plans, func(i, j int) bool {
		return plans[i].SortDate.After(plans[j].SortDate)
	})
	s.plans = plans

	names := make([]string, len(plans))
	for i, p := range plans {
		title := p.Title
		if title == "" {
			title = "(untitled)"
		}
		names[i] = fmt.Sprintf("%s — %s", p.SortDate.Format("2006-01-02"), title)
	}

	// Auto-select the earliest upcoming plan (closest to today).
	today := time.Now().Truncate(24 * time.Hour)
	autoSelect := ""
	for i, p := range plans {
		if !p.SortDate.Before(today) {
			autoSelect = names[i] // keep overwriting; last match is closest to today
		}
	}
	if autoSelect == "" && len(names) > 0 {
		autoSelect = names[0] // nothing upcoming — pick most recent
	}

	fyne.Do(func() {
		s.planSelect.Options = names
		if len(names) == 0 {
			s.planSelect.PlaceHolder = "No plans found"
			s.planSelect.Disable()
			s.statusLabel.SetText("No plans found")
		} else {
			s.planSelect.PlaceHolder = "Select a plan"
			s.planSelect.Enable()
			s.statusLabel.SetText(fmt.Sprintf("%d plans", len(plans)))
		}
		s.planSelect.Refresh()
		if autoSelect != "" {
			s.planSelect.SetSelected(autoSelect)
		}
	})
}

func (s *mainWindow) onPlanSelected(name string) {
	s.selectedPlan = nil
	for i := range s.plans {
		p := &s.plans[i]
		title := p.Title
		if title == "" {
			title = "(untitled)"
		}
		if fmt.Sprintf("%s — %s", p.SortDate.Format("2006-01-02"), title) == name {
			s.selectedPlan = p
			break
		}
	}
	if s.selectedPlan == nil {
		return
	}
	s.planItems = nil
	s.itemList.Refresh()
	s.genButton.Disable()

	outputDir := s.app.Preferences().String("outputDir")
	if outputDir == "" {
		outputDir, _ = os.UserHomeDir()
	}
	s.outputEntry.SetText(filepath.Join(outputDir, generate.DefaultOutputPath(s.selectedPlan)))
	go s.loadPlanItems(context.Background())
}

func (s *mainWindow) loadPlanItems(ctx context.Context) {
	if s.selectedST == nil || s.selectedPlan == nil || s.client == nil {
		return
	}

	dlg := s.showSpinner("Loading Plan", "Fetching plan items…")
	defer fyne.Do(func() { dlg.Hide() })

	items, err := s.client.GetPlanItems(ctx, s.selectedST.ID, s.selectedPlan.ID)
	if err != nil {
		fyne.Do(func() {
			s.statusLabel.SetText("Failed to load plan items")
			dialog.ShowError(err, s.win)
		})
		return
	}

	attachments, err := s.client.GetPlanAttachments(ctx, s.selectedST.ID, s.selectedPlan.ID)
	if err != nil {
		fyne.Do(func() { s.statusLabel.SetText("Warning: could not load plan attachments") })
		attachments = nil
	}

	s.planItems = items
	s.planAttachments = attachments
	fyne.Do(func() {
		s.itemList.Refresh()
		s.attachmentList.Refresh()
		s.genButton.Enable()
		s.statusLabel.SetText(fmt.Sprintf("Ready — %d items, %d attachments", len(items), len(attachments)))
	})
}

// ── Generate ──────────────────────────────────────────────────────────────────

func (s *mainWindow) runGenerate() {
	if s.selectedST == nil || s.selectedPlan == nil || s.client == nil {
		return
	}

	output := s.outputEntry.Text
	if output == "" {
		output = generate.DefaultOutputPath(s.selectedPlan)
	}

	cfg := generate.Config{
		ServiceTypeID: s.selectedST.ID,
		PlanID:        s.selectedPlan.ID,
		OutputPath:    output,
		NoHeaders:     s.noHeaders,
		NoCache:       s.noCache,
	}

	// Overwrite check
	if _, err := os.Stat(output); err == nil {
		dialog.ShowConfirm("File already exists",
			fmt.Sprintf("%s already exists.\nOverwrite it?", filepath.Base(output)),
			func(ok bool) {
				if ok {
					s.doGenerate(cfg)
				}
			}, s.win)
		return
	}
	s.doGenerate(cfg)
}

func (s *mainWindow) doGenerate(cfg generate.Config) {
	progressLabel := widget.NewLabel("Starting…")
	progressLabel.Wrapping = fyne.TextWrapWord
	progressLabel.Alignment = fyne.TextAlignCenter
	d := dialog.NewCustomWithoutButtons("Generating Service File",
		container.NewVBox(widget.NewProgressBarInfinite(), progressLabel), s.win)
	d.Resize(fyne.NewSize(420, 140))
	d.Show()
	s.genButton.Disable()

	go func() {
		outPath, err := generate.Run(context.Background(), s.client, cfg, func(msg string) {
			fyne.Do(func() { progressLabel.SetText(msg) })
		})
		fyne.Do(func() {
			d.Hide()
			s.genButton.Enable()
			if err != nil {
				dialog.ShowError(fmt.Errorf("generation failed: %w", err), s.win)
				return
			}
			fi, _ := os.Stat(outPath)
			sizeStr := ""
			if fi != nil {
				sizeStr = fmt.Sprintf(" (%s)", generate.FormatSize(fi.Size()))
			}
			s.statusLabel.SetText("Done — " + filepath.Base(outPath))
			s.showGenerateDone(outPath, sizeStr)
		})
	}()
}

// ── Post-generation dialog ────────────────────────────────────────────────────

func (s *mainWindow) showGenerateDone(outPath, sizeStr string) {
	msg := widget.NewLabel(fmt.Sprintf("Service file written:\n%s%s", filepath.Base(outPath), sizeStr))
	msg.Alignment = fyne.TextAlignCenter
	msg.Wrapping = fyne.TextWrapWord

	var d *dialog.CustomDialog

	revealBtn := widget.NewButton("Show in Finder", func() {
		exec.Command("open", "-R", outPath).Start()
		d.Hide()
	})
	closeBtn := widget.NewButton("Close", func() { d.Hide() })
	closeBtn.Importance = widget.LowImportance

	buttons := container.NewHBox(revealBtn, closeBtn)
	d = dialog.NewCustomWithoutButtons("Done", container.NewVBox(msg, buttons), s.win)
	d.Resize(fyne.NewSize(400, 130))
	d.Show()
}

// ── Browse for output path ────────────────────────────────────────────────────

func (s *mainWindow) browseOutput() {
	d := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil || writer == nil {
			return
		}
		defer writer.Close()
		s.outputEntry.SetText(writer.URI().Path())
	}, s.win)

	// Use what's already in the filename bar, falling back to the plan default.
	filename := filepath.Base(s.outputEntry.Text)
	if filename == "" || filename == "." {
		if s.selectedPlan != nil {
			filename = generate.DefaultOutputPath(s.selectedPlan)
		} else {
			filename = "service.osz"
		}
	}
	d.SetFileName(filename)
	d.SetFilter(storage.NewExtensionFileFilter([]string{".osz"}))

	if dir := s.app.Preferences().String("outputDir"); dir != "" {
		if uri, err := storage.ParseURI("file://" + dir); err == nil {
			if luri, err := storage.ListerForURI(uri); err == nil {
				d.SetLocation(luri)
			}
		}
	}
	d.Show()
}

