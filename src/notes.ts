import { mkdir, readdir, readFile, rm, writeFile } from "node:fs/promises";
import { join } from "node:path";
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

function parseNote(raw: string): NoteRecord | null {
  try {
    const parsed = JSON.parse(raw) as NoteRecord;
    if (
      typeof parsed?.id === "string" &&
      typeof parsed?.date === "string" &&
      typeof parsed?.body === "string" &&
      (NOTE_TYPES as readonly string[]).includes(parsed?.type)
    ) {
      return parsed;
    }
    return null;
  } catch {
    return null;
  }
}

function compareNotes(a: NoteRecord, b: NoteRecord): number {
  if (a.date !== b.date) return a.date < b.date ? -1 : 1;
  if (a.id !== b.id) return a.id < b.id ? -1 : 1;
  return 0;
}

export async function writeNote(paths: DataPaths, note: NoteRecord): Promise<void> {
  await mkdir(paths.notesDir, { recursive: true });
  await writeFile(join(paths.notesDir, `${note.id}.json`), `${serializeNote(note)}\n`, "utf-8");
}

async function readLooseNotes(paths: DataPaths): Promise<Map<string, NoteRecord>> {
  const notes = new Map<string, NoteRecord>();
  let files: string[];
  try {
    files = await readdir(paths.notesDir);
  } catch {
    return notes;
  }
  for (const f of files) {
    if (!f.endsWith(".json")) continue;
    try {
      const note = parseNote(await readFile(join(paths.notesDir, f), "utf-8"));
      if (note != null) notes.set(note.id, note);
    } catch {}
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

export async function compactNotes(
  paths: DataPaths,
  now: Date,
): Promise<{ archived: number; years: number[] }> {
  const cutoff = new Date(now);
  cutoff.setDate(cutoff.getDate() - COMPACT_AGE_DAYS);
  const cutoffKey = localISO(cutoff);

  const loose = await readLooseNotes(paths);
  const eligible = [...loose.values()].filter((n) => n.date < cutoffKey);
  if (eligible.length === 0) return { archived: 0, years: [] };

  const byYear = new Map<number, NoteRecord[]>();
  for (const note of eligible) {
    const year = Number(note.date.slice(0, 4));
    if (!byYear.has(year)) byYear.set(year, []);
    byYear.get(year)!.push(note);
  }

  const archived = await readArchivedNotes(paths);
  await mkdir(paths.archiveDir, { recursive: true });
  for (const [year, notes] of byYear) {
    const merged = new Map<string, NoteRecord>();
    for (const n of archived.values()) {
      if (Number(n.date.slice(0, 4)) === year) merged.set(n.id, n);
    }
    for (const n of notes) merged.set(n.id, n);
    const lines = [...merged.values()].sort(compareNotes).map(serializeNote);
    await writeFile(paths.archiveFile(year), `${lines.join("\n")}\n`, "utf-8");
  }

  for (const note of eligible) {
    await rm(join(paths.notesDir, `${note.id}.json`), { force: true });
  }

  return { archived: eligible.length, years: [...byYear.keys()].sort() };
}
