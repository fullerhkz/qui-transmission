---
sidebar_position: 5
title: Link Directories
description: How qui-Transmission lays out hardlink/reflink trees on disk.
---

# Link Directories

When **Hardlink mode** or **Reflink mode** is enabled for a Transmission instance, qui-Transmission creates a directory tree that matches the incoming torrentâ€™s expected layout, then adds the torrent pointing at that tree.

Because these modes add torrents with an explicit `savepath` (the link-tree root), AutoTMM is always disabled for torrents added via hardlink/reflink mode.

This applies to:
- Cross-seed searches (RSS, completion, manual, scan)
- Directory scan (dirscan) injections

## Settings

Configured per Transmission instance in **Cross-Seed â†’ Hardlink Mode**:

- **Base directory** (`HardlinkBaseDir`): root path where link trees are created.
- **Directory preset** (`HardlinkDirPreset`): controls how trees are grouped below the base directory.
- **Fallback to regular mode** (`FallbackToRegularMode`): if link-tree creation fails, qui-Transmission can fall back to â€œregular modeâ€ instead of skipping/failing.

## Directory Presets

qui-Transmission supports three presets:

- `flat`: one folder per torrent under the base directory
  - Example: `base/Torrent.Name--abcdef12/...`
- `by-tracker`: groups by tracker display name, then optional isolation folder
  - Example: `base/TrackerName/Torrent.Name--abcdef12/...`
- `by-instance`: groups by instance name, then optional isolation folder
  - Example: `base/MyInstance/Torrent.Name--abcdef12/...`

### Tracker Names (by-tracker)

For `by-tracker`, qui-Transmission resolves the folder name using the same fallback chain as cross-seed statistics:

1. **Tracker customization display name** (Settings â†’ Tracker Customizations)
2. Indexer name (from Prowlarr/Jackett)
3. Raw announce domain

Folder names are sanitized to be filesystem-safe.

### Isolation Folders

For `by-tracker` and `by-instance`, qui-Transmission adds an isolation folder only when needed:

- Torrents with a common root folder donâ€™t need isolation.
- â€œRootlessâ€ torrents (top-level files) use an isolation folder to avoid collisions.

For `flat`, an isolation folder is always used.

## Fallback to Regular Mode

If **Fallback to regular mode** is enabled, qui-Transmission will fall back to adding the torrent with a normal `savepath` (pointing at the matched source files) when link-tree creation fails.

This is particularly useful when hardlinking can intermittently fail due to filesystem/device boundaries (for example: pooled mounts where two paths look the same but resolve to different underlying devices).

Because this fallback uses regular source-file paths instead of the link-tree directory, qui-Transmission adds the torrent paused, rechecks it, and only auto-resumes after Transmission reports 100% complete. If **Skip recheck** is enabled, these fallback candidates are skipped.

If fallback is disabled, qui-Transmission skips/fails the candidate when link-tree creation fails.
