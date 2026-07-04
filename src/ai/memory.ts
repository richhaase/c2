import { appendFile, mkdir, readFile } from "node:fs/promises";
import { join } from "node:path";
import { configDir } from "../config.ts";

export interface CoachNote {
  date: string;
  note: string;
}

export function coachDir(): string {
  return join(configDir(), "coach");
}

function profilePath(): string {
  return join(coachDir(), "profile.md");
}

function notesPath(): string {
  return join(coachDir(), "notes.jsonl");
}

export async function readProfile(): Promise<string> {
  try {
    return await readFile(profilePath(), "utf-8");
  } catch (err: unknown) {
    if ((err as NodeJS.ErrnoException).code === "ENOENT") return "";
    throw err;
  }
}

export function parseNotes(text: string, limit: number): CoachNote[] {
  const notes: CoachNote[] = [];
  for (const line of text.split("\n")) {
    if (line.trim() === "") continue;
    try {
      const parsed = JSON.parse(line) as CoachNote;
      if (typeof parsed?.date === "string" && typeof parsed?.note === "string") {
        notes.push(parsed);
      }
    } catch {}
  }
  return notes.slice(-limit);
}

export async function readNotes(limit: number): Promise<CoachNote[]> {
  try {
    const text = await readFile(notesPath(), "utf-8");
    return parseNotes(text, limit);
  } catch (err: unknown) {
    if ((err as NodeJS.ErrnoException).code === "ENOENT") return [];
    throw err;
  }
}

export async function appendNote(note: string, now: Date): Promise<void> {
  await mkdir(coachDir(), { recursive: true });
  const entry: CoachNote = { date: now.toISOString(), note };
  await appendFile(notesPath(), `${JSON.stringify(entry)}\n`, "utf-8");
}
