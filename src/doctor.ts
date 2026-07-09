import { readdir, readFile } from "node:fs/promises";
import { join } from "node:path";
import { isNoteShaped } from "./notes.ts";
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

function badLines(text: string): number[] {
  const bad: number[] = [];
  text.split("\n").forEach((line, i) => {
    if (line.trim() === "") return;
    try {
      JSON.parse(line);
    } catch {
      bad.push(i + 1);
    }
  });
  return bad;
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
    for (const line of badLines(workouts)) {
      issues.push(`workouts.jsonl: line ${line} is not valid JSON`);
    }
  }

  for (const f of await listDir(paths.strokesDir, "strokes/", issues)) {
    if (!f.endsWith(".jsonl")) continue;
    const text = await readOrNull(join(paths.strokesDir, f), `strokes/${f}`, issues);
    if (text == null) continue;
    checkedFiles++;
    for (const line of badLines(text)) {
      issues.push(`strokes/${f}: line ${line} is not valid JSON`);
    }
  }

  for (const f of await listDir(paths.notesDir, "notes/", issues)) {
    if (!f.endsWith(".json")) continue;
    const text = await readOrNull(join(paths.notesDir, f), `notes/${f}`, issues);
    if (text == null) continue;
    checkedFiles++;
    try {
      const parsed = JSON.parse(text) as { id?: string };
      if (!isNoteShaped(parsed)) {
        issues.push(`notes/${f}: malformed note record`);
      } else if (`${parsed.id}.json` !== f) {
        issues.push(`notes/${f}: filename does not match note id ${parsed.id}`);
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
    let prevKey = "";
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
      const key = `${parsed.date} ${parsed.id}`;
      if (prevKey !== "" && key < prevKey) {
        issues.push(`notes/archive/${f}: line ${lineNo} out of (date, id) order`);
      }
      prevKey = key;
      if (archiveIds.has(parsed.id)) {
        issues.push(`notes/archive/${f}: duplicate note id ${parsed.id}`);
      }
      archiveIds.add(parsed.id);
    }
  }

  return { issues, checkedFiles };
}
