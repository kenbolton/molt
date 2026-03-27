# molt sync — Scheduled Backup Spec

> v0.1.0-draft

`molt sync` is a one-way backup daemon. It exports `.molt` bundles from a running installation to a configurable destination on a cron or interval schedule. The first export is a full bundle; subsequent exports are delta bundles containing only what changed. `molt restore` assembles any full + delta chain to reach a specific point in time.

Scope: one-way export for backup and disaster recovery. Two-way sync and cross-arch mirroring are explicitly out of scope.

---

## CLI

```
molt sync init <destination>    write .molt-sync.json with defaults, print next steps
molt sync start                 launch the daemon (daemonizes, writes PID file)
molt sync stop                  stop the daemon gracefully (SIGTERM, waits for in-progress export)
molt sync status                show daemon state, last run, next run, bundle count
molt sync run                   trigger an immediate sync (foreground, exits when done)
molt sync list                  list all saved bundles at the destination with timestamps

molt restore                    restore from the latest bundle at the configured destination
  --at   <timestamp>            restore to this point in time (ISO 8601; default: latest)
  --from <destination>          destination URI to restore from (default: from .molt-sync.json)
  --to   <source-dir>           installation to restore into (default: auto-detect)
  --dry-run                     print the bundle chain that would be applied without importing
```

`init` is idempotent — safe to re-run, will not overwrite existing config unless `--force` is passed.

---

## Configuration

Config is looked up in order:

1. `<source_dir>/.molt-sync.json` — co-located with the installation (commit this)
2. `~/.molt/sync.json` — global fallback

```json
{
  "destination": "s3://my-bucket/backups/nanoclaw",
  "schedule": "0 * * * *",
  "full_every": "7d",
  "retention": {
    "keep_bundles": 168,
    "keep_full": 4
  },
  "arch": "nanoclaw",
  "source_dir": ""
}
```

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `destination` | string | — | Destination URI (required) |
| `schedule` | string | `"0 * * * *"` | Cron expression or interval (`"1h"`, `"15m"`) |
| `full_every` | string | `"7d"` | How often to write a full bundle; in between: delta |
| `retention.keep_bundles` | int | `168` | Total bundles to retain; oldest pruned first |
| `retention.keep_full` | int | `4` | Minimum full bundles always retained regardless of age |
| `arch` | string | auto-detect | Driver arch name |
| `source_dir` | string | `""` | Installation path (empty = auto-detect) |

`keep_full` acts as a floor: even if `keep_bundles` would prune a full bundle, it is kept until `keep_full` is satisfied.

---

## Destinations

| Scheme | Example | Notes |
|--------|---------|-------|
| `file://` | `file:///backups/nanoclaw` | Local or network-mounted path |
| `s3://` | `s3://bucket/prefix` | S3 or S3-compatible (Cloudflare R2, Minio); credentials via standard env vars or `~/.aws/credentials` |
| `ssh://` | `ssh://host/path` | Via rsync; uses `ssh-agent` |

All adapters implement three operations:

```
Put(name string, r io.Reader) error
Get(name string, w io.Writer) error
List() ([]BundleEntry, error)
```

`BundleEntry` carries `Name`, `Timestamp`, `Type` (`"full"` or `"delta"`), `Size`, and `BaseHash`.

---

## Bundle naming

```
<arch>-<timestamp>-full.molt
<arch>-<timestamp>-delta-<base-hash8>.molt
```

`<timestamp>` is `YYYYMMDDTHHmmssZ`. `<base-hash8>` is the first 8 hex characters of the SHA-256 of the base full bundle.

Example sequence (hourly schedule, weekly full):

```
nanoclaw-20260327T090000Z-full.molt
nanoclaw-20260327T100000Z-delta-3f8a1c2d.molt
nanoclaw-20260327T110000Z-delta-3f8a1c2d.molt
nanoclaw-20260327T120000Z-delta-3f8a1c2d.molt
  ...
nanoclaw-20260403T090000Z-full.molt         ← weekly full resets base
nanoclaw-20260403T100000Z-delta-9b4e72a1.molt
```

All deltas between two consecutive full bundles share the same `<base-hash8>`.

---

## Incremental mechanism

### State file

`<source_dir>/.molt-sync-state.json` — written atomically after each successful export.

```json
{
  "last_sync_at": "2026-03-27T10:00:00Z",
  "last_full_at": "2026-03-27T09:00:00Z",
  "last_bundle": "nanoclaw-20260327T100000Z-delta-3f8a1c2d.molt",
  "base_hash": "3f8a1c2d",
  "bundles": [
    {
      "name": "nanoclaw-20260327T090000Z-full.molt",
      "timestamp": "2026-03-27T09:00:00Z",
      "type": "full",
      "hash": "3f8a1c2d"
    },
    {
      "name": "nanoclaw-20260327T100000Z-delta-3f8a1c2d.molt",
      "timestamp": "2026-03-27T10:00:00Z",
      "type": "delta",
      "base": "3f8a1c2d"
    }
  ]
}
```

### Export flow

On each tick:

1. Read `.molt-sync-state.json`; determine whether this run is a full or delta based on `last_full_at` and `full_every`
2. Call the driver with `export_request`; for delta runs, include `"since": "<last_sync_at>"` — the driver returns only content modified after that timestamp
3. Assemble the bundle; for delta bundles, add to `manifest.json`:
   ```json
   {
     "bundle_type": "delta",
     "base_bundle": "3f8a1c2d",
     "since": "2026-03-27T09:00:00Z"
   }
   ```
   Full bundles use `"bundle_type": "full"` and omit `base_bundle` and `since`.
4. Upload to destination via adapter `Put`
5. Write updated state file atomically
6. Prune bundles at destination per retention policy (delete oldest when `keep_bundles` exceeded, respecting `keep_full` floor)

If `since` is not supported by a driver, it returns all content and molt falls back to a full bundle for that run.

---

## Restore

```
molt restore --at 2026-03-27T10:30 --from s3://my-bucket/backups/nanoclaw --to ~/src/nanoclaw
```

Steps:

1. `List()` all bundles at the destination
2. Find the latest full bundle whose timestamp is at or before `--at`
3. Collect all delta bundles whose `base` matches that full bundle's hash and whose timestamp is ≤ `--at`
4. Sort deltas by timestamp ascending
5. Assemble: start with the full bundle, layer each delta in order (later content wins per group/conversation)
6. Run `molt import --overwrite` into `--to`

With `--dry-run`:

```
Restore chain for 2026-03-27T10:30:
  [full]  nanoclaw-20260327T090000Z-full.molt         (2026-03-27 09:00)
  [delta] nanoclaw-20260327T100000Z-delta-3f8a1c2d.molt (2026-03-27 10:00)
  2 bundles · 3 groups · ~42MB assembled
  (dry run — no changes made)
```

---

## Daemon

`molt sync start` forks a background process and writes its PID to `~/.molt/sync.pid`.

On SIGTERM: finishes any in-progress export, then exits. State is written before the process exits so the next startup resumes correctly.

`molt sync status` reads `.molt-sync-state.json` and the PID file only — no IPC required:

```
Daemon:      running (pid 12345)
Destination: s3://my-bucket/backups/nanoclaw
Schedule:    0 * * * * (next run in 23m)
Last sync:   2026-03-27T10:00:00Z (delta, 1.2MB, 0.4s)
Last full:   2026-03-27T09:00:00Z
Bundles:     14 stored (4 full, 10 delta)
```

Platform integration (v1 polish, not in initial implementation):

- macOS: `molt sync install` writes a launchd plist to `~/Library/LaunchAgents/dev.molt.sync.plist`
- Linux: `molt sync install` writes a systemd user unit to `~/.config/systemd/user/molt-sync.service`

---

## Error handling

| Failure | Behaviour |
|---------|-----------|
| Driver export fails | Log error, retain previous state, retry on next tick |
| Destination write fails | Preserve local temp bundle, retry upload on next tick; do not advance state |
| Destination pruning fails | Log warning, continue — stale bundles are harmless |
| Restore assembly fails | Abort before `molt import` runs; installation unchanged |
| `molt import` fails | Transactional — no partial state written to disk |
| State file corrupted | Fall back to full export on next run; log warning |

Errors are written to `<source_dir>/.molt-sync.log` (rotated at 10MB, 3 files kept).
