import { expect, test } from "bun:test";
import { hrAtPace, splitShape, splitTable, strokeSummary } from "./analysis.ts";
import { projectGoal } from "./commands/stats.ts";
import type { StrokeData, Workout, WorkoutSplit } from "./models.ts";
import type { GoalProgress } from "./stats.ts";

function split(timeTenths: number, distance: number, spm: number, hr: number): WorkoutSplit {
  return {
    type: "distance",
    time: timeTenths,
    distance,
    stroke_rate: spm,
    heart_rate: { average: hr, max: hr + 8 },
  };
}

function workoutWithSplits(splits: WorkoutSplit[]): Workout {
  return {
    id: 1,
    user_id: 1,
    date: "2026-07-03 12:00:00",
    distance: splits.reduce((s, x) => s + (x.distance ?? 0), 0),
    type: "rower",
    time: splits.reduce((s, x) => s + x.time, 0),
    time_formatted: "34:39.1",
    workout: { targets: { pace: 1750 }, splits },
  };
}

const NEGATIVE_SPLIT_WORKOUT = workoutWithSplits([
  split(4280, 1200, 24, 93),
  split(4210, 1200, 25, 107),
  split(4140, 1200, 25, 115),
  split(4076, 1200, 26, 120),
  split(4088, 1200, 26, 122),
]);

test("splitTable computes per-split pace from tenths", () => {
  const rows = splitTable(NEGATIVE_SPLIT_WORKOUT);
  expect(rows.length).toBe(5);
  expect(rows[0]!.pace_500m_seconds).toBe(178.3);
  expect(rows[0]!.pace_500m).toBe("2:58.3");
  expect(rows[3]!.pace_500m).toBe("2:49.8");
  expect(rows[0]!.hr_avg).toBe(93);
  expect(rows[0]!.hr_max).toBe(101);
});

test("splitShape detects the 07/03-style negative split", () => {
  expect(splitShape(splitTable(NEGATIVE_SPLIT_WORKOUT))).toBe("negative");
});

test("splitShape detects even, positive, variable, and unknown", () => {
  const even = workoutWithSplits([
    split(4200, 1200, 25, 110),
    split(4210, 1200, 25, 112),
    split(4195, 1200, 25, 113),
    split(4205, 1200, 25, 114),
  ]);
  expect(splitShape(splitTable(even))).toBe("even");

  const positive = workoutWithSplits([
    split(4000, 1200, 26, 118),
    split(4100, 1200, 25, 120),
    split(4250, 1200, 24, 121),
    split(4350, 1200, 23, 122),
  ]);
  expect(splitShape(splitTable(positive))).toBe("positive");

  const variable = workoutWithSplits([
    split(4000, 1200, 26, 118),
    split(4400, 1200, 22, 110),
    split(3950, 1200, 27, 125),
    split(4380, 1200, 22, 112),
  ]);
  expect(splitShape(splitTable(variable))).toBe("variable");

  expect(splitShape(splitTable(workoutWithSplits([split(4200, 1200, 25, 110)])))).toBe("unknown");
  expect(splitShape([])).toBe("unknown");
});

test("strokeSummary aggregates samples", () => {
  const strokes: StrokeData[] = [
    { t: 100, d: 500, p: 1750, spm: 24, hr: 110 },
    { t: 200, d: 1000, p: 1730, spm: 25, hr: 118 },
    { t: 300, d: 1500, p: 1710, spm: 26, hr: 126 },
    { t: 400, d: 2000, p: 0, spm: 0, hr: 0 },
  ];
  const s = strokeSummary(strokes);
  expect(s.samples).toBe(4);
  expect(s.avg_pace_500m_seconds).toBe(173);
  expect(s.avg_pace_500m).toBe("2:53.0");
  expect(s.avg_spm).toBe(25);
  expect(s.avg_hr).toBe(118);
  expect(s.max_hr).toBe(126);
});

function goalFixture(overrides: Partial<GoalProgress>): GoalProgress {
  return {
    target: 1_000_000,
    totalMeters: 900_000,
    progress: 0.9,
    weeksElapsed: 26,
    totalWeeks: 52,
    remainingMeters: 100_000,
    remainingWeeks: 26,
    requiredPace: 3846,
    currentAvgPace: 20_000,
    onPace: true,
    ...overrides,
  };
}

const PROJECT_NOW = new Date("2026-07-06T12:00:00");

function daysAfterNow(n: number): Date {
  return new Date(PROJECT_NOW.getTime() + n * 24 * 60 * 60 * 1000);
}

test("projectGoal projects over actual remaining time", () => {
  const active = projectGoal(goalFixture({}), daysAfterNow(26 * 7), PROJECT_NOW);
  expect(active.projected_total_meters).toBe(900_000 + 26 * 20_000);
  expect(active.shortfall_meters).toBe(0);
});

test("projectGoal adds nothing after the goal window ends", () => {
  const expired = projectGoal(
    goalFixture({ weeksElapsed: 60, remainingWeeks: 1 }),
    daysAfterNow(-2),
    PROJECT_NOW,
  );
  expect(expired.projected_total_meters).toBe(900_000);
  expect(expired.projected_pct).toBe(90);
  expect(expired.shortfall_meters).toBe(100_000);
});

test("projectGoal handles fractional final weeks without over-projecting", () => {
  const halfWeek = projectGoal(goalFixture({}), daysAfterNow(3.5), PROJECT_NOW);
  expect(halfWeek.projected_total_meters).toBe(900_000 + 10_000);
});

function steadyWorkout(id: number, date: string, paceSecs: number, hr: number): Workout {
  const distance = 6000;
  return {
    id,
    user_id: 1,
    date,
    distance,
    type: "rower",
    time: Math.round(paceSecs * (distance / 500) * 10),
    time_formatted: "x",
    heart_rate: { average: hr },
  };
}

const NOW = new Date("2026-07-05T12:00:00");

test("hrAtPace buckets steady work and reports early-late drift", () => {
  const workouts: Workout[] = [
    steadyWorkout(1, "2026-05-20 08:00:00", 173, 120),
    steadyWorkout(2, "2026-05-27 08:00:00", 174, 118),
    steadyWorkout(3, "2026-06-24 08:00:00", 172, 112),
    steadyWorkout(4, "2026-07-01 08:00:00", 171, 110),
    steadyWorkout(5, "2026-07-03 08:00:00", 168, 115),
    { ...steadyWorkout(6, "2026-07-02 08:00:00", 150, 140), rest_time: 600 },
    steadyWorkout(7, "2026-01-01 08:00:00", 173, 130),
  ];
  const bands = hrAtPace(workouts, NOW, 8);

  expect(bands.length).toBe(2);
  const band170 = bands.find((b) => b.band_start_seconds === 170)!;
  expect(band170.workouts).toBe(4);
  expect(band170.early_avg_hr).toBe(119);
  expect(band170.late_avg_hr).toBe(111);
  expect(band170.hr_delta).toBe(-8);
  expect(band170.band).toBe("2:50.0–2:55.0");

  const band165 = bands.find((b) => b.band_start_seconds === 165)!;
  expect(band165.workouts).toBe(1);
  expect(band165.hr_delta).toBeNull();
});
