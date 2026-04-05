---
title: CLI Reference
nav_order: 6
---

# CLI Reference

pco2olp can be used entirely from the command line. Running it with arguments enters CLI mode; running it without arguments opens the GUI.

## Authentication

The CLI uses the same OAuth token store as the GUI. If you have already authenticated in the GUI, the CLI will use the same token automatically.

Set your credentials via environment variables:

```
export PCO_CLIENT_ID=your-client-id
export PCO_CLIENT_SECRET=your-client-secret
```

## Basic usage

```
pco2olp --service-type "Sunday" --plan 2026-04-06
```

## Flags

### Service selection

| Flag | Description |
|---|---|
| `--service-type <name\|id>` | Service type name or numeric ID |
| `--plan <id\|YYYY-MM-DD>` | Plan ID or date |
| `--all` | When listing plans, show all (default: recent and upcoming only) |

### Output

| Flag | Description |
|---|---|
| `--output <file.osz>` | Output file path (default: `YYYY-MM-DD-Title.osz` in current directory) |
| `--dry-run` | Preview what would be generated without creating the file |

### Content

| Flag | Description |
|---|---|
| `--no-headers` | Exclude header items from the service |

### Media

| Flag | Description |
|---|---|
| `--no-cache` | Bypass the media cache and re-download all files |
| `--cache-info` | Show cache statistics and location |
| `--clean-cache` | Delete all cached media files |

### Listing

| Flag | Description |
|---|---|
| `--list-service-types` | List all available service types |
| `--list-plans` | List recent plans for the given service type |

### Other

| Flag | Description |
|---|---|
| `--debug` | Enable verbose debug logging |
| `--version` | Show version, commit, and build time |

## Examples

```bash
# List service types
pco2olp --list-service-types

# List recent plans for a service type
pco2olp --service-type "Sunday Morning" --list-plans

# List all plans (full history)
pco2olp --service-type "Sunday Morning" --list-plans --all

# Generate a service file for a specific date
pco2olp --service-type "Sunday Morning" --plan 2026-04-06

# Generate by plan ID
pco2olp --service-type "Sunday Morning" --plan 12345678

# Generate without headers
pco2olp --service-type "Sunday Morning" --plan 2026-04-06 --no-headers

# Preview without generating
pco2olp --service-type "Sunday Morning" --plan 2026-04-06 --dry-run

# Re-download all media
pco2olp --service-type "Sunday Morning" --plan 2026-04-06 --no-cache

# Save to a specific path
pco2olp --service-type "Sunday Morning" --plan 2026-04-06 --output ~/Desktop/service.osz

# Show cache info
pco2olp --cache-info

# Clear the cache
pco2olp --clean-cache
```

## Environment variables

| Variable | Description |
|---|---|
| `PCO_CLIENT_ID` | OAuth Client ID (required unless set via GUI Settings) |
| `PCO_CLIENT_SECRET` | OAuth Client Secret (required unless set via GUI Settings) |

Token storage: `$XDG_CONFIG_HOME/pco2olp/tokens.json` (Linux), `~/Library/Application Support/pco2olp/tokens.json` (macOS), `%AppData%\pco2olp\tokens.json` (Windows).
