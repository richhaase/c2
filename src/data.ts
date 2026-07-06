import { cp, mkdir, readdir, rm, stat, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { calendarDay } from "./models.ts";
import type { DataPaths } from "./paths.ts";
import { readMeta, readWorkouts, SCHEMA_VERSION, writeMeta } from "./storage.ts";

export type DirState = "missing" | "store" | "empty" | "foreign";

export interface DirInspection {
  path: string;
  state: DirState;
  writable: boolean;
}

export interface StoreSummary {
  workouts: number;
  firstDate: string;
  lastDate: string;
  strokeFiles: number;
  notes: number;
  schemaVersion: number | null;
  lastSync: string | null;
}

export async function inspectDataDir(paths: DataPaths): Promise<DirInspection> {
  let state: DirState;
  try {
    const s = await stat(paths.root);
    if (!s.isDirectory()) {
      return { path: paths.root, state: "foreign", writable: false };
    }
    const meta = await readMeta(paths);
    if (meta != null) {
      state = "store";
    } else {
      const entries = (await readdir(paths.root)).filter((e) => !e.startsWith("."));
      const legacyStore =
        entries.length > 0 &&
        entries.every((e) => e === "workouts.jsonl" || e === "strokes" || e === "notes");
      if (entries.length === 0) state = "empty";
      else if (legacyStore) state = "store";
      else state = "foreign";
    }
  } catch (err: unknown) {
    if ((err as NodeJS.ErrnoException).code !== "ENOENT") throw err;
    return { path: paths.root, state: "missing", writable: await parentWritable(paths.root) };
  }
  return { path: paths.root, state, writable: await probeWritable(paths.root) };
}

async function parentWritable(path: string): Promise<boolean> {
  const parent = join(path, "..");
  try {
    await stat(parent);
    return probeWritable(parent);
  } catch {
    return false;
  }
}

async function probeWritable(dir: string): Promise<boolean> {
  const probe = join(dir, `.c2-probe-${process.pid}`);
  try {
    await writeFile(probe, "", "utf-8");
    await rm(probe);
    return true;
  } catch {
    return false;
  }
}

export async function initStore(paths: DataPaths, now: Date): Promise<void> {
  await mkdir(paths.strokesDir, { recursive: true });
  await mkdir(paths.archiveDir, { recursive: true });
  await mkdir(paths.reportsDir, { recursive: true });
  const meta = await readMeta(paths);
  if (meta == null) {
    await writeMeta(paths, { schema_version: SCHEMA_VERSION, created: now.toISOString() });
  }
}

export async function storeSummary(paths: DataPaths): Promise<StoreSummary> {
  const workouts = await readWorkouts(paths);
  const days = workouts.map(calendarDay).sort();
  let strokeFiles = 0;
  try {
    strokeFiles = (await readdir(paths.strokesDir)).filter((f) => f.endsWith(".jsonl")).length;
  } catch {}
  let notes = 0;
  try {
    notes = (await readdir(paths.notesDir)).filter((f) => f.endsWith(".json")).length;
  } catch {}
  const meta = await readMeta(paths);
  return {
    workouts: workouts.length,
    firstDate: days[0] ?? "",
    lastDate: days[days.length - 1] ?? "",
    strokeFiles,
    notes,
    schemaVersion: meta?.schema_version ?? null,
    lastSync: meta?.last_sync ?? null,
  };
}

async function treeStats(dir: string): Promise<{ files: number; bytes: number }> {
  let files = 0;
  let bytes = 0;
  const entries = await readdir(dir, { withFileTypes: true });
  for (const e of entries) {
    const full = join(dir, e.name);
    if (e.isDirectory()) {
      const sub = await treeStats(full);
      files += sub.files;
      bytes += sub.bytes;
    } else {
      files++;
      bytes += (await stat(full)).size;
    }
  }
  return { files, bytes };
}

export async function moveStore(
  from: DataPaths,
  to: DataPaths,
): Promise<{ files: number; bytes: number }> {
  const target = await inspectDataDir(to);
  if (target.state === "store" || target.state === "foreign") {
    throw new Error(`target ${to.root} is not empty`);
  }
  if (!target.writable) {
    throw new Error(`target ${to.root} is not writable`);
  }
  await mkdir(to.root, { recursive: true });
  await cp(from.root, to.root, { recursive: true });
  const src = await treeStats(from.root);
  const dst = await treeStats(to.root);
  if (src.files !== dst.files || src.bytes !== dst.bytes) {
    throw new Error(
      `copy verification failed: source ${src.files} files/${src.bytes} bytes, target ${dst.files} files/${dst.bytes} bytes`,
    );
  }
  return dst;
}
