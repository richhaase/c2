export interface HeartRate {
  average?: number;
  min?: number;
  max?: number;
}

/**
 * A single result from the Concept2 Logbook API.
 *
 * Time fields are in tenths of a second. For interval workouts (those with
 * `rest_time > 0` or a `workout_type` containing "Interval"):
 * - `time` is **work time only** (total across all reps, excluding rest)
 * - `time_formatted` is **elapsed time including rest**
 * - `distance` is total work meters (excluding rest rowing)
 *
 * This means pace (`time / distance`) is correctly work pace, but
 * `time_formatted` cannot be used for pace math on interval workouts.
 * Use `isIntervalWorkout()` to classify and `restSeconds()` to recover
 * elapsed vs work time when needed.
 */
export interface Workout {
  id: number;
  user_id: number;
  date: string; // "2026-03-02 17:41:00"
  timezone?: string;
  distance: number; // meters of work (rest rowing NOT included)
  type: string; // "rower"
  time: number; // tenths of a second of work time (rest NOT included)
  time_formatted: string; // elapsed time including rest for interval workouts
  workout_type?: string; // e.g., "FixedDistanceSplits", "FixedDistanceInterval"
  source?: string;
  weight_class?: string;
  stroke_rate?: number;
  stroke_count?: number;
  calories_total?: number;
  drag_factor?: number;
  heart_rate?: HeartRate;
  stroke_data?: boolean;
  /** Total rest time across all rest intervals, in tenths of a second. */
  rest_time?: number;
  /** Total meters rowed during rest intervals (easy rowing between reps). */
  rest_distance?: number;
  comments?: string;
}

export interface StrokeData {
  t?: number; // cumulative time
  d?: number; // cumulative distance
  p?: number; // pace
  spm?: number; // strokes per minute
  hr?: number; // heart rate
}

export interface UserProfile {
  id: number;
  username: string;
  first_name?: string;
  last_name?: string;
  email?: string;
}

export interface UserResponse {
  data: UserProfile;
}

export interface Pagination {
  total: number;
  count: number;
  per_page: number;
  current_page: number;
  total_pages: number;
}

export interface ResultsMeta {
  pagination?: Pagination;
}

export interface ResultsResponse {
  data: Workout[];
  meta?: ResultsMeta;
}

/** API wraps stroke data in {"data": [...]} */
export interface StrokeDataResponse {
  data: StrokeData[];
}

const TENTHS_PER_SECOND = 10;
const PACE_DISTANCE = 500;

export function parsedDate(w: Workout): Date {
  return new Date(w.date.replace(" ", "T"));
}

export function calendarDay(w: Workout): string {
  return w.date.slice(0, 10);
}

export function pace500mSeconds(w: Workout): number {
  if (w.distance === 0 || w.time === 0) return 0;
  return (w.time / TENTHS_PER_SECOND) * (PACE_DISTANCE / w.distance);
}

export function pace500m(w: Workout): string {
  const secs = pace500mSeconds(w);
  if (secs === 0) return "-";
  return formatSeconds(secs);
}

/**
 * Returns true if this workout is an interval workout — i.e., multiple
 * work reps separated by rest. Interval workouts have `time` as work time
 * only and `time_formatted` as elapsed time including rest.
 *
 * Detection is inclusive: any of the following is sufficient.
 * - `workout_type` contains "Interval" (the Concept2 API's own classification)
 * - `rest_time` is > 0
 * - `rest_distance` is > 0
 */
export function isIntervalWorkout(w: Workout): boolean {
  if (w.workout_type?.includes("Interval")) return true;
  if (w.rest_time != null && w.rest_time > 0) return true;
  if (w.rest_distance != null && w.rest_distance > 0) return true;
  return false;
}

/** Total rest time in seconds (0 when not an interval workout). */
export function restSeconds(w: Workout): number {
  return (w.rest_time ?? 0) / TENTHS_PER_SECOND;
}

/** Total work time (excluding rest) in seconds. */
export function workSeconds(w: Workout): number {
  return w.time / TENTHS_PER_SECOND;
}

/**
 * Format a duration in seconds as `M:SS.S` (matching the Concept2 API's
 * `time_formatted` style). Returns `"0:00.0"` for zero.
 */
export function formatSeconds(totalSeconds: number): string {
  if (totalSeconds <= 0) return "0:00.0";
  const mins = Math.floor(totalSeconds / 60);
  const rem = totalSeconds - mins * 60;
  return `${mins}:${rem.toFixed(1).padStart(4, "0")}`;
}
