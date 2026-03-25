# Molt Roadmap

## v0.1.0 — NanoClaw foundation

- [ ] Core molt binary (Go or Rust)
- [ ] Bundle format implemented
- [ ] `molt export` — NanoClaw source driver
- [ ] `molt import` — NanoClaw target driver
- [ ] `molt upgrade` — bundle version migration
- [ ] `molt archs` — list installed drivers
- [ ] Slug collision detection + `--rename` flag
- [ ] `secrets-template.env` generation
- [ ] `--dry-run` support
- [ ] Session best-effort export

## v0.2.0 — Cross-arch

- [ ] ZeptoClaw driver (import from OpenClaw → ZeptoClaw already exists; extend to NanoClaw source)
- [ ] OpenClaw driver (read existing OpenClaw installations)
- [ ] PicoClaw driver

## v0.3.0 — Polish

- [ ] `molt diff <bundle1> <bundle2>` — show what changed between exports
- [ ] Incremental export (only export changes since last molt)
- [ ] Per-group exclude: `--exclude <slug>`
- [ ] Shell completions

## Future (v1+)

- [ ] Optional driver registry (ClawHub-style, opt-in)
- [ ] `molt schedule` — scheduled/automatic exports as backup
- [ ] Encrypted bundles (`--encrypt`)
- [ ] Multi-source merge: import from two bundles into one installation
