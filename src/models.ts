export interface HeartRate {
  average?: number;
  min?: number;
  max?: number;
}

/** A single result from the Concept2 Logbook API. Time is in tenths of a second. */
export interface Workout {
  id: number;
  user_id: number;
  date: string; // "2026-03-02 17:41:00"
  timezone?: string;
  distance: number;
  type: string; // "rower"
  time: number; // tenths of a second
  time_formatted: string;
  workout_type?: string;
  source?: string;
  weight_class?: string;
  stroke_rate?: number;
  stroke_count?: number;
  calories_total?: number;
  drag_factor?: number;
  heart_rate?: HeartRate;
  stroke_data?: boolean;
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
  const mins = Math.floor(secs / 60);
  const rem = secs - mins * 60;
  return `${mins}:${rem.toFixed(1).padStart(4, "0")}`;
}
