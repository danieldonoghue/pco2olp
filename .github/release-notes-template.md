## Installation

Download the binary for your platform from the assets below.

### macOS (Apple Silicon)

```sh
# Download
curl -L -o pco2olp https://github.com/danieldonoghue/pco2olp/releases/latest/download/pco2olp-darwin-arm64
chmod +x pco2olp
sudo mv pco2olp /usr/local/bin/
```

### macOS (Intel)

```sh
curl -L -o pco2olp https://github.com/danieldonoghue/pco2olp/releases/latest/download/pco2olp-darwin-amd64
chmod +x pco2olp
sudo mv pco2olp /usr/local/bin/
```

### Windows

Download `pco2olp-windows-amd64.exe` from the assets below and place it wherever you keep executables (e.g. `C:\Tools\`). Add that folder to your `PATH` if you want to run it from any terminal.

### Linux (amd64)

```sh
curl -L -o pco2olp https://github.com/danieldonoghue/pco2olp/releases/latest/download/pco2olp-linux-amd64
chmod +x pco2olp
sudo mv pco2olp /usr/local/bin/
```

### Linux (arm64 / Raspberry Pi)

```sh
curl -L -o pco2olp https://github.com/danieldonoghue/pco2olp/releases/latest/download/pco2olp-linux-arm64
chmod +x pco2olp
sudo mv pco2olp /usr/local/bin/
```

---

## First-Run Setup

### macOS — Gatekeeper (unsigned binary)

> **Note:** If the release was built with code signing enabled (see repository secrets), this step is not needed.

macOS will block the binary on first run because it is not notarized by Apple. To allow it:

**Option A — one-time terminal command (recommended):**
```sh
xattr -d com.apple.quarantine /usr/local/bin/pco2olp
```

**Option B — via System Settings:**
1. Run `pco2olp` once — macOS will show a "cannot be opened" alert.
2. Open **System Settings → Privacy & Security**.
3. Scroll down to the *Security* section and click **Allow Anyway** next to the `pco2olp` message.
4. Run `pco2olp` again and click **Open** in the confirmation dialog.

### macOS — Keynote Automation Permission

`pco2olp` converts Keynote and PowerPoint files to slides using Keynote on macOS. On first use it will ask you to grant **Automation** permission so it can control Keynote.

If you see a permission dialog — click **OK** to allow it. If you accidentally denied it:

1. Open **System Settings → Privacy & Security → Automation**.
2. Find **pco2olp** (or your terminal app) and enable **Keynote**.

### macOS — PDF Conversion (pdftoppm)

PDF files are converted using `pdftoppm` from [poppler](https://poppler.freedesktop.org/). Install it with Homebrew if you need PDF support:

```sh
brew install poppler
```

### Windows — SmartScreen

Windows may show a SmartScreen warning on first run. Click **More info → Run anyway** to proceed. This only appears once.

### Windows — PowerPoint Automation

`pco2olp` uses PowerPoint (via PowerShell COM automation) to convert `.pptx` files. Microsoft PowerPoint must be installed. If PowerPoint is not installed, `.pdf` files are still supported via `pdftoppm` (install `poppler` via [Scoop](https://scoop.sh/): `scoop install poppler`).

---

## Code Signing (maintainers)

To enable code signing and notarization for macOS builds, add the following secrets to the repository:

| Secret | Description |
|--------|-------------|
| `APPLE_CERTIFICATE_P12_BASE64` | Base64-encoded Developer ID Application `.p12` certificate |
| `APPLE_CERTIFICATE_PASSWORD` | Password for the `.p12` file |
| `APPLE_TEAM_ID` | Apple Developer Team ID |
| `APPLE_NOTARIZATION_APPLE_ID` | Apple ID (email) used for notarization |
| `APPLE_NOTARIZATION_PASSWORD` | App-specific password for notarization |

Generate the base64 value with: `base64 -i certificate.p12 | pbcopy`

When these secrets are present the release workflow will automatically sign and notarize the macOS binaries, removing the need for the Gatekeeper workaround above.
