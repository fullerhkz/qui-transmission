---
sidebar_position: 3
title: Base URL
---

# Base URL Configuration

If you need to serve qui-Transmission from a subdirectory (e.g., `https://example.com/qui-Transmission/`), you can configure the base URL.

## Using Environment Variable

```bash
QUI__BASE_URL=/qui-Transmission/ ./qui-Transmission
```

## Using Configuration File

Edit your `config.toml`:

```toml
baseUrl = "/qui-Transmission/"
```

## With Nginx Reverse Proxy

```nginx
# Redirect /qui-Transmission to /qui-Transmission/ for proper SPA routing
location = /qui-Transmission {
    return 301 /qui-Transmission/;
}

location /qui-Transmission/ {
    proxy_pass http://localhost:7476/qui-Transmission/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```
