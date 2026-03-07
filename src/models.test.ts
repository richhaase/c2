import { describe, expect, test } from "bun:test";
import {
  parsedDate,
  calendarDay,
  pace500mSeconds,
  pace500m,
} from "./models.ts";
import type { Workout } from "./models.ts";

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
});
