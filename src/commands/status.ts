import type { Command } from "commander";
import { loadConfig, parseGoalDate } from "../config.ts";
import { readWorkouts } from "../storage.ts";
import {
  formatMeters,
  formatPercent,
  formatMetersPerWeek,
} from "../display.ts";
import { calendarDay } from "../models.ts";
import type { Workout } from "../models.ts";

function mondayOf(t: Date): Date {
  const d = new Date(t.getFullYear(), t.getMonth(), t.getDate());
  const offset = (d.getDay() + 6) % 7;
  d.setDate(d.getDate() - offset);
  return d;
}

function workoutsInRange(
  workouts: Workout[],
  from: Date,
  to: Date,
): Workout[] {
  return workouts.filter((w) => {
    const t = new Date(w.date.replace(" ", "T"));
    return t >= from && t < to;
  });
}

export function registerStatus(program: Command): void {
  program
    .command("status")
    .description("Show progress toward million-meter goal")
    .action(async () => {
      const cfg = await loadConfig();
      const workouts = await readWorkouts();

      const target = cfg.goal.target_meters;
      const start = parseGoalDate(cfg.goal.start_date);
      const end = parseGoalDate(cfg.goal.end_date);
      const today = new Date();

      let totalMeters = 0;
      for (const w of workouts) {
        const t = new Date(w.date.replace(" ", "T"));
        if (t >= start && t <= end) {
          totalMeters += w.distance;
        }
      }

      const progress = totalMeters / target;
      const totalDays = (end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24);
      const totalWeeks = Math.ceil(totalDays / 7);

      let weeksElapsed = 0;
      if (today > start) {
        weeksElapsed = Math.floor(
          (today.getTime() - start.getTime()) / (1000 * 60 * 60 * 24 * 7),
        );
      }

      let remainingMeters = target - totalMeters;
      if (remainingMeters < 0) remainingMeters = 0;
      let remainingWeeks = totalWeeks - weeksElapsed;
      if (remainingWeeks < 1) remainingWeeks = 1;
      const requiredPace = Math.floor(remainingMeters / remainingWeeks);

      console.log(`Goal: ${formatMeters(target)}m`);
      console.log(`Season start: ${start.toISOString().slice(0, 10)}`);
      console.log(
        `Progress: ${formatMeters(totalMeters)} / ${formatMeters(target)} (${formatPercent(progress)})`,
      );
      console.log(`Weeks elapsed: ${weeksElapsed} / ${totalWeeks}`);
      console.log(`Required pace: ${formatMetersPerWeek(requiredPace)}`);
      console.log();

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
        console.log(
          `  Week of ${mm}/${dd}: ${formatMeters(meters)} (${sessions} sessions)`,
        );
      }
      console.log();

      if (weeksElapsed > 0) {
        const avg = Math.floor(totalMeters / weeksElapsed);
        const targetWeekly = target / totalWeeks;
        const onPace = avg >= targetWeekly;
        const indicator = onPace ? "on pace \u2713" : "behind pace \u2717";
        console.log(
          `Current avg: ${formatMetersPerWeek(avg)} \u2014 ${indicator}`,
        );
      }
    });
}
