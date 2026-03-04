# c2cli

Concept2 Logbook CLI — personal tool for rowing data sync and analysis.

## Build & Test

```bash
make help            # list available targets
make build           # build to bin/ with version ldflags
make install         # install to GOPATH/bin
make check           # fmt, vet, lint, test
make test            # run tests only
make clean           # remove bin/ and test cache
```

## Architecture

- **Language:** Go
- **Storage:** JSONL files at `~/.config/c2cli/data/`
- **Auth:** Static personal access token from log.concept2.com
- **Config:** TOML at `~/.config/c2cli/config.toml`

## Source Layout

```
cmd/c2/
└── main.go              # CLI entry point (binary: c2)
internal/
├── api/                 # Concept2 API client
│   └── client.go
├── commands/            # Cobra command implementations
│   ├── root.go
│   ├── auth.go
│   ├── sync.go
│   ├── log.go
│   ├── status.go
│   ├── trend.go
│   └── export.go
├── config/              # Config + token management
│   └── config.go
├── display/             # Formatting helpers
│   └── display.go
├── models/              # Data types
│   └── models.go
└── storage/             # JSONL read/write
    └── storage.go
```

## Key Decisions

- JSONL for storage (append-friendly, portable, small enough to parse fully)
- Static personal access token (no OAuth2 flow needed — C2 provides one at log.concept2.com)
- Custom goal dates independent of C2 season (May 1 – Apr 30)
- `time` field from API is in tenths of a second
- Commands use `init()` registration with `rootCmd.AddCommand()` (plonk pattern)
- `cmd/<binary>/main.go` entry point with `debug.ReadBuildInfo()` fallback for `go install`
