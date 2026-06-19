---
sidebar_position: 6
title: autobrr Integration
---

# autobrr Integration

qui-Transmission integrates with autobrr through webhook endpoints, enabling real-time cross-seed detection when autobrr announces new releases.

## How It Works

1. autobrr sees a new release from a tracker
2. autobrr sends the torrent name and indexer identifier to qui-Transmission's `/api/cross-seed/webhook/check` endpoint
3. qui-Transmission searches your Transmission instances for matching content
4. qui-Transmission responds with:
   - `200 OK` â€“ matching torrent is complete and ready to cross-seed
   - `202 Accepted` â€“ matching torrent exists but still downloading; retry later
   - `404 Not Found` â€“ no matching torrent exists
5. On `200 OK`, autobrr sends the torrent file to `/api/cross-seed/apply`

## Setup

### 1. Create an API Key in qui-Transmission

- Go to **Settings â†’ API Keys**
- Click **Create API Key**
- Name it (e.g., "autobrr webhook")
- Copy the generated key

### 2. Configure autobrr External Filter

:::important
Create a **new autobrr filter dedicated to qui-Transmission**.
:::

:::note
The **External** webhook (`/api/cross-seed/webhook/check`) only answers: "is this ready to cross-seed?" It does **not** add a torrent to Transmission.

You must also set up the **Action** in [Apply Endpoint](#apply-endpoint).
:::

:::tip
**Docker Compose:** if autobrr and qui-Transmission are both containers, `localhost` inside autobrr is the autobrr container, not qui-Transmission.

Use your qui-Transmission container hostname instead (often the Compose service name), for example: `http://qui-Transmission:7476/api/cross-seed/webhook/check`.
:::

In your new autobrr filter, go to **External** tab â†’ **Add new**:

| Field                     | Value                                                |
| ------------------------- | ---------------------------------------------------- |
| Type                      | `Webhook`                                            |
| Name                      | `qui-Transmission`                                                |
| On Error                  | `Reject`                                             |
| Endpoint                  | `http://localhost:7476/api/cross-seed/webhook/check` |
| HTTP Method               | `POST`                                               |
| HTTP Request Headers      | `X-API-Key=YOUR_QUI_API_KEY`                         |
| Expected HTTP Status Code | `200`                                                |

**Data (JSON):**

```json
{
  "torrentName": {{ toRawJson .TorrentName }},
  "instanceIds": [1],
  "indexer": {{ toRawJson .Indexer }}
}
```

To search all instances, omit `instanceIds`:

```json
{
  "torrentName": {{ toRawJson .TorrentName }},
  "indexer": {{ toRawJson .Indexer }}
}
```

**Field descriptions:**

- `torrentName` (required): The release name as announced
- `instanceIds` (optional): Transmission instance IDs to scan. Omit to search all instances.
- `indexer` (optional): autobrr indexer identifier (for example `hdb`). Required for qui-Transmission's HDBits-specific missing-collection fallback on `/check`.
- `findIndividualEpisodes` (optional): Override the global episode matching setting

### 3. Configure Retry Handling

Use autobrr's **Retry** block to handle `202 Accepted` responses:

- **Retry HTTP status code(s):** `202`
- **Maximum retry attempts:** `10`
- **Retry delay in seconds:** `4`

## Apply Endpoint

When `/check` returns `200 OK`, send the torrent to `/api/cross-seed/apply`:

**Action setup in autobrr:**

| Field       | Value                                                                |
| ----------- | -------------------------------------------------------------------- |
| Action Type | `Webhook`                                                            |
| Name        | `qui-Transmission cross-seed`                                                     |
| Endpoint    | `http://localhost:7476/api/cross-seed/apply?apikey=YOUR_QUI_API_KEY` |

**Payload (JSON):**

```json
{
  "torrentData": "{{ .TorrentDataRawBytes | toString | b64enc }}",
  "instanceIds": [1],
  "indexer": {{ toRawJson .Indexer }}
}
```

**Field descriptions:**

- `torrentData` (required) - Base64-encoded torrent file bytes
- `instanceIds` (optional) - Target instances (omit to apply to any matching instance)
- `indexer` (optional) - autobrr indexer identifier (for example `hdb`). When "Use indexer name as category" mode is enabled, qui-Transmission uses this identifier value as the category; ignored otherwise
- `tags` (optional) - Override webhook tags from settings
- `category` (optional) - Override category. Takes precedence over `indexer`
- `startPaused` (optional) - Override whether torrents are added paused
- `skipIfExists` (optional) - Skip adding if the torrent already exists
- `findIndividualEpisodes` (optional) - Override the global episode matching setting

Cross-seeded torrents are added paused with `skip_checking=true`. qui-Transmission polls the torrent state and auto-resumes if progress meets the size tolerance threshold. If progress is too low, it remains paused for manual review.

### Troubleshooting: autobrr matches, but nothing gets added to Transmission

Use this when autobrr shows the filter accepted the release (or your Discord notification fires), but you never see a new torrent in Transmission.

1. **Confirm you added the `/apply` Action**
   - The External webhook (`/check`) does not add torrents.
   - You need an autobrr **Action** (Webhook) that calls `/api/cross-seed/apply` (above).
2. **Fix Docker networking if you're using containers**
   - `http://localhost:7476/...` only works if autobrr can reach qui-Transmission on its own `localhost`.
   - In Docker Compose, use the qui-Transmission service hostname (example): `http://qui-Transmission:7476/api/cross-seed/apply?apikey=...`.
3. **Double-check auth**
   - `/check`: header `X-API-Key=...`
   - `/apply`: query string `?apikey=...` (as shown in this guide)
4. **Verify qui-Transmission can talk to Transmission**
   - qui-Transmission UI: **Settings â†’ Instances â†’ Test Connection**
5. **Check paused torrents**
   - Cross-seeds are often added **paused**. Look in Transmission's paused list (and any cross-seed tag/category you configured).

If you still can't see why, jump to [Cross-Seed Troubleshooting](./troubleshooting.md).

## Webhook Source Filters

By default, the webhook endpoint scans **all** torrents on your instances when looking for matches. You can configure filters to exclude certain categories or tags from being matched:

- **Exclude Categories:** Skip torrents in specific categories (e.g., `cross-seed-link`)
- **Exclude Tags:** Skip torrents with specific tags (e.g., `no-cross-seed`)
- **Include Categories:** Only match against torrents in these categories (leave empty for all)
- **Include Tags:** Only match against torrents with these tags (leave empty for all)

This is useful when:

- You have a legacy cross-seed category that shouldn't be re-matched
- Certain content types should never be considered for cross-seeding
- You want to exclude torrents with specific metadata tags

:::note
Exclude filters take precedence over include filters. Tag matching is case-sensitive. When both category and tag include filters are configured, a torrent must pass both filter checks (matching at least one allowed category AND at least one allowed tag).
:::

Configure in qui-Transmission UI: **Cross-Seed â†’ Auto â†’ Webhook / autobrr**

## Season Pack Webhook

qui-Transmission also supports a dedicated season-pack flow through separate endpoints. When autobrr announces a season pack, qui-Transmission checks your instances for matching individual episodes, links whatever is already local, and lets Transmission fetch the remainder after recheck when coverage is sufficient.

This uses different endpoints (`/api/cross-seed/season-pack/check` and `/api/cross-seed/season-pack/apply`) and requires a separate autobrr filter.

See [Season Packs](./season-packs.md) for full setup instructions.
