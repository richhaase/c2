import { mkdtemp, readFile, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import type { Command } from "commander";
import { splitShape, splitTable } from "../analysis.ts";
import { loadConfig, parseGoalDate } from "../config.ts";
import { rejectForeignStore } from "../data.ts";
import { formatMeters, workoutJSON } from "../display.ts";
import { printJSON } from "../envelope.ts";
import type { Workout } from "../models.ts";
import { calendarDay, pace500m, pace500mSeconds } from "../models.ts";
import { filterNotes, type NoteRecord, readAllNotes } from "../notes.ts";
import type { DataPaths } from "../paths.ts";
import { dataPaths } from "../paths.ts";
import { sessionCount } from "../sessions.ts";
import {
  buildWeekSummaries,
  computeGoalProgress,
  type GoalProgress,
  mondayOf,
  type WeekSummary,
  weekSummaryData,
  workoutsInRange,
} from "../stats.ts";
import { readWorkouts } from "../storage.ts";
import { listNarratives } from "./docs.ts";
import { projectGoal } from "./stats.ts";

const MONTH_NAMES = [
  "Jan",
  "Feb",
  "Mar",
  "Apr",
  "May",
  "Jun",
  "Jul",
  "Aug",
  "Sep",
  "Oct",
  "Nov",
  "Dec",
];

function shortDate(d: Date): string {
  return `${MONTH_NAMES[d.getMonth()]} ${d.getDate()}`;
}

function fullDate(d: Date): string {
  return `${MONTH_NAMES[d.getMonth()]} ${d.getDate()}, ${d.getFullYear()}`;
}

function fmtPace(secs: number): string {
  if (secs === 0) return "-";
  const mins = Math.floor(secs / 60);
  const rem = secs - mins * 60;
  return `${mins}:${rem.toFixed(1).padStart(4, "0")}`;
}

function avgPaceForWorkouts(workouts: Workout[]): number {
  let sum = 0;
  let count = 0;
  for (const w of workouts) {
    const p = pace500mSeconds(w);
    if (p > 0) {
      sum += p;
      count++;
    }
  }
  return count > 0 ? sum / count : 0;
}

function avgHRForWorkouts(workouts: Workout[]): number {
  let sum = 0;
  let count = 0;
  for (const w of workouts) {
    if (w.heart_rate?.average && w.heart_rate.average > 0) {
      sum += w.heart_rate.average;
      count++;
    }
  }
  return count > 0 ? Math.round(sum / count) : 0;
}

function esc(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

function buildStatsCards(
  goal: GoalProgress,
  sessions: number,
  avgPace: number,
  avgHR: number,
): string {
  const paceClass = goal.onPace ? "green" : "red";
  return `<div class="stats-grid">
  <div class="stat-card">
    <div class="label">Total Meters</div>
    <div class="value">${formatMeters(goal.totalMeters)} <span class="unit">m</span></div>
  </div>
  <div class="stat-card">
    <div class="label">Sessions</div>
    <div class="value">${sessions}</div>
  </div>
  <div class="stat-card">
    <div class="label">Avg Pace</div>
    <div class="value">${fmtPace(avgPace)} <span class="unit">/500m</span></div>
  </div>
  <div class="stat-card">
    <div class="label">Avg Heart Rate</div>
    <div class="value">${avgHR > 0 ? avgHR : "-"} <span class="unit">bpm</span></div>
  </div>
  <div class="stat-card">
    <div class="label">Current Weekly Avg</div>
    <div class="value ${paceClass}">${formatMeters(goal.currentAvgPace)} <span class="unit">m/wk</span></div>
  </div>
  <div class="stat-card">
    <div class="label">Required Weekly Pace</div>
    <div class="value blue">${formatMeters(goal.requiredPace)} <span class="unit">m/wk</span></div>
  </div>
</div>`;
}

function fmtShortNum(n: number): string {
  if (n === 0) return "0";
  if (n >= 1_000_000 && n % 1_000_000 === 0) return `${n / 1_000_000}M`;
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1000) return `${Math.round(n / 1000)}K`;
  return String(n);
}

function buildGoalProgress(goal: GoalProgress): string {
  const pct = (goal.progress * 100).toFixed(1);
  const onPacePct = ((goal.weeksElapsed / goal.totalWeeks) * 100).toFixed(1);
  const diff = (goal.progress * 100 - parseFloat(onPacePct)).toFixed(1);
  const diffLabel =
    parseFloat(diff) >= 0
      ? `${diff}% ahead of pace`
      : `${Math.abs(parseFloat(diff)).toFixed(1)}% behind pace`;
  const diffClass = parseFloat(diff) >= 0 ? "green" : "red";

  const q = goal.target / 4;

  return `<div class="section">
  <h2>Goal Progress</h2>
  <div style="display:flex; justify-content:space-between; font-size:13px; margin-bottom:4px;">
    <span class="${diffClass}" style="font-weight:600;">${formatMeters(goal.totalMeters)}m &mdash; ${pct}%</span>
    <span class="muted">${formatMeters(goal.target)}m</span>
  </div>
  <div class="progress-container">
    <div class="progress-fill" style="width: ${pct}%;"></div>
    <div class="progress-marker" style="left: ${onPacePct}%;">
      <div class="progress-marker-label">On Pace (${onPacePct}%)</div>
    </div>
  </div>
  <div class="progress-label-row">
    <span>${fmtShortNum(0)}</span>
    <span>${fmtShortNum(q)}</span>
    <span>${fmtShortNum(q * 2)}</span>
    <span>${fmtShortNum(q * 3)}</span>
    <span>${fmtShortNum(goal.target)}</span>
  </div>
  <div style="margin-top: 12px; font-size: 13px;">
    <span class="${diffClass}">&#9632;</span> Actual &nbsp;&nbsp;
    <span class="green">|</span> On-pace target (week ${goal.weeksElapsed} of ${goal.totalWeeks})
    &mdash; <span class="${diffClass}" style="font-weight:600;">${diffLabel}</span>
  </div>
</div>`;
}

function buildWeeklyVolume(summaries: WeekSummary[], requiredPace: number): string {
  const maxM = Math.max(...summaries.map((w) => w.meters), requiredPace * 1.25);
  const scale = maxM > 0 ? maxM : 1;
  const targetPct = ((requiredPace / scale) * 100).toFixed(1);
  const lastIdx = summaries.length - 1;

  const rows = summaries
    .map((ws, i) => {
      const pct = ((ws.meters / scale) * 100).toFixed(1);
      const barClass = ws.meters >= requiredPace ? "on-pace" : "behind";
      const isLast = i === lastIdx;
      const labelStyle = isLast ? ' style="color:#c9d1d9; font-weight:600;"' : "";
      const nowTag = isLast ? ' <span style="color:#58a6ff; font-size:10px;">(now)</span>' : "";
      return `  <div class="week-row">
    <div class="week-label"${labelStyle}>${shortDate(ws.weekStart)}</div>
    <div class="week-bar-container">
      <div class="week-bar ${barClass}" style="width: ${pct}%;"></div>
      <div class="week-target-line" style="left: ${targetPct}%;"></div>
    </div>
    <div class="week-meta"><span class="meters">${formatMeters(ws.meters)}</span> m &middot; ${ws.sessions} sess${nowTag}</div>
  </div>`;
    })
    .join("\n\n");

  return `<div class="section">
  <h2>Weekly Volume</h2>
  <div class="target-legend">
    <span class="target-legend-line"></span>
    <span>Target: ${formatMeters(requiredPace)} m/wk</span>
  </div>

${rows}
</div>`;
}

function buildWeeklyTrends(summaries: WeekSummary[]): string {
  let bestVolume = 0;
  let bestPace = Infinity;
  for (const ws of summaries) {
    if (ws.meters > bestVolume) bestVolume = ws.meters;
    if (ws.paceCount > 0) {
      const avg = ws.paceSum / ws.paceCount;
      if (avg < bestPace) bestPace = avg;
    }
  }

  const rows = summaries
    .map((ws) => {
      const avgPace = ws.paceCount > 0 ? ws.paceSum / ws.paceCount : 0;
      const avgSPM = ws.spmCount > 0 ? (ws.spmSum / ws.spmCount).toFixed(1) : "-";
      const avgHR = ws.hrCount > 0 ? Math.round(ws.hrSum / ws.hrCount).toString() : "-";

      const volStyle = ws.meters === bestVolume && ws.meters > 0 ? ' style="color:#3fb950;"' : "";
      const paceStyle = avgPace === bestPace && avgPace > 0 ? ' style="color:#3fb950;"' : "";

      return `      <tr>
        <td>${shortDate(ws.weekStart)}</td>
        <td class="r"${volStyle}>${formatMeters(ws.meters)}m</td>
        <td class="r"${paceStyle}>${avgPace > 0 ? fmtPace(avgPace) : "-"}</td>
        <td class="r">${esc(String(avgSPM))}</td>
        <td class="r">${esc(String(avgHR))}</td>
      </tr>`;
    })
    .join("\n");

  const firstPace = summaries.find((w) => w.paceCount > 0);
  const lastPace = [...summaries].reverse().find((w) => w.paceCount > 0);
  let trendNote = "";
  if (firstPace && lastPace && firstPace !== lastPace) {
    const fp = firstPace.paceSum / firstPace.paceCount;
    const lp = lastPace.paceSum / lastPace.paceCount;
    const diff = Math.abs(fp - lp);
    const direction = lp < fp ? "faster" : "slower";
    trendNote = `\n  <div style="margin-top:12px; font-size:12px; color:#8b949e;">
    Pace trending ${direction}: <span class="${lp < fp ? "green" : "red"}">${fmtPace(fp)} &rarr; ${fmtPace(lp)}</span> &mdash; ${Math.round(diff)} seconds ${direction === "faster" ? "improvement" : "decline"} over ${summaries.length} weeks
  </div>`;
  }

  return `<div class="section">
  <h2>Weekly Trends</h2>
  <table>
    <thead>
      <tr>
        <th>Week</th>
        <th class="r">Volume</th>
        <th class="r">Avg Pace /500m</th>
        <th class="r">Avg SPM</th>
        <th class="r">Avg HR</th>
      </tr>
    </thead>
    <tbody>
${rows}
    </tbody>
  </table>${trendNote}
</div>`;
}

function buildRecentWorkouts(workouts: Workout[], count: number): string {
  const sorted = [...workouts].sort((a, b) => b.date.localeCompare(a.date));
  const recent = sorted.slice(0, count).reverse();

  const dayCounts = new Map<string, number>();
  for (const w of recent) {
    const day = calendarDay(w);
    dayCounts.set(day, (dayCounts.get(day) || 0) + 1);
  }

  const rows = recent
    .map((w) => {
      const day = calendarDay(w);
      const d = new Date(w.date.replace(" ", "T"));
      const dateLabel = `${MONTH_NAMES[d.getMonth()]} ${d.getDate()}`;
      const pace = pace500m(w);
      const paceS = pace500mSeconds(w);
      const spm = w.stroke_rate ?? "-";
      const hr = w.heart_rate?.average ?? "-";

      const multiDay = (dayCounts.get(day) || 0) > 1;
      if (!multiDay) {
        return `      <tr>
        <td>${esc(dateLabel)}</td>
        <td class="r">${formatMeters(w.distance)}m</td>
        <td class="r">${esc(pace)}</td>
        <td class="r">${spm}</td>
        <td class="r">${hr}</td>
      </tr>`;
      }

      const isShort = w.distance <= 1500;
      const isHard = paceS > 0 && paceS < 160;
      let annotation = "";
      let rowStyle = "";
      let paceStyle = "";
      let hrStyle = "";

      if (isShort && !isHard) {
        const dayWorkouts = recent
          .filter((r) => calendarDay(r) === day)
          .sort((a, b) => a.date.localeCompare(b.date));
        const idx = dayWorkouts.indexOf(w);
        if (idx === 0) annotation = "warmup";
        else if (idx === dayWorkouts.length - 1) annotation = "cooldown";
        else annotation = "warmup";
        rowStyle = ' style="color:#8b949e;"';
      } else if (isHard) {
        annotation = "hard";
        paceStyle = ' style="color:#3fb950;"';
        if (typeof hr === "number" && hr >= 135) {
          hrStyle = ' style="color:#f85149;"';
        }
      }

      const dateCell = annotation
        ? `${esc(dateLabel)} <span style="font-size:10px;${isHard ? " color:#3fb950;" : ""}">(${annotation})</span>`
        : esc(dateLabel);

      return `      <tr${rowStyle}>
        <td>${dateCell}</td>
        <td class="r">${formatMeters(w.distance)}m</td>
        <td class="r"${paceStyle}>${esc(pace)}</td>
        <td class="r">${spm}</td>
        <td class="r"${hrStyle}>${hr}</td>
      </tr>`;
    })
    .join("\n");

  return `<div class="section">
  <h2>Recent Workouts</h2>
  <table>
    <thead>
      <tr>
        <th>Date</th>
        <th class="r">Distance</th>
        <th class="r">Pace /500m</th>
        <th class="r">SPM</th>
        <th class="r">HR</th>
      </tr>
    </thead>
    <tbody>
${rows}
    </tbody>
  </table>
</div>`;
}

function buildProjection(goal: GoalProgress, workouts: Workout[]): string {
  const projectedAtCurrent = goal.currentAvgPace * goal.remainingWeeks + goal.totalMeters;
  const projectedPct = ((projectedAtCurrent / goal.target) * 100).toFixed(1);
  const shortfall = goal.target - projectedAtCurrent;
  const avgSessionDist =
    workouts.length > 0
      ? Math.round(workouts.reduce((s, w) => s + w.distance, 0) / workouts.length)
      : 5000;
  const sessionsPerWeek =
    avgSessionDist > 0 ? (goal.requiredPace / avgSessionDist).toFixed(1) : "-";
  const increaseNeeded =
    goal.currentAvgPace > 0
      ? (((goal.requiredPace - goal.currentAvgPace) / goal.currentAvgPace) * 100).toFixed(0)
      : "-";

  const currentClass = projectedAtCurrent >= goal.target ? "green" : "red";

  return `<div class="section">
  <h2>Year-End Projection</h2>
  <div class="projection-grid">
    <div class="projection-card">
      <h3 class="${currentClass}">At Current Pace</h3>
      <div class="big-num ${currentClass}">~${formatMeters(Math.round(projectedAtCurrent / 1000) * 1000)}m</div>
      <div class="detail">
        ${formatMeters(goal.currentAvgPace)} m/wk &times; ${goal.remainingWeeks} remaining + ${formatMeters(goal.totalMeters)}<br>
        ${shortfall > 0 ? `${formatMeters(Math.round(shortfall))}m short of goal` : "On track to exceed goal"}<br>
        <span class="${currentClass}" style="font-weight:600;">${projectedPct}% of target</span>
      </div>
    </div>
    <div class="projection-card">
      <h3 class="green">To Hit ${formatMeters(goal.target)}m</h3>
      <div class="big-num green">${formatMeters(goal.requiredPace)} <span style="font-size:16px; font-weight:400;">m/wk</span></div>
      <div class="detail">
        ${formatMeters(goal.remainingMeters)}m remaining over ${goal.remainingWeeks} weeks<br>
        ~${sessionsPerWeek} sessions of ${formatMeters(avgSessionDist)}m per week<br>
        <span class="green" style="font-weight:600;">${Number(increaseNeeded) > 0 ? `+${increaseNeeded}% increase needed` : "Pace is sufficient"}</span>
      </div>
    </div>
  </div>
</div>`;
}

export interface CoachingContent {
  narrative: { date: string; text: string } | null;
  notes: NoteRecord[];
  planExcerpt: string | null;
}

const EMPTY_COACHING: CoachingContent = { narrative: null, notes: [], planExcerpt: null };

const RECENT_NOTE_DAYS = 14;
const MAX_RECENT_NOTES = 20;
const PLAN_EXCERPT_MAX_CHARS = 1500;

async function readIfExists(path: string): Promise<string | null> {
  try {
    return await readFile(path, "utf-8");
  } catch (err: unknown) {
    const code = (err as NodeJS.ErrnoException).code;
    if (code === "ENOENT" || code === "ENOTDIR") return null;
    throw err;
  }
}

export async function gatherCoaching(paths: DataPaths, now: Date): Promise<CoachingContent> {
  const since = new Date(now);
  since.setDate(since.getDate() - RECENT_NOTE_DAYS);
  const sinceKey = `${since.getFullYear()}-${String(since.getMonth() + 1).padStart(2, "0")}-${String(since.getDate()).padStart(2, "0")}`;
  const notes = filterNotes(await readAllNotes(paths), { since: sinceKey }).slice(
    -MAX_RECENT_NOTES,
  );

  let narrative: CoachingContent["narrative"] = null;
  const dates = await listNarratives(paths);
  const latest = dates[dates.length - 1];
  if (latest != null) {
    const text = await readIfExists(paths.narrativeFile(latest));
    if (text != null && text.trim() !== "") narrative = { date: latest, text };
  }

  let planExcerpt: string | null = null;
  const plan = await readIfExists(paths.plan);
  if (plan != null && plan.trim() !== "") {
    const sections = plan.split(/\n(?=## )/);
    let excerpt = sections[0]!.trim();
    if (excerpt.length > PLAN_EXCERPT_MAX_CHARS) {
      excerpt = `${excerpt.slice(0, PLAN_EXCERPT_MAX_CHARS)}…`;
    } else if (sections.length > 1) {
      excerpt = `${excerpt}\n\n_(full plan: \`c2 plan show\`)_`;
    }
    planExcerpt = excerpt;
  }

  return { narrative, notes, planExcerpt };
}

function mdLite(text: string): string {
  const blocks: string[] = [];
  let paragraph: string[] = [];
  let list: string[] = [];

  const flushParagraph = () => {
    if (paragraph.length > 0) {
      blocks.push(`<p>${paragraph.join(" ")}</p>`);
      paragraph = [];
    }
  };
  const flushList = () => {
    if (list.length > 0) {
      blocks.push(`<ul>${list.map((i) => `<li>${i}</li>`).join("")}</ul>`);
      list = [];
    }
  };

  for (const rawLine of text.split("\n")) {
    const line = esc(rawLine.trim());
    if (line === "") {
      flushParagraph();
      flushList();
      continue;
    }
    const heading = /^(#{1,4})\s+(.*)$/.exec(line);
    if (heading != null) {
      flushParagraph();
      flushList();
      const level = heading[1]!.length <= 2 ? "h3" : "h4";
      blocks.push(`<${level}>${heading[2]}</${level}>`);
      continue;
    }
    const item = /^[-*]\s+(.*)$/.exec(line);
    if (item != null) {
      flushParagraph();
      list.push(item[1]!);
      continue;
    }
    flushList();
    paragraph.push(line);
  }
  flushParagraph();
  flushList();
  return blocks.join("\n");
}

function buildNarrativeSection(narrative: { date: string; text: string }): string {
  return `<div class="section">
  <h2>Coach's Report &mdash; ${esc(narrative.date)}</h2>
  <div class="prose">
${mdLite(narrative.text)}
  </div>
</div>`;
}

function buildNotesSection(notes: NoteRecord[]): string {
  const rows = notes
    .map((n) => {
      const workout = n.workout_id != null ? ` &middot; workout ${n.workout_id}` : "";
      return `  <div class="note-row">
    <div class="note-meta">${esc(n.date.slice(0, 10))} &middot; ${esc(n.type)} (${esc(n.author)})${workout}</div>
    <div class="note-body">${esc(n.body)}</div>
  </div>`;
    })
    .join("\n");
  return `<div class="section">
  <h2>Recent Notes</h2>
${rows}
</div>`;
}

function buildPlanSection(excerpt: string): string {
  return `<div class="section">
  <h2>Training Plan</h2>
  <div class="prose">
${mdLite(excerpt)}
  </div>
</div>`;
}

function buildHTML(
  goal: GoalProgress,
  summaries: WeekSummary[],
  allWorkouts: Workout[],
  windowedWorkouts: Workout[],
  recentCount: number,
  coaching: CoachingContent,
): string {
  const sessions = sessionCount(windowedWorkouts);
  const avgPace = avgPaceForWorkouts(windowedWorkouts);
  const avgHR = avgHRForWorkouts(windowedWorkouts);
  const today = new Date();

  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Rowing Progress — ${today.getFullYear()}</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    background: #0d1117;
    color: #c9d1d9;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    line-height: 1.6;
    padding: 24px;
    max-width: 960px;
    margin: 0 auto;
  }
  h1 { color: #f0f6fc; font-size: 28px; font-weight: 700; }
  h2 { color: #f0f6fc; font-size: 20px; font-weight: 600; margin-bottom: 16px; }
  .subtitle { color: #8b949e; font-size: 15px; margin-top: 4px; }
  .date { color: #8b949e; font-size: 13px; margin-top: 2px; }
  .muted { color: #8b949e; }
  .green { color: #3fb950; }
  .red { color: #f85149; }
  .blue { color: #58a6ff; }

  header { margin-bottom: 32px; }

  .stats-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 12px;
    margin-bottom: 32px;
  }
  .stat-card {
    background: #161b22;
    border: 1px solid #30363d;
    border-radius: 8px;
    padding: 16px;
  }
  .stat-card .label {
    color: #8b949e;
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 4px;
  }
  .stat-card .value {
    color: #f0f6fc;
    font-size: 24px;
    font-weight: 700;
  }
  .stat-card .unit {
    color: #8b949e;
    font-size: 13px;
    font-weight: 400;
  }

  .section {
    background: #161b22;
    border: 1px solid #30363d;
    border-radius: 8px;
    padding: 20px;
    margin-bottom: 24px;
  }

  .prose { font-size: 14px; }
  .prose p { margin-bottom: 10px; }
  .prose h3 { color: #f0f6fc; font-size: 15px; font-weight: 600; margin: 14px 0 6px; }
  .prose h4 { color: #c9d1d9; font-size: 13px; font-weight: 600; margin: 12px 0 4px; }
  .prose ul { margin: 0 0 10px 20px; }
  .prose li { margin-bottom: 4px; }

  .note-row { padding: 10px 0; border-bottom: 1px solid #21262d; }
  .note-row:last-child { border-bottom: none; }
  .note-meta { color: #8b949e; font-size: 11px; text-transform: uppercase; letter-spacing: 0.3px; margin-bottom: 3px; }
  .note-body { font-size: 13px; }

  .progress-container {
    position: relative;
    background: #21262d;
    border-radius: 6px;
    height: 32px;
    margin: 16px 0 8px;
    overflow: visible;
  }
  .progress-fill {
    height: 100%;
    border-radius: 6px;
    background: #f85149;
    position: relative;
    z-index: 1;
    min-width: 2px;
  }
  .progress-marker {
    position: absolute;
    top: -6px;
    height: 44px;
    width: 2px;
    background: #3fb950;
    z-index: 2;
  }
  .progress-marker-label {
    position: absolute;
    top: -22px;
    transform: translateX(-50%);
    font-size: 11px;
    color: #3fb950;
    white-space: nowrap;
    font-weight: 600;
  }
  .progress-label-row {
    display: flex;
    justify-content: space-between;
    font-size: 12px;
    color: #8b949e;
    margin-top: 4px;
  }

  .week-row {
    display: flex;
    align-items: center;
    margin-bottom: 8px;
    font-size: 13px;
  }
  .week-label {
    width: 70px;
    flex-shrink: 0;
    color: #8b949e;
    font-size: 12px;
    text-align: right;
    padding-right: 10px;
  }
  .week-bar-container {
    flex: 1;
    position: relative;
    height: 24px;
    background: #21262d;
    border-radius: 4px;
    overflow: visible;
  }
  .week-bar {
    height: 100%;
    border-radius: 4px;
    min-width: 2px;
  }
  .week-bar.on-pace { background: #238636; }
  .week-bar.behind { background: #8b2a2d; }
  .week-meta {
    width: 140px;
    flex-shrink: 0;
    text-align: right;
    font-size: 12px;
    color: #8b949e;
    padding-left: 8px;
  }
  .week-meta .meters { color: #c9d1d9; font-weight: 500; }
  .week-target-line {
    position: absolute;
    top: -2px;
    height: 28px;
    width: 0;
    border-left: 2px dashed #58a6ff;
    z-index: 2;
    opacity: 0.7;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
  }
  thead th {
    text-align: left;
    color: #8b949e;
    font-weight: 600;
    padding: 8px 10px;
    border-bottom: 1px solid #30363d;
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }
  th.r, td.r { text-align: right; }
  tbody td {
    padding: 8px 10px;
    border-bottom: 1px solid #21262d;
    font-variant-numeric: tabular-nums;
  }
  tbody tr:last-child td { border-bottom: none; }
  tbody tr:hover { background: #1c2128; }

  .projection-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }
  .projection-card {
    background: #21262d;
    border-radius: 6px;
    padding: 16px;
  }
  .projection-card h3 {
    font-size: 14px;
    font-weight: 600;
    margin-bottom: 8px;
  }
  .projection-card .big-num {
    font-size: 28px;
    font-weight: 700;
    margin-bottom: 4px;
  }
  .projection-card .detail {
    font-size: 12px;
    color: #8b949e;
    line-height: 1.8;
  }

  .target-legend {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    color: #8b949e;
    margin-bottom: 12px;
    justify-content: flex-end;
    padding-right: 140px;
  }
  .target-legend-line {
    width: 16px;
    border-top: 2px dashed #58a6ff;
  }

  @media (max-width: 640px) {
    .stats-grid { grid-template-columns: repeat(2, 1fr); }
    .projection-grid { grid-template-columns: 1fr; }
    body { padding: 16px; }
  }
</style>
</head>
<body>

<header>
  <h1>Rowing Progress</h1>
  <div class="subtitle">${today.getFullYear()} Season &mdash; ${formatMeters(goal.target)}m Goal</div>
  <div class="date">${fullDate(today)}</div>
</header>

${buildStatsCards(goal, sessions, avgPace, avgHR)}

${buildGoalProgress(goal)}

${coaching.narrative != null ? buildNarrativeSection(coaching.narrative) : ""}

${buildWeeklyVolume(summaries, goal.requiredPace)}

${buildWeeklyTrends(summaries)}

${buildRecentWorkouts(allWorkouts, recentCount)}

${coaching.notes.length > 0 ? buildNotesSection(coaching.notes) : ""}

${buildProjection(goal, allWorkouts)}

${coaching.planExcerpt != null ? buildPlanSection(coaching.planExcerpt) : ""}

<div style="text-align: center; color: #484f58; font-size: 12px; margin-top: 32px; padding-bottom: 16px;">
  Generated by c2 &middot; Data from Concept2 Logbook &middot; ${fullDate(today)}
</div>

</body>
</html>`;
}

export function registerReport(program: Command): void {
  program
    .command("report")
    .description("Generate HTML progress report and open in browser")
    .option("-o, --output <file>", "save to a specific file instead of a temp file")
    .option("-w, --weeks <n>", "weeks of history to show", "12")
    .option("--data", "emit the report content as JSON instead of HTML")
    .option("--no-open", "don't open in browser")
    .action(async (opts: { output?: string; weeks: string; data?: boolean; open: boolean }) => {
      const cfg = await loadConfig();
      if (!cfg.goal.start_date || !cfg.goal.end_date) {
        console.error("Goal dates not configured. Run `c2 setup` to set start and end dates.");
        process.exit(1);
      }
      const paths = dataPaths(cfg);
      const foreign = await rejectForeignStore(paths);
      if (foreign != null) {
        console.error(foreign);
        process.exit(1);
      }
      const workouts = await readWorkouts(paths);

      if (workouts.length === 0) {
        console.log("No workouts found. Run `c2 sync` first.");
        return;
      }

      const weeks = parseInt(opts.weeks, 10);
      if (Number.isNaN(weeks) || weeks < 1) {
        console.error("Error: --weeks must be a positive integer.");
        process.exit(1);
      }
      const now = new Date();
      const goal = computeGoalProgress(workouts, cfg, now);
      const summaries = buildWeekSummaries(workouts, now, weeks);
      const thisMonday = mondayOf(now);
      const cutoff = new Date(thisMonday);
      cutoff.setDate(cutoff.getDate() - (weeks - 1) * 7);
      const windowedWorkouts = workoutsInRange(workouts, cutoff, now);
      const coaching = await gatherCoaching(paths, now);

      if (opts.data) {
        const sorted = [...workouts].sort((a, b) => b.date.localeCompare(a.date));
        const latest = sorted[0]!;
        const latestSplits = splitTable(latest);
        printJSON("c2.report.v1", {
          period: { weeks, to: calendarDay(latest) },
          summary: {
            total_meters: goal.totalMeters,
            sessions: sessionCount(windowedWorkouts),
            avg_pace_500m_seconds:
              Math.round(avgPaceForWorkouts(windowedWorkouts) * 10) / 10 || null,
            avg_hr: avgHRForWorkouts(windowedWorkouts) || null,
          },
          goal,
          projection: projectGoal(goal, parseGoalDate(cfg.goal.end_date), now),
          weekly: summaries.map(weekSummaryData),
          recent_workouts: sorted.slice(0, 10).map(workoutJSON),
          latest_splits:
            latestSplits.length > 0
              ? {
                  workout_id: latest.id,
                  date: latest.date,
                  split_shape: splitShape(latestSplits),
                  splits: latestSplits,
                }
              : null,
          narrative: coaching.narrative,
          notes: coaching.notes,
          plan_excerpt: coaching.planExcerpt,
        });
        return;
      }

      const html = buildHTML(goal, summaries, workouts, windowedWorkouts, 10, coaching);

      let outPath: string;
      if (opts.output) {
        outPath = resolve(opts.output);
      } else {
        const dir = await mkdtemp(join(tmpdir(), "c2-report-"));
        outPath = join(dir, "report.html");
      }
      await writeFile(outPath, html, "utf-8");

      if (opts.open) {
        const { spawn } = await import("node:child_process");
        const cmd =
          process.platform === "darwin"
            ? "open"
            : process.platform === "win32"
              ? "start"
              : "xdg-open";
        const child = spawn(cmd, [outPath], { stdio: "ignore", detached: true });
        child.on("error", (err) => {
          console.error(`Could not open report: ${err.message}`);
        });
        child.unref();
      } else {
        console.log(`Report written to: ${outPath}`);
      }
    });
}
