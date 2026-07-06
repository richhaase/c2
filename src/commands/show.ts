import type { Command } from "commander";
import { splitShape, splitTable, strokeSummary } from "../analysis.ts";
import { loadConfig } from "../config.ts";
import { formatMeters, formatWorkoutLine, workoutJSON } from "../display.ts";
import { printJSON } from "../envelope.ts";
import type { Workout } from "../models.ts";
import { formatSeconds } from "../models.ts";
import { dataPaths } from "../paths.ts";
import { readStrokeData, readWorkouts } from "../storage.ts";

export function resolveWorkout(workouts: Workout[], ref: string): Workout | null {
  if (ref === "last") {
    if (workouts.length === 0) return null;
    return workouts.reduce((latest, w) => (w.date > latest.date ? w : latest));
  }
  const id = Number(ref);
  if (!Number.isInteger(id)) return null;
  return workouts.find((w) => w.id === id) ?? null;
}

export function registerShow(program: Command): void {
  program
    .command("show <id>")
    .description("Show full detail for one workout (use a workout id or 'last')")
    .option("--json", "output as JSON")
    .action(async (ref: string, opts: { json?: boolean }) => {
      const cfg = await loadConfig();
      const paths = dataPaths(cfg);
      const workouts = await readWorkouts(paths);
      const w = resolveWorkout(workouts, ref);
      if (w == null) {
        console.error(
          ref === "last"
            ? "No workouts found. Run `c2 sync` first."
            : `No workout with id ${ref}. Use \`c2 log --json\` to list ids.`,
        );
        process.exit(1);
      }

      const splits = splitTable(w);
      const shape = splitShape(splits);
      const strokes = await readStrokeData(paths, w.id);
      const strokesSummary = strokes.length > 0 ? strokeSummary(strokes) : null;

      if (opts.json) {
        printJSON("c2.show.v1", {
          workout: workoutJSON(w),
          raw: w,
          target_pace_500m_seconds: w.workout?.targets?.pace ? w.workout.targets.pace / 10 : null,
          splits,
          split_shape: shape,
          stroke_summary: strokesSummary,
        });
        return;
      }

      console.log(formatWorkoutLine(w, cfg.display.date_format));
      console.log();
      console.log(`Id: ${w.id}`);
      console.log(`Date: ${w.date}${w.timezone ? ` (${w.timezone})` : ""}`);
      if (w.workout_type) console.log(`Type: ${w.workout_type}`);
      if (w.source) console.log(`Source: ${w.source}`);
      if (w.stroke_count) console.log(`Strokes: ${formatMeters(w.stroke_count)}`);
      if (w.calories_total) console.log(`Calories: ${w.calories_total}`);
      if (w.heart_rate?.average) {
        const hr = w.heart_rate;
        const parts = [
          hr.min != null ? `min ${hr.min}` : null,
          `avg ${hr.average}`,
          hr.max != null ? `max ${hr.max}` : null,
          hr.ending != null ? `ending ${hr.ending}` : null,
        ].filter(Boolean);
        console.log(`Heart rate: ${parts.join(", ")}`);
      }
      if (w.rest_time != null && w.rest_time > 0) {
        console.log(`Interval rest: ${formatSeconds(w.rest_time / 10)}`);
      }
      if (w.rest_distance != null && w.rest_distance > 0) {
        console.log(`Interval rest distance: ${formatMeters(w.rest_distance)}m`);
      }
      if (w.workout?.targets?.pace) {
        console.log(`Target pace: ${formatSeconds(w.workout.targets.pace / 10)}/500m`);
      }
      if (w.comments) console.log(`Comments: ${w.comments}`);

      if (splits.length > 0) {
        console.log();
        console.log(`Splits (${shape}):`);
        console.log("    #      dist      time     pace/500m   spm    hr");
        for (const s of splits) {
          const dist = s.distance != null ? `${formatMeters(s.distance)}m` : "-";
          const pace = s.pace_500m ?? "-";
          const spm = s.stroke_rate ?? "-";
          const hr =
            s.hr_avg != null ? `${s.hr_avg}${s.hr_max != null ? `/${s.hr_max}` : ""}` : "-";
          console.log(
            `  ${String(s.index).padStart(3)}  ${dist.padStart(8)}  ${formatSeconds(s.time_seconds).padStart(8)}  ${pace.padStart(10)}  ${String(spm).padStart(4)}  ${hr.padStart(7)}`,
          );
        }
      }

      if (strokesSummary != null) {
        console.log();
        console.log(
          `Stroke data: ${strokesSummary.samples} samples, avg ${strokesSummary.avg_pace_500m ?? "-"}/500m, ${strokesSummary.avg_spm ?? "-"}spm, HR avg ${strokesSummary.avg_hr ?? "-"} max ${strokesSummary.max_hr ?? "-"}`,
        );
      }
    });
}
