---
title: Introduction
nav_order: 1
---

<img src="{{ '/assets/images/logo.svg' | relative_url }}" alt="pco2olp logo" width="80" style="display:block; margin-bottom:1rem;">

# pco2olp User Guide

pco2olp connects to [Planning Center Online](https://planningcenteronline.com) and generates [OpenLP](https://openlp.org) service files (`.osz`) ready to open directly in OpenLP on the day of your service.

## What it does

1. Authenticates with your Planning Center account using OAuth
2. Lets you browse your service types and plans
3. Shows you all items in a plan — songs, headers, media, presentations
4. Downloads associated media and attachments
5. Produces an `.osz` file containing everything OpenLP needs

## Key features

- **GUI** — a simple point-and-click interface (no command line needed)
- **CLI** — a full command-line interface for scripted or automated use
- **Media embedding** — video, audio, images, and slide decks downloaded and bundled into the service file
- **Presentation conversion** — Keynote, PowerPoint, and PDF files automatically converted to image slides
- **Media caching** — downloaded files are cached locally so re-generating is fast
- **Selective items** — include or exclude individual items before generating
- **External media mode** — optionally keep media files outside the `.osz` for manual placement

## Supported platforms

| Platform | GUI | CLI |
|---|---|---|
| macOS (Apple Silicon) | ✓ | ✓ |
| macOS (Intel) | ✓ | ✓ |
| Windows (x64) | ✓ | ✓ |
| Windows (ARM64) | ✓ ¹ | ✓ |
| Linux (x64) | ✓ | ✓ |
| Linux (ARM64) | — | ✓ |

*¹ Windows ARM64 runs the x64 GUI binary via Windows 11's built-in x64 emulation.*

## Quick start

1. [Download and install](installation) the app for your platform
2. Open pco2olp — it will launch your browser to sign in to Planning Center
3. Select your service type and plan
4. Click **Generate Service File**
5. Open the resulting `.osz` file in OpenLP

{: .note }
If you are using an organisation-specific build (e.g. provided by your church), PCO credentials are already baked in — you only need to sign in.
