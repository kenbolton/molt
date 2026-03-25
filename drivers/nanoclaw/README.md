# molt-driver-nanoclaw

The molt driver for [NanoClaw](https://github.com/qwibitai/nanoclaw) installations.

Ships with NanoClaw. Also available standalone for use with other molt-compatible tools.

## Installation

```bash
# Included with NanoClaw — symlink to PATH
ln -s ~/src/nanoclaw/scripts/molt-driver-nanoclaw ~/.molt/drivers/molt-driver-nanoclaw

# Or install globally
cp molt-driver-nanoclaw /usr/local/bin/
```

## What it reads

- `store/messages.db` — registered groups, scheduled tasks
- `groups/<slug>/` — CLAUDE.md, conversations, memory files
- `data/sessions/<slug>/` — Claude session cache (best-effort)
- `.env` — secret key names (not values)

## Status

🚧 In development — see [molt roadmap](../../spec/ROADMAP.md)
