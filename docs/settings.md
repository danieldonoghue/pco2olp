---
title: Settings
nav_order: 5
---

# Settings

Open Settings from the app menu (**pco2olp → Settings…** on macOS, or ⌘,).

<img src="{{ '/assets/images/settings.png' | relative_url }}" alt="Settings dialog" class="screenshot">

---

## Planning Center Credentials

| Field | Description |
|---|---|
| **Client ID** | Your PCO OAuth application Client ID |
| **Client Secret** | Your PCO OAuth application Client Secret |

Click **Sign In to Planning Center** to authenticate (or re-authenticate after changing credentials).

{: .note }
In organisation-specific builds, credentials are baked in and cannot be changed. Only the **Re-authenticate** button is shown.

---

## Output

| Field | Description |
|---|---|
| **Default output folder** | Where `.osz` files are saved by default. Leave blank to use your home directory. |
| **Media cache** | Shows the number of cached files, total size, and cache location. |

Click **Clear Cache** to delete all cached media files. You will be asked to confirm.

---

## Media

### Download missing media

When enabled (the default), pco2olp downloads media files from Planning Center and embeds them in the `.osz` file during generation.

When disabled, the service file is generated without any media — items referencing media will still appear in OpenLP but without the associated files.

{: .note }
When "Download missing media" is off, the **Re-download existing media** option on the main window is disabled.

### Use external media folder

When enabled, media files are **not** embedded in the `.osz` file. Instead, pco2olp copies them into the configured external folder, organised as:

```
<external folder>/
  media/           ← video and audio files
  presentations/   ← converted slide images (one subfolder per presentation)
```

A note is added to each affected service item referencing the external file location.

Set a folder using the **Browse…** button next to **External folder**.

{: .tip }
External media mode is useful when your OpenLP machine and the machine running pco2olp share a network drive, or when you want to manage media separately from the service file.
