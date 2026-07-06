import type { Command } from "commander";
import { hrAtPace, splitShape, splitTable } from "../analysis.ts";
import { loadConfig } from "../config.ts";
import { formatMeters } from "../display.ts";
import { printJSON } from "../envelope.ts";
import { formatSeconds } from "../models.ts";
import { dataPaths } from "../paths.ts";
import type { GoalProgress } from "../stats.ts";
import {
  buildWeekSummaries,
  computeGoalProgress,
  localYMD,
  recentWeeks,
  weekSummaryData,
} from "../stats.ts";
import { readWorkouts } from "../storage.ts";
import { resolveWorkout } from "./show.ts";

export interface GoalProjection {
  projected_total_meters: number;
  projected_pct: number;
  shortfall_meters: number;
}

export function projectGoal(goal: GoalProgress): GoalProjection {
  const remaining = Math.max(0, goal.totalWeeks - goal.weeksElapsed);
  const projected = Math.round(goal.currentAvgPace * remaining + goal.totalMeters);
  return {
    projected_total_meters: projected,
    projected_pct: Math.round((projected / goal.target) * 1000) / 10,
    shortfall_meters: Math.max(0, goal.target - projected),
  };
}

function parseWeeks(raw: string): number {
  const weeks = parseInt(raw, 10);
  if (Number.isNaN(weeks) || weeks < 1) {
    console.error("Error: --weeks must be a positive integer.");
    process.exit(1);
  }
  return weeks;
}

export function registerStats(program: Command): void {
  const stats = program.command("stats").description("Derived training statistics");

  stats
    .command("weekly")
    .description("Weekly volume, sessions, pace, SPM, and HR")
    .option("-w, --weeks <n>", "number of weeks", "12")
    .option("--json", "output as JSON")
    .action(async (opts: { weeks: string; json?: boolean }) => {
      const weeks = parseWeeks(opts.weeks);
      const cfg = await loadConfig();
      const workouts = await readWorkouts(dataPaths(cfg));
      const summaries = buildWeekSummaries(workouts, new Date(), weeks).map(weekSummaryData);

      if (opts.json) {
        printJSON("c2.stats.weekly.v1", { weeks: summaries });
        return;
      }
      console.log("week        meters  sess  pace/500m   spm    hr");
      for (const s of summaries) {
        console.log(
          `${s.week_start}  ${formatMeters(s.meters).padStart(8)}  ${String(s.sessions).padStart(4)}  ${(s.avg_pace_500m ?? "-").padStart(9)}  ${String(s.avg_spm ?? "-").padStart(4)}  ${String(s.avg_hr ?? "-").padStart(4)}`,
        );
      }
    });

  stats
    .command("goal")
    .description("Goal trajectory and projection")
    .option("--json", "output as JSON")
    .action(async (opts: { json?: boolean }) => {
      const cfg = await loadConfig();
      if (!cfg.goal.start_date || !cfg.goal.end_date) {
        console.error("Goal dates not configured. Run `c2 setup` to set start and end dates.");
        process.exit(1);
      }
      const workouts = await readWorkouts(dataPaths(cfg));
      const now = new Date();
      const goal = computeGoalProgress(workouts, cfg, now);
      const projection = projectGoal(goal);
      const weeks = recentWeeks(workouts, now, 4);
      const thisWeek = weeks[0]!;

      if (opts.json) {
        printJSON("c2.stats.goal.v1", {
          goal,
          projection,
          this_week: {
            week_start: localYMD(thisWeek.weekStart),
            meters: thisWeek.meters,
            sessions: thisWeek.sessions,
          },
        });
        return;
      }
      console.log(
        `Progress: ${formatMeters(goal.totalMeters)} / ${formatMeters(goal.target)} (${((goal.progress ?? 0) * 100).toFixed(1)}%)`,
      );
      console.log(`Required pace: ${formatMeters(goal.requiredPace)} m/wk`);
      console.log(`Recent average: ${formatMeters(goal.currentAvgPace)} m/wk`);
      console.log(
        `Projection at current pace: ${formatMeters(projection.projected_total_meters)} m (${projection.projected_pct}%)`,
      );
      if (projection.shortfall_meters > 0) {
        console.log(`Projected shortfall: ${formatMeters(projection.shortfall_meters)} m`);
      } else {
        console.log("On track to exceed goal.");
      }
      console.log(
        `This week so far: ${formatMeters(thisWeek.meters)} m (${thisWeek.sessions} sessions)`,
      );
    });

  stats
    .command("splits <id>")
    .description("Split analysis for one workout (id or 'last')")
    .option("--json", "output as JSON")
    .action(async (ref: string, opts: { json?: boolean }) => {
      const cfg = await loadConfig();
      const workouts = await readWorkouts(dataPaths(cfg));
      const w = resolveWorkout(workouts, ref);
      if (w == null) {
        console.error(
          ref === "last" ? "No workouts found. Run `c2 sync` first." : `No workout with id ${ref}.`,
        );
        process.exit(1);
      }
      const rows = splitTable(w);
      const shape = splitShape(rows);

      if (opts.json) {
        printJSON("c2.stats.splits.v1", {
          workout_id: w.id,
          date: w.date,
          distance: w.distance,
          split_shape: shape,
          splits: rows,
        });
        return;
      }
      if (rows.length === 0) {
        console.log(`Workout ${w.id} has no split data.`);
        return;
      }
      console.log(`Workout ${w.id} — ${w.date} — ${formatMeters(w.distance)}m — ${shape} splits`);
      for (const s of rows) {
        console.log(
          `  ${s.index}: ${s.distance != null ? `${formatMeters(s.distance)}m` : "-"}  ${s.pace_500m ?? "-"}/500m  ${s.stroke_rate ?? "-"}spm  HR ${s.hr_avg ?? "-"}`,
        );
      }
    });

  stats
    .command("hr-pace")
    .description("Average heart rate by steady pace band (fitness proxy)")
    .option("-w, --weeks <n>", "window in weeks", "8")
    .option("--json", "output as JSON")
    .action(async (opts: { weeks: string; json?: boolean }) => {
      const weeks = parseWeeks(opts.weeks);
      const cfg = await loadConfig();
      const workouts = await readWorkouts(dataPaths(cfg));
      const bands = hrAtPace(workouts, new Date(), weeks);

      if (opts.json) {
        printJSON("c2.stats.hr-pace.v1", { weeks, bands });
        return;
      }
      if (bands.length === 0) {
        console.log("No steady workouts with heart rate data in the window.");
        return;
      }
      console.log(`Steady pace bands over the last ${weeks} weeks (HR avg, early→late half):`);
      for (const b of bands) {
        const trend =
          b.hr_delta == null
            ? ""
            : b.hr_delta < 0
              ? `  ↓${Math.abs(b.hr_delta)} (improving)`
              : b.hr_delta > 0
                ? `  ↑${b.hr_delta} (watch)`
                : "  → flat";
        const halves =
          b.early_avg_hr != null && b.late_avg_hr != null
            ? ` (${b.early_avg_hr}→${b.late_avg_hr})`
            : "";
        console.log(
          `  ${b.band}/500m: HR ${b.avg_hr}${halves} across ${b.workouts} workout${b.workouts === 1 ? "" : "s"}${trend}`,
        );
      }
    });
}
