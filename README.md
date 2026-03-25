# molt

Portable migration tool for claw agent architectures.

Move your agents, groups, memory, skills, and config between NanoClaw, OpenClaw, ZeptoClaw, PicoClaw, and others — without starting over.

```
molt <source> <dest> --arch <target>
```

## The problem

Every claw architecture has its own format for groups, memory, config, and credentials. When you need to move — new machine, new architecture, scaling up or down — you're on your own. Existing tools only know about OpenClaw, and nothing knows how to read NanoClaw.

`molt` fixes that.

## How it works

`molt` exports your installation into a portable `.molt` bundle, then imports it into the target architecture using a driver for that arch. Drivers are standalone binaries that ship with each claw architecture. No registry, no network dependency, no trust issues.

```
molt export ~/nanoclaw-install --out ~/my-agents.molt
molt import ~/my-agents.molt ~/new-install --arch zepto
```

Or combined:

```
molt ~/old-install ~/new-install --arch zepto
```

## What moves

| Item | Moves? | Notes |
|------|--------|-------|
| Group configs (name, trigger, JID) | ✓ | Normalized format |
| Per-group memory (CLAUDE.md, files) | ✓ | Verbatim |
| Global memory | ✓ | Verbatim |
| Conversation history | ✓ | Common schema + arch extensions |
| Scheduled tasks | ✓ | Normalized cron/interval |
| Skills | ✓ | Arch-neutral |
| Sessions (Claude session cache) | ⚠️ | Best-effort, with warning |
| Secrets / API keys | ✗ | `secrets-template.env` provided |
| Container images | ✗ | Rebuilt by target arch |

## Bundle format

A `.molt` file is a gzipped tar with a `manifest.json` and a predictable directory layout. Human-readable. Version-upgradeable via `molt upgrade`.

See [spec/BUNDLE.md](spec/BUNDLE.md) for the full format.

## Drivers

Each claw architecture ships a `molt-driver` binary that implements the driver interface. `molt` locates drivers via `$PATH` or `~/.molt/drivers/`.

```
molt-driver-nanoclaw   # ships with NanoClaw
molt-driver-zepto      # ships with ZeptoClaw
molt-driver-openclaw   # ships with OpenClaw
molt-driver-pico       # ships with PicoClaw
```

See [spec/DRIVER.md](spec/DRIVER.md) for the driver interface spec.

## Naming collisions

If a group slug already exists in the destination, molt aborts with a ready-to-run fix:

```
Error: agent slug collision — "main" already exists in dest.
Re-run with:
  molt import bundle.molt /dest --arch nanoclaw --rename main=main-imported
```

## Commands

```
molt export <source>            Export to bundle
  --out <file>                  Output path (default: <source-basename>.molt)
  --arch <name>                 Override source arch detection

molt import <bundle> <dest>     Import from bundle
  --arch <name>                 Target architecture (required)
  --rename <old>=<new>          Rename group slug on import (repeatable)
  --dry-run                     Show what would happen, make no changes

molt upgrade <bundle>           Upgrade bundle to current format version
  --out <file>                  Output path (default: overwrites in place)

molt archs                      List installed drivers and their versions

molt <source> <dest>            Export + import in one step
  --arch <name>                 Target architecture (required)
  --rename <old>=<new>          Rename group slug
  --dry-run                     Dry run
```

## Status

Early spec / pre-alpha. NanoClaw driver in active development.

Drivers planned: NanoClaw, ZeptoClaw, OpenClaw, PicoClaw.

Contributions welcome — especially drivers for architectures we haven't seen yet.

## Future

A driver registry (ClawHub-style) is on the roadmap for discovery and auto-install of drivers. The core tool will remain peer-to-peer; the registry is opt-in.
