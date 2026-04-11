import type { Command } from "commander";
import type { Workout } from "../models.ts";
import { pace500m } from "../models.ts";
import { readWorkouts } from "../storage.ts";

export function filterByDate(workouts: Workout[], from: string, to: string): Workout[] {
  if (!from && !to) return workouts;
  return workouts.filter((w) => {
    const date = w.date.slice(0, 10);
    if (from && date < from) return false;
    if (to && date > to) return false;
    return true;
  });
}

export function escapeCSV(s: string): string {
  if (s.includes(",") || s.includes('"') || s.includes("\n")) {
    return `"${s.replace(/"/g, '""')}"`;
  }
  return s;
}

export const CSV_HEADER = [
  "id",
  "date",
  "distance",
  "time_tenths",
  "time_formatted",
  "pace_500m",
  "stroke_rate",
  "stroke_count",
  "calories",
  "drag_factor",
  "hr_avg",
  "hr_min",
  "hr_max",
  "workout_type",
  "rest_time_tenths",
  "rest_distance",
  "machine_type",
  "comments",
] as const;

export function buildCSVRow(w: Workout): string[] {
  return [
    String(w.id),
    w.date,
    String(w.distance),
    String(w.time),
    w.time_formatted,
    pace500m(w),
    String(w.stroke_rate ?? ""),
    String(w.stroke_count ?? ""),
    String(w.calories_total ?? ""),
    String(w.drag_factor ?? ""),
    w.heart_rate?.average ? String(w.heart_rate.average) : "",
    w.heart_rate?.min ? String(w.heart_rate.min) : "",
    w.heart_rate?.max ? String(w.heart_rate.max) : "",
    w.workout_type ?? "",
    w.rest_time != null ? String(w.rest_time) : "",
    w.rest_distance != null ? String(w.rest_distance) : "",
    w.type ?? "",
    escapeCSV(w.comments ?? ""),
  ];
}

function exportCSV(workouts: Workout[]): void {
  console.log(CSV_HEADER.join(","));
  for (const w of workouts) {
    console.log(buildCSVRow(w).join(","));
  }
}

function exportJSON(workouts: Workout[]): void {
  console.log(JSON.stringify(workouts, null, 2));
}

function exportJSONL(workouts: Workout[]): void {
  for (const w of workouts) {
    console.log(JSON.stringify(w));
  }
}

export function registerExport(program: Command): void {
  program
    .command("export")
    .description("Export workouts to CSV or JSON")
    .option("-f, --format <fmt>", "output format: csv, json, or jsonl", "csv")
    .option("--from <date>", "filter workouts from date (YYYY-MM-DD)")
    .option("--to <date>", "filter workouts to date (YYYY-MM-DD)")
    .action(async (opts: { format: string; from?: string; to?: string }) => {
      let workouts = await readWorkouts();
      if (workouts.length === 0) {
        console.error("No workouts found. Run `c2 sync` first.");
        process.exit(1);
      }

      workouts = filterByDate(workouts, opts.from ?? "", opts.to ?? "");
      if (workouts.length === 0) {
        console.error("No workouts match the specified date range.");
        process.exit(1);
      }

      // Sort oldest first for export
      workouts.sort((a, b) => a.date.localeCompare(b.date));

      switch (opts.format) {
        case "csv":
          exportCSV(workouts);
          break;
        case "json":
          exportJSON(workouts);
          break;
        case "jsonl":
          exportJSONL(workouts);
          break;
        default:
          console.error(`Unsupported format "${opts.format}": must be csv, json, or jsonl`);
          process.exit(1);
      }
    });
}
