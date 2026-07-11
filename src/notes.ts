import { mkdir, readdir, readFile, rm, writeFile } from "node:fs/promises";
import { join } from "node:path";
import { isValidYMD } from "./models.ts";
import type { DataPaths } from "./paths.ts";

export const NOTE_TYPES = ["subjective", "observation", "lesson"] as const;
export type NoteType = (typeof NOTE_TYPES)[number];

export const NOTE_AUTHORS = ["athlete", "coach"] as const;
export type NoteAuthor = (typeof NOTE_AUTHORS)[number];

export interface NoteRecord {
  id: string;
  date: string;
  type: NoteType;
  workout_id?: number;
  tags?: string[];
  body: string;
  author: NoteAuthor;
}

const CROCKFORD = "0123456789ABCDEFGHJKMNPQRSTVWXYZ";

export function ulid(now: Date): string {
  let ms = now.getTime();
  let time = "";
  for (let i = 0; i < 10; i++) {
    time = CROCKFORD[ms % 32] + time;
    ms = Math.floor(ms / 32);
  }
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  let rand = "";
  for (const b of bytes) {
    rand += CROCKFORD[b % 32];
  }
  return time + rand;
}

export function localISO(d: Date): string {
  const pad = (n: number, width = 2) => String(Math.abs(n)).padStart(width, "0");
  const offsetMinutes = -d.getTimezoneOffset();
  const sign = offsetMinutes >= 0 ? "+" : "-";
  const offset = `${sign}${pad(Math.floor(Math.abs(offsetMinutes) / 60))}:${pad(Math.abs(offsetMinutes) % 60)}`;
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}${offset}`;
}

export function serializeNote(note: NoteRecord): string {
  const ordered: Record<string, unknown> = { id: note.id, date: note.date, type: note.type };
  if (note.workout_id != null) ordered.workout_id = note.workout_id;
  if (note.tags != null && note.tags.length > 0) ordered.tags = note.tags;
  ordered.body = note.body;
  ordered.author = note.author;
  return JSON.stringify(ordered);
}

export function isNoteShaped(parsed: unknown): parsed is NoteRecord {
  const note = parsed as NoteRecord;
  return (
    typeof note?.id === "string" &&
    typeof note?.date === "string" &&
    typeof note?.body === "string" &&
    (NOTE_TYPES as readonly string[]).includes(note?.type) &&
    (NOTE_AUTHORS as readonly string[]).includes(note?.author) &&
    /^\d{4}-\d{2}-\d{2}T.*(?:Z|[+-]\d{2}:\d{2})$/.test(note.date) &&
    isValidYMD(note.date.slice(0, 10)) &&
    !Number.isNaN(new Date(note.date).getTime()) &&
    (note.workout_id === undefined ||
      (typeof note.workout_id === "number" && Number.isFinite(note.workout_id))) &&
    (note.tags === undefined ||
      (Array.isArray(note.tags) && note.tags.every((t) => typeof t === "string")))
  );
}

function parseNote(raw: string): NoteRecord | null {
  try {
    const parsed = JSON.parse(raw) as unknown;
    return isNoteShaped(parsed) ? parsed : null;
  } catch {
    return null;
  }
}

function compareNotes(a: NoteRecord, b: NoteRecord): number {
  const aMs = new Date(a.date).getTime();
  const bMs = new Date(b.date).getTime();
  if (aMs !== bMs) return aMs < bMs ? -1 : 1;
  if (a.id !== b.id) return a.id < b.id ? -1 : 1;
  return 0;
}

export async function writeNote(paths: DataPaths, note: NoteRecord): Promise<void> {
  await mkdir(paths.notesDir, { recursive: true });
  await writeFile(join(paths.notesDir, `${note.id}.json`), `${serializeNote(note)}\n`, "utf-8");
}

interface LooseEntry {
  note: NoteRecord;
  file: string;
}

async function readLooseEntries(paths: DataPaths): Promise<LooseEntry[]> {
  const entries: LooseEntry[] = [];
  let files: string[];
  try {
    files = await readdir(paths.notesDir);
  } catch {
    return entries;
  }
  for (const f of files.sort()) {
    if (!f.endsWith(".json")) continue;
    try {
      const note = parseNote(await readFile(join(paths.notesDir, f), "utf-8"));
      if (note != null) entries.push({ note, file: f });
    } catch {}
  }
  return entries;
}

async function readLooseNotes(paths: DataPaths): Promise<Map<string, NoteRecord>> {
  const notes = new Map<string, NoteRecord>();
  for (const entry of await readLooseEntries(paths)) {
    notes.set(entry.note.id, entry.note);
  }
  return notes;
}

async function readArchivedNotes(paths: DataPaths): Promise<Map<string, NoteRecord>> {
  const notes = new Map<string, NoteRecord>();
  let files: string[];
  try {
    files = await readdir(paths.archiveDir);
  } catch {
    return notes;
  }
  for (const f of files) {
    if (!f.endsWith(".jsonl")) continue;
    let text: string;
    try {
      text = await readFile(join(paths.archiveDir, f), "utf-8");
    } catch {
      continue;
    }
    for (const line of text.split("\n")) {
      if (line.trim() === "") continue;
      const note = parseNote(line);
      if (note != null && !notes.has(note.id)) notes.set(note.id, note);
    }
  }
  return notes;
}

export async function readAllNotes(paths: DataPaths): Promise<NoteRecord[]> {
  const archived = await readArchivedNotes(paths);
  const loose = await readLooseNotes(paths);
  for (const [id, note] of loose) {
    archived.set(id, note);
  }
  return [...archived.values()].sort(compareNotes);
}

export interface NoteFilter {
  type?: string;
  since?: string;
  workoutId?: number;
}

export function filterNotes(notes: NoteRecord[], filter: NoteFilter): NoteRecord[] {
  return notes.filter((n) => {
    if (filter.type && n.type !== filter.type) return false;
    if (filter.since && n.date.slice(0, 10) < filter.since) return false;
    if (filter.workoutId != null && n.workout_id !== filter.workoutId) return false;
    return true;
  });
}

const COMPACT_AGE_DAYS = 7;

async function readArchiveYear(
  paths: DataPaths,
  year: number,
): Promise<{ notes: NoteRecord[]; safeToRewrite: boolean }> {
  const notes: NoteRecord[] = [];
  let text: string;
  try {
    text = await readFile(paths.archiveFile(year), "utf-8");
  } catch (err: unknown) {
    const code = (err as NodeJS.ErrnoException).code;
    return { notes, safeToRewrite: code === "ENOENT" };
  }
  const seen = new Set<string>();
  for (const line of text.split("\n")) {
    if (line.trim() === "") continue;
    const note = parseNote(line);
    if (note == null) return { notes, safeToRewrite: false };
    if (seen.has(note.id)) return { notes, safeToRewrite: false };
    seen.add(note.id);
    notes.push(note);
  }
  return { notes, safeToRewrite: true };
}

export interface CompactResult {
  archived: number;
  years: number[];
  skippedYears: number[];
}

export async function compactNotes(paths: DataPaths, now: Date): Promise<CompactResult> {
  const cutoff = new Date(now);
  cutoff.setDate(cutoff.getDate() - COMPACT_AGE_DAYS);
  const cutoffMs = cutoff.getTime();

  const entries = await readLooseEntries(paths);
  const deduped = new Map<string, NoteRecord>();
  const contentById = new Map<string, string>();
  const divergent = new Set<string>();
  for (const e of entries) {
    const content = serializeNote(e.note);
    const prior = contentById.get(e.note.id);
    if (prior != null && prior !== content) divergent.add(e.note.id);
    contentById.set(e.note.id, content);
    deduped.set(e.note.id, e.note);
  }
  const eligible = [...deduped.values()].filter(
    (n) => !divergent.has(n.id) && new Date(n.date).getTime() < cutoffMs,
  );
  if (eligible.length === 0) return { archived: 0, years: [], skippedYears: [] };

  const byYear = new Map<number, NoteRecord[]>();
  for (const note of eligible) {
    const year = Number(note.date.slice(0, 4));
    if (!byYear.has(year)) byYear.set(year, []);
    byYear.get(year)!.push(note);
  }

  await mkdir(paths.archiveDir, { recursive: true });
  let archivedCount = 0;
  const years: number[] = [];
  const skippedYears: number[] = [];
  for (const [year, notes] of byYear) {
    const existing = await readArchiveYear(paths, year);
    if (!existing.safeToRewrite) {
      skippedYears.push(year);
      continue;
    }
    const merged = new Map<string, NoteRecord>();
    for (const n of existing.notes) merged.set(n.id, n);
    for (const n of notes) merged.set(n.id, n);
    const lines = [...merged.values()].sort(compareNotes).map(serializeNote);
    try {
      await writeFile(paths.archiveFile(year), `${lines.join("\n")}\n`, "utf-8");
    } catch {
      skippedYears.push(year);
      continue;
    }
    const archivedIds = new Set(notes.map((n) => n.id));
    for (const e of entries) {
      if (archivedIds.has(e.note.id)) {
        try {
          await rm(join(paths.notesDir, e.file), { force: true });
        } catch {}
      }
    }
    archivedCount += notes.length;
    years.push(year);
  }

  return { archived: archivedCount, years: years.sort(), skippedYears: skippedYears.sort() };
}
