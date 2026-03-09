import { appendFile, readFile, stat, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { dataDir } from "./config.ts";
import type { StrokeData, Workout } from "./models.ts";

function workoutsPath(): string {
  return join(dataDir(), "workouts.jsonl");
}

function strokesPath(workoutId: number): string {
  return join(dataDir(), "strokes", `${workoutId}.jsonl`);
}

export async function readWorkouts(): Promise<Workout[]> {
  try {
    const text = await readFile(workoutsPath(), "utf-8");
    return text
      .split("\n")
      .filter((line) => line.trim() !== "")
      .map((line) => JSON.parse(line) as Workout);
  } catch (err: unknown) {
    if ((err as NodeJS.ErrnoException).code === "ENOENT") return [];
    throw err;
  }
}

export async function appendWorkouts(newWorkouts: Workout[]): Promise<number> {
  const existing = await readWorkouts();
  const seen = new Set(existing.map((w) => w.id));

  const toWrite = newWorkouts.filter((w) => !seen.has(w.id));
  if (toWrite.length === 0) return 0;

  const lines = `${toWrite.map((w) => JSON.stringify(w)).join("\n")}\n`;
  await appendFile(workoutsPath(), lines, "utf-8");
  return toWrite.length;
}

export async function workoutCount(): Promise<number> {
  try {
    const text = await readFile(workoutsPath(), "utf-8");
    return text.split("\n").filter((line) => line.trim() !== "").length;
  } catch (err: unknown) {
    if ((err as NodeJS.ErrnoException).code === "ENOENT") return 0;
    throw err;
  }
}

export async function hasStrokeData(workoutId: number): Promise<boolean> {
  try {
    await stat(strokesPath(workoutId));
    return true;
  } catch (err: unknown) {
    if ((err as NodeJS.ErrnoException).code === "ENOENT") return false;
    throw err;
  }
}

export async function writeStrokeData(workoutId: number, strokes: StrokeData[]): Promise<void> {
  const lines = `${strokes.map((s) => JSON.stringify(s)).join("\n")}\n`;
  await writeFile(strokesPath(workoutId), lines, "utf-8");
}

export async function readStrokeData(workoutId: number): Promise<StrokeData[]> {
  try {
    const text = await readFile(strokesPath(workoutId), "utf-8");
    return text
      .split("\n")
      .filter((line) => line.trim() !== "")
      .map((line) => JSON.parse(line) as StrokeData);
  } catch (err: unknown) {
    if ((err as NodeJS.ErrnoException).code === "ENOENT") return [];
    throw err;
  }
}
