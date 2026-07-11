import { appendFile, readFile, stat, writeFile } from "node:fs/promises";
import type { StrokeData, Workout } from "./models.ts";
import type { DataPaths } from "./paths.ts";

export interface StoreMeta {
  schema_version: number;
  created: string;
  last_sync?: string;
}

export const SCHEMA_VERSION = 1;

export async function readWorkouts(paths: DataPaths): Promise<Workout[]> {
  try {
    const text = await readFile(paths.workouts, "utf-8");
    return text
      .split("\n")
      .filter((line) => line.trim() !== "")
      .map((line) => JSON.parse(line) as Workout);
  } catch (err: unknown) {
    const code = (err as NodeJS.ErrnoException).code;
    if (code === "ENOENT" || code === "ENOTDIR") return [];
    throw err;
  }
}

export async function appendWorkouts(paths: DataPaths, newWorkouts: Workout[]): Promise<number> {
  const existing = await readWorkouts(paths);
  const seen = new Set(existing.map((w) => w.id));

  const toWrite = newWorkouts.filter((w) => !seen.has(w.id));
  if (toWrite.length === 0) return 0;

  const lines = `${toWrite.map((w) => JSON.stringify(w)).join("\n")}\n`;
  await appendFile(paths.workouts, lines, "utf-8");
  return toWrite.length;
}

export async function workoutCount(paths: DataPaths): Promise<number> {
  try {
    const text = await readFile(paths.workouts, "utf-8");
    return text.split("\n").filter((line) => line.trim() !== "").length;
  } catch (err: unknown) {
    const code = (err as NodeJS.ErrnoException).code;
    if (code === "ENOENT" || code === "ENOTDIR") return 0;
    throw err;
  }
}

export async function hasStrokeData(paths: DataPaths, workoutId: number): Promise<boolean> {
  try {
    await stat(paths.strokeFile(workoutId));
    return true;
  } catch (err: unknown) {
    const code = (err as NodeJS.ErrnoException).code;
    if (code === "ENOENT" || code === "ENOTDIR") return false;
    throw err;
  }
}

export async function writeStrokeData(
  paths: DataPaths,
  workoutId: number,
  strokes: StrokeData[],
): Promise<void> {
  const lines = `${strokes.map((s) => JSON.stringify(s)).join("\n")}\n`;
  await writeFile(paths.strokeFile(workoutId), lines, "utf-8");
}

export async function readStrokeData(paths: DataPaths, workoutId: number): Promise<StrokeData[]> {
  let text: string;
  try {
    text = await readFile(paths.strokeFile(workoutId), "utf-8");
  } catch (err: unknown) {
    const code = (err as NodeJS.ErrnoException).code;
    if (code === "ENOENT" || code === "ENOTDIR") return [];
    throw err;
  }
  const strokes: StrokeData[] = [];
  for (const line of text.split("\n")) {
    if (line.trim() === "") continue;
    try {
      const parsed = JSON.parse(line) as unknown;
      if (parsed != null && typeof parsed === "object" && !Array.isArray(parsed)) {
        strokes.push(parsed as StrokeData);
      }
    } catch {}
  }
  return strokes;
}

export async function readMeta(paths: DataPaths): Promise<StoreMeta | null> {
  let text: string;
  try {
    text = await readFile(paths.meta, "utf-8");
  } catch (err: unknown) {
    const code = (err as NodeJS.ErrnoException).code;
    if (code !== "ENOENT" && code !== "ENOTDIR") {
      console.error(`Warning: ${paths.meta} is unreadable and will be ignored.`);
    }
    return null;
  }
  try {
    return JSON.parse(text) as StoreMeta;
  } catch {
    console.error(`Warning: ${paths.meta} is corrupt and will be ignored.`);
    return null;
  }
}

export async function writeMeta(paths: DataPaths, meta: StoreMeta): Promise<void> {
  await writeFile(paths.meta, `${JSON.stringify(meta, null, 2)}\n`, "utf-8");
}
