import { cp, mkdir, readdir, rm, stat, writeFile } from "node:fs/promises";
import { basename, dirname, join } from "node:path";
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

const STORE_MARKERS = new Set([
  "workouts.jsonl",
  "strokes",
  "notes",
  "reports",
  "plan.md",
  "playbook.md",
]);

export async function inspectDataDir(paths: DataPaths): Promise<DirInspection> {
  let state: DirState;
  try {
    const s = await stat(paths.root);
    if (!s.isDirectory()) {
      return { path: paths.root, state: "foreign", writable: false };
    }
    const meta = await readMeta(paths);
    const entries = (await readdir(paths.root)).filter((e) => !e.startsWith("."));
    if (meta != null && typeof meta.schema_version === "number") {
      state = "store";
    } else if (entries.length === 0) {
      state = "empty";
    } else {
      const hasStrongMarker = entries.includes("workouts.jsonl") || entries.includes("strokes");
      const allKnown = entries.every((e) => STORE_MARKERS.has(e) || e === "meta.json");
      state = hasStrongMarker && allKnown ? "store" : "foreign";
    }
  } catch (err: unknown) {
    const code = (err as NodeJS.ErrnoException).code;
    if (code === "ENOTDIR") {
      return { path: paths.root, state: "foreign", writable: false };
    }
    if (code !== "ENOENT") throw err;
    return { path: paths.root, state: "missing", writable: await parentWritable(paths.root) };
  }
  return { path: paths.root, state, writable: await probeWritable(paths.root) };
}

async function parentWritable(path: string): Promise<boolean> {
  let current = path;
  for (;;) {
    const parent = dirname(current);
    if (parent === current) return false;
    try {
      await stat(parent);
      return probeWritable(parent);
    } catch (err: unknown) {
      if ((err as NodeJS.ErrnoException).code !== "ENOENT") return false;
      current = parent;
    }
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

export async function ensureStoreForWrite(paths: DataPaths, now: Date): Promise<string | null> {
  const inspection = await inspectDataDir(paths);
  if (inspection.state === "foreign") {
    return `${paths.root} exists but is not a c2 data store. Fix data_dir via \`c2 setup\`.`;
  }
  if (!inspection.writable) {
    return `Cannot write to ${paths.root}.`;
  }
  await initStore(paths, now);
  return null;
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

async function listIfPresent(dir: string): Promise<string[]> {
  try {
    return await readdir(dir);
  } catch (err: unknown) {
    const code = (err as NodeJS.ErrnoException).code;
    if (code === "ENOENT" || code === "ENOTDIR") return [];
    throw err;
  }
}

export async function storeSummary(paths: DataPaths): Promise<StoreSummary> {
  const workouts = await readWorkouts(paths);
  const days = workouts.map(calendarDay).sort();
  const strokeFiles = (await listIfPresent(paths.strokesDir)).filter((f) =>
    f.endsWith(".jsonl"),
  ).length;
  const notes = (await listIfPresent(paths.notesDir)).filter((f) => f.endsWith(".json")).length;
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
  await cp(from.root, to.root, {
    recursive: true,
    filter: (src) => src === from.root || !basename(src).startsWith("."),
  });
  return verifyCopy(from.root, to.root);
}

async function verifyCopy(
  fromDir: string,
  toDir: string,
): Promise<{ files: number; bytes: number }> {
  let files = 0;
  let bytes = 0;
  for (const e of await readdir(fromDir, { withFileTypes: true })) {
    if (e.name.startsWith(".")) continue;
    const src = join(fromDir, e.name);
    const dst = join(toDir, e.name);
    if (e.isDirectory()) {
      const sub = await verifyCopy(src, dst);
      files += sub.files;
      bytes += sub.bytes;
    } else {
      const srcStat = await stat(src);
      let dstStat: Awaited<ReturnType<typeof stat>>;
      try {
        dstStat = await stat(dst);
      } catch {
        throw new Error(`copy verification failed: ${dst} is missing`);
      }
      if (dstStat.size !== srcStat.size) {
        throw new Error(
          `copy verification failed: ${dst} has ${dstStat.size} bytes, expected ${srcStat.size}`,
        );
      }
      files++;
      bytes += srcStat.size;
    }
  }
  return { files, bytes };
}
