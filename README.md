# c2 — Concept2 Logbook CLI

A CLI tool for syncing and analyzing rowing data from your [Concept2 Logbook](https://log.concept2.com). Built with [Bun](https://bun.sh).

## Install

Requires [Bun](https://bun.sh) v1.0+.

```bash
# Install globally
bun install -g @richhaase/c2

# Or install from source
git clone https://github.com/richhaase/c2.git
cd c2
bun install
bun link
```

## Quick Start

```bash
# Configure token and goals
bun src/index.ts setup

# Sync workouts
bun src/index.ts sync

# Check your progress
bun src/index.ts status
```

## Usage

### Setup

Configure your token and goal settings:

```bash
c2 setup
```

Get your personal access token from [log.concept2.com](https://log.concept2.com) under Settings → Developer. The setup wizard will prompt for your token, goal target, and date range.

### Sync Workouts

Pull new workouts from the Concept2 API:

```bash
c2 sync
```

### View Workouts

```bash
# Show last 10 workouts
c2 log

# Show last 25 workouts
c2 log -n 25
```

Interval workouts are tagged with `[IVL rest M:SS.S]` so they're visually
distinct from continuous pieces. The rest duration is surfaced because, for
interval workouts, the displayed time (`time_formatted`) is elapsed time
including rest, while the displayed pace is correctly computed from work
time only. For example:

```
04/11   5,000m   28:35.4   2:51.5/500m  24spm  112bpm  107df
04/11   3,000m   20:22.6   2:23.8/500m  30spm  152bpm  108df  [IVL rest 6:00.0]
```

The second row is 6x500m with ~1 min rest between reps: 20:22.6 elapsed =
14:22.6 work + 6:00 rest. The `2:23.8/500m` pace is the work pace.

### Goal Progress

Track progress toward your distance goal:

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

### HTML Report

Generate a self-contained HTML progress report:

```bash
# Generate report.html in current directory
c2 report

# Custom output path
c2 report -o ~/Desktop/rowing.html

# Generate and open in browser
c2 report --open

# Show more weeks of history
c2 report -w 16
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

The CSV export includes `workout_type`, `rest_time_tenths`, and
`rest_distance` columns so interval workouts are fully distinguishable from
continuous pieces without having to consult the full JSON export.

## Configuration

Config lives at `~/.config/c2/config.json`. Created automatically on `c2 setup`.

```json
{
  "api": {
    "base_url": "https://log.concept2.com",
    "token": "YOUR_TOKEN"
  },
  "sync": {
    "machine_type": "rower"
  },
  "goal": {
    "target_meters": 1000000,
    "start_date": "2026-01-01",
    "end_date": "2026-12-31"
  },
  "display": {
    "date_format": "%m/%d"
  }
}
```

## Development

```bash
# Install dependencies
bun install

# Type check
bun run check

# Run tests
bun test

# Run directly
bun src/index.ts <command>
```

## License

MIT — see [LICENSE](LICENSE)
