# c2 — Concept2 Logbook CLI

A CLI tool for syncing and analyzing rowing data from your [Concept2 Logbook](https://log.concept2.com).

## Quick Start

```bash
# Install
go install github.com/richhaase/c2cli/cmd/c2@latest

# Authenticate (get your token from log.concept2.com → Settings → Developer)
c2 auth YOUR_TOKEN

# Sync workouts
c2 sync

# Check your progress
c2 status
```

## Installation

### From Source

```bash
go install github.com/richhaase/c2cli/cmd/c2@latest
```

### Build from Checkout

```bash
git clone https://github.com/richhaase/c2cli.git
cd c2cli
make install
```

## Usage

### Authentication

Get a personal access token from [log.concept2.com](https://log.concept2.com) under Settings → Developer, then save it:

```bash
c2 auth YOUR_TOKEN
```

### Sync Workouts

Pull new workouts from the Concept2 API:

```bash
c2 sync
```

To backfill per-stroke data for workouts that are missing it:

```bash
c2 sync --backfill-strokes
```

### View Workouts

```bash
# Show last 10 workouts
c2 log

# Show last 25 workouts
c2 log -n 25
```

### Goal Progress

Track progress toward a million-meter annual goal:

```bash
c2 status
```

### Training Trends

View weekly trends for pace, volume, stroke rate, and heart rate:

```bash
# Last 8 weeks (default)
c2 trend

# Last 12 weeks
c2 trend -w 12
```

### Export Data

Export workouts to CSV, JSON, or JSONL:

```bash
# CSV to stdout
c2 export

# JSON format
c2 export -f json

# Filter by date range
c2 export --from 2026-01-01 --to 2026-03-01

# Pipe to file
c2 export -f jsonl > workouts.jsonl
```

## Configuration

Config lives at `~/.config/c2cli/config.toml`. Created automatically on `c2 auth`.

```toml
[api]
base_url = "https://log.concept2.com"
token = "YOUR_TOKEN"

[sync]
machine_type = "rower"

[goal]
target_meters = 1000000
start_date = "2026-01-01"
end_date = "2026-12-31"

[display]
date_format = "%m/%d"
```

## Development

```bash
# List available targets
make help

# Build with version info (outputs to bin/)
make build

# Install to GOPATH/bin
make install

# Run all quality checks (format, vet, lint, tests)
make check

# Run tests
make test

# Run linter
make lint

# Format code
make fmt

# Clean build artifacts
make clean
```

## License

MIT — see [LICENSE](LICENSE)
