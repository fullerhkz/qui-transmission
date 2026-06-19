---
sidebar_position: 3
title: Docker
description: Run qui-Transmission in Docker with compose or standalone.
---

import CodeBlock from '@theme/CodeBlock';
import DockerCompose from '!!raw-loader!@site/../distrib/docker/docker-compose.yml';
import DockerComposePostgres from '!!raw-loader!@site/../distrib/docker/docker-compose.postgres.yml';
import LocalFilesystemDocker from "../_partials/_local-filesystem-docker.mdx";

# Docker

## Docker Compose

<CodeBlock language="yaml" title="docker-compose.yml">{DockerCompose}</CodeBlock>

```bash
docker compose up -d
```

## Docker Compose With Postgres

<CodeBlock language="yaml" title="docker-compose.postgres.yml">{DockerComposePostgres}</CodeBlock>

```bash
docker compose -f docker-compose.postgres.yml up -d
```

## Standalone

```bash
docker run -d \
  -p 7476:7476 \
  -v $(pwd)/config:/config \
  ghcr.io/fullerhkz/qui-transmission:latest
```

## Local Filesystem Access

<LocalFilesystemDocker />

If Transmission runs in a different container or host, configure path mappings
so qui-Transmission can translate Transmission download paths to paths visible
inside the qui-Transmission container.

## Updating

```bash
docker compose pull && docker compose up -d
```
