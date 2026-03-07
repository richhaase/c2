import { describe, expect, test } from "bun:test";
import type { Workout } from "./models.ts";
import type { Config } from "./config.ts";
import { defaultConfig } from "./config.ts";
import {
  mondayOf,
  workoutsInRange,
  buildWeekSummaries,
  computeGoalProgress,
} from "./stats.ts";

function makeWorkout(id: number, date: string, distance: number, time?: number): Workout {
  return {
    id,
    user_id: 1,
    date,
    distance,
    type: "rower",
    time: time ?? Math.round(distance * 3.5),
    time_formatted: "0:00.0",
  };
}

function makeGoalConfig(overrides: Partial<Config["goal"]> = {}): Config {
  const cfg = defaultConfig();
  cfg.goal = { ...cfg.goal, start_date: "2026-01-01", end_date: "2026-12-31", target_meters: 1_000_000, ...overrides };
  return cfg;
}

describe("mondayOf", () => {
  test("monday returns same date", () => {
    const d = new Date(2026, 2, 2); // Mon Mar 2
    const m = mondayOf(d);
    expect(m.getDay()).toBe(1); // Monday
    expect(m.getDate()).toBe(2);
  });

  test("wednesday returns previous monday", () => {
    const d = new Date(2026, 2, 4); // Wed Mar 4
    const m = mondayOf(d);
    expect(m.getDay()).toBe(1);
    expect(m.getDate()).toBe(2);
  });

  test("sunday returns previous monday", () => {
    const d = new Date(2026, 2, 8); // Sun Mar 8
    const m = mondayOf(d);
    expect(m.getDay()).toBe(1);
    expect(m.getDate()).toBe(2);
  });

  test("handles month boundary", () => {
    const d = new Date(2026, 2, 1); // Mar 1
    const m = mondayOf(d);
    expect(m.getDay()).toBe(1);
    expect(m.getTime()).toBeLessThanOrEqual(d.getTime());
  });
});

describe("workoutsInRange", () => {
  test("filters workouts within date range", () => {
    const workouts = [
      makeWorkout(1, "2026-01-15 10:00:00", 5000),
      makeWorkout(2, "2026-02-15 10:00:00", 5000),
      makeWorkout(3, "2026-03-15 10:00:00", 5000),
    ];
    const from = new Date(2026, 1, 1); // Feb 1
    const to = new Date(2026, 2, 1); // Mar 1
    const result = workoutsInRange(workouts, from, to);
    expect(result).toHaveLength(1);
    expect(result[0]!.id).toBe(2);
  });

  test("returns empty for no matches", () => {
    const workouts = [makeWorkout(1, "2026-06-15 10:00:00", 5000)];
    const from = new Date(2026, 0, 1);
    const to = new Date(2026, 1, 1);
    expect(workoutsInRange(workouts, from, to)).toHaveLength(0);
  });
});

describe("buildWeekSummaries", () => {
  test("buckets workouts into correct weeks", () => {
    const now = new Date(2026, 2, 7); // Sat Mar 7
    const workouts = [
      makeWorkout(1, "2026-03-02 10:00:00", 5000), // Mon of current week
      makeWorkout(2, "2026-02-23 10:00:00", 6000), // Previous week
    ];
    const summaries = buildWeekSummaries(workouts, now, 2);
    expect(summaries).toHaveLength(2);
    expect(summaries[0]!.meters).toBe(6000);
    expect(summaries[1]!.meters).toBe(5000);
  });

  test("counts sessions as unique days", () => {
    const now = new Date(2026, 2, 7);
    const workouts = [
      makeWorkout(1, "2026-03-02 09:00:00", 1000),
      makeWorkout(2, "2026-03-02 10:00:00", 2000), // same day
      makeWorkout(3, "2026-03-04 10:00:00", 3000), // different day
    ];
    const summaries = buildWeekSummaries(workouts, now, 1);
    expect(summaries[0]!.meters).toBe(6000);
    expect(summaries[0]!.sessions).toBe(2); // 2 unique days
  });

  test("returns empty summaries for no workouts", () => {
    const now = new Date(2026, 2, 7);
    const summaries = buildWeekSummaries([], now, 4);
    expect(summaries).toHaveLength(4);
    expect(summaries.every(s => s.meters === 0)).toBe(true);
  });
});

describe("computeGoalProgress", () => {
  test("computes progress for mid-season", () => {
    const cfg = makeGoalConfig();
    const workouts = [
      makeWorkout(1, "2026-03-01 10:00:00", 100_000),
      makeWorkout(2, "2026-02-01 10:00:00", 100_000),
    ];
    const now = new Date(2026, 2, 7); // Mar 7
    const goal = computeGoalProgress(workouts, cfg, now);
    expect(goal.totalMeters).toBe(200_000);
    expect(goal.target).toBe(1_000_000);
    expect(goal.progress).toBeCloseTo(0.2, 2);
    expect(goal.remainingMeters).toBe(800_000);
    expect(goal.weeksElapsed).toBeGreaterThan(0);
  });

  test("clamps remainingMeters to 0 when goal exceeded", () => {
    const cfg = makeGoalConfig({ target_meters: 100_000 });
    const workouts = [makeWorkout(1, "2026-03-01 10:00:00", 150_000)];
    const now = new Date(2026, 2, 7);
    const goal = computeGoalProgress(workouts, cfg, now);
    expect(goal.remainingMeters).toBe(0);
    expect(goal.progress).toBeGreaterThan(1);
  });

  test("excludes workouts outside goal date range", () => {
    const cfg = makeGoalConfig();
    const workouts = [
      makeWorkout(1, "2025-12-01 10:00:00", 50_000), // before start
      makeWorkout(2, "2026-03-01 10:00:00", 100_000), // in range
      makeWorkout(3, "2027-02-01 10:00:00", 50_000), // after end
    ];
    const now = new Date(2026, 2, 7);
    const goal = computeGoalProgress(workouts, cfg, now);
    expect(goal.totalMeters).toBe(100_000);
  });

  test("before start date: weeksElapsed is 0", () => {
    const cfg = makeGoalConfig({ start_date: "2026-06-01" });
    const workouts: Workout[] = [];
    const now = new Date(2026, 2, 7); // before start
    const goal = computeGoalProgress(workouts, cfg, now);
    expect(goal.weeksElapsed).toBe(0);
    expect(goal.currentAvgPace).toBe(0);
  });
});
