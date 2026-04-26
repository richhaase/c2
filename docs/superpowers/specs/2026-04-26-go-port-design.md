# c2 Go Port Design

Date: 2026-04-26

## Objective

Replace the Bun/TypeScript implementation of `c2` with a Go-native CLI and TUI.
The port should improve distribution, maintainability, versioning, release flow, and
interactive data browsing while preserving the core purpose of the tool: syncing,
viewing, reporting, and exporting Concept2 Logbook data.

This is a redesign, not a mechanical translation. Behavior changes are allowed when
they make the Go app clearer or more useful, but they should be explicit and covered
by tests.

## Non-Objectives

- Keep npm or Bun packaging.
- Maintain TypeScript source in parallel.
- Add unrelated Concept2 features during the port.
- Change the local data directory unless a migration need is discovered.
- Preserve old default behavior where bare `c2` generated an HTML report.
- Add a `c2 tui` alias.

## Reference Patterns

Follow the Go tool patterns used in the local reference repositories:

- `cmd/<name>/main.go` binary entrypoint.
- Internal packages for CLI, domain logic, storage, API, output, and TUI code.
- `Makefile` targets patterned after `reach` and `bigboard`: `build`, `install`,
  `test`, `test-coverage`, `fmt`, `lint`, `vet`, `tidy`, `clean`, `staticcheck`,
  and `check`.
- Version injection via `-ldflags` into `main.version`, `main.commit`, and
  `main.date`.
- `runtime/debug.ReadBuildInfo` fallback so `go install` builds still show useful
  version metadata.
- GoReleaser config patterned after `bigboard`/`acr`: darwin and linux builds for
  amd64 and arm64, tar archives, checksums, GitHub release notes, and optional
  future Homebrew support.

## Command Surface

Bare `c2` launches the TUI. The TUI is the primary app surface.

Scriptable commands remain available for automation and static output:

- `c2 setup`: configure token, goal, and preferences.
- `c2 sync`: sync workouts and stroke data from Concept2.
- `c2 report`: generate the static self-contained HTML report.
- `c2 export`: export data as CSV, JSON, or JSONL.
- `c2 log`: print recent workouts.
- `c2 status`: print goal progress.
- `c2 trend`: print weekly trends.

There will be no `c2 tui` alias in the initial port. If an explicit TUI command
becomes useful later, it can be added as a deliberate follow-up.

## Architecture

Use Cobra for the CLI, matching `reach` and `acr`.

Proposed package layout:

```text
cmd/c2/
  main.go
  version.go

internal/cli/
  root.go
  setup.go
  sync.go
  log.go
  status.go
  trend.go
  export.go
  report.go

internal/api/
  client.go

internal/config/
  config.go
  paths.go

internal/model/
  workout.go
  pace.go

internal/storage/
  workouts.go
  strokes.go
  migrate.go

internal/stats/
  sessions.go
  weekly.go
  goals.go

internal/display/
  format.go
  tables.go

internal/report/
  html.go
  templates.go

internal/export/
  csv.go
  json.go
  filter.go

internal/tui/
  app.go
  model.go
  views.go
  actions.go
  styles.go
```

Shared packages must stay independent of terminal UI concerns. The CLI commands,
HTML report, and TUI should all call the same storage, stats, API, report, and
export services instead of duplicating calculations or shelling out to subcommands.

## Data And Compatibility

Keep the current local paths:

- Config: `~/.config/c2/config.json`
- Workouts: `~/.config/c2/data/workouts.jsonl`
- Strokes: `~/.config/c2/data/strokes/<workout_id>.jsonl`

Existing JSONL data must remain readable. Go structs should use JSON tags matching
the Concept2 API and the current stored data. Unknown JSON fields may be ignored.
Optional fields should use pointers where field absence matters, especially
`rest_time`, `rest_distance`, stroke fields, and heart-rate details.

Config should remain readable in its current shape. The Go loader should apply
defaults during load, like the TypeScript implementation. A versioned config schema
is not required for the initial port unless implementation reveals a concrete
migration need. `setup` may rewrite config using Go JSON formatting.

## TUI

The TUI is first-class and launches from bare `c2`.

Primary screens:

- Dashboard: goal progress, total meters, session count, average pace, required
  weekly pace, and sync status.
- Workouts: searchable/filterable workout list with interval indicators and key
  metrics.
- Trends: weekly volume, pace, SPM, and heart-rate summaries.
- Detail: selected workout details, interval/rest fields, comments, and stroke-data
  availability.
- Actions: sync, generate HTML report, export data, and run setup/config flow.

TUI actions should call shared Go services directly:

- Sync uses the Concept2 API client and storage writer, shows progress and errors,
  and updates visible totals when complete.
- Report generation uses `internal/report`, writes the current HTML report format,
  and reports the output path.
- Export uses `internal/export`, supports CSV, JSON, and JSONL, and reports the
  output path.
- Setup/config should at least handle missing-token states gracefully and guide the
  user through setup. A full TUI form is allowed if it stays focused.

Long-running actions must not freeze the UI. Use Bubble Tea commands for sync,
report generation, and export work.

## Static HTML Report

`c2 report` remains the static HTML output method. The Go report should initially
preserve the current self-contained report style and metrics as closely as
practical. Large HTML and CSS generation should live in `internal/report`, not in
the command handler.

Report generation from the TUI and from `c2 report` must use the same code path.

## Build And Release

Add a Go `Makefile` with these targets:

- `build`: build `bin/c2` with version metadata.
- `install`: install `c2` with version metadata.
- `test`: run `go test ./...`.
- `test-coverage`: generate coverage output.
- `fmt`: run `go fmt ./...`.
- `lint`: run `golangci-lint` via `go run`, pinned like the reference repos.
- `vet`: run `go vet ./...`.
- `tidy`: run `go mod tidy`.
- `staticcheck`: run `staticcheck` via `go run`.
- `clean`: remove build and coverage artifacts.
- `check`: run formatting, lint, vet, staticcheck, and tests.

Add `.goreleaser.yaml` for darwin/linux amd64/arm64 archives and checksums.
Homebrew cask support can be added in the initial config if it is low-friction, but
it is not required for the port to be considered complete.

## Testing

Port the current TypeScript tests into Go package tests:

- Model helpers: date parsing, pace, interval detection, rest/work seconds,
  duration formatting.
- Sessions: grouping and session counts.
- Stats: weekly summaries and goal progress.
- Display: meters, percentages, arrows, interval tags, workout lines.
- Export: CSV escaping, date filtering, headers, interval columns, JSON/JSONL.
- Config: defaults, date parsing, load/default merge behavior.
- Storage: JSONL read/write, duplicate avoidance, missing-file behavior.

Add Go-specific tests:

- CLI root behavior: bare `c2` enters TUI mode, explicit commands route correctly.
- Version formatting and `debug.ReadBuildInfo` fallback behavior.
- TUI model update tests for state transitions, action completion, and error
  display.
- Report smoke tests that assert expected sections and escaped content.

Verification before completion:

- `make check`
- `make build`
- `bin/c2 --version`
- `bin/c2 --help`
- `bin/c2 report -o <temp-file>` with fixture data where practical
- `bin/c2 export` with fixture data where practical

## Implementation Sequence

1. Create an isolated worktree and baseline check.
2. Initialize Go module, Makefile, versioning, and GoReleaser config.
3. Port model, config, storage, and stats packages with tests.
4. Port API client and sync service.
5. Port scriptable CLI commands.
6. Port static HTML report generation.
7. Add Bubble Tea TUI with dashboard/workouts/trends/detail views.
8. Add TUI actions for sync, report generation, export, and setup/config handling.
9. Remove Bun/TypeScript packaging and update README/AGENTS/CLAUDE docs.
10. Run full verification and address regressions.

## Risks

- The TUI increases scope compared with a simple port. The implementation should
  build reliable shared services and CLI behavior before interactive polish.
- Current HTML report generation is large and string-heavy. Moving it to Go should
  preserve output semantics first, then improve template structure.
- Concept2 API behavior cannot be fully tested without live credentials. API tests
  should use fake HTTP servers.
- Storage compatibility matters because existing local history is the source of
  truth. Tests should include fixture JSONL matching current output.

