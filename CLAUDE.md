# c2

Concept2 Logbook CLI.

## Build & Test

```bash
make check          # fmt-check + vet + lint + race tests (all-in-one)
make test           # tests only
make lint           # golangci-lint only
make fmt            # auto-format
make build          # build ./bin/c2 with version info
make run ARGS="log -n 5"
```

## Architecture

- **Language:** Go
- **Storage:** JSONL files under a user-chosen data directory
- **Auth:** Static personal access token from log.concept2.com
- **Config:** JSON at `~/.config/c2/config.json`, mode 600
- **Dependencies:** cobra (CLI), golang.org/x/term (TTY detection)
- **Linting:** golangci-lint
- **Release:** goreleaser
- **CI:** GitHub Actions (test + lint + govulncheck + cross-platform build)

## Source Layout

```
cmd/c2/main.go            # entry point; ldflags version injection
internal/
├── cli/                  # one file per command; a leaf layer
│   ├── root.go           # command tree, error/exit-code handling
│   ├── common.go         # shared command helpers
│   └── setup.go sync.go log.go status.go trend.go export.go
│       report.go data.go show.go stats.go note.go docs.go version.go
├── models/               # data types + workout helpers (filter/resolve)
├── config/               # config load/save (machine-local, holds data_dir)
├── paths/                # data-store path resolution from data_dir
├── storage/              # JSONL read/write + store meta.json
├── store/                # store inspect/init/summary/move
├── jsonx/                # JSON encoding used everywhere (see Hard Rules)
├── envelope/             # versioned JSON output envelope
├── analysis/             # split/stroke/HR-at-pace analysis
├── stats/                # weekly summaries, sessions, goal progress + projection
├── notes/                # coaching notes: ULID ids, per-file hot set, yearly archives
├── documents/            # plan / playbook / narrative reads from the store
├── doctor/               # store validation checks
├── display/              # formatting helpers
├── terminal/             # TTY detection
└── api/                  # Concept2 API client
```

## Hard Rules

- **No comments in source code, ever.** No exceptions. Not `//`, not `/* */`, not
  doc comments on exported symbols — Go convention does not override this. If
  code needs explanation, rename the variable, extract a function, or write a
  test that encodes the invariant. If context is load-bearing, put it in the
  commit message. Rich does not read comments.
- **All JSON encoding goes through `internal/jsonx`.** Go's `encoding/json`
  escapes `<`, `>` and `&` by default, which corrupts note bodies and comments.
  jsonx disables that.
- **JSON payloads are structs, never `map[string]any`.** Go sorts map keys
  alphabetically; struct fields marshal in declaration order, and the envelope
  schemas depend on field order being stable.
- **User-visible decimals go through `models.ToFixed`.** Paces and split times
  round halves away from zero, which is what a reader expects from a stopwatch
  figure. Go's `strconv` rounds half-to-even, so a pace landing exactly on
  2:54.25 would display as 2:54.2 rather than 2:54.3. ToFixed rounds the
  shortest decimal that identifies the float, not its exact binary expansion,
  so a percentage computed as 11500/1000000 shows 1.2% rather than 1.1%.

## Key Decisions

- Go is the long-term runtime; scaffolding comes from https://github.com/richhaase/go-cli-template
- JSONL for storage (append-friendly, portable, small enough to parse fully)
- Static personal access token (no OAuth2 flow — C2 provides one at log.concept2.com)
- Custom goal dates independent of C2 season (May 1 – Apr 30)
- `time` field from API is in tenths of a second
- Workout and stroke records keep their raw JSON and re-marshal it verbatim, so
  fields the struct does not model survive a sync round-trip. The API currently
  sends `date_utc`, `privacy`, `ranked`, `real_time` and `verified` on workouts,
  none of which are modelled. A consequence: records written by a sync keep the
  API's escaping, so `"America\/Denver"` is stored with the escaped slash. That
  is the same string once parsed, and normalising it would mean re-encoding,
  which would reorder keys and drop the unmodelled fields — a worse trade
- Dates parse in `time.Local` throughout; week bucketing and calendar-day
  grouping depend on it
- Session grouping: workouts on the same calendar day form one session
- Stroke data fields use abbreviated names from API (`t`, `d`, `p`, `spm`, `hr`)
- Data store location is user-chosen (`data_dir` in config, validated by `c2 setup`); config with secrets stays machine-local at `~/.config/c2/` mode 600
- Machine-readable output via `--json` with versioned envelopes (`c2.<command>.v1`); `export -f json` emits `c2.export.v1`; `export -f jsonl` stays one workout per line for streaming
- Bare `c2` prints help; unknown commands are errors (no default command)
- `-v` is `--version`, not `--verbose` — c2 has no logging subsystem
- Commands are built by constructor, not `init()` globals, so the command tree
  can be built repeatedly in one test process
- Store state (`meta.json`: schema_version, last_sync) lives in the data dir, not config
- `meta.json`'s `schema_version` is a `*int`: absent and zero must stay
  distinguishable or a foreign meta.json gets adopted as a c2 store
- Coaching notes: one JSON file per note (sync-conflict-safe) for the last 7 days, then deterministic compaction into `notes/archive/<year>.jsonl`; reads union both and dedup by id
- Note dates use local-offset ISO timestamps so calendar days display correctly
- plan.md / playbook.md / reports/<date>.md are whole-file managed documents
