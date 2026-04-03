# pco2olp

Convert [Planning Center Online](https://www.planningcenteronline.com/) service plans to [OpenLP](https://openlp.org/) service files.

A single-binary CLI tool that authenticates with the PCO API, fetches your service plan, and generates an `.osz` file you can open directly in OpenLP 3.x. No installation required — just download and run.

## Features

- OAuth 2.0 authentication with PCO (browser-based login, tokens cached locally)
- List service types and plans from your PCO account
- Generate `.osz` service files with songs (OpenLyrics XML), custom slides, images, media, and presentations
- Automatically downloads and caches media from PCO (videos, images, presentations)
- Preview service plans without generating files (dry-run mode)
- Cross-platform: macOS (Intel + Apple Silicon), Linux (x86_64 + ARM64), Windows

## Quick Start

### 1. Register a PCO OAuth Application

1. Go to https://api.planningcenteronline.com/oauth/applications
2. Create a new application (requires Organization Administrator)
3. Set the redirect URI to `http://localhost:11019/callback`
4. Note the **Client ID** and **Secret**

### 2. Set Up

```bash
export PCO_CLIENT_ID="your-client-id-here"
export PCO_CLIENT_SECRET="your-client-secret-here"
```

### 3. Use

```bash
# List your service types
pco2olp --list-service-types

# List plans for a service type (shows recent plans, highlights next upcoming)
pco2olp --service-type "Sunday Service" --list-plans

# List all plans (full history)
pco2olp --service-type "Sunday Service" --list-plans --all

# Preview a plan
pco2olp --service-type "Sunday Service" --plan 2026-04-05 --dry-run

# Generate an .osz file (defaults to <date>-<title>.osz)
pco2olp --service-type "Sunday Service" --plan 2026-04-05

# Generate by plan ID instead of date
pco2olp --service-type "Sunday Service" --plan 12345678

# Custom output filename
pco2olp --service-type "Sunday Service" --plan 2026-04-05 --output sunday.osz

# Exclude section headers from the service
pco2olp --service-type "Sunday Service" --plan 2026-04-05 --no-headers

# Force re-download of all media (bypass cache)
pco2olp --service-type "Sunday Service" --plan 2026-04-05 --no-cache

# Show media cache info
pco2olp --cache-info

# Clear media cache
pco2olp --clean-cache
```

## Installation

Download the binary for your platform from the [releases page](https://github.com/danieldonoghue/pco2olp/releases), or build from source:

```bash
# Build for your current platform
make build

# Cross-compile for all platforms
make release

# Binaries go to dist/
ls dist/
# pco2olp-darwin-amd64    pco2olp-linux-amd64    pco2olp-windows-amd64.exe
# pco2olp-darwin-arm64    pco2olp-linux-arm64
```

### Requirements

- Go 1.21+ (build only)
- A Planning Center Online account with Services access
- PCO OAuth application (Client ID and Secret)

## How It Works

1. **Authentication**: On first run, pco2olp opens your browser for PCO login. Tokens are cached in your platform's config directory and auto-refreshed.
2. **Fetch**: The tool queries the PCO Services API for your plan's items — songs (with lyrics and arrangements), headers, custom items, and media references.
3. **Download**: Media attachments (videos, images, presentations) are downloaded and cached locally. Subsequent runs skip unchanged files.
4. **Convert**: PCO items are mapped to OpenLP service items. Song lyrics are parsed into structured verses and converted to OpenLyrics XML. Media files are mapped to the correct OpenLP plugin (images, media, or presentations).
5. **Generate**: Everything is packaged into an `.osz` file (ZIP archive containing `service_data.osj` + media files) that OpenLP 3.x can open directly.

## Token Storage

Tokens are stored in your platform's config directory:

| Platform | Path |
|----------|------|
| macOS    | `~/Library/Application Support/pco2olp/tokens.json` |
| Linux    | `~/.config/pco2olp/tokens.json` |
| Windows  | `%APPDATA%\pco2olp\tokens.json` |

Delete `tokens.json` to force re-authentication.

## Media Cache

Media files (videos, images, presentations) downloaded from PCO are cached locally to avoid redundant downloads:

| Platform | Path |
|----------|------|
| macOS    | `~/Library/Caches/pco2olp/media/` |
| Linux    | `~/.cache/pco2olp/media/` |
| Windows  | `%LOCALAPPDATA%\pco2olp\media\` |

Files are stored by SHA256 hash. The cache automatically detects when files have been updated in PCO. Use `--no-cache` to bypass the cache, or `--clean-cache` to clear it entirely.

## CLI Reference

| Flag | Description |
|------|-------------|
| `--service-type <name\|id>` | Service type name (fuzzy match) or numeric ID |
| `--plan <id\|YYYY-MM-DD>` | Plan ID or date to generate |
| `--output <path>` | Output file path (default: `<date>-<title>.osz`) |
| `--no-headers` | Exclude header items from the generated service |
| `--no-cache` | Force re-download of all media (bypass cache) |
| `--list-service-types` | List available PCO service types |
| `--list-plans` | List plans for the specified service type (recent by default) |
| `--all` | Show all plans instead of just recent ones |
| `--dry-run` | Preview plan items without generating a file |
| `--cache-info` | Show media cache directory, file count, and total size |
| `--clean-cache` | Delete all cached media files |
| `--debug` | Enable verbose debug logging |
| `--version` | Show version information |

## Building

```bash
make build          # Build local binary
make test           # Run all tests
make lint           # Run go vet
make release        # Cross-compile for all platforms → dist/
make clean          # Remove build artifacts
```

### Organisation Builds

You can bake your PCO OAuth credentials and organisation name into the binary so your org members don't need to set environment variables:

```bash
# Build for current platform with embedded credentials
make build-org PCO_CLIENT_ID=your-id PCO_CLIENT_SECRET=your-secret ORG_NAME="My Church"

# Cross-compile all platforms with embedded credentials
make release-org PCO_CLIENT_ID=your-id PCO_CLIENT_SECRET=your-secret ORG_NAME="My Church"
```

`ORG_NAME` is optional. When set, it appears in the version output:

```
pco2olp v1.2.0 (commit: abc1234, built: 2026-04-03T09:00:00Z)
Built for: My Church
```

Environment variables (`PCO_CLIENT_ID`, `PCO_CLIENT_SECRET`) always take precedence over baked-in defaults, so users can still override if needed.

## License

MIT
