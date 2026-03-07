import type { Workout } from "./models.ts";
import { calendarDay } from "./models.ts";

export interface Session {
  date: string; // "2026-03-07"
  workouts: Workout[];
  totalDistance: number;
  totalTime: number; // tenths of seconds
}

export function groupIntoSessions(workouts: Workout[]): Session[] {
  const byDay = new Map<string, Workout[]>();

  for (const w of workouts) {
    const day = calendarDay(w);
    const existing = byDay.get(day);
    if (existing) {
      existing.push(w);
    } else {
      byDay.set(day, [w]);
    }
  }

  return Array.from(byDay.entries())
    .map(([date, ws]) => ({
      date,
      workouts: ws.sort((a, b) => a.date.localeCompare(b.date)),
      totalDistance: ws.reduce((sum, w) => sum + w.distance, 0),
      totalTime: ws.reduce((sum, w) => sum + w.time, 0),
    }))
    .sort((a, b) => a.date.localeCompare(b.date));
}

/** Count unique calendar days (sessions) in a list of workouts. */
export function sessionCount(workouts: Workout[]): number {
  const days = new Set(workouts.map(calendarDay));
  return days.size;
}
