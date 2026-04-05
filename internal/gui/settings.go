//go:build gui

package gui

import (
	"fmt"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/danieldonoghue/pco2olp/internal/auth"
	"github.com/danieldonoghue/pco2olp/internal/cache"
	"github.com/danieldonoghue/pco2olp/internal/generate"
)

func showSettingsDialog(win fyne.Window, a fyne.App, reauth func(func(error)), isAuthenticated bool, onSettingsChanged func()) {
	prefs := a.Preferences()

	// ── PCO Credentials ───────────────────────────────────────────────────
	clientIDEntry := widget.NewEntry()
	clientIDEntry.SetPlaceHolder("OAuth Client ID")
	clientSecretEntry := widget.NewPasswordEntry()
	clientSecretEntry.SetPlaceHolder("OAuth Client Secret")

	credHint := widget.NewLabel("Register an app at planningcenteronline.com/oauth/applications")
	credHint.TextStyle = fyne.TextStyle{Italic: true}

	bakedIn := auth.HasBakedInCredentials()

	var initialID, initialSecret string
	if bakedIn {
		clientIDEntry.SetText("(set at build time)")
		clientIDEntry.Disable()
		clientSecretEntry.SetText("(set at build time)")
		clientSecretEntry.Disable()
	} else {
		// Prefs take priority; fall back to env vars so existing env-based users see their values.
		initialID = prefs.String("clientID")
		if initialID == "" {
			initialID = os.Getenv("PCO_CLIENT_ID")
		}
		initialSecret = prefs.String("clientSecret")
		if initialSecret == "" {
			initialSecret = os.Getenv("PCO_CLIENT_SECRET")
		}
		clientIDEntry.SetText(initialID)
		clientSecretEntry.SetText(initialSecret)
	}

	authStatusLabel := widget.NewLabel("")
	authStatusLabel.TextStyle = fyne.TextStyle{Italic: true}
	authStatusLabel.Alignment = fyne.TextAlignCenter

	signInBtn := widget.NewButton("Sign In to Planning Center", nil)

	// Disable the button if already signed in and credentials are unchanged.
	credentialsChanged := func() bool {
		return clientIDEntry.Text != initialID || clientSecretEntry.Text != initialSecret
	}
	updateSignInBtn := func() {
		if isAuthenticated && !bakedIn && !credentialsChanged() {
			signInBtn.Disable()
		} else {
			signInBtn.Enable()
		}
	}
	if isAuthenticated && !bakedIn {
		signInBtn.Disable()
		authStatusLabel.SetText("Signed in.")
	}
	if bakedIn && isAuthenticated {
		signInBtn.Disable()
		authStatusLabel.SetText("Signed in.")
	}

	clientIDEntry.OnChanged = func(_ string) { updateSignInBtn() }
	clientSecretEntry.OnChanged = func(_ string) { updateSignInBtn() }

	signInBtn.OnTapped = func() {
		if !bakedIn {
			if strings.TrimSpace(clientIDEntry.Text) == "" || strings.TrimSpace(clientSecretEntry.Text) == "" {
				dialog.ShowError(fmt.Errorf("Client ID and Client Secret are required"), win)
				return
			}
			prefs.SetString("clientID", strings.TrimSpace(clientIDEntry.Text))
			prefs.SetString("clientSecret", strings.TrimSpace(clientSecretEntry.Text))
		}
		// Always clear the stored token so the new credentials trigger a fresh browser flow.
		if err := auth.ClearStoredToken(); err != nil {
			dialog.ShowError(fmt.Errorf("clearing stored token: %w", err), win)
			return
		}
		signInBtn.Disable()
		authStatusLabel.SetText("Opening browser for authentication…")
		reauth(func(err error) {
			fyne.Do(func() {
				if err != nil {
					signInBtn.Enable()
					authStatusLabel.SetText("Authentication failed.")
				} else {
					isAuthenticated = true
					// Update initial values so button returns to disabled state.
					initialID = strings.TrimSpace(clientIDEntry.Text)
					initialSecret = strings.TrimSpace(clientSecretEntry.Text)
					updateSignInBtn()
					authStatusLabel.SetText("Authentication successful.")
				}
			})
		})
	}
	if bakedIn {
		signInBtn.SetText("Re-authenticate")
	}

	// ── Output directory ──────────────────────────────────────────────────
	outputDirEntry := widget.NewEntry()
	outputDirEntry.SetPlaceHolder("Default: home directory")
	if dir := prefs.String("outputDir"); dir != "" {
		outputDirEntry.SetText(dir)
	}

	browseDirBtn := widget.NewButton("Browse…", func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			outputDirEntry.SetText(uri.Path())
		}, win)
		d.Show()
	})

	outputRow := container.NewBorder(nil, nil, nil, browseDirBtn, outputDirEntry)

	// ── Cache info ────────────────────────────────────────────────────────
	cacheInfoLabel := widget.NewLabel(cacheInfoText())
	cacheInfoLabel.Wrapping = fyne.TextWrapWord

	cleanBtn := widget.NewButton("Clear Cache", func() {
		c, err := cache.NewCache()
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		dialog.ShowConfirm("Clear Cache", "Delete all cached media files?", func(ok bool) {
			if !ok {
				return
			}
			if err := c.Clean(); err != nil {
				dialog.ShowError(err, win)
				return
			}
			cacheInfoLabel.SetText(cacheInfoText())
			dialog.ShowInformation("Cache", "Media cache cleared.", win)
		}, win)
	})

	// ── Media ─────────────────────────────────────────────────────────────
	dlMediaCheck := widget.NewCheck("Download missing media", nil)
	dlMediaCheck.SetChecked(prefs.BoolWithFallback("downloadMediaDefault", true))
	dlMediaCheck.OnChanged = func(v bool) {
		prefs.SetBool("downloadMediaDefault", v)
		onSettingsChanged()
	}

	externalDirEntry := widget.NewEntry()
	externalDirEntry.SetPlaceHolder("Select a folder…")
	if dir := prefs.String("externalMediaDir"); dir != "" {
		externalDirEntry.SetText(dir)
	}

	browseExternalBtn := widget.NewButton("Browse…", func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			externalDirEntry.SetText(uri.Path())
		}, win)
		d.Show()
	})

	externalRow := container.NewBorder(nil, nil, nil, browseExternalBtn, externalDirEntry)

	useExternalCheck := widget.NewCheck("Use external media folder", nil)
	useExternalCheck.SetChecked(prefs.BoolWithFallback("externalMediaEnabled", false))
	setExternalRowEnabled := func(enabled bool) {
		if enabled {
			externalDirEntry.Enable()
			browseExternalBtn.Enable()
		} else {
			externalDirEntry.Disable()
			browseExternalBtn.Disable()
		}
	}
	setExternalRowEnabled(useExternalCheck.Checked)
	useExternalCheck.OnChanged = func(v bool) {
		prefs.SetBool("externalMediaEnabled", v)
		setExternalRowEnabled(v)
	}

	// ── Layout ────────────────────────────────────────────────────────────
	var credSection fyne.CanvasObject
	if bakedIn {
		credSection = container.NewVBox(
			widget.NewForm(
				widget.NewFormItem("Client ID", clientIDEntry),
				widget.NewFormItem("Client Secret", clientSecretEntry),
			),
			widget.NewLabel("Credentials are set at build time and cannot be changed."),
			signInBtn,
			authStatusLabel,
		)
	} else {
		credSection = container.NewVBox(
			widget.NewForm(
				widget.NewFormItem("Client ID", clientIDEntry),
				widget.NewFormItem("Client Secret", clientSecretEntry),
			),
			credHint,
			signInBtn,
			authStatusLabel,
		)
	}

	content := container.NewVBox(
		widget.NewLabel("Planning Center Credentials"),
		widget.NewSeparator(),
		credSection,
		widget.NewSeparator(),
		widget.NewLabel("Output"),
		widget.NewSeparator(),
		widget.NewForm(
			widget.NewFormItem("Default output folder", outputRow),
			widget.NewFormItem("Media cache", cacheInfoLabel),
		),
		cleanBtn,
		widget.NewSeparator(),
		widget.NewLabel("Media"),
		widget.NewSeparator(),
		dlMediaCheck,
		useExternalCheck,
		widget.NewForm(widget.NewFormItem("External folder", externalRow)),
	)

	d := dialog.NewCustom("Settings", "Close", container.NewVScroll(container.NewPadded(content)), win)
	d.SetOnClosed(func() {
		prefs.SetString("outputDir", outputDirEntry.Text)
		prefs.SetString("externalMediaDir", externalDirEntry.Text)
		if !bakedIn {
			prefs.SetString("clientID", strings.TrimSpace(clientIDEntry.Text))
			prefs.SetString("clientSecret", strings.TrimSpace(clientSecretEntry.Text))
		}
		onSettingsChanged()
	})
	d.Resize(fyne.NewSize(520, 650))
	d.Show()
}

func cacheInfoText() string {
	c, err := cache.NewCache()
	if err != nil {
		return "Unable to read cache info"
	}
	dir, count, size, err := c.Info()
	if err != nil {
		return "Unable to read cache info"
	}
	return fmt.Sprintf("%d files, %s\n%s", count, generate.FormatSize(size), dir)
}
