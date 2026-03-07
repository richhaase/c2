import type { Command } from "commander";
import { readWorkouts } from "../storage.ts";
import {
  formatMeters,
  sparkBar,
  trendArrow,
  paceArrow,
} from "../display.ts";
import { buildWeekSummaries, type WeekSummary } from "../stats.ts";

function fmtDate(d: Date): string {
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${mm}/${dd}`;
}

function maxMeters(summaries: WeekSummary[]): number {
  let max = 0;
  for (const ws of summaries) {
    if (ws.meters > max) max = ws.meters;
  }
  return max;
}

function printVolumeTrend(summaries: WeekSummary[]): void {
  console.log("Volume (meters/week):");
  let prevMeters = 0;
  for (const ws of summaries) {
    const arrow = trendArrow(prevMeters, ws.meters);
    const bar = sparkBar(ws.meters, maxMeters(summaries));
    console.log(
      `  ${fmtDate(ws.weekStart)}  ${arrow} ${formatMeters(ws.meters).padStart(7)}  ${bar}  (${ws.sessions} sessions)`,
    );
    prevMeters = ws.meters;
  }
}

function printPaceTrend(summaries: WeekSummary[]): void {
  console.log("Avg Pace (/500m):");
  let prevPace = 0;
  for (const ws of summaries) {
    if (ws.paceCount === 0) {
      console.log(`  ${fmtDate(ws.weekStart)}    -`);
      continue;
    }
    const avgPace = ws.paceSum / ws.paceCount;
    const arrow = paceArrow(prevPace, avgPace);
    const mins = Math.floor(avgPace / 60);
    const secs = avgPace - mins * 60;
    console.log(
      `  ${fmtDate(ws.weekStart)}  ${arrow} ${mins}:${secs.toFixed(1).padStart(4, "0")}`,
    );
    prevPace = avgPace;
  }
}

function printSPMTrend(summaries: WeekSummary[]): void {
  console.log("Avg Stroke Rate (spm):");
  let prevSPM = 0;
  for (const ws of summaries) {
    if (ws.spmCount === 0) {
      console.log(`  ${fmtDate(ws.weekStart)}    -`);
      continue;
    }
    const avg = ws.spmSum / ws.spmCount;
    const arrow = trendArrow(prevSPM, avg);
    console.log(
      `  ${fmtDate(ws.weekStart)}  ${arrow} ${avg.toFixed(1).padStart(4)}`,
    );
    prevSPM = avg;
  }
}

function printHRTrend(summaries: WeekSummary[]): void {
  console.log("Avg Heart Rate (bpm):");
  let hasAny = false;
  let prevHR = 0;
  for (const ws of summaries) {
    if (ws.hrCount === 0) {
      console.log(`  ${fmtDate(ws.weekStart)}    -`);
      continue;
    }
    hasAny = true;
    const avg = ws.hrSum / ws.hrCount;
    const arrow = trendArrow(prevHR, avg);
    console.log(
      `  ${fmtDate(ws.weekStart)}  ${arrow} ${avg.toFixed(1).padStart(5)}`,
    );
    prevHR = avg;
  }
  if (!hasAny) {
    console.log("  No heart rate data available.");
  }
}

export function registerTrend(program: Command): void {
  program
    .command("trend")
    .description("Show training trends over time")
    .option("-w, --weeks <n>", "number of weeks to display", "8")
    .action(async (opts: { weeks: string }) => {
      const workouts = await readWorkouts();
      if (workouts.length === 0) {
        console.log("No workouts found. Run `c2 sync` first.");
        return;
      }

      const weeks = parseInt(opts.weeks, 10);
      if (isNaN(weeks) || weeks < 1) {
        console.error("Error: --weeks must be a positive integer.");
        process.exit(1);
      }
      const summaries = buildWeekSummaries(workouts, new Date(), weeks);

      printVolumeTrend(summaries);
      console.log();
      printPaceTrend(summaries);
      console.log();
      printSPMTrend(summaries);
      console.log();
      printHRTrend(summaries);
    });
}
