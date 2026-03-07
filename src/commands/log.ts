import type { Command } from "commander";
import { loadConfig } from "../config.ts";
import { readWorkouts } from "../storage.ts";
import { formatWorkoutLine } from "../display.ts";

export function registerLog(program: Command): void {
  program
    .command("log")
    .description("Show recent workouts")
    .option("-n, --count <n>", "number of workouts to display", "10")
    .action(async (opts: { count: string }) => {
      const cfg = await loadConfig();
      const workouts = await readWorkouts();
      if (workouts.length === 0) {
        console.log("No workouts found. Run `c2 sync` first.");
        return;
      }

      workouts.sort((a, b) => b.date.localeCompare(a.date));

      const count = parseInt(opts.count, 10);
      if (isNaN(count) || count < 1) {
        console.error("Error: --count must be a positive integer.");
        process.exit(1);
      }
      const n = Math.min(count, workouts.length);
      for (const w of workouts.slice(0, n)) {
        console.log(formatWorkoutLine(w, cfg.display.date_format));
      }
    });
}
