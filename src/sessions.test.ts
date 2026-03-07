import { describe, expect, test } from "bun:test";
import { groupIntoSessions, sessionCount } from "./sessions.ts";
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

describe("groupIntoSessions", () => {
  test("groups 3 workouts on same day into 1 session", () => {
    const workouts = [
      makeWorkout(1, "2026-03-07 09:21:00", 1000),
      makeWorkout(2, "2026-03-07 09:45:00", 2500),
      makeWorkout(3, "2026-03-07 09:53:00", 1000),
    ];
    const sessions = groupIntoSessions(workouts);
    expect(sessions).toHaveLength(1);
    expect(sessions[0]!.date).toBe("2026-03-07");
    expect(sessions[0]!.totalDistance).toBe(4500);
    expect(sessions[0]!.workouts).toHaveLength(3);
  });

  test("keeps different days as separate sessions", () => {
    const workouts = [
      makeWorkout(1, "2026-03-05 10:00:00", 5000),
      makeWorkout(2, "2026-03-07 09:00:00", 5000),
    ];
    const sessions = groupIntoSessions(workouts);
    expect(sessions).toHaveLength(2);
    expect(sessions[0]!.date).toBe("2026-03-05");
    expect(sessions[1]!.date).toBe("2026-03-07");
  });

  test("sorts sessions by date", () => {
    const workouts = [
      makeWorkout(2, "2026-03-07 09:00:00", 5000),
      makeWorkout(1, "2026-03-05 10:00:00", 5000),
    ];
    const sessions = groupIntoSessions(workouts);
    expect(sessions[0]!.date).toBe("2026-03-05");
    expect(sessions[1]!.date).toBe("2026-03-07");
  });

  test("single workout produces single session", () => {
    const workouts = [makeWorkout(1, "2026-03-07 09:00:00", 5500)];
    const sessions = groupIntoSessions(workouts);
    expect(sessions).toHaveLength(1);
    expect(sessions[0]!.totalDistance).toBe(5500);
    expect(sessions[0]!.workouts).toHaveLength(1);
  });

  test("empty input returns empty array", () => {
    expect(groupIntoSessions([])).toEqual([]);
  });
});

describe("sessionCount", () => {
  test("counts unique calendar days", () => {
    const workouts = [
      makeWorkout(1, "2026-03-07 09:21:00", 1000),
      makeWorkout(2, "2026-03-07 09:45:00", 2500),
      makeWorkout(3, "2026-03-07 09:53:00", 1000),
      makeWorkout(4, "2026-03-05 14:00:00", 5000),
    ];
    expect(sessionCount(workouts)).toBe(2);
  });

  test("returns 0 for empty list", () => {
    expect(sessionCount([])).toBe(0);
  });
});
