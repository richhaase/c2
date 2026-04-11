import { describe, expect, test } from "bun:test";
import {
  formatIntervalTag,
  formatMeters,
  formatMetersPerWeek,
  formatPercent,
  formatWorkoutLine,
  paceArrow,
  sparkBar,
  trendArrow,
} from "./display.ts";
import type { Workout } from "./models.ts";

function makeWorkout(overrides: Partial<Workout> = {}): Workout {
  return {
    id: 1,
    user_id: 1,
    date: "2026-04-09 07:00:00",
    distance: 5000,
    type: "rower",
    time: 17155,
    time_formatted: "28:35.4",
    workout_type: "FixedDistanceSplits",
    stroke_rate: 24,
    heart_rate: { average: 112 },
    drag_factor: 107,
    ...overrides,
  };
}

describe("formatMeters", () => {
  test.each([
    [0, "0"],
    [500, "500"],
    [1000, "1,000"],
    [12345, "12,345"],
    [1000000, "1,000,000"],
  ])("formatMeters(%i) = %s", (input, expected) => {
    expect(formatMeters(input)).toBe(expected);
  });
});

describe("formatPercent", () => {
  test.each([
    [0, "0.0%"],
    [0.5, "50.0%"],
    [1.0, "100.0%"],
    [0.1234, "12.3%"],
    [0.131, "13.1%"],
  ])("formatPercent(%f) = %s", (input, expected) => {
    expect(formatPercent(input)).toBe(expected);
  });
});

describe("formatMetersPerWeek", () => {
  test("formats with unit", () => {
    expect(formatMetersPerWeek(20212)).toBe("20,212m/week");
  });
});

describe("sparkBar", () => {
  test("returns empty string when max is 0", () => {
    expect(sparkBar(100, 0)).toBe("");
  });

  test("returns full bar for max value", () => {
    const bar = sparkBar(100, 100);
    expect(bar).toBe("\u2588".repeat(20));
  });

  test("returns half bar for 50%", () => {
    const bar = sparkBar(50, 100);
    expect(bar.length).toBe(20);
    expect(bar).toBe("\u2588".repeat(10) + "\u2591".repeat(10));
  });
});

describe("trendArrow", () => {
  test("returns space when prev is 0", () => {
    expect(trendArrow(0, 100)).toBe(" ");
  });

  test("returns up arrow for increase", () => {
    expect(trendArrow(100, 110)).toBe("\u2191");
  });

  test("returns down arrow for decrease", () => {
    expect(trendArrow(100, 90)).toBe("\u2193");
  });

  test("returns right arrow for stable", () => {
    expect(trendArrow(100, 101)).toBe("\u2192");
  });
});

describe("paceArrow", () => {
  test("reversed: lower pace shows up arrow (improvement)", () => {
    expect(paceArrow(180, 170)).toBe("\u2191");
  });

  test("reversed: higher pace shows down arrow", () => {
    expect(paceArrow(170, 180)).toBe("\u2193");
  });
});

describe("formatIntervalTag", () => {
  test("returns empty string for a continuous piece", () => {
    const w = makeWorkout({ workout_type: "FixedDistanceSplits" });
    expect(formatIntervalTag(w)).toBe("");
  });

  test("returns tag with rest duration for interval workout", () => {
    const w = makeWorkout({
      workout_type: "FixedDistanceInterval",
      rest_time: 3600,
    });
    expect(formatIntervalTag(w)).toBe("[IVL rest 6:00.0]");
  });

  test("returns bare [IVL] when rest_time is missing but type is Interval", () => {
    const w = makeWorkout({ workout_type: "FixedDistanceInterval" });
    expect(formatIntervalTag(w)).toBe("[IVL]");
  });

  test("detects interval via rest_distance alone", () => {
    const w = makeWorkout({
      workout_type: undefined,
      rest_distance: 660,
    });
    expect(formatIntervalTag(w)).toBe("[IVL]");
  });
});

describe("formatWorkoutLine", () => {
  test("formats a continuous piece without an interval tag", () => {
    const w = makeWorkout({
      date: "2026-04-09 07:00:00",
      distance: 5000,
      time: 17155,
      time_formatted: "28:35.4",
      workout_type: "FixedDistanceSplits",
      stroke_rate: 24,
      heart_rate: { average: 112 },
      drag_factor: 107,
    });
    const line = formatWorkoutLine(w, "01/02");
    expect(line).toContain("04/09");
    expect(line).toContain("5,000m");
    expect(line).toContain("28:35.4");
    expect(line).toContain("2:51.6/500m");
    expect(line).toContain("24spm");
    expect(line).toContain("112bpm");
    expect(line).toContain("107df");
    expect(line).not.toContain("[IVL");
  });

  test("appends [IVL rest M:SS.S] tag for interval workouts", () => {
    // Real 2026-04-11 session: 6x500m, 14:22.6 work, 6:00 rest.
    const w = makeWorkout({
      date: "2026-04-11 09:14:00",
      distance: 3000,
      time: 8626,
      time_formatted: "20:22.6",
      workout_type: "FixedDistanceInterval",
      rest_time: 3600,
      rest_distance: 660,
      stroke_rate: 30,
      heart_rate: { average: 152 },
      drag_factor: 108,
    });
    const line = formatWorkoutLine(w, "01/02");
    expect(line).toContain("[IVL rest 6:00.0]");
    // Pace comes from work time, not elapsed — must be 2:23.8, not 3:23.8.
    expect(line).toContain("2:23.8/500m");
  });

  test("handles missing stroke_rate / heart_rate / drag_factor gracefully", () => {
    const w = makeWorkout({
      stroke_rate: undefined,
      heart_rate: undefined,
      drag_factor: undefined,
    });
    const line = formatWorkoutLine(w, "01/02");
    expect(line).toContain("    -");
  });
});
