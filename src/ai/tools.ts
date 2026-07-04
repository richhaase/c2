import type { C2Client } from "../api/client.ts";
import type { Config } from "../config.ts";
import { formatDate } from "../display.ts";
import {
  calendarDay,
  isIntervalWorkout,
  pace500m,
  restSeconds,
  type StrokeData,
  type Workout,
} from "../models.ts";
import { buildWeekSummaries, computeGoalProgress } from "../stats.ts";
import { readStrokeData, writeStrokeData } from "../storage.ts";
import type { ToolDef } from "./client.ts";
import { appendNote } from "./memory.ts";

export interface ToolContext {
  cfg: Config;
  workouts: Workout[];
  api: C2Client;
  now: Date;
}

export type ToolEvent = (message: string) => void;

const MAX_STROKE_SAMPLES = 300;

export function downsampleStrokes(strokes: StrokeData[], max = MAX_STROKE_SAMPLES): StrokeData[] {
  if (strokes.length <= max) return strokes;
  const stride = strokes.length / max;
  const out = Array.from({ length: max }, (_, i) => strokes[Math.floor(i * stride)]!);
  out[out.length - 1] = strokes[strokes.length - 1]!;
  return out;
}

function compactWorkout(w: Workout) {
  return {
    id: w.id,
    date: w.date,
    distance: w.distance,
    time: w.time_formatted,
    pace_500m: pace500m(w),
    spm: w.stroke_rate ?? null,
    hr_avg: w.heart_rate?.average ?? null,
    drag_factor: w.drag_factor ?? null,
    interval: isIntervalWorkout(w),
    rest_seconds: isIntervalWorkout(w) ? Math.round(restSeconds(w)) : null,
    comments: w.comments ?? null,
    has_strokes: w.stroke_data ?? false,
  };
}

export function buildTools(ctx: ToolContext, onEvent: ToolEvent) {
  const defs: ToolDef[] = [
    {
      type: "function",
      function: {
        name: "goal_progress",
        description:
          "Current progress toward the athlete's distance goal: total meters, percent complete, weeks elapsed/remaining, required weekly pace, and recent average weekly pace.",
        parameters: { type: "object", properties: {}, additionalProperties: false },
      },
    },
    {
      type: "function",
      function: {
        name: "weekly_summaries",
        description:
          "Per-week training summaries (volume, average pace, stroke rate, heart rate, session count) for the most recent N weeks.",
        parameters: {
          type: "object",
          properties: {
            weeks: {
              type: "integer",
              description: "Number of recent weeks (default 12).",
              minimum: 1,
            },
          },
          additionalProperties: false,
        },
      },
    },
    {
      type: "function",
      function: {
        name: "list_workouts",
        description:
          "List individual workouts, optionally filtered by date range. Returns compact per-workout metrics. Use this to inspect specific sessions.",
        parameters: {
          type: "object",
          properties: {
            from: { type: "string", description: "Inclusive start date YYYY-MM-DD." },
            to: { type: "string", description: "Inclusive end date YYYY-MM-DD." },
            limit: {
              type: "integer",
              description: "Max workouts, most recent first (default 30).",
              minimum: 1,
            },
          },
          additionalProperties: false,
        },
      },
    },
    {
      type: "function",
      function: {
        name: "get_strokes",
        description:
          "Stroke-by-stroke data for one workout: elapsed time (tenths s), distance (m), pace (tenths s/500m), stroke rate, and heart rate per sample. Use to analyze pacing, splits, drift, and fade within a single piece.",
        parameters: {
          type: "object",
          properties: {
            workout_id: { type: "integer", description: "The workout id." },
          },
          required: ["workout_id"],
          additionalProperties: false,
        },
      },
    },
    {
      type: "function",
      function: {
        name: "save_coach_note",
        description:
          "Persist a durable observation about the athlete (training patterns, goals, constraints, progress) so future coaching sessions retain it. Use for insights worth remembering, not conversational chatter.",
        parameters: {
          type: "object",
          properties: {
            note: { type: "string", description: "The observation to remember." },
          },
          required: ["note"],
          additionalProperties: false,
        },
      },
    },
  ];

  async function dispatch(name: string, rawArgs: string): Promise<string> {
    let args: Record<string, unknown> = {};
    if (rawArgs && rawArgs.trim() !== "") {
      try {
        args = JSON.parse(rawArgs) as Record<string, unknown>;
      } catch {
        return JSON.stringify({ error: `invalid arguments: ${rawArgs}` });
      }
    }

    switch (name) {
      case "goal_progress": {
        onEvent("checking goal progress");
        const goal = computeGoalProgress(ctx.workouts, ctx.cfg, ctx.now);
        return JSON.stringify(goal);
      }
      case "weekly_summaries": {
        const weeks = typeof args.weeks === "number" ? args.weeks : 12;
        onEvent(`summarizing last ${weeks} weeks`);
        const summaries = buildWeekSummaries(ctx.workouts, ctx.now, weeks).map((ws) => ({
          week_start: formatDate(ws.weekStart, "%Y-%m-%d"),
          meters: ws.meters,
          sessions: ws.sessions,
          avg_pace_500m:
            ws.paceCount > 0 ? Math.round((ws.paceSum / ws.paceCount) * 10) / 10 : null,
          avg_spm: ws.spmCount > 0 ? Math.round((ws.spmSum / ws.spmCount) * 10) / 10 : null,
          avg_hr: ws.hrCount > 0 ? Math.round(ws.hrSum / ws.hrCount) : null,
        }));
        return JSON.stringify(summaries);
      }
      case "list_workouts": {
        const from = typeof args.from === "string" ? args.from : "";
        const to = typeof args.to === "string" ? args.to : "";
        const limit = typeof args.limit === "number" ? args.limit : 30;
        onEvent(`listing workouts${from || to ? ` ${from || "…"}→${to || "…"}` : ""}`);
        let filtered = ctx.workouts;
        if (from) filtered = filtered.filter((w) => calendarDay(w) >= from);
        if (to) filtered = filtered.filter((w) => calendarDay(w) <= to);
        const sorted = [...filtered].sort((a, b) => b.date.localeCompare(a.date)).slice(0, limit);
        return JSON.stringify({ count: sorted.length, workouts: sorted.map(compactWorkout) });
      }
      case "get_strokes": {
        const id = typeof args.workout_id === "number" ? args.workout_id : NaN;
        if (Number.isNaN(id)) return JSON.stringify({ error: "workout_id required" });
        let strokes = await readStrokeData(id);
        if (strokes.length === 0) {
          onEvent(`fetching stroke data for ${id} from Concept2`);
          try {
            strokes = await ctx.api.getStrokes(id);
            if (strokes.length > 0) await writeStrokeData(id, strokes);
          } catch (err) {
            return JSON.stringify({ error: `could not fetch strokes: ${(err as Error).message}` });
          }
        } else {
          onEvent(`reading stroke data for ${id}`);
        }
        const sampled = downsampleStrokes(strokes);
        return JSON.stringify({
          workout_id: id,
          sample_count: strokes.length,
          returned_samples: sampled.length,
          downsampled: sampled.length < strokes.length,
          strokes: sampled,
        });
      }
      case "save_coach_note": {
        const note = typeof args.note === "string" ? args.note.trim() : "";
        if (!note) return JSON.stringify({ error: "note required" });
        await appendNote(note, ctx.now);
        onEvent("saved a note to memory");
        return JSON.stringify({ saved: true });
      }
      default:
        return JSON.stringify({ error: `unknown tool: ${name}` });
    }
  }

  return { defs, dispatch };
}
