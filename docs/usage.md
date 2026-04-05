---
title: Using the App
nav_order: 4
---

# Using the App

## Launching

Open pco2olp from your Applications folder (macOS), Start menu (Windows), or by running the binary (Linux).

On first launch, your browser will open for authentication. See [Authentication](authentication) for details.

## Main window

The main window has three sections:

- **Top** — service type and plan selectors
- **Centre** — service items list and plan attachments list
- **Bottom** — output path, options, and the Generate button

{: .note }
📸 *Screenshot: main window with a plan loaded and items visible.*

## Selecting a service type

Use the **Service type** dropdown to pick the Planning Center service type (e.g. "Sunday Morning", "Youth"). The list is populated automatically once you are authenticated.

## Selecting a plan

Once a service type is selected, recent and upcoming plans appear in the **Plan** dropdown. The next upcoming service is selected automatically.

To see older plans, check **Show all**.

## Service items

The items list shows all items in the selected plan:

- **Headers** are shown with a bold label and a shaded background
- **Songs**, **media**, **presentations**, and **custom items** are shown with their type icon

Each item has a checkbox — uncheck any items you want to exclude from the generated service file.

Use **All** and **None** buttons to select or deselect everything at once.

### Including headers

By default, header items are hidden. Check **Include headers** to show them in the list.

When Include headers is turned on, all header items are set to included. You can then uncheck individual headers you do not want.

## Plan attachments

Files attached to the plan itself (rather than to individual items) appear in the **Plan attachments** section. These are always included.

## Output path

The output path is automatically set to your configured output folder (or your home directory) with a filename of `YYYY-MM-DD-Title.osz`.

Click **Browse…** to pick a different location or filename.

## Options

| Option | Description |
|---|---|
| **Include headers** | Show and optionally include header items |
| **Re-download existing media** | Bypass the media cache and re-download all files from Planning Center |

{: .note }
"Re-download existing media" is only available when "Download missing media" is enabled in Settings.

## Generating the service file

Click **Generate Service File** to produce the `.osz` file.

A progress dialog shows each step: authenticating, fetching items, downloading media, converting presentations, and writing the file.

When complete, a dialog shows the filename and size. Click **Show in Finder** (macOS) or **Show in Explorer** (Windows) to reveal the file.

### Alternative media mode

The dropdown arrow (▾) next to the Generate button offers an alternative:

- If your default is **embedded media**, the dropdown offers **Generate with external media**
- If your default is **external media**, the dropdown offers **Generate with embedded media**

See [Settings → Media](settings#media) for how to set your default.

## Opening in OpenLP

Open OpenLP and use **File → Open Service** (or drag the `.osz` file onto OpenLP) to load the generated service.
