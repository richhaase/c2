import type { StrokeData, Workout } from "./models.ts";
import { formatSeconds, isIntervalWorkout, pace500mSeconds, parsedDate } from "./models.ts";

const TENTHS_PER_SECOND = 10;
const PACE_DISTANCE = 500;

export interface SplitRow {
  index: number;
  distance: number | null;
  time_seconds: number;
  pace_500m_seconds: number | null;
  pace_500m: string | null;
  stroke_rate: number | null;
  hr_avg: number | null;
  hr_max: number | null;
}

export type SplitShape = "even" | "negative" | "positive" | "variable" | "unknown";

const SHAPE_DRIFT_THRESHOLD_SECONDS = 1.5;
const SHAPE_SPREAD_THRESHOLD_SECONDS = 6;

export function splitTable(w: Workout): SplitRow[] {
  const splits = w.workout?.splits ?? [];
  return splits.map((s, i) => {
    const seconds = s.time / TENTHS_PER_SECOND;
    const pace =
      s.distance != null && s.distance > 0 ? seconds * (PACE_DISTANCE / s.distance) : null;
    return {
      index: i + 1,
      distance: s.distance ?? null,
      time_seconds: Math.round(seconds * 10) / 10,
      pace_500m_seconds: pace != null ? Math.round(pace * 10) / 10 : null,
      pace_500m: pace != null ? formatSeconds(pace) : null,
      stroke_rate: s.stroke_rate ?? null,
      hr_avg: s.heart_rate?.average ?? null,
      hr_max: s.heart_rate?.max ?? null,
    };
  });
}

export function splitShape(rows: SplitRow[]): SplitShape {
  const paces = rows.map((r) => r.pace_500m_seconds).filter((p): p is number => p != null);
  if (paces.length < 2) return "unknown";

  const mid = Math.floor(paces.length / 2);
  const firstHalf = paces.slice(0, mid);
  const secondHalf = paces.slice(paces.length - mid);
  const avg = (xs: number[]) => xs.reduce((a, b) => a + b, 0) / xs.length;
  const drift = avg(firstHalf) - avg(secondHalf);
  const spread = Math.max(...paces) - Math.min(...paces);

  if (Math.abs(drift) <= SHAPE_DRIFT_THRESHOLD_SECONDS) {
    return spread > SHAPE_SPREAD_THRESHOLD_SECONDS ? "variable" : "even";
  }
  return drift > 0 ? "negative" : "positive";
}

export interface StrokeSummary {
  samples: number;
  avg_pace_500m_seconds: number | null;
  avg_pace_500m: string | null;
  avg_spm: number | null;
  avg_hr: number | null;
  max_hr: number | null;
}

export function strokeSummary(strokes: StrokeData[]): StrokeSummary {
  let paceSum = 0;
  let paceCount = 0;
  let spmSum = 0;
  let spmCount = 0;
  let hrSum = 0;
  let hrCount = 0;
  let maxHR = 0;
  for (const s of strokes) {
    if (s.p != null && s.p > 0) {
      paceSum += s.p / TENTHS_PER_SECOND;
      paceCount++;
    }
    if (s.spm != null && s.spm > 0) {
      spmSum += s.spm;
      spmCount++;
    }
    if (s.hr != null && s.hr > 0) {
      hrSum += s.hr;
      hrCount++;
      if (s.hr > maxHR) maxHR = s.hr;
    }
  }
  const avgPace = paceCount > 0 ? paceSum / paceCount : null;
  return {
    samples: strokes.length,
    avg_pace_500m_seconds: avgPace != null ? Math.round(avgPace * 10) / 10 : null,
    avg_pace_500m: avgPace != null ? formatSeconds(avgPace) : null,
    avg_spm: spmCount > 0 ? Math.round((spmSum / spmCount) * 10) / 10 : null,
    avg_hr: hrCount > 0 ? Math.round(hrSum / hrCount) : null,
    max_hr: maxHR > 0 ? maxHR : null,
  };
}

export interface HRPaceBand {
  band_start_seconds: number;
  band: string;
  workouts: number;
  avg_hr: number;
  early_avg_hr: number | null;
  late_avg_hr: number | null;
  hr_delta: number | null;
}

const BAND_WIDTH_SECONDS = 5;
const MIN_STEADY_DISTANCE = 1000;

export function hrAtPace(workouts: Workout[], now: Date, weeks: number): HRPaceBand[] {
  const cutoff = new Date(now);
  cutoff.setDate(cutoff.getDate() - weeks * 7);
  const midpoint = new Date(cutoff.getTime() + (now.getTime() - cutoff.getTime()) / 2);

  const steady = workouts.filter((w) => {
    if (isIntervalWorkout(w)) return false;
    if (w.distance < MIN_STEADY_DISTANCE) return false;
    if (!w.heart_rate?.average || w.heart_rate.average <= 0) return false;
    const t = parsedDate(w);
    return t >= cutoff && t <= now && pace500mSeconds(w) > 0;
  });

  const bands = new Map<number, { all: number[]; early: number[]; late: number[] }>();
  for (const w of steady) {
    const pace = pace500mSeconds(w);
    const bandStart = Math.floor(pace / BAND_WIDTH_SECONDS) * BAND_WIDTH_SECONDS;
    if (!bands.has(bandStart)) bands.set(bandStart, { all: [], early: [], late: [] });
    const bucket = bands.get(bandStart)!;
    const hr = w.heart_rate!.average!;
    bucket.all.push(hr);
    if (parsedDate(w) < midpoint) bucket.early.push(hr);
    else bucket.late.push(hr);
  }

  const avg = (xs: number[]) =>
    xs.length > 0 ? Math.round(xs.reduce((a, b) => a + b, 0) / xs.length) : null;

  return [...bands.entries()]
    .sort(([a], [b]) => a - b)
    .map(([bandStart, bucket]) => {
      const early = avg(bucket.early);
      const late = avg(bucket.late);
      return {
        band_start_seconds: bandStart,
        band: `${formatSeconds(bandStart)}–${formatSeconds(bandStart + BAND_WIDTH_SECONDS)}`,
        workouts: bucket.all.length,
        avg_hr: avg(bucket.all)!,
        early_avg_hr: early,
        late_avg_hr: late,
        hr_delta: early != null && late != null ? late - early : null,
      };
    });
}
