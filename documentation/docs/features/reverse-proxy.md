---
sidebar_position: 6
title: Reverse Proxy
description: Let external apps access Transmission through qui-Transmission without credentials.
---

# Reverse Proxy for External Applications

qui-Transmission includes a built-in reverse proxy that allows external applications like autobrr, Sonarr, Radarr, and other tools to connect to your Transmission instances **without needing Transmission credentials**.

## How It Works

qui-Transmission maintains a shared session with Transmission and proxies requests from your external apps. This eliminates login thrash - automation tools reuse the live session instead of racing to re-authenticate.

## Setup Instructions

### 1. Create a Client Proxy API Key

1. Open qui-Transmission in your browser
2. Go to **Settings â†’ Client Proxy Keys**
3. Click **"Create Client API Key"**
4. Enter a name for the client (e.g., "Sonarr")
5. Choose the Transmission instance you want to proxy
6. Click **"Create Client API Key"**
7. **Copy the generated proxy url immediately** - it's only shown once

### 2. Configure Your External Application

Use qui-Transmission as the Transmission host with the special proxy URL format:

**Complete URL example:**
```
http://localhost:7476/proxy/abc123def456ghi789jkl012mno345pqr678stu901vwx234yz
```

## Application-Specific Setup

### Sonarr / Radarr

1. Go to `Settings â†’ Download Clients`
2. Select `Show Advanced`
3. Add a new **Transmission** client
4. Set the host and port of qui-Transmission
5. Add URL Base (`/proxy/...`) - remember to include `/qui-Transmission/` if you use custom baseurl
6. Click **Test** and then **Save** once the test succeeds

### autobrr

1. Open `Settings â†’ Download Clients`
2. Add **Transmission** (or edit an existing one)
3. Enter the full url like: `http://localhost:7476/proxy/abc123def456ghi789jkl012mno345pqr678stu901vwx234yz`
4. Leave username/password blank and press **Test**
5. Leave basic auth blank since qui-Transmission handles that

For cross-seed integration with autobrr, see the [Cross-Seed](./cross-seed/autobrr.md) section.

### cross-seed

1. Open cross-seed config file
2. Add or edit the `torrentClients` section
3. Append the full url following the documentation:
   ```
   torrentClients: ["Transmission:http://localhost:7476/proxy/abc123def456ghi789jkl012mno345pqr678stu901vwx234yz"],
   ```
4. Save the config file and restart cross-seed

### Upload Assistant

1. Open the Upload Assistant config file
2. Add or edit `qui_proxy_url` under the Transmission client settings
3. Append the full url like: `"qui_proxy_url": "http://localhost:7476/proxy/abc123def456ghi789jkl012mno345pqr678stu901vwx234yz",`
4. All other auth type can remain unchanged
5. Save the config file

## Supported Applications

This reverse proxy will work with any application that supports Transmission's Web API.

## Security Features

- **API Key Authentication** - Each client requires a unique key
- **Instance Isolation** - Keys are tied to specific Transmission instances
- **Usage Tracking** - Monitor which clients are accessing your instances
- **Revocation** - Disable access instantly by deleting the API key
- **No Credential Exposure** - Transmission passwords never leave qui-Transmission

## Intercepted Endpoints

The proxy intercepts certain Transmission API endpoints to improve performance and enable qui-Transmission-specific features. Most requests are forwarded transparently to Transmission.

### Read Operations (Served from qui-Transmission)

These endpoints are served directly from qui-Transmission's sync manager for faster response times:

| Endpoint | Description |
|----------|-------------|
| `/api/v2/torrents/info` | Torrent list with standard Transmission filtering |
| `/api/v2/torrents/search` | Enhanced torrent list with fuzzy search (qui-Transmission-specific) |
| `/api/v2/torrents/categories` | Category list from synchronized data |
| `/api/v2/torrents/tags` | Tag list from synchronized data |
| `/api/v2/torrents/properties` | Torrent properties |
| `/api/v2/torrents/trackers` | Torrent trackers with icon discovery |
| `/api/v2/torrents/files` | Torrent file list |

These endpoints proxy to Transmission and update qui-Transmission's local state:

| Endpoint | Description |
|----------|-------------|
| `/api/v2/sync/maindata` | Full sync data (updates qui-Transmission's cache) |
| `/api/v2/sync/torrentPeers` | Peer data (updates qui-Transmission's peer state) |

### Write Operations

| Endpoint | Behavior |
|----------|----------|
| `/api/v2/auth/login` | No-op, returns success if instance is healthy |
| `/api/v2/torrents/reannounce` | Delegated to reannounce service when tracker monitoring is enabled |
| `/api/v2/torrents/setLocation` | Forwards to Transmission, invalidates file cache |
| `/api/v2/torrents/renameFile` | Forwards to Transmission, invalidates file cache |
| `/api/v2/torrents/renameFolder` | Forwards to Transmission, invalidates file cache |
| `/api/v2/torrents/delete` | Forwards to Transmission, invalidates file cache |

All other endpoints are forwarded transparently to Transmission.