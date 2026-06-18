---
sidebar_position: 1
title: Introduction
description: Fast, modern interface for Transmission with multi-instance management, automations, and backups.
---

# qui-Transmission

qui-Transmission is a self-hosted web interface for Transmission. It is derived
from autobrr/qui-Transmission and keeps the same multi-instance UI, but talks to Transmission
RPC instead of Transmission WebAPI.

## Features

- **Single binary**: Build once and run the backend plus embedded frontend.
- **Multi-instance support**: Manage all Transmission instances from one place.
- **Large collections**: Efficient torrent lists with filtering and live updates.
- **Add torrents**: Upload `.torrent` files, paste URLs, or open magnet links.
- **Torrent operations**: Start, stop, recheck, reannounce, move, rename, delete,
  edit labels, groups, trackers, priorities, and limits.
- **Automations**: Rule-based torrent management with cross-seed awareness.
- **Orphan scan**: Find local files not associated with managed torrents.
- **Backups and restore**: Snapshot instance metadata and restore selected data.
- **Reverse proxy**: Give external apps scoped access through qui-Transmission.
- **Multi-language UI**: English, German, French, Italian, and Simplified Chinese.

## Supported Torrent Client

qui-Transmission supports Transmission RPC endpoints. Transmission 4.1 or newer
is recommended because the compatibility layer targets the JSON-RPC 2.0 API.

Categories in the original qui-Transmission UI map to Transmission groups. Tags map to
Transmission labels.

## Limitations

Transmission does not provide direct equivalents for every Transmission feature.
Native Transmission RSS management, torrent creation, path autocomplete,
subcategories, and libtorrent-specific metadata are limited or unavailable.

## Quick Start

1. [Install qui-Transmission](./getting-started/installation.md).
2. Open `http://localhost:7476`.
3. Create your local admin account.
4. Add a Transmission RPC instance.
5. Start managing your torrents.

## License

GPL-2.0-or-later.
