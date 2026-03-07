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

      const n = Math.min(parseInt(opts.count, 10), workouts.length);
      for (const w of workouts.slice(0, n)) {
        console.log(formatWorkoutLine(w, cfg.display.date_format));
      }
    });
}
