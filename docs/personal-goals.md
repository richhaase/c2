# Personal goals and portable coaching

C2 keeps goals, workouts, and coaching in your chosen data folder. Commands
and the HTML report evaluate the same goal evidence. The first version uses
whole-workout average pace; it does not treat a fast split as the average
pace of the entire workout.

## Prepare an existing store

Run this on the machine whose current goal and calendar settings you want
to preserve:

```sh
c2 data prepare --timezone America/Denver
```

This writes `goals.json` with the analysis timezone, account identity when
known, and the existing dated distance goal. It preserves that goal's date
range and all-equipment eligibility. Preparation is idempotent. Ordinary
reads do not migrate the store. An explicit empty goal list stays empty.

The timezone determines calendar windows and today; recorded workout dates
retain their original local calendar meaning. Preparation does not convert
historical dates. Choose the timezone previously used for analysis to retain
calendar behavior. A prepared store's timezone cannot be changed by running
prepare again.

For a new store, the first goal can supply `--timezone` instead. Credentials
and the local data-folder path stay in the machine's configuration.

## Goals

```sh
c2 goal add "Annual distance" --kind volume --target 1000000 --from 2026-01-01 --to 2026-12-31
c2 goal add "Reach ten kilometers" --kind distance --target 10000
c2 goal add "Comfortable pace" --kind pace --target 2:30
c2 goal list
c2 goal show <id> --json
c2 goal update <id> --min-distance 5000
c2 goal archive <id>
c2 goal list --all
c2 goal update <id> --archived=false
```

Names are independent of targets. `volume` sums qualifying meters;
`distance` finds the longest qualifying workout; `pace` finds the fastest
qualifying whole-workout average in seconds per 500m. New goals default to
`rower`; use `--equipment all` or another Concept2 equipment type explicitly.
The current API sync requests rower results; selecting a goal's equipment
does not expand what sync downloads.

Start and end dates are independently optional and inclusive. Without a
start, existing recorded history counts; without an end, there is no
deadline. Future calendar dates do not count as completed work. Clear a
bound with `--from ''` or `--to ''`. Revising a goal recalculates its progress
from the new definition; archive it and create a new goal if you want to
retain the old target as a separate goal.

Distance goals default to `--effort continuous`. Known interval workouts and
workouts with insufficient type information cannot establish continuity.
Use `--effort workout` explicitly when a recorded workout's work total is
the intended accomplishment. Pace and volume default to workout evidence;
pace can also require continuity or `--min-distance`. Interval averages
exclude rest and are labeled accordingly. Minimum distance and pace always
refer to the same workout.

Progress includes the evidence workout and achievement date where
available. Cumulative goals have a numerical zero with no workouts;
performance goals instead report no qualifying evidence. Volume forecasts
are shown only for an active window with both dates. Pace and single-effort
distance never receive a volume forecast.

Dated volume details and forecast eligibility use the same inclusive calendar
dates as goal evidence. A skipped or repeated midnight cannot shorten the goal
window or start its forecast on the previous calendar date.

Required weekly volume and projections use the same fractional calendar time
remaining through the end of the inclusive deadline, including the rest of
today. Required volume rounds up to whole meters per week; projected volume
rounds down to whole meters. The on-pace flag tests whether the recent completed
weeks' average can cover the meters still needed in that time. It accounts for
progress already accumulated rather than comparing with a full-period average.
Completed goals remain on pace. After the deadline, an unmet goal is off pace
and has no future required pace; its numeric required pace and remaining weeks
are zero, and the legacy text/HTML labels the goal window ended.

The existing integer `remainingWeeks` field is the ceiling of the actual
remaining weeks, retained for compatibility. It is not the divisor for required
pace. The projection's `remaining_weeks` display rounds the actual horizon to
one decimal; calculations use the unrounded value. Whole-week elapsed/total
fields remain coarse counters. The HTML timeline uses elapsed calendar time.

## Reports and compatibility

`c2 report` writes a self-contained HTML overview; it works without goals
or with only undated goals. It separates the inclusive recent-activity
window from individual goal windows, counts workouts and training days
separately, and distinguishes generation time, last successful sync, and
latest recorded workout. Recent workout details are separately labeled as
the latest across recorded history.

Activity totals and weekly rows use the recorded calendar dates, including
the full final date even when the analysis timezone skips or repeats midnight.
The analysis timezone determines today's date and the report's freshness time.

The new default schemas are `c2.report.v2` and `c2.status.v2`.
`c2 report --json` and the existing `--data` are equivalent.
`c2 stats goal` now provides the same multi-goal view as `c2 goal list`,
using `c2.goal.v1`.

For scripts that require the original single-goal schemas:

```sh
c2 report --legacy --data
c2 status --legacy --json
c2 stats goal --legacy --json
```

These retain the original report/status/stats-goal v1 payloads. On a
prepared store, they read its migrated `legacy` goal rather than a stale
machine goal. If that goal is retired or changed to an incompatible kind
or scope, legacy mode fails explicitly. Other command schemas are unchanged.

## Transfer

1. Prepare the source store and run `c2 data doctor`.
2. Stop writes, then copy the entire data folder or let your storage provider
   finish synchronizing it. Include `goals.json`, metadata, workouts, strokes,
   loose and archived notes, plan, playbook, and dated narratives.
3. On the destination run `c2 data use /path/to/copied-store`.

Adoption validates the store before changing the local path and requires
no API token. It does not import the destination machine's old goal into
the selected store. Configure authentication separately if you want to
sync there. Sync checks account ownership before updating records or its
incremental cursor. Missing or conflicting workout account IDs require
reconciliation rather than guessing ownership.

`c2 data move` remains verified local relocation and retains the source.
Prepare first when moving for cross-machine use; an unprepared legacy
store still relies on its machine's goal configuration. The workout-only
`export` command is not a complete coaching backup. Doctor checks integrity,
not whether an external copy includes every file from its source.

Sync can re-download damaged stroke-cache files for workouts that advertise
stroke data. It still validates goals, metadata, workouts, and coaching records
before writing. Doctor and transfer checks continue to report damaged strokes;
sync the source store to repair those files before transferring it.

If `goals.json` is damaged, `c2 data doctor` still scans the store and reports
the goals error alongside other problems. `c2 setup` can save updated credentials
or select another valid store without rewriting the damaged goals file.

Use one active writer, and complete provider synchronization before switching
machines. Cloud folders transport files; C2 does not reconcile simultaneous
offline edits. Shared yearly note archives and whole-document writes can
conflict. Adoption and relocation reject detected corruption or divergent
note copies rather than silently resolving them. A dedicated backup/restore
system and distributed synchronization are outside this change.
