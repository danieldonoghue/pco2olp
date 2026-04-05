---
title: Authentication
nav_order: 3
---

# Authentication

pco2olp connects to Planning Center Online using OAuth 2.0. You authenticate once and your token is stored securely on your computer — you won't need to sign in again unless your session expires (tokens last up to 90 days).

## Getting PCO credentials

You need a PCO **OAuth Application** registered in your Planning Center account. This requires Org Admin access.

1. Go to [planningcenteronline.com/oauth/applications](https://planningcenteronline.com/oauth/applications)
2. Click **New Application**
3. Give it a name (e.g. "pco2olp")
4. Set the **Redirect URI** to:
   ```
   http://localhost:11019/callback
   ```
5. Note the **Client ID** and **Client Secret**

{: .note }
If you are using an organisation-specific build, credentials are already baked in and you do not need to register an application.

## Entering credentials

1. Open **Settings** (⌘, on macOS, or via the app menu)
2. Enter your **Client ID** and **Client Secret** in the PCO Credentials section
3. Click **Sign In to Planning Center**

## Signing in

When you click Sign In (or on first launch), pco2olp opens your web browser at the Planning Center login page. Sign in with your PCO account and click **Authorize**.

Your browser will redirect to `http://localhost:11019/callback` — this is handled automatically by pco2olp and you can close the browser tab once you see "Authentication successful".

### Firewall note

Port 11019 is only open briefly during authentication and is bound to `127.0.0.1` (localhost only) — it is not accessible from the network. If your firewall prompts you, allow access for localhost connections only.

## Re-authentication

Your access token is automatically refreshed in the background. If your refresh token expires (after 90 days of inactivity), you will need to sign in again:

1. Open **Settings**
2. Click **Sign In to Planning Center**

You can also sign in again after changing your Client ID or Secret.
