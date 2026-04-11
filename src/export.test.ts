import { describe, expect, test } from "bun:test";
import { buildCSVRow, CSV_HEADER, escapeCSV, filterByDate } from "./commands/export.ts";
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

describe("CSV_HEADER", () => {
  test("includes rest_time_tenths and rest_distance columns", () => {
    expect(CSV_HEADER).toContain("rest_time_tenths");
    expect(CSV_HEADER).toContain("rest_distance");
  });

  test("includes workout_type column", () => {
    expect(CSV_HEADER).toContain("workout_type");
  });
});

describe("buildCSVRow", () => {
  test("row length matches header length", () => {
    const w = makeWorkout(1, "2026-01-15 10:00:00", 5000);
    const row = buildCSVRow(w);
    expect(row).toHaveLength(CSV_HEADER.length);
  });

  test("leaves rest columns empty for a continuous piece", () => {
    const w: Workout = {
      id: 1,
      user_id: 1,
      date: "2026-04-09 07:00:00",
      distance: 5000,
      type: "rower",
      time: 17155,
      time_formatted: "28:35.4",
      workout_type: "FixedDistanceSplits",
    };
    const row = buildCSVRow(w);
    const restTimeIdx = CSV_HEADER.indexOf("rest_time_tenths");
    const restDistanceIdx = CSV_HEADER.indexOf("rest_distance");
    expect(row[restTimeIdx]).toBe("");
    expect(row[restDistanceIdx]).toBe("");
  });

  test("populates rest columns for an interval workout", () => {
    const w: Workout = {
      id: 2,
      user_id: 1,
      date: "2026-04-11 09:14:00",
      distance: 3000,
      type: "rower",
      time: 8626,
      time_formatted: "20:22.6",
      workout_type: "FixedDistanceInterval",
      rest_time: 3600,
      rest_distance: 660,
    };
    const row = buildCSVRow(w);
    const workoutTypeIdx = CSV_HEADER.indexOf("workout_type");
    const restTimeIdx = CSV_HEADER.indexOf("rest_time_tenths");
    const restDistanceIdx = CSV_HEADER.indexOf("rest_distance");
    expect(row[workoutTypeIdx]).toBe("FixedDistanceInterval");
    expect(row[restTimeIdx]).toBe("3600");
    expect(row[restDistanceIdx]).toBe("660");
  });

  test("preserves explicit zero rest_time", () => {
    const w: Workout = {
      id: 3,
      user_id: 1,
      date: "2026-04-09 07:00:00",
      distance: 5000,
      type: "rower",
      time: 17155,
      time_formatted: "28:35.4",
      rest_time: 0,
      rest_distance: 0,
    };
    const row = buildCSVRow(w);
    const restTimeIdx = CSV_HEADER.indexOf("rest_time_tenths");
    const restDistanceIdx = CSV_HEADER.indexOf("rest_distance");
    expect(row[restTimeIdx]).toBe("0");
    expect(row[restDistanceIdx]).toBe("0");
  });
});
