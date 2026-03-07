import type { Workout } from "./models.ts";
import type { Config } from "./config.ts";
import { parseGoalDate } from "./config.ts";
import { pace500mSeconds, calendarDay } from "./models.ts";

export interface WeekSummary {
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

export interface GoalProgress {
  target: number;
  totalMeters: number;
  progress: number; // ratio 0–1
  weeksElapsed: number;
  totalWeeks: number;
  remainingMeters: number;
  remainingWeeks: number;
  requiredPace: number; // meters/week needed going forward
  currentAvgPace: number; // meters/week so far
  onPace: boolean;
}

export function mondayOf(t: Date): Date {
  const d = new Date(t.getFullYear(), t.getMonth(), t.getDate());
  const offset = (d.getDay() + 6) % 7;
  d.setDate(d.getDate() - offset);
  return d;
}

export function workoutsInRange(
  workouts: Workout[],
  from: Date,
  to: Date,
): Workout[] {
  return workouts.filter((w) => {
    const t = new Date(w.date.replace(" ", "T"));
    return t >= from && t < to;
  });
}

export function buildWeekSummaries(
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

  for (const [idx, days] of daysByWeek) {
    summaries[idx]!.sessions = days.size;
  }

  return summaries;
}

export function computeGoalProgress(
  workouts: Workout[],
  cfg: Config,
): GoalProgress {
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
  const totalDays =
    (end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24);
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

  const currentAvgPace = weeksElapsed > 0 ? Math.floor(totalMeters / weeksElapsed) : 0;
  const targetWeekly = target / totalWeeks;
  const onPace = currentAvgPace >= targetWeekly;

  return {
    target,
    totalMeters,
    progress,
    weeksElapsed,
    totalWeeks,
    remainingMeters,
    remainingWeeks,
    requiredPace,
    currentAvgPace,
    onPace,
  };
}
