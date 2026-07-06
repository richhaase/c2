import type { Command } from "commander";
import { loadConfig } from "../config.ts";
import { formatWorkoutLine, workoutJSON } from "../display.ts";
import { printJSON } from "../envelope.ts";
import { isValidYMD } from "../models.ts";
import { dataPaths } from "../paths.ts";
import { readWorkouts } from "../storage.ts";
import { filterByDate } from "./export.ts";

export function registerLog(program: Command): void {
  program
    .command("log")
    .description("Show recent workouts")
    .option("-n, --count <n>", "number of workouts to display", "10")
    .option("--from <date>", "only workouts on or after date (YYYY-MM-DD)")
    .option("--to <date>", "only workouts on or before date (YYYY-MM-DD)")
    .option("--json", "output as JSON")
    .action(async (opts: { count: string; from?: string; to?: string; json?: boolean }) => {
      const cfg = await loadConfig();
      const paths = dataPaths(cfg);

      for (const [flag, value] of [
        ["--from", opts.from],
        ["--to", opts.to],
      ] as const) {
        if (value && !isValidYMD(value)) {
          console.error(`Error: invalid ${flag} date "${value}" (expected YYYY-MM-DD).`);
          process.exit(1);
        }
      }

      const count = parseInt(opts.count, 10);
      if (Number.isNaN(count) || count < 1) {
        console.error("Error: --count must be a positive integer.");
        process.exit(1);
      }

      const all = await readWorkouts(paths);
      const workouts = filterByDate(all, opts.from ?? "", opts.to ?? "");
      workouts.sort((a, b) => b.date.localeCompare(a.date));
      const shown = workouts.slice(0, Math.min(count, workouts.length));

      if (opts.json) {
        printJSON("c2.log.v1", { count: shown.length, workouts: shown.map(workoutJSON) });
        return;
      }

      if (all.length === 0) {
        console.log("No workouts found. Run `c2 sync` first.");
        return;
      }
      if (shown.length === 0) {
        console.log("No workouts match the specified date range.");
        return;
      }

      for (const w of shown) {
        const line = formatWorkoutLine(w, cfg.display.date_format);
        console.log(w.comments ? `${line}  — ${w.comments}` : line);
      }
    });
}
