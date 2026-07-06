import type { Command } from "commander";
import { loadConfig } from "../config.ts";
import { formatMeters, formatMetersPerWeek, formatPercent } from "../display.ts";
import { printJSON } from "../envelope.ts";
import { dataPaths } from "../paths.ts";
import { computeGoalProgress, localYMD, recentWeeks } from "../stats.ts";
import { readWorkouts } from "../storage.ts";

export function registerStatus(program: Command): void {
  program
    .command("status")
    .description("Show progress toward your distance goal")
    .option("--json", "output as JSON")
    .action(async (opts: { json?: boolean }) => {
      const cfg = await loadConfig();
      if (!cfg.goal.start_date || !cfg.goal.end_date) {
        console.error("Goal dates not configured. Run `c2 setup` to set start and end dates.");
        process.exit(1);
      }
      const paths = dataPaths(cfg);
      const workouts = await readWorkouts(paths);
      if (workouts.length === 0) {
        console.log("No workouts found. Run `c2 sync` first.");
        return;
      }
      const now = new Date();
      const goal = computeGoalProgress(workouts, cfg, now);
      const weeks = recentWeeks(workouts, now, 4);
      const thisWeek = weeks[0]!;

      if (opts.json) {
        printJSON("c2.status.v1", {
          goal,
          this_week: {
            week_start: localYMD(thisWeek.weekStart),
            meters: thisWeek.meters,
            sessions: thisWeek.sessions,
          },
          recent_weeks: weeks.map((w) => ({
            week_start: localYMD(w.weekStart),
            meters: w.meters,
            sessions: w.sessions,
          })),
        });
        return;
      }

      console.log(`Goal: ${formatMeters(goal.target)}m`);
      console.log(`Season start: ${cfg.goal.start_date}`);
      console.log(
        `Progress: ${formatMeters(goal.totalMeters)} / ${formatMeters(goal.target)} (${formatPercent(goal.progress)})`,
      );
      console.log(`Weeks elapsed: ${goal.weeksElapsed} / ${goal.totalWeeks}`);
      console.log(`Required pace: ${formatMetersPerWeek(goal.requiredPace)}`);
      console.log(
        `This week so far: ${formatMeters(thisWeek.meters)} (${thisWeek.sessions} sessions)`,
      );
      console.log();

      console.log("Last 4 weeks:");
      for (const w of weeks) {
        const mm = String(w.weekStart.getMonth() + 1).padStart(2, "0");
        const dd = String(w.weekStart.getDate()).padStart(2, "0");
        console.log(`  Week of ${mm}/${dd}: ${formatMeters(w.meters)} (${w.sessions} sessions)`);
      }
      console.log();

      if (goal.weeksElapsed > 0) {
        const indicator = goal.onPace ? "on pace ✓" : "behind pace ✗";
        console.log(`Current avg: ${formatMetersPerWeek(goal.currentAvgPace)} — ${indicator}`);
      }
    });
}
