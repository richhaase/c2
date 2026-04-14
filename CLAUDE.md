# c2

Concept2 Logbook CLI — `@richhaase/c2` on npm.

## Build & Test

```bash
bun run check       # typecheck + lint + test (all-in-one)
bun test            # tests only
bun run lint        # lint only (biome)
bun run fmt         # auto-fix lint + format
bun src/index.ts    # run directly during dev
```

## Architecture

- **Runtime:** Bun (TypeScript)
- **Storage:** JSONL files at `~/.config/c2/data/`
- **Auth:** Static personal access token from log.concept2.com
- **Config:** JSON at `~/.config/c2/config.json`
- **Dependencies:** commander (CLI)
- **Linting:** biome (lint + format)
- **CI:** GitHub Actions (typecheck + lint + test)

## Source Layout

```
src/
├── index.ts              # CLI entry point (binary: c2)
├── models.ts             # Data types + helpers
├── config.ts             # JSON config load/save
├── storage.ts            # JSONL read/write
├── display.ts            # Formatting helpers
├── sessions.ts           # Session grouping (same-day merge)
├── stats.ts              # Weekly summaries + goal progress
├── api/
│   └── client.ts         # Concept2 API client
├── commands/
│   ├── setup.ts
│   ├── sync.ts
│   ├── log.ts
│   ├── status.ts
│   ├── trend.ts
│   ├── export.ts
│   └── report.ts
└── *.test.ts             # Colocated tests
```

## Hard Rules

- **No comments in source code, ever.** No exceptions. Not `//`, not `/* */`, not
  JSDoc. If code needs explanation, rename the variable, extract a function, or
  write a test that encodes the invariant. If context is load-bearing, put it in
  the commit message. Rich does not read comments.

## Key Decisions

- JSONL for storage (append-friendly, portable, small enough to parse fully)
- Static personal access token (no OAuth2 flow — C2 provides one at log.concept2.com)
- Custom goal dates independent of C2 season (May 1 – Apr 30)
- `time` field from API is in tenths of a second
- Session grouping: workouts on the same calendar day form one session
- Stroke data fields use abbreviated names from API (`t`, `d`, `p`, `spm`, `hr`)
