import type { Command } from "commander";
import { loadConfig } from "../config.ts";
import { formatMeters, formatMetersPerWeek, formatPercent } from "../display.ts";
import { calendarDay } from "../models.ts";
import { computeGoalProgress, mondayOf, workoutsInRange } from "../stats.ts";
import { readWorkouts } from "../storage.ts";

export function registerStatus(program: Command): void {
  program
    .command("status")
    .description("Show progress toward your distance goal")
    .action(async () => {
      const cfg = await loadConfig();
      if (!cfg.goal.start_date || !cfg.goal.end_date) {
        console.error("Goal dates not configured. Run `c2 setup` to set start and end dates.");
        process.exit(1);
      }
      const workouts = await readWorkouts();
      if (workouts.length === 0) {
        console.log("No workouts found. Run `c2 sync` first.");
        return;
      }
      const goal = computeGoalProgress(workouts, cfg);

      console.log(`Goal: ${formatMeters(goal.target)}m`);
      console.log(`Season start: ${cfg.goal.start_date}`);
      console.log(
        `Progress: ${formatMeters(goal.totalMeters)} / ${formatMeters(goal.target)} (${formatPercent(goal.progress)})`,
      );
      console.log(`Weeks elapsed: ${goal.weeksElapsed} / ${goal.totalWeeks}`);
      console.log(`Required pace: ${formatMetersPerWeek(goal.requiredPace)}`);
      console.log();

      const today = new Date();
      console.log("Last 4 weeks:");
      for (let i = 0; i < 4; i++) {
        const weekStart = mondayOf(today);
        weekStart.setDate(weekStart.getDate() - i * 7);
        const weekEnd = new Date(weekStart);
        weekEnd.setDate(weekEnd.getDate() + 7);

        const weekWorkouts = workoutsInRange(workouts, weekStart, weekEnd);
        const meters = weekWorkouts.reduce((sum, w) => sum + w.distance, 0);
        const sessions = new Set(weekWorkouts.map(calendarDay)).size;

        const mm = String(weekStart.getMonth() + 1).padStart(2, "0");
        const dd = String(weekStart.getDate()).padStart(2, "0");
        console.log(`  Week of ${mm}/${dd}: ${formatMeters(meters)} (${sessions} sessions)`);
      }
      console.log();

      if (goal.weeksElapsed > 0) {
        const indicator = goal.onPace ? "on pace \u2713" : "behind pace \u2717";
        console.log(`Current avg: ${formatMetersPerWeek(goal.currentAvgPace)} \u2014 ${indicator}`);
      }
    });
}
