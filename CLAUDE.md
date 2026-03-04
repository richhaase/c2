# c2cli

Concept2 Logbook CLI — personal tool for rowing data sync and analysis.

## Build & Test

```bash
cargo build          # compile
cargo test           # run tests
cargo clippy         # lint
cargo fmt --check    # format check
cargo check          # type check (fast)
```

## Architecture

- **Language:** Rust
- **Storage:** JSONL files at `~/.config/c2cli/data/`
- **Auth:** OAuth2 with token refresh, tokens at `~/.config/c2cli/tokens.json`
- **Config:** TOML at `~/.config/c2cli/config.toml`

## Source Layout

```
src/
├── main.rs          # CLI entry point (clap)
├── cli.rs           # Command definitions
├── api.rs           # Concept2 API client
├── auth.rs          # OAuth2 flow and token management
├── config.rs        # Config file reading/writing
├── storage.rs       # JSONL read/write operations
├── models.rs        # Data types (Workout, StrokeData, etc.)
├── commands/        # Command implementations
│   ├── mod.rs
│   ├── auth.rs
│   ├── sync.rs
│   ├── log.rs
│   ├── status.rs
│   ├── trend.rs
│   └── export.rs
└── display.rs       # Formatting and terminal output
```

## Key Decisions

- JSONL for storage (append-friendly, portable, small enough to parse fully)
- OAuth2 with local redirect server for initial auth
- Custom goal dates independent of C2 season (May 1 – Apr 30)
- `time` field from API is in tenths of a second
