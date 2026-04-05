---
title: Installation
nav_order: 2
---

# Installation

## Downloading

Download the latest release from the [GitHub Releases page](https://github.com/danieldonoghue/pco2olp/releases/latest).

| Platform | File |
|---|---|
| macOS (Universal) | `pco2olp-gui-vX.Y.Z-darwin-universal.zip` |
| Windows (x64) | `pco2olp-gui-windows-amd64.exe` |
| Windows (ARM64) | `pco2olp-gui-windows-arm64.exe` |
| Linux (x64 GUI) | `pco2olp-gui-linux-amd64` |
| Linux (x64 CLI) | `pco2olp-linux-amd64` |
| Linux (ARM64 CLI) | `pco2olp-linux-arm64` |

---

## macOS

1. Download and unzip `pco2olp-gui-vX.Y.Z-darwin-universal.zip`
2. Drag `pco2olp.app` to your **Applications** folder
3. Double-click to open

### Gatekeeper warning

Because the app is downloaded from the internet, macOS may block it on first launch.

**Option A — System Settings (recommended)**
1. Try to open the app — you will see a warning dialog
2. Open **System Settings → Privacy & Security**
3. Scroll down to the Security section and click **Open Anyway**
4. Click **Open** in the confirmation dialog

**Option B — Terminal**
```
xattr -dr com.apple.quarantine /Applications/pco2olp.app
```

### Keynote / presentation conversion

To convert Keynote (`.key`) and PowerPoint (`.pptx`) files into slides, pco2olp uses AppleScript to drive Keynote.app. The first time you generate a file with presentations, macOS will ask:

> *"pco2olp" wants access to control "Keynote". Allowing control will provide access to documents and data in "Keynote", and to perform actions within that app.*

Click **OK** to allow. You can manage this permission later in **System Settings → Privacy & Security → Automation**.

### PDF conversion (optional)

PDF files are converted using `pdftoppm` from the [poppler](https://poppler.freedesktop.org/) toolkit. Install it with Homebrew:

```
brew install poppler
```

If poppler is not installed, PDF plan attachments will be included as-is rather than converted to slides.

---

## Windows

1. Download `pco2olp-gui-windows-amd64.exe` (or the ARM64 version)
2. Move it to a convenient location (e.g. `C:\Program Files\pco2olp\`)
3. Double-click to run

### SmartScreen warning

Windows may show a SmartScreen warning on first run:

1. Click **More info**
2. Click **Run anyway**

### PDF conversion (optional)

Install `pdftoppm` via the [poppler Windows builds](https://github.com/oschwartz10612/poppler-windows/releases) and add the `bin/` folder to your system `PATH`.

### Firewall prompt

pco2olp opens a local port (11019) briefly during authentication to receive the OAuth callback from your browser. Windows Firewall may ask permission — click **Allow access**.

---

## Linux

1. Download the appropriate binary
2. Make it executable:
   ```
   chmod +x pco2olp-gui-linux-amd64
   ```
3. Run it:
   ```
   ./pco2olp-gui-linux-amd64
   ```

### GUI dependencies

The GUI requires X11 or Wayland libraries. On Debian/Ubuntu:

```
sudo apt-get install libgl1-mesa-dev xorg-dev
```

### PDF conversion (optional)

```
sudo apt-get install poppler-utils
```
