import { describe, expect, test } from "bun:test";
import type { Workout } from "./models.ts";
import {
  calendarDay,
  formatSeconds,
  isIntervalWorkout,
  pace500m,
  pace500mSeconds,
  parsedDate,
  restSeconds,
  workSeconds,
} from "./models.ts";

function makeWorkout(overrides: Partial<Workout> = {}): Workout {
  return {
    id: 1,
    user_id: 1,
    date: "2026-03-07 09:21:00",
    distance: 5500,
    type: "rower",
    time: 19122,
    time_formatted: "31:52.2",
    ...overrides,
  };
}

describe("parsedDate", () => {
  test("parses workout date string", () => {
    const w = makeWorkout({ date: "2026-03-07 09:21:00" });
    const d = parsedDate(w);
    expect(d.getFullYear()).toBe(2026);
    expect(d.getMonth()).toBe(2); // March = 2 (0-indexed)
    expect(d.getDate()).toBe(7);
    expect(d.getHours()).toBe(9);
    expect(d.getMinutes()).toBe(21);
  });
});

describe("calendarDay", () => {
  test("extracts date portion", () => {
    const w = makeWorkout({ date: "2026-03-07 09:21:00" });
    expect(calendarDay(w)).toBe("2026-03-07");
  });
});

describe("pace500mSeconds", () => {
  test("computes pace for normal workout", () => {
    const w = makeWorkout({ distance: 5500, time: 19122 });
    const pace = pace500mSeconds(w);
    // 19122 / 10 * 500 / 5500 = 173.836...
    expect(pace).toBeCloseTo(173.836, 2);
  });

  test("returns 0 for zero distance", () => {
    const w = makeWorkout({ distance: 0 });
    expect(pace500mSeconds(w)).toBe(0);
  });

  test("returns 0 for zero time", () => {
    const w = makeWorkout({ time: 0 });
    expect(pace500mSeconds(w)).toBe(0);
  });
});

describe("pace500m", () => {
  test("formats pace as M:SS.S", () => {
    const w = makeWorkout({ distance: 5500, time: 19122 });
    expect(pace500m(w)).toBe("2:53.8");
  });

  test("returns dash for zero values", () => {
    const w = makeWorkout({ distance: 0, time: 0 });
    expect(pace500m(w)).toBe("-");
  });

  test("formats 3+ minute pace correctly", () => {
    const w = makeWorkout({ distance: 1000, time: 3706 });
    // 3706 / 10 * 500 / 1000 = 185.3
    expect(pace500m(w)).toBe("3:05.3");
  });

  test("uses work time (not elapsed) for interval workout pace", () => {
    // Real 2026-04-11 session: 6x500m, 14:22.6 work, 6:00 rest.
    // API returns `time: 8626` tenths (work), `time_formatted: "20:22.6"` (elapsed).
    // Pace must be computed from `time`, not `time_formatted`.
    const w = makeWorkout({
      distance: 3000,
      time: 8626,
      time_formatted: "20:22.6",
      workout_type: "FixedDistanceInterval",
      rest_time: 3600,
      rest_distance: 660,
    });
    // 8626 / 10 * 500 / 3000 = 143.77
    expect(pace500m(w)).toBe("2:23.8");
  });
});

describe("isIntervalWorkout", () => {
  test("returns false for a continuous piece", () => {
    const w = makeWorkout({
      workout_type: "FixedDistanceSplits",
    });
    expect(isIntervalWorkout(w)).toBe(false);
  });

  test("returns true for FixedDistanceInterval workout_type", () => {
    const w = makeWorkout({ workout_type: "FixedDistanceInterval" });
    expect(isIntervalWorkout(w)).toBe(true);
  });

  test("returns true for FixedTimeInterval workout_type", () => {
    const w = makeWorkout({ workout_type: "FixedTimeInterval" });
    expect(isIntervalWorkout(w)).toBe(true);
  });

  test("returns true when rest_time > 0 even without workout_type", () => {
    const w = makeWorkout({ rest_time: 3600 });
    expect(isIntervalWorkout(w)).toBe(true);
  });

  test("returns true when rest_distance > 0 even without workout_type", () => {
    const w = makeWorkout({ rest_distance: 660 });
    expect(isIntervalWorkout(w)).toBe(true);
  });

  test("returns false when rest_time is explicitly 0", () => {
    const w = makeWorkout({
      workout_type: "FixedDistanceSplits",
      rest_time: 0,
      rest_distance: 0,
    });
    expect(isIntervalWorkout(w)).toBe(false);
  });

  test("returns false when workout_type is missing and no rest", () => {
    const w = makeWorkout({});
    expect(isIntervalWorkout(w)).toBe(false);
  });
});

describe("restSeconds", () => {
  test("returns 0 when rest_time is undefined", () => {
    const w = makeWorkout({});
    expect(restSeconds(w)).toBe(0);
  });

  test("converts tenths to seconds", () => {
    const w = makeWorkout({ rest_time: 3600 });
    expect(restSeconds(w)).toBe(360);
  });
});

describe("workSeconds", () => {
  test("converts time tenths to seconds", () => {
    // 8626 tenths = 862.6 seconds = 14:22.6
    const w = makeWorkout({ time: 8626 });
    expect(workSeconds(w)).toBe(862.6);
  });
});

describe("formatSeconds", () => {
  test.each([
    [0, "0:00.0"],
    [-5, "0:00.0"],
    [5.5, "0:05.5"],
    [65.3, "1:05.3"],
    [360, "6:00.0"],
    [862.6, "14:22.6"],
  ])("formatSeconds(%f) = %s", (input, expected) => {
    expect(formatSeconds(input)).toBe(expected);
  });
});
