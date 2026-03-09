import type { Workout } from "./models.ts";
import { pace500m, parsedDate } from "./models.ts";

export function formatMeters(m: number): string {
  return m.toLocaleString("en-US");
}

export function formatPercent(ratio: number): string {
  return `${(ratio * 100).toFixed(1)}%`;
}

export function formatMetersPerWeek(m: number): string {
  return `${formatMeters(m)}m/week`;
}

export function formatDate(d: Date, fmt: string): string {
  // Go-style reference time: "01/02" means month/day
  if (fmt === "01/02" || fmt === "%m/%d") {
    const mm = String(d.getMonth() + 1).padStart(2, "0");
    const dd = String(d.getDate()).padStart(2, "0");
    return `${mm}/${dd}`;
  }
  if (fmt === "%Y-%m-%d") {
    const y = d.getFullYear();
    const mm = String(d.getMonth() + 1).padStart(2, "0");
    const dd = String(d.getDate()).padStart(2, "0");
    return `${y}-${mm}-${dd}`;
  }
  // Fallback: MM/DD
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${mm}/${dd}`;
}

export function formatWorkoutLine(w: Workout, dateFormat: string): string {
  const d = parsedDate(w);
  const dateStr = formatDate(d, dateFormat);
  const distance = `${formatMeters(w.distance)}m`;
  const pace = pace500m(w);
  const spm = w.stroke_rate ? `${w.stroke_rate}spm` : "-";
  const hr = w.heart_rate?.average && w.heart_rate.average > 0 ? `${w.heart_rate.average}bpm` : "-";
  const df = w.drag_factor ? `${w.drag_factor}df` : "-";

  return `${dateStr}  ${distance.padStart(7)}  ${w.time_formatted.padStart(8)}  ${pace.padStart(7)}/500m  ${spm.padStart(5)}  ${hr.padStart(6)}  ${df.padStart(4)}`;
}

export function sparkBar(value: number, max: number): string {
  if (max === 0) return "";
  const BAR_WIDTH = 20;
  const filled = Math.round((value / max) * BAR_WIDTH);
  return "\u2588".repeat(filled) + "\u2591".repeat(BAR_WIDTH - filled);
}

const TREND_THRESHOLD = 0.02;

export function trendArrow(prev: number, curr: number): string {
  if (prev === 0) return " ";
  const diff = (curr - prev) / prev;
  if (diff > TREND_THRESHOLD) return "\u2191";
  if (diff < -TREND_THRESHOLD) return "\u2193";
  return "\u2192";
}

export function paceArrow(prev: number, curr: number): string {
  if (prev === 0) return " ";
  return trendArrow(curr, prev); // reversed: lower pace is better
}
