# qui-Transmission

qui-Transmission is a self-hosted web interface for managing multiple
Transmission instances from one place. It is based on the original
[autobrr/qui](https://github.com/autobrr/qui) project and keeps the same
single-binary, multi-instance workflow while replacing the qBittorrent API
integration with a Transmission RPC compatibility layer.

> All examples below use placeholders such as `<admin-user>`,
> `<admin-password>`, and `<transmission-host>`. Replace them with your own
> values and do not commit real credentials or private server addresses.

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

- A Transmission daemon with RPC enabled.
- Transmission 4.x is recommended. Newer RPC formats are preferred, but classic
  Transmission RPC responses are supported.
- For Docker installs: Docker Engine 24+ and Docker Compose v2.
- For source builds: Go 1.26+, Node.js 24+, pnpm 11.1.2, and `make`.
- Network access from qui-Transmission to each Transmission RPC endpoint.

## Quick Start With Docker Compose

Create a `docker-compose.yml`:

```yaml
services:
  qui-transmission:
    image: ghcr.io/fullerhkz/qui-transmission:latest
    container_name: qui-transmission
    restart: unless-stopped
    ports:
      - "<host-port>:7476"
    volumes:
      - ./config:/config
    environment:
      QUI__HOST: 0.0.0.0
      QUI__PORT: 7476
```

Start the container:

```sh
docker compose up -d
```

Open:

```text
http://<server-host>:<host-port>
```

You can create the first local account in the web UI. For unattended setup, use
the CLI inside the container:

```sh
docker compose run --rm qui-transmission generate-config --config-dir /config
docker compose run --rm qui-transmission create-user \
  --config-dir /config \
  --data-dir /config \
  --username "<admin-user>" \
  --password "<admin-password>"
```

Prefer omitting `--password` on shared shells so the command prompts for it
without storing it in shell history.

## Install From Source

```sh
git clone https://github.com/fullerhkz/qui-transmission.git
cd qui-transmission
corepack enable
corepack prepare pnpm@11.1.2 --activate
make build
./qui-transmission generate-config
./qui-transmission create-user --username "<admin-user>"
./qui-transmission serve
```

Default config and data directories:

| Platform | Directory |
| --- | --- |
| Linux/macOS | `~/.config/qui-transmission` |
| Windows | `%APPDATA%\qui-transmission` |
| Docker | `/config` |

Useful environment variables:

```sh
QUI__HOST=0.0.0.0
QUI__PORT=7476
QUI__BASE_URL=/
```

## Add A Transmission Instance

1. Open qui-Transmission in your browser.
2. Go to **Settings -> Instances -> Add Instance**.
3. Enter a display name.
4. Enter the Transmission RPC URL:

   ```text
   http://<transmission-host>:<rpc-port>
   ```

   or, when your Transmission setup requires the explicit RPC path:

   ```text
   http://<transmission-host>:<rpc-port>/transmission/rpc
   ```

5. If Transmission RPC authentication is enabled, enter the Transmission RPC
   username and password.
6. Save, then use **Test Connection**.

Categories from the original qui UI map to Transmission groups. Tags map to
Transmission labels.

## Updating

Docker Compose:

```sh
docker compose pull
docker compose up -d
```

Source build:

```sh
git pull --ff-only
corepack enable
corepack prepare pnpm@11.1.2 --activate
make build
```

Back up your config/data directory before major upgrades. It contains the SQLite
database, settings, API keys, encrypted instance credentials, and backups.

## Security Notes

- Do not expose qui-Transmission to the public internet without authentication
  and a reverse proxy with TLS.
- Do not commit `config.toml`, `qui-transmission.db`, `.env`, Docker override
  files, or screenshots/logs containing private hosts, tokens, usernames, or
  passwords.
- Use placeholders in public examples: `<server-host>`, `<host-port>`,
  `<transmission-host>`, `<rpc-port>`, `<admin-user>`, and `<admin-password>`.

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
