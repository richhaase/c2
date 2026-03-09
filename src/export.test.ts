import { describe, expect, test } from "bun:test";
import { escapeCSV, filterByDate } from "./commands/export.ts";
import type { Workout } from "./models.ts";

function makeWorkout(id: number, date: string, distance: number): Workout {
  return {
    id,
    user_id: 1,
    date,
    distance,
    type: "rower",
    time: Math.round(distance * 3.5),
    time_formatted: "0:00.0",
  };
}

describe("escapeCSV", () => {
  test("passes through plain string", () => {
    expect(escapeCSV("hello")).toBe("hello");
  });

  test("quotes string with commas", () => {
    expect(escapeCSV("a,b")).toBe('"a,b"');
  });

  test("escapes double quotes", () => {
    expect(escapeCSV('say "hi"')).toBe('"say ""hi"""');
  });

  test("quotes string with newlines", () => {
    expect(escapeCSV("line1\nline2")).toBe('"line1\nline2"');
  });

  test("handles empty string", () => {
    expect(escapeCSV("")).toBe("");
  });
});

describe("filterByDate", () => {
  const workouts = [
    makeWorkout(1, "2026-01-15 10:00:00", 5000),
    makeWorkout(2, "2026-02-15 10:00:00", 5000),
    makeWorkout(3, "2026-03-15 10:00:00", 5000),
  ];

  test("filters with both from and to", () => {
    const result = filterByDate(workouts, "2026-02-01", "2026-02-28");
    expect(result).toHaveLength(1);
    expect(result[0]!.id).toBe(2);
  });

  test("filters with from only", () => {
    const result = filterByDate(workouts, "2026-02-01", "");
    expect(result).toHaveLength(2);
    expect(result[0]!.id).toBe(2);
    expect(result[1]!.id).toBe(3);
  });

  test("filters with to only", () => {
    const result = filterByDate(workouts, "", "2026-02-28");
    expect(result).toHaveLength(2);
    expect(result[0]!.id).toBe(1);
    expect(result[1]!.id).toBe(2);
  });

  test("returns all when no bounds", () => {
    const result = filterByDate(workouts, "", "");
    expect(result).toHaveLength(3);
  });
});
