---
sidebar_position: 1
title: Installation
description: Build and run qui-Transmission.
---

# Installation

## Requirements

- Transmission daemon with RPC enabled.
- Transmission 4.1+ recommended.
- For source builds: Go 1.26+, Node.js 24+, and pnpm 11.1.2.

## Build From Source

```bash
git clone https://github.com/fullerhkz/qui-transmission.git
cd qui-transmission
corepack enable
corepack prepare pnpm@11.1.2 --activate
make build
```

## Run

```bash
./qui-transmission serve
```

The web interface will be available at `http://localhost:7476`.

## Generate Config

```bash
./qui-transmission generate-config
```

Default config directories:

- Linux/macOS: `~/.config/qui-transmission`
- Windows: `%APPDATA%\qui-transmission`
- Docker: `/config`

## Updating

```bash
./qui-transmission update
```

## First Setup

1. Open `http://localhost:7476`.
2. Create your account.
3. Add your Transmission RPC instance.
4. Start managing torrents.
