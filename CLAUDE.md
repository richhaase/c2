# c2

Concept2 Logbook CLI.

## Build & Test

```bash
make build       # build bin/c2
make install     # install c2 to GOBIN/GOPATH
make test        # tests only
make check       # full check: fmt-check + lint + vet + staticcheck + test
go run ./cmd/c2  # run directly during dev
```

## Architecture

- **Runtime:** Go
- **Entry point:** `cmd/c2`
- **Storage:** JSONL files at `~/.config/c2/data/`
- **Auth:** Static personal access token from log.concept2.com
- **Config:** JSON at `~/.config/c2/config.json`
- **CLI/TUI:** Cobra commands with a Bubble Tea terminal UI as the default command
- **CI:** GitHub Actions (`make check`)

## Source Layout

```text
cmd/c2/                 # command entry point and version metadata
internal/api/           # Concept2 API client
internal/cli/           # Cobra commands and command wiring
internal/config/        # JSON config load/save and paths
internal/display/       # formatting and table helpers
internal/export/        # CSV/JSON/JSONL export helpers
internal/model/         # data types and pace helpers
internal/report/        # static HTML report generation
internal/stats/         # sessions, weekly summaries, goal progress
internal/storage/       # JSONL workout and stroke storage
internal/sync/          # sync service
internal/tui/           # default terminal UI
```

## Hard Rules

- **No comments in source code, ever.** No exceptions. Not `//`, not `/* */`, not
  JSDoc. If code needs explanation, rename the variable, extract a function, or
  write a test that encodes the invariant. If context is load-bearing, put it in
  the commit message. Rich does not read comments.

## Key Decisions

- JSONL for storage (append-friendly, portable, small enough to parse fully)
- Static personal access token (no OAuth2 flow; C2 provides one at log.concept2.com)
- Custom goal dates independent of C2 season (May 1 - Apr 30)
- `time` field from API is in tenths of a second
- Session grouping: workouts on the same calendar day form one session
- Stroke data fields use abbreviated names from API (`t`, `d`, `p`, `spm`, `hr`)
