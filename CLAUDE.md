# c2cli

Concept2 Logbook CLI — personal tool for rowing data sync and analysis.

## Build & Test

```bash
go build ./...       # compile
go test ./...        # run tests
go vet ./...         # lint
```

## Architecture

- **Language:** Go
- **Storage:** JSONL files at `~/.config/c2cli/data/`
- **Auth:** Static personal access token from log.concept2.com
- **Config:** TOML at `~/.config/c2cli/config.toml`

## Source Layout

```
main.go              # CLI entry point
internal/
├── api/             # Concept2 API client
│   └── client.go
├── config/          # Config + token management
│   └── config.go
├── display/         # Formatting helpers
│   └── display.go
├── models/          # Data types
│   └── models.go
├── storage/         # JSONL read/write
│   └── storage.go
└── cmd/             # Command implementations
    ├── auth.go
    ├── sync.go
    ├── log.go
    └── status.go
```

## Key Decisions

- JSONL for storage (append-friendly, portable, small enough to parse fully)
- Static personal access token (no OAuth2 flow needed — C2 provides one at log.concept2.com)
- Custom goal dates independent of C2 season (May 1 – Apr 30)
- `time` field from API is in tenths of a second
