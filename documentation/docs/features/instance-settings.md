---
sidebar_position: 3
title: Instance Settings
description: Configure Transmission instance connections in qui-Transmission.
---

# Instance Settings

Add and configure Transmission instances that qui-Transmission connects to. Each instance represents a separate Transmission WebUI that qui-Transmission can manage.

## Adding an Instance

1. Open qui-Transmission and go to **Settings â†’ Instances**
2. Click **Add Instance**
3. Enter connection details and click **Save**

## Instance Configuration

On the Dashboard, click the gear icon next to an instance name. In **Settings â†’ Instances**, click the three-dot menu and select **Edit**.

### Connection Settings

| Field | Description |
|-------|-------------|
| **Name** | Display name shown in qui-Transmission's sidebar and instance selector. |
| **Host** | Full URL to Transmission WebUI (e.g., `http://localhost:8080`). |
| **Skip TLS Verification** | Bypass certificate validation for self-signed certificates. |
| **Local Filesystem Access** | Enable for features requiring direct file access. |

### Authentication

qui-Transmission supports multiple authentication methods depending on your setup:

| Option | When to Use |
|--------|-------------|
| **Transmission Login** | Enable and enter credentials for standard WebUI authentication. Disable if Transmission bypasses auth for localhost or whitelisted IPs. |
| **HTTP Basic Auth** | Enable when a reverse proxy adds Basic Authentication in front of Transmission. |

:::note
HTTP Basic Auth is separate from Transmission's built-in auth. Enable it when your reverse proxy (nginx, Caddy, etc.) requires credentials before reaching Transmission.
:::

## Local Filesystem Access

When enabled, qui-Transmission can access the same filesystem as Transmission. This unlocks several features:

- **Content File Download** - Download individual files from a torrent's content directly through the browser (right-click a file in the Content tab).
- **Hardlink Detection** - Automations can detect whether torrent files have hardlinks to your media library.
- **Orphan Scan** - Find files on disk that aren't tracked by any torrent.
- **Free Space (Path)** - Automation rules can check free space on specific mount points instead of relying on Transmission's reported value.

:::warning
Only enable this if qui-Transmission runs on the same machine (or has the same mounts) as Transmission. If paths don't match, features will fail silently or produce incorrect results.
:::

For Docker deployments, ensure the container has the necessary volume mounts. See [Docker configuration](../getting-started/docker.md) for details.

## Instance Actions

At the bottom of the settings panel:

- **Enable / Disable** - Toggle whether qui-Transmission actively connects to and manages this instance.
- **Delete** - Remove the instance from qui-Transmission. This does not affect Transmission itself.

## Transmission Preferences

The settings dialog includes tabs for configuring Transmission's application preferences (speed limits, queue management, connection settings, etc.). These are passed directly to Transmission's API and behave identically to the native WebUI settings.
