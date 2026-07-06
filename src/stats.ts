import type { Config } from "./config.ts";
import { parseGoalDate } from "./config.ts";
import type { Workout } from "./models.ts";
import { calendarDay, formatSeconds, pace500mSeconds, parsedDate } from "./models.ts";

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
  progress: number;
  weeksElapsed: number;
  totalWeeks: number;
  remainingMeters: number;
  remainingWeeks: number;
  requiredPace: number;
  currentAvgPace: number;
  onPace: boolean;
}

const RECENT_WEEKS = 4;

export function mondayOf(t: Date): Date {
  const d = new Date(t.getFullYear(), t.getMonth(), t.getDate());
  const offset = (d.getDay() + 6) % 7;
  d.setDate(d.getDate() - offset);
  return d;
}

export function workoutsInRange(workouts: Workout[], from: Date, to: Date): Workout[] {
  return workouts.filter((w) => {
    const t = parsedDate(w);
    return t >= from && t < to;
  });
}

export function buildWeekSummaries(workouts: Workout[], now: Date, weeks: number): WeekSummary[] {
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
    const diffDays = Math.round((monday.getTime() - cutoff.getTime()) / (1000 * 60 * 60 * 24));
    const idx = Math.floor(diffDays / 7);
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

export interface RecentWeek {
  weekStart: Date;
  meters: number;
  sessions: number;
}

export function recentWeeks(workouts: Workout[], now: Date, count: number): RecentWeek[] {
  const out: RecentWeek[] = [];
  for (let i = 0; i < count; i++) {
    const weekStart = mondayOf(now);
    weekStart.setDate(weekStart.getDate() - i * 7);
    const weekEnd = new Date(weekStart);
    weekEnd.setDate(weekEnd.getDate() + 7);
    const weekWorkouts = workoutsInRange(workouts, weekStart, weekEnd);
    out.push({
      weekStart,
      meters: weekWorkouts.reduce((sum, w) => sum + w.distance, 0),
      sessions: new Set(weekWorkouts.map(calendarDay)).size,
    });
  }
  return out;
}

export interface WeekSummaryData {
  week_start: string;
  meters: number;
  sessions: number;
  avg_pace_500m_seconds: number | null;
  avg_pace_500m: string | null;
  avg_spm: number | null;
  avg_hr: number | null;
}

export function localYMD(d: Date): string {
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${d.getFullYear()}-${mm}-${dd}`;
}

export function weekSummaryData(ws: WeekSummary): WeekSummaryData {
  const avgPace = ws.paceCount > 0 ? ws.paceSum / ws.paceCount : null;
  return {
    week_start: localYMD(ws.weekStart),
    meters: ws.meters,
    sessions: ws.sessions,
    avg_pace_500m_seconds: avgPace != null ? Math.round(avgPace * 10) / 10 : null,
    avg_pace_500m: avgPace != null ? formatSeconds(avgPace) : null,
    avg_spm: ws.spmCount > 0 ? Math.round((ws.spmSum / ws.spmCount) * 10) / 10 : null,
    avg_hr: ws.hrCount > 0 ? Math.round(ws.hrSum / ws.hrCount) : null,
  };
}

export function computeGoalProgress(workouts: Workout[], cfg: Config, now?: Date): GoalProgress {
  const target = cfg.goal.target_meters;
  const start = parseGoalDate(cfg.goal.start_date);
  const end = parseGoalDate(cfg.goal.end_date);
  const today = now ?? new Date();

  let totalMeters = 0;
  for (const w of workouts) {
    const t = parsedDate(w);
    if (t >= start && t <= end) {
      totalMeters += w.distance;
    }
  }

  const progress = totalMeters / target;
  const totalDays = (end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24);
  const totalWeeks = Math.ceil(totalDays / 7);

  let weeksElapsed = 0;
  if (today > start) {
    weeksElapsed = Math.floor((today.getTime() - start.getTime()) / (1000 * 60 * 60 * 24 * 7));
  }

  let remainingMeters = target - totalMeters;
  if (remainingMeters < 0) remainingMeters = 0;
  let remainingWeeks = totalWeeks - weeksElapsed;
  if (remainingWeeks < 1) remainingWeeks = 1;
  const requiredPace = Math.floor(remainingMeters / remainingWeeks);

  let currentAvgPace = 0;
  if (weeksElapsed > 0) {
    const thisMonday = mondayOf(today);
    const recentStart = new Date(thisMonday);
    recentStart.setDate(recentStart.getDate() - RECENT_WEEKS * 7);
    const windowStart = recentStart < start ? start : recentStart;
    const windowMs = thisMonday.getTime() - windowStart.getTime();
    const weeksInWindow = Math.max(1, Math.round(windowMs / (1000 * 60 * 60 * 24 * 7)));
    let recentMeters = 0;
    for (const w of workouts) {
      const t = parsedDate(w);
      if (t >= windowStart && t < thisMonday) {
        recentMeters += w.distance;
      }
    }
    currentAvgPace = Math.floor(recentMeters / weeksInWindow);
  }
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
