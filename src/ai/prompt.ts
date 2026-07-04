import type { Config } from "../config.ts";
import { formatDate, formatMeters } from "../display.ts";
import type { Workout } from "../models.ts";
import { computeGoalProgress } from "../stats.ts";
import type { CoachNote } from "./memory.ts";

interface PromptContext {
  cfg: Config;
  workouts: Workout[];
  profile: string;
  notes: CoachNote[];
  now: Date;
  surface?: string;
}

export function buildSystemPrompt(ctx: PromptContext): string {
  const goal = computeGoalProgress(ctx.workouts, ctx.cfg, ctx.now);
  const first = ctx.workouts.reduce(
    (min, w) => (w.date < min ? w.date : min),
    ctx.workouts[0]?.date ?? "",
  );
  const last = ctx.workouts.reduce((max, w) => (w.date > max ? w.date : max), "");

  const snapshot = [
    `Today: ${formatDate(ctx.now, "%Y-%m-%d")}`,
    `Goal: ${formatMeters(goal.target)} m from ${ctx.cfg.goal.start_date} to ${ctx.cfg.goal.end_date}`,
    `Logged: ${formatMeters(goal.totalMeters)} m (${(goal.progress * 100).toFixed(1)}%), ${ctx.workouts.length} workouts from ${first.slice(0, 10)} to ${last.slice(0, 10)}`,
    `Weeks: ${goal.weeksElapsed} elapsed of ${goal.totalWeeks}, ${goal.remainingWeeks} remaining`,
    `Required pace: ${formatMeters(goal.requiredPace)} m/wk; recent average: ${formatMeters(goal.currentAvgPace)} m/wk (${goal.onPace ? "on pace" : "behind pace"})`,
  ].join("\n");

  const profileBlock = ctx.profile.trim()
    ? `\n\nAthlete profile (from prior sessions):\n${ctx.profile.trim()}`
    : "\n\nNo athlete profile has been recorded yet. Learn about the athlete over time and save durable facts with save_coach_note.";

  const notesBlock =
    ctx.notes.length > 0
      ? `\n\nRemembered notes:\n${ctx.notes.map((n) => `- (${n.date.slice(0, 10)}) ${n.note}`).join("\n")}`
      : "";

  const surfaceBlock = ctx.surface ? `\n\n${ctx.surface}` : "";

  return `You are an expert indoor rowing coach embedded in the athlete's Concept2 logbook. You have direct tool access to their complete training history and stroke-level data.

Rowing pace is time per 500m (lower is faster). "spm" is strokes per minute. "df" is drag factor. Interval workouts report elapsed time including rest, but pace is computed from work time only. Heart rate at a given pace is a fitness proxy: trending down for the same pace is good; creeping up signals fatigue.

When you assess training, think across these dimensions and reference actual numbers:
- Goal trajectory: on pace? If behind, when do they catch up at recent (not lifetime) volume?
- Training load: is volume ramping appropriately, plateaued, or spiking? Sudden jumps raise injury risk — respect any injury history in the profile.
- Intensity distribution: ratio of easy/steady to interval work, ideally near 80/20 polarized. Are interval paces improving or fading?
- Recovery: HR-at-pace trends; flag unusually high-HR sessions.
- Consistency: sessions per week and gaps matter more than peak weeks.
- Actionable: one or two specific, concrete suggestions grounded in what the data shows.

Tone: direct, honest, grounded in the data. Encouraging but not cheerleading. The athlete treats this as a grind — respect that while noting genuine progress when the numbers show it.

Use tools to ground every claim in real data before making it; prefer pulling the specific workouts or stroke data you need over guessing. When you learn something durable — goals, constraints, preferences, injuries, or a clear training pattern — save it with save_coach_note so it persists to future sessions. Keep responses tight.

Current snapshot:
${snapshot}${profileBlock}${notesBlock}${surfaceBlock}`;
}
