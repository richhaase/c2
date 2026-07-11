import { readdir, readFile } from "node:fs/promises";
import { join } from "node:path";
import { isNoteShaped, serializeNote } from "./notes.ts";
import type { DataPaths } from "./paths.ts";

export interface DoctorReport {
  issues: string[];
  checkedFiles: number;
}

async function readOrNull(path: string, label: string, issues: string[]): Promise<string | null> {
  try {
    return await readFile(path, "utf-8");
  } catch (err: unknown) {
    const code = (err as NodeJS.ErrnoException).code;
    if (code !== "ENOENT") {
      issues.push(`${label}: unreadable (${code ?? (err as Error).message})`);
    }
    return null;
  }
}

async function listDir(dir: string, label: string, issues: string[]): Promise<string[]> {
  try {
    return await readdir(dir);
  } catch (err: unknown) {
    const code = (err as NodeJS.ErrnoException).code;
    if (code !== "ENOENT") {
      issues.push(`${label}: unreadable (${code ?? (err as Error).message})`);
    }
    return [];
  }
}

const STROKE_FIELDS = ["t", "d", "p", "spm", "hr"] as const;

function isStrokeShaped(parsed: unknown): boolean {
  if (parsed == null || typeof parsed !== "object" || Array.isArray(parsed)) return false;
  const row = parsed as Record<string, unknown>;
  return STROKE_FIELDS.every((k) => row[k] === undefined || typeof row[k] === "number");
}

export async function runDoctor(paths: DataPaths): Promise<DoctorReport> {
  const issues: string[] = [];
  let checkedFiles = 0;

  const meta = await readOrNull(paths.meta, "meta.json", issues);
  if (meta != null) {
    checkedFiles++;
    try {
      const parsed = JSON.parse(meta);
      if (typeof parsed?.schema_version !== "number") {
        issues.push("meta.json: missing numeric schema_version");
      }
    } catch {
      issues.push("meta.json: not valid JSON");
    }
  }

  const workouts = await readOrNull(paths.workouts, "workouts.jsonl", issues);
  if (workouts != null) {
    checkedFiles++;
    let lineNo = 0;
    for (const line of workouts.split("\n")) {
      lineNo++;
      if (line.trim() === "") continue;
      let parsed: unknown;
      try {
        parsed = JSON.parse(line);
      } catch {
        issues.push(`workouts.jsonl: line ${lineNo} is not valid JSON`);
        continue;
      }
      const w = parsed as { id?: unknown; date?: unknown; distance?: unknown; time?: unknown };
      if (
        parsed == null ||
        typeof parsed !== "object" ||
        typeof w.id !== "number" ||
        typeof w.date !== "string" ||
        typeof w.distance !== "number" ||
        typeof w.time !== "number"
      ) {
        issues.push(`workouts.jsonl: line ${lineNo} malformed workout record`);
      }
    }
  }

  for (const f of await listDir(paths.strokesDir, "strokes/", issues)) {
    if (!f.endsWith(".jsonl")) continue;
    const text = await readOrNull(join(paths.strokesDir, f), `strokes/${f}`, issues);
    if (text == null) continue;
    checkedFiles++;
    let lineNo = 0;
    for (const line of text.split("\n")) {
      lineNo++;
      if (line.trim() === "") continue;
      let parsed: unknown;
      try {
        parsed = JSON.parse(line);
      } catch {
        issues.push(`strokes/${f}: line ${lineNo} is not valid JSON`);
        continue;
      }
      if (!isStrokeShaped(parsed)) {
        issues.push(`strokes/${f}: line ${lineNo} malformed stroke record`);
      }
    }
  }

  const looseContent = new Map<string, { content: string; file: string }>();
  const divergentIds = new Set<string>();
  for (const f of await listDir(paths.notesDir, "notes/", issues)) {
    if (!f.endsWith(".json")) continue;
    const text = await readOrNull(join(paths.notesDir, f), `notes/${f}`, issues);
    if (text == null) continue;
    checkedFiles++;
    try {
      const parsed = JSON.parse(text) as unknown;
      if (!isNoteShaped(parsed)) {
        issues.push(`notes/${f}: malformed note record`);
      } else {
        const content = serializeNote(parsed);
        const prior = looseContent.get(parsed.id);
        if (prior != null && prior.content !== content && !divergentIds.has(parsed.id)) {
          divergentIds.add(parsed.id);
          issues.push(
            `notes: divergent copies of note ${parsed.id} (${prior.file}, ${f}); reconcile before they compact`,
          );
        }
        looseContent.set(parsed.id, { content, file: f });
      }
    } catch {
      issues.push(`notes/${f}: not valid JSON`);
    }
  }

  const archiveIds = new Set<string>();
  for (const f of await listDir(paths.archiveDir, "notes/archive/", issues)) {
    if (!f.endsWith(".jsonl")) continue;
    const text = await readOrNull(join(paths.archiveDir, f), `notes/archive/${f}`, issues);
    if (text == null) continue;
    checkedFiles++;
    let prevMs = Number.NEGATIVE_INFINITY;
    let prevId = "";
    let lineNo = 0;
    for (const line of text.split("\n")) {
      lineNo++;
      if (line.trim() === "") continue;
      let parsed: unknown;
      try {
        parsed = JSON.parse(line);
      } catch {
        issues.push(`notes/archive/${f}: line ${lineNo} is not valid JSON`);
        continue;
      }
      if (!isNoteShaped(parsed)) {
        issues.push(`notes/archive/${f}: line ${lineNo} malformed note record`);
        continue;
      }
      const ms = new Date(parsed.date).getTime();
      if (ms < prevMs || (ms === prevMs && parsed.id < prevId)) {
        issues.push(`notes/archive/${f}: line ${lineNo} out of (date, id) order`);
      }
      prevMs = ms;
      prevId = parsed.id;
      if (archiveIds.has(parsed.id)) {
        issues.push(`notes/archive/${f}: duplicate note id ${parsed.id}`);
      }
      archiveIds.add(parsed.id);
    }
  }

  return { issues, checkedFiles };
}
