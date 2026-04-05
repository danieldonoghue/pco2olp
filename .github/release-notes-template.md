## Downloads

### Graphical Interface (GUI)

| Platform | Download |
|----------|----------|
| macOS (Universal) | `pco2olp-gui-VERSION-darwin-universal.zip` — unzip and drag to Applications |
| Windows (x64) | `pco2olp-gui-windows-amd64.exe` |
| Windows (arm64) | `pco2olp-gui-windows-arm64.exe` |
| Linux (amd64) | `pco2olp-gui-linux-amd64` |

### Command-Line (CLI)

| Platform | Download |
|----------|----------|
| macOS (Apple Silicon) | `pco2olp-darwin-arm64` |
| macOS (Intel) | `pco2olp-darwin-amd64` |
| Windows (x64) | `pco2olp-windows-amd64.exe` |
| Windows (arm64) | `pco2olp-windows-arm64.exe` |
| Linux (amd64) | `pco2olp-linux-amd64` |
| Linux (arm64) | `pco2olp-linux-arm64` |

---

## CLI Installation

### macOS

```sh
curl -L -o pco2olp https://github.com/danieldonoghue/pco2olp/releases/latest/download/pco2olp-darwin-arm64
chmod +x pco2olp
sudo mv pco2olp /usr/local/bin/
```

*(Replace `arm64` with `amd64` for Intel Macs.)*

### Windows

Download the appropriate `.exe` from the assets below (`amd64` for x64 systems, `arm64` for ARM-based Windows devices) and place it wherever you keep executables (e.g. `C:\Tools\`). Add that folder to your `PATH` if you want to run it from any terminal.

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

## Authentication

`pco2olp` uses OAuth 2.0 to connect to your Planning Center account. You will need a PCO OAuth **Client ID** (and **Client Secret** for the generic build). See the [README](https://github.com/danieldonoghue/pco2olp#quick-start) for how to register a PCO application if you don't have one.

### Setting your credentials

```sh
export PCO_CLIENT_ID="your-client-id"
export PCO_CLIENT_SECRET="your-client-secret"
```

Add these to your shell profile (`~/.zprofile`, `~/.bashrc`, etc.) to avoid setting them each session. On Windows, set them as user environment variables via **System Properties → Environment Variables**.

> If you are using an organisation-specific build provided by your church, credentials are already baked in — skip this step.

### Logging in

On first run, `pco2olp` will open your browser automatically and take you to the Planning Center login page. Log in with your PCO account, approve the access request, and the browser will redirect back to the tool. The terminal will confirm authentication and continue.

Tokens are cached so you only need to do this once. They refresh automatically in the background.

| Platform | Token location |
|----------|----------------|
| macOS    | `~/Library/Application Support/pco2olp/tokens.json` |
| Linux    | `~/.config/pco2olp/tokens.json` |
| Windows  | `%APPDATA%\pco2olp\tokens.json` |

Delete `tokens.json` to force re-authentication (e.g. if you switch PCO accounts).

---

## First-Run Setup

### macOS — Gatekeeper

> **Note:** If the release was built with code signing enabled, this step is not needed.

macOS will block unsigned binaries on first run. To allow it:

**Option A — one-time terminal command (recommended):**
```sh
xattr -d com.apple.quarantine /usr/local/bin/pco2olp
```

**Option B — via System Settings:**
1. Run `pco2olp` once — macOS will show a "cannot be opened" alert.
2. Open **System Settings → Privacy & Security**.
3. Scroll down to the *Security* section and click **Allow Anyway**.
4. Run `pco2olp` again and click **Open**.

### macOS — Keynote Automation Permission

`pco2olp` converts Keynote and PowerPoint files to slides using Keynote on macOS. On first use it will ask you to grant **Automation** permission so it can control Keynote. Click **OK** to allow it. If you accidentally denied it:

1. Open **System Settings → Privacy & Security → Automation**.
2. Find **pco2olp** and enable **Keynote**.

### macOS — PDF Conversion (pdftoppm)

PDF files are converted using `pdftoppm` from [poppler](https://poppler.freedesktop.org/):

```sh
brew install poppler
```

### Windows — SmartScreen

Windows may show a SmartScreen warning on first run. Click **More info → Run anyway** to proceed.

### Windows — PowerPoint Automation

`pco2olp` uses PowerPoint (via PowerShell COM automation) to convert `.pptx` files. Microsoft PowerPoint must be installed. PDFs are supported via `pdftoppm` (install via [Scoop](https://scoop.sh/): `scoop install poppler`).
