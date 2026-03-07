import type { Command } from "commander";
import { writeFile } from "node:fs/promises";
import { resolve } from "node:path";
import { loadConfig } from "../config.ts";
import { readWorkouts } from "../storage.ts";
import { formatMeters, formatPercent } from "../display.ts";
import { pace500mSeconds, pace500m, calendarDay } from "../models.ts";
import type { Workout } from "../models.ts";
import { sessionCount } from "../sessions.ts";
import {
  buildWeekSummaries,
  computeGoalProgress,
  type WeekSummary,
  type GoalProgress,
} from "../stats.ts";

const MONTH_NAMES = [
  "Jan", "Feb", "Mar", "Apr", "May", "Jun",
  "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
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

function buildGoalProgress(goal: GoalProgress): string {
  const pct = (goal.progress * 100).toFixed(1);
  const onPacePct = ((goal.weeksElapsed / goal.totalWeeks) * 100).toFixed(1);
  const diff = (goal.progress * 100 - parseFloat(onPacePct)).toFixed(1);
  const diffLabel =
    parseFloat(diff) >= 0
      ? `${diff}% ahead of pace`
      : `${Math.abs(parseFloat(diff)).toFixed(1)}% behind pace`;
  const diffClass = parseFloat(diff) >= 0 ? "green" : "red";

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
    <span>0</span>
    <span>250K</span>
    <span>500K</span>
    <span>750K</span>
    <span>1M</span>
  </div>
  <div style="margin-top: 12px; font-size: 13px;">
    <span class="${diffClass}">&#9632;</span> Actual &nbsp;&nbsp;
    <span class="green">|</span> On-pace target (week ${goal.weeksElapsed} of ${goal.totalWeeks})
    &mdash; <span class="${diffClass}" style="font-weight:600;">${diffLabel}</span>
  </div>
</div>`;
}

function buildWeeklyVolume(
  summaries: WeekSummary[],
  requiredPace: number,
): string {
  const maxM = Math.max(...summaries.map((w) => w.meters), requiredPace * 1.25);
  const scale = maxM > 0 ? maxM : 1;
  const targetPct = ((requiredPace / scale) * 100).toFixed(1);
  const lastIdx = summaries.length - 1;

  const rows = summaries
    .map((ws, i) => {
      const pct = ((ws.meters / scale) * 100).toFixed(1);
      const barClass = ws.meters >= requiredPace ? "on-pace" : "behind";
      const isLast = i === lastIdx;
      const labelStyle = isLast
        ? ' style="color:#c9d1d9; font-weight:600;"'
        : "";
      const nowTag = isLast
        ? ' <span style="color:#58a6ff; font-size:10px;">(now)</span>'
        : "";
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
  // Find best volume week and best pace week for highlighting
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
      const avgPace =
        ws.paceCount > 0 ? ws.paceSum / ws.paceCount : 0;
      const avgSPM =
        ws.spmCount > 0 ? (ws.spmSum / ws.spmCount).toFixed(1) : "-";
      const avgHR =
        ws.hrCount > 0 ? Math.round(ws.hrSum / ws.hrCount).toString() : "-";

      const volStyle =
        ws.meters === bestVolume && ws.meters > 0
          ? ' style="color:#3fb950;"'
          : "";
      const paceStyle =
        avgPace === bestPace && avgPace > 0
          ? ' style="color:#3fb950;"'
          : "";

      return `      <tr>
        <td>${shortDate(ws.weekStart)}</td>
        <td class="r"${volStyle}>${formatMeters(ws.meters)}m</td>
        <td class="r"${paceStyle}>${avgPace > 0 ? fmtPace(avgPace) : "-"}</td>
        <td class="r">${esc(String(avgSPM))}</td>
        <td class="r">${esc(String(avgHR))}</td>
      </tr>`;
    })
    .join("\n");

  // Pace trend summary
  const firstPace =
    summaries.find((w) => w.paceCount > 0);
  const lastPace = [...summaries]
    .reverse()
    .find((w) => w.paceCount > 0);
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

  // Detect same-day groups for session-aware formatting
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

      // If multiple workouts on same day, annotate
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

      // Session-aware: short warmup/cooldown get muted style
      const isShort = w.distance <= 1500;
      const isHard = paceS > 0 && paceS < 160; // sub-2:40 is hard
      let annotation = "";
      let rowStyle = "";
      let paceStyle = "";
      let hrStyle = "";

      if (isShort && !isHard) {
        // Likely warmup or cooldown — check position
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

function buildProjection(goal: GoalProgress): string {
  const projectedAtCurrent =
    goal.currentAvgPace * goal.remainingWeeks + goal.totalMeters;
  const projectedPct = ((projectedAtCurrent / goal.target) * 100).toFixed(1);
  const shortfall = goal.target - projectedAtCurrent;
  const sessionsPerWeek =
    goal.requiredPace > 0 ? (goal.requiredPace / 5500).toFixed(1) : "-";
  const increaseNeeded =
    goal.currentAvgPace > 0
      ? (
          ((goal.requiredPace - goal.currentAvgPace) / goal.currentAvgPace) *
          100
        ).toFixed(0)
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
        ~${sessionsPerWeek} sessions of 5,500m per week<br>
        <span class="green" style="font-weight:600;">${Number(increaseNeeded) > 0 ? `+${increaseNeeded}% increase needed` : "Pace is sufficient"}</span>
      </div>
    </div>
  </div>
  <div style="margin-top: 16px; padding: 12px; background: #21262d; border-radius: 6px; font-size: 13px; line-height: 1.8;">
    <strong style="color: #f0f6fc;">The path forward:</strong>
    4 sessions/week at 5,500m = 22,000m/week. That covers the gap with margin.
  </div>
</div>`;
}

function buildHTML(
  goal: GoalProgress,
  summaries: WeekSummary[],
  allWorkouts: Workout[],
  recentCount: number,
): string {
  const sessions = sessionCount(allWorkouts);
  const avgPace = avgPaceForWorkouts(allWorkouts);
  const avgHR = avgHRForWorkouts(allWorkouts);
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

${buildWeeklyVolume(summaries, goal.requiredPace)}

${buildWeeklyTrends(summaries)}

${buildRecentWorkouts(allWorkouts, recentCount)}

${buildProjection(goal)}

<div style="text-align: center; color: #484f58; font-size: 12px; margin-top: 32px; padding-bottom: 16px;">
  Generated by c2cli &middot; Data from Concept2 Logbook &middot; ${fullDate(today)}
</div>

</body>
</html>`;
}

export function registerReport(program: Command): void {
  program
    .command("report")
    .description("Generate HTML progress report")
    .option("-o, --output <file>", "output file path", "report.html")
    .option("-w, --weeks <n>", "weeks of history to show", "12")
    .option("--open", "open report in default browser")
    .action(
      async (opts: { output: string; weeks: string; open?: boolean }) => {
        const cfg = await loadConfig();
        const workouts = await readWorkouts();

        if (workouts.length === 0) {
          console.log("No workouts found. Run `c2 sync` first.");
          return;
        }

        const goal = computeGoalProgress(workouts, cfg);
        const weeks = parseInt(opts.weeks, 10);
        const summaries = buildWeekSummaries(workouts, new Date(), weeks);
        const html = buildHTML(goal, summaries, workouts, 10);

        const outPath = resolve(opts.output);
        await writeFile(outPath, html, "utf-8");
        console.log(`Report written to: ${outPath}`);

        if (opts.open) {
          const { exec } = await import("node:child_process");
          exec(`open "${outPath}"`);
        }
      },
    );
}
