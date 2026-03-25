# Molt Driver Interface

Version: 0.1.0

A molt driver is a standalone executable that knows how to read from and write to a specific claw architecture. `molt` locates drivers by looking for `molt-driver-<arch>` in `$PATH` or `~/.molt/drivers/`.

## Naming convention

```
molt-driver-nanoclaw
molt-driver-zepto
molt-driver-openclaw
molt-driver-pico
```

## Protocol

Drivers communicate with `molt` via stdin/stdout using newline-delimited JSON. All messages have a `type` field.

### Export (molt → driver)

```json
{"type": "export_request", "source_dir": "/path/to/install"}
```

Driver responds with a stream of bundle parts:

```json
{"type": "group", "slug": "main", "config": {...}, "files": [...]}
{"type": "task_list", "tasks": [...]}
{"type": "skill", "name": "downwind", "files": [...]}
{"type": "session", "slug": "main", "best_effort": true, "data": {...}}
{"type": "secrets_keys", "keys": ["ANTHROPIC_API_KEY", "SIGNAL_ACCOUNT"]}
{"type": "export_complete", "warnings": []}
```

### Import (molt → driver)

```json
{"type": "import_request", "dest_dir": "/path/to/new/install", "bundle": {...}}
```

Driver responds:

```json
{"type": "progress", "message": "Registering group: main"}
{"type": "collision", "slug": "main"}          // molt handles rename prompt
{"type": "progress", "message": "Importing tasks"}
{"type": "import_complete", "warnings": ["Session import best-effort: main"]}
```

### Version check

```json
{"type": "version_request"}
```

```json
{"type": "version_response", "arch": "nanoclaw", "arch_version": "1.4.2", "driver_version": "0.1.0", "molt_protocol": "0.1.0"}
```

## Collision handling

When a driver emits `{"type": "collision", "slug": "..."}`, molt pauses, resolves via `--rename` flag or aborts with the ready-to-run fix message. The driver waits for either:

```json
{"type": "collision_resolved", "original": "main", "renamed_to": "main-imported"}
```

or

```json
{"type": "abort"}
```

## Error handling

Drivers emit errors as:

```json
{"type": "error", "code": "SOURCE_NOT_FOUND", "message": "No NanoClaw installation found at /path"}
```

Fatal errors terminate the stream. Non-fatal errors are collected as warnings in the final `export_complete` / `import_complete` message.

## Sessions

Session export/import is always best-effort. Drivers MUST emit `"best_effort": true` on session messages. `molt` surfaces a warning to the user:

```
⚠ Sessions exported best-effort — Claude session IDs may not be valid in target arch.
```

## Auto-detection

When `--arch` is not specified for export, `molt` calls each installed driver's version endpoint and asks it to probe the source directory:

```json
{"type": "probe_request", "source_dir": "/path/to/install"}
```

```json
{"type": "probe_response", "confidence": 0.95, "arch": "nanoclaw"}
```

Highest confidence wins. Drivers should return 0.0 if they don't recognize the installation.

## Minimal viable driver

A driver MUST implement:
- `version_request` / `version_response`
- `export_request` with at least groups and secrets_keys
- `import_request` with group registration

Sessions, tasks, and skills are optional (emit warnings if skipped).
