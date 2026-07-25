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
├── models.ts             # Data types + workout helpers (filter/resolve)
├── config.ts             # JSON config load/save (machine-local, holds data_dir)
├── paths.ts              # Data-store path resolution from data_dir
├── storage.ts            # JSONL read/write + store meta.json
├── data.ts               # Store inspect/init/summary/move
├── envelope.ts           # Versioned JSON output envelope
├── analysis.ts           # Split/stroke/HR-at-pace analysis
├── notes.ts              # Coaching notes: ULID ids, per-file hot set, yearly archives
├── documents.ts          # plan / playbook / narrative reads from the store
├── doctor.ts             # Store validation checks
├── display.ts            # Formatting helpers
├── stats.ts              # Weekly summaries, sessions, goal progress + projection
├── api/
│   └── client.ts         # Concept2 API client
├── commands/
│   ├── setup.ts
│   ├── sync.ts
│   ├── log.ts
│   ├── status.ts
│   ├── trend.ts
│   ├── export.ts
│   ├── report.ts
│   ├── data.ts
│   ├── show.ts
│   ├── stats.ts
│   ├── note.ts
│   └── docs.ts           # plan / playbook / narrative documents
└── *.test.ts             # Colocated tests
```

## Hard Rules

- **No comments in source code, ever.** No exceptions. Not `//`, not `/* */`, not
  JSDoc. If code needs explanation, rename the variable, extract a function, or
  write a test that encodes the invariant. If context is load-bearing, put it in
  the commit message. Rich does not read comments.

## Key Decisions

- Bun/TypeScript is the long-term runtime
- JSONL for storage (append-friendly, portable, small enough to parse fully)
- Static personal access token (no OAuth2 flow — C2 provides one at log.concept2.com)
- Custom goal dates independent of C2 season (May 1 – Apr 30)
- `time` field from API is in tenths of a second
- Session grouping: workouts on the same calendar day form one session
- Stroke data fields use abbreviated names from API (`t`, `d`, `p`, `spm`, `hr`)
- Data store location is user-chosen (`data_dir` in config, validated by `c2 setup`); config with secrets stays machine-local at `~/.config/c2/` mode 600
- Machine-readable output via `--json` with versioned envelopes (`c2.<command>.v1`); `export -f json` emits `c2.export.v1` (the raw array retired with the last legacy consumer); `export -f jsonl` stays one workout per line for streaming
- Bare `c2` prints help; unknown commands are errors (no default command)
- Store state (`meta.json`: schema_version, last_sync) lives in the data dir, not config
- Coaching notes: one JSON file per note (sync-conflict-safe) for the last 7 days, then deterministic compaction into `notes/archive/<year>.jsonl`; reads union both and dedup by id
- Note dates use local-offset ISO timestamps so calendar days display correctly
- plan.md / playbook.md / reports/<date>.md are whole-file managed documents
- `commands/` is a leaf layer: commands import library modules, never each other
- Goal projection has one implementation (`projectGoal`), shared by the HTML report, `report --data`, and `stats goal`
- `display.date_format` accepts `%m/%d` (default) and `%Y-%m-%d`
