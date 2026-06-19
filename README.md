# qui-Transmission

qui-Transmission is a self-hosted web interface for managing multiple
Transmission instances from one place. It is based on the original
[autobrr/qui](https://github.com/autobrr/qui) project and keeps the same
single-binary, multi-instance workflow while replacing the qBittorrent API
integration with a Transmission RPC compatibility layer.

## Features

- Manage multiple Transmission instances from one dashboard.
- Add torrents from files, URLs, and magnet links.
- Start, stop, recheck, reannounce, rename, move, and delete torrents.
- Edit labels, groups, trackers, priorities, speed limits, ratio limits, and
  queue state where Transmission exposes the matching RPC operation.
- View torrent details, files, trackers, peers, transfer stats, and history.
- Use automations, cross-seed helpers, orphan scans, notifications, backups,
  API keys, reverse proxy routes, and the existing qui web UI workflow.

## Requirements

- Transmission daemon with RPC enabled.
- Transmission 4.1 or newer is recommended because qui-Transmission targets the
  JSON-RPC 2.0 API and snake_case RPC fields.
- For source builds: Go 1.26+, Node.js 24+, and pnpm 11.1.2.
- Optional: Docker or Docker Compose for container deployment.

## Install From Source

```sh
git clone https://github.com/fullerhkz/qui-transmission.git
cd qui-transmission
corepack enable
corepack prepare pnpm@11.1.2 --activate
make build
./qui-transmission serve
```

By default the config directory is:

- Linux/macOS: `~/.config/qui-transmission`
- Windows: `%APPDATA%\qui-transmission`
- Docker: `/config`

Generate a starter config:

```sh
./qui-transmission generate-config
```

## Docker Compose

```yaml
services:
  qui-transmission:
    image: ghcr.io/fullerhkz/qui-transmission:latest
    container_name: qui-transmission
    restart: unless-stopped
    ports:
      - "7476:7476"
    volumes:
      - ./qui-transmission:/config
```

Start it with:

```sh
docker compose up -d
```

## Add A Transmission Instance

1. Open `http://localhost:7476`.
2. Complete the local admin setup.
3. Go to Settings -> Instances -> Add Instance.
4. Use the Transmission RPC URL, for example:
   - `http://localhost:9091`
   - `http://localhost:9091/transmission/rpc`
5. Enter the Transmission RPC username and password if authentication is
   enabled.
6. Save, then use "Test Connection" to verify access.

Categories from the original qui UI map to Transmission groups. Tags map to
Transmission labels.

## Known Differences From qBittorrent qui

Transmission does not expose every qBittorrent WebAPI feature. The compatibility
layer keeps the UI and behavior aligned where possible, but these areas are
limited or unavailable:

- Native qBittorrent RSS feed management is not available in Transmission RPC.
- Torrent creation is not exposed by Transmission RPC.
- Some qBittorrent-specific preferences, path autocomplete, subcategories, and
  libtorrent-specific metadata do not have direct Transmission equivalents.
- Local filesystem features still require qui-Transmission to see the same
  paths as the Transmission instance, or matching path mappings.

## Development

Common commands:

```sh
make dev
make frontend
make backend
make precommit
make build
```

Run Go tests with:

```sh
go test -race -count=1 ./...
```

Frontend commands live in `web/` and use pnpm:

```sh
cd web
pnpm install
pnpm build
pnpm test
```

## License

GPL-2.0-or-later. This project is derived from autobrr/qui and keeps the
original license.
