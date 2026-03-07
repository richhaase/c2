# c2cli

Concept2 Logbook CLI — personal tool for rowing data sync and analysis.

## Build & Test

```bash
bun test            # run tests
bun run check       # typecheck (tsc --noEmit)
bun src/index.ts    # run directly during dev
```

## Architecture

- **Runtime:** Bun (TypeScript)
- **Storage:** JSONL files at `~/.config/c2cli/data/`
- **Auth:** Static personal access token from log.concept2.com
- **Config:** TOML at `~/.config/c2cli/config.toml`
- **Dependencies:** commander (CLI), smol-toml (config)

## Source Layout

```
src/
├── index.ts              # CLI entry point (binary: c2)
├── models.ts             # Data types + helpers
├── config.ts             # TOML config load/save
├── storage.ts            # JSONL read/write
├── display.ts            # Formatting helpers
├── sessions.ts           # Session grouping (same-day merge)
├── api/
│   └── client.ts         # Concept2 API client
├── commands/
│   ├── setup.ts
│   ├── sync.ts
│   ├── log.ts
│   ├── status.ts
│   ├── trend.ts
│   └── export.ts
└── *.test.ts             # Colocated tests
```

## Key Decisions

- JSONL for storage (append-friendly, portable, small enough to parse fully)
- Static personal access token (no OAuth2 flow — C2 provides one at log.concept2.com)
- Custom goal dates independent of C2 season (May 1 – Apr 30)
- `time` field from API is in tenths of a second
- Session grouping: workouts on the same calendar day form one session
- Stroke data fields use abbreviated names from API (`t`, `d`, `p`, `spm`, `hr`)
