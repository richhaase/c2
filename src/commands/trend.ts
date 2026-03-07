import type { Command } from "commander";
import { readWorkouts } from "../storage.ts";
import type { Workout } from "../models.ts";
import { pace500mSeconds, calendarDay } from "../models.ts";
import {
  formatMeters,
  sparkBar,
  trendArrow,
  paceArrow,
} from "../display.ts";

interface WeekSummary {
  weekStart: Date;
  meters: number;
  sessions: number;
  paceSum: number;
  paceCount: number;
  spmSum: number;
  spmCount: number;
  hrSum: number;
  hrCount: number;
}

function mondayOf(t: Date): Date {
  const d = new Date(t.getFullYear(), t.getMonth(), t.getDate());
  const offset = (d.getDay() + 6) % 7;
  d.setDate(d.getDate() - offset);
  return d;
}

function buildWeekSummaries(
  workouts: Workout[],
  now: Date,
  weeks: number,
): WeekSummary[] {
  const thisMonday = mondayOf(now);
  const cutoff = new Date(thisMonday);
  cutoff.setDate(cutoff.getDate() - (weeks - 1) * 7);

  const summaries: WeekSummary[] = [];
  for (let i = 0; i < weeks; i++) {
    const ws = new Date(thisMonday);
    ws.setDate(ws.getDate() - (weeks - 1 - i) * 7);
    summaries.push({
      weekStart: ws,
      meters: 0,
      sessions: 0,
      paceSum: 0,
      paceCount: 0,
      spmSum: 0,
      spmCount: 0,
      hrSum: 0,
      hrCount: 0,
    });
  }

  // Track unique days per week for session counting
  const daysByWeek = new Map<number, Set<string>>();

  for (const w of workouts) {
    const t = new Date(w.date.replace(" ", "T"));
    if (t < cutoff || t > now) continue;

    const monday = mondayOf(t);
    const idx = Math.floor(
      (monday.getTime() - cutoff.getTime()) / (1000 * 60 * 60 * 24 * 7),
    );
    if (idx < 0 || idx >= weeks) continue;

    const ws = summaries[idx]!;
    ws.meters += w.distance;

    if (!daysByWeek.has(idx)) daysByWeek.set(idx, new Set());
    daysByWeek.get(idx)!.add(calendarDay(w));

    const pace = pace500mSeconds(w);
    if (pace > 0) {
      ws.paceSum += pace;
      ws.paceCount++;
    }
    if (w.stroke_rate && w.stroke_rate > 0) {
      ws.spmSum += w.stroke_rate;
      ws.spmCount++;
    }
    if (w.heart_rate?.average && w.heart_rate.average > 0) {
      ws.hrSum += w.heart_rate.average;
      ws.hrCount++;
    }
  }

  // Set session counts from unique days
  for (const [idx, days] of daysByWeek) {
    summaries[idx]!.sessions = days.size;
  }

  return summaries;
}

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
