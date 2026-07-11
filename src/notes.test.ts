import { expect, test } from "bun:test";
import { chmod, mkdir, mkdtemp, readdir, readFile, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import {
  compactNotes,
  filterNotes,
  localISO,
  type NoteRecord,
  readAllNotes,
  serializeNote,
  ulid,
  writeNote,
} from "./notes.ts";
import { pathsFor } from "./paths.ts";

const NOW = new Date("2026-07-06T12:00:00");

async function tempStore() {
  const base = await mkdtemp(join(tmpdir(), "c2-notes-test-"));
  const paths = pathsFor(join(base, "store"));
  await mkdir(paths.archiveDir, { recursive: true });
  return paths;
}

function record(id: string, date: string, body: string, type = "observation"): NoteRecord {
  return { id, date, type: type as NoteRecord["type"], body, author: "athlete" };
}

test("ulid is 26 chars and time-ordered", () => {
  const a = ulid(new Date("2026-07-01T00:00:00Z"));
  const b = ulid(new Date("2026-07-02T00:00:00Z"));
  expect(a.length).toBe(26);
  expect(b.length).toBe(26);
  expect(a < b).toBe(true);
});

test("localISO renders the local calendar day with offset", () => {
  const iso = localISO(new Date(2026, 6, 6, 21, 30, 0));
  expect(iso.startsWith("2026-07-06T21:30:00")).toBe(true);
  expect(/[+-]\d{2}:\d{2}$/.test(iso)).toBe(true);
});

test("notes sort by instant across mixed offsets", async () => {
  const paths = await tempStore();
  await writeNote(paths, record("MINUS6", "2026-07-05T23:30:00-06:00", "later instant"));
  await writeNote(paths, record("PLUS2", "2026-07-06T01:00:00+02:00", "earlier instant"));
  const notes = await readAllNotes(paths);
  expect(notes.map((n) => n.id)).toEqual(["PLUS2", "MINUS6"]);
});

test("mixed-offset archives pass doctor after compaction", async () => {
  const { runDoctor } = await import("./doctor.ts");
  const paths = await tempStore();
  await writeNote(paths, record("TZ1", "2026-06-05T23:30:00-06:00", "denver"));
  await writeNote(paths, record("TZ2", "2026-06-06T01:00:00+02:00", "europe"));
  const result = await compactNotes(paths, NOW);
  expect(result.archived).toBe(2);
  const report = await runDoctor(paths);
  expect(report.issues).toEqual([]);
});

test("notes round-trip and sort by date then id", async () => {
  const paths = await tempStore();
  await writeNote(paths, record("B", "2026-07-05T10:00:00-06:00", "second"));
  await writeNote(paths, record("A", "2026-07-05T10:00:00-06:00", "first"));
  await writeNote(paths, record("C", "2026-07-01T08:00:00-06:00", "oldest"));

  const notes = await readAllNotes(paths);
  expect(notes.map((n) => n.id)).toEqual(["C", "A", "B"]);
});

test("corrupt loose notes are skipped", async () => {
  const paths = await tempStore();
  await writeNote(paths, record("GOOD", "2026-07-05T10:00:00-06:00", "fine"));
  await writeFile(join(paths.notesDir, "BAD.json"), "{ nope", "utf-8");
  await writeFile(join(paths.notesDir, "SHAPE.json"), '{"id":"SHAPE"}', "utf-8");

  const notes = await readAllNotes(paths);
  expect(notes.length).toBe(1);
  expect(notes[0]!.id).toBe("GOOD");
});

test("filterNotes filters by type, since, and workout", () => {
  const notes = [
    { ...record("A", "2026-07-01T08:00:00-06:00", "x", "subjective"), workout_id: 7 },
    record("B", "2026-07-03T08:00:00-06:00", "y", "lesson"),
    record("C", "2026-07-05T08:00:00-06:00", "z", "observation"),
  ];
  expect(filterNotes(notes, { type: "lesson" }).map((n) => n.id)).toEqual(["B"]);
  expect(filterNotes(notes, { since: "2026-07-03" }).map((n) => n.id)).toEqual(["B", "C"]);
  expect(filterNotes(notes, { workoutId: 7 }).map((n) => n.id)).toEqual(["A"]);
});

test("compaction archives old notes by year, keeps the hot set, dedups reads", async () => {
  const paths = await tempStore();
  await writeNote(paths, record("OLD1", "2026-06-20T08:00:00-06:00", "old june"));
  await writeNote(paths, record("OLD2", "2025-12-30T08:00:00-07:00", "old last year"));
  await writeNote(paths, record("NEW1", "2026-07-05T08:00:00-06:00", "recent"));

  const result = await compactNotes(paths, NOW);
  expect(result.archived).toBe(2);
  expect(result.years).toEqual([2025, 2026]);

  const looseFiles = (await readdir(paths.notesDir)).filter((f) => f.endsWith(".json"));
  expect(looseFiles).toEqual(["NEW1.json"]);

  const all = await readAllNotes(paths);
  expect(all.map((n) => n.id)).toEqual(["OLD2", "OLD1", "NEW1"]);

  const again = await compactNotes(paths, NOW);
  expect(again.archived).toBe(0);
});

test("compaction is deterministic and merge-idempotent", async () => {
  const storeA = await tempStore();
  const storeB = await tempStore();
  const notes = [
    record("01A", "2026-06-01T08:00:00-06:00", "one"),
    record("01B", "2026-06-15T08:00:00-06:00", "two"),
    record("01C", "2026-05-20T08:00:00-06:00", "three"),
  ];
  for (const store of [storeA, storeB]) {
    for (const n of [...notes].reverse()) {
      await writeNote(store, n);
    }
  }

  await compactNotes(storeA, NOW);
  await compactNotes(storeB, NOW);
  const a = await readFile(storeA.archiveFile(2026), "utf-8");
  const b = await readFile(storeB.archiveFile(2026), "utf-8");
  expect(a).toBe(b);

  await writeNote(storeA, record("01D", "2026-06-20T08:00:00-06:00", "late arrival"));
  await compactNotes(storeA, NOW);
  const merged = await readAllNotes(storeA);
  expect(merged.filter((n) => n.date.startsWith("2026-06")).length).toBe(3);
});

test("notes with invalid authors are skipped by the reader", async () => {
  const paths = await tempStore();
  await writeNote(paths, record("GOOD", "2026-07-05T10:00:00-06:00", "fine"));
  await writeFile(
    join(paths.notesDir, "LLM.json"),
    JSON.stringify({
      id: "LLM",
      date: "2026-07-05T10:00:00-06:00",
      type: "observation",
      body: "x",
      author: "llm",
    }),
    "utf-8",
  );
  await writeFile(
    join(paths.notesDir, "BADTAGS.json"),
    JSON.stringify({
      id: "BADTAGS",
      date: "2026-07-05T10:00:00-06:00",
      type: "observation",
      tags: "not-an-array",
      body: "x",
      author: "athlete",
    }),
    "utf-8",
  );
  await writeFile(
    join(paths.notesDir, "BADWID.json"),
    JSON.stringify({
      id: "BADWID",
      date: "2026-07-05T10:00:00-06:00",
      type: "observation",
      workout_id: "seven",
      body: "x",
      author: "athlete",
    }),
    "utf-8",
  );
  const notes = await readAllNotes(paths);
  expect(notes.map((n) => n.id)).toEqual(["GOOD"]);
});

test("compaction refuses to rewrite archives containing corrupt lines", async () => {
  const paths = await tempStore();
  const garbage = "{ this is not json";
  await writeFile(
    paths.archiveFile(2026),
    `${serializeNote(record("OLDARCH", "2026-01-01T08:00:00-07:00", "archived"))}\n${garbage}\n`,
    "utf-8",
  );
  await writeNote(paths, record("OLDLOOSE", "2026-06-01T08:00:00-06:00", "wants archiving"));

  const result = await compactNotes(paths, NOW);
  expect(result.archived).toBe(0);
  expect(result.skippedYears).toEqual([2026]);

  const archiveText = await readFile(paths.archiveFile(2026), "utf-8");
  expect(archiveText).toContain(garbage);
  const looseFiles = (await readdir(paths.notesDir)).filter((f) => f.endsWith(".json"));
  expect(looseFiles).toEqual(["OLDLOOSE.json"]);
});

test("parseNoteDate uses the target date's own offset and rejects partial dates", async () => {
  const { parseNoteDate } = await import("./commands/note.ts");
  expect(parseNoteDate("2026-01-15")).toBe(localISO(new Date("2026-01-15T12:00:00")));
  expect(parseNoteDate("2026-07-15")).toBe(localISO(new Date("2026-07-15T12:00:00")));
  expect(parseNoteDate("2026-02-31")).toBeNull();
  expect(parseNoteDate("2026-07")).toBeNull();
  expect(parseNoteDate("2026")).toBeNull();
  expect(parseNoteDate("2026-07-15T08:30:00")).toBe(localISO(new Date("2026-07-15T08:30:00")));
  expect(parseNoteDate("2026-07-05T10:00:00+02:00")).toBe("2026-07-05T10:00:00+02:00");
  expect(parseNoteDate("2026-07-05T10:00:00Z")).toBe("2026-07-05T10:00:00+00:00");
  expect(parseNoteDate("2026-07-05 10:00:00 +02:00")).toBeNull();
});

test("doctor validates stroke row shapes and reader skips junk rows", async () => {
  const { runDoctor } = await import("./doctor.ts");
  const { readStrokeData } = await import("./storage.ts");
  const paths = await tempStore();
  await mkdir(paths.strokesDir, { recursive: true });
  await writeFile(
    paths.strokeFile(42),
    `${JSON.stringify({ t: 100, d: 500, p: 1750, spm: 24, hr: 110 })}\nnull\n${JSON.stringify({ t: "x" })}\n`,
    "utf-8",
  );
  const report = await runDoctor(paths);
  expect(report.issues).toContain("strokes/42.jsonl: line 2 malformed stroke record");
  expect(report.issues).toContain("strokes/42.jsonl: line 3 malformed stroke record");

  const strokes = await readStrokeData(paths, 42);
  expect(strokes.length).toBe(2);
  expect(strokes[0]!.p).toBe(1750);
});

test("compaction compares note age as instants, not offset strings", async () => {
  const paths = await tempStore();
  const cutoffInstant = new Date(NOW);
  cutoffInstant.setDate(cutoffInstant.getDate() - 7);
  const oldInstant = new Date(cutoffInstant.getTime() - 11 * 60 * 60 * 1000);
  const pad = (n: number) => String(n).padStart(2, "0");
  const inPlus13 = new Date(oldInstant.getTime() + 13 * 60 * 60 * 1000);
  const farOffsetDate = `${inPlus13.getUTCFullYear()}-${pad(inPlus13.getUTCMonth() + 1)}-${pad(inPlus13.getUTCDate())}T${pad(inPlus13.getUTCHours())}:${pad(inPlus13.getUTCMinutes())}:00+13:00`;

  await writeNote(paths, record("FAROFF", farOffsetDate, "written abroad"));
  const result = await compactNotes(paths, NOW);
  expect(result.archived).toBe(1);
});

test("notes with unparseable or non-ISO dates are skipped and never reach compaction", async () => {
  const paths = await tempStore();
  await writeFile(
    join(paths.notesDir, "BADDATE.json"),
    JSON.stringify({
      id: "BADDATE",
      date: "sometime last spring",
      type: "observation",
      body: "x",
      author: "athlete",
    }),
    "utf-8",
  );
  await writeFile(
    join(paths.notesDir, "PROSEDATE.json"),
    JSON.stringify({
      id: "PROSEDATE",
      date: "July 1, 2026",
      type: "observation",
      body: "parseable but not ISO",
      author: "athlete",
    }),
    "utf-8",
  );
  expect((await readAllNotes(paths)).length).toBe(0);
  const result = await compactNotes(paths, NOW);
  expect(result.archived).toBe(0);
  const looseFiles = (await readdir(paths.notesDir)).filter((f) => f.endsWith(".json"));
  expect(looseFiles.sort()).toEqual(["BADDATE.json", "PROSEDATE.json"]);
  const archives = await readdir(paths.archiveDir);
  expect(archives).not.toContain("NaN.jsonl");
  const { runDoctor } = await import("./doctor.ts");
  const report = await runDoctor(paths);
  expect(report.issues).toContain("notes/BADDATE.json: malformed note record");
  expect(report.issues).toContain("notes/PROSEDATE.json: malformed note record");
});

test("doctor accepts conflict-copy filenames and survives null workout rows", async () => {
  const { runDoctor } = await import("./doctor.ts");
  const paths = await tempStore();
  const n = record("CONF", "2026-07-05T10:00:00-06:00", "conflict copy content");
  await writeFile(join(paths.notesDir, "CONF conflict copy 2.json"), serializeNote(n), "utf-8");
  const clean = await runDoctor(paths);
  expect(clean.issues).toEqual([]);

  await writeFile(paths.workouts, 'null\n"just a string"\n', "utf-8");
  const report = await runDoctor(paths);
  expect(report.issues).toContain("workouts.jsonl: line 1 malformed workout record");
  expect(report.issues).toContain("workouts.jsonl: line 2 malformed workout record");
});

test("doctor flags malformed workout records", async () => {
  const { runDoctor } = await import("./doctor.ts");
  const paths = await tempStore();
  await writeFile(
    paths.workouts,
    `${JSON.stringify({ id: 1, date: "2026-07-01 08:00:00", distance: 8000, time: 12000 })}\n${JSON.stringify({ id: "two", date: 5 })}\n`,
    "utf-8",
  );
  const report = await runDoctor(paths);
  expect(report.issues).toContain("workouts.jsonl: line 2 malformed workout record");
});

test("offset-less note timestamps are rejected everywhere", async () => {
  const { runDoctor } = await import("./doctor.ts");
  const paths = await tempStore();
  await writeFile(
    join(paths.notesDir, "NOOFFSET.json"),
    JSON.stringify({
      id: "NOOFFSET",
      date: "2026-07-05T10:00:00",
      type: "observation",
      body: "parses as local time",
      author: "athlete",
    }),
    "utf-8",
  );
  expect((await readAllNotes(paths)).length).toBe(0);
  const report = await runDoctor(paths);
  expect(report.issues).toContain("notes/NOOFFSET.json: malformed note record");
});

test("compaction skips read-only archives without failing", async () => {
  const paths = await tempStore();
  await writeFile(
    paths.archiveFile(2026),
    `${serializeNote(record("RO", "2026-01-01T08:00:00-07:00", "existing"))}\n`,
    "utf-8",
  );
  await chmod(paths.archiveFile(2026), 0o444);
  await writeNote(paths, record("ROLOOSE", "2026-06-01T08:00:00-06:00", "wants in"));
  try {
    const result = await compactNotes(paths, NOW);
    expect(result.archived).toBe(0);
    expect(result.skippedYears).toEqual([2026]);
  } finally {
    await chmod(paths.archiveFile(2026), 0o644);
  }
  const looseFiles = (await readdir(paths.notesDir)).filter((f) => f.endsWith(".json"));
  expect(looseFiles).toEqual(["ROLOOSE.json"]);
});

test("divergent conflict copies are preserved and flagged, identical ones compact", async () => {
  const { runDoctor } = await import("./doctor.ts");
  const paths = await tempStore();
  const original = record("DIV", "2026-06-01T08:00:00-06:00", "original text");
  await writeNote(paths, original);
  await writeFile(
    join(paths.notesDir, "DIV conflict.json"),
    serializeNote({ ...original, body: "edited on the other machine" }),
    "utf-8",
  );

  const report = await runDoctor(paths);
  expect(report.issues.some((i) => i.includes("divergent copies of note DIV"))).toBe(true);

  const result = await compactNotes(paths, NOW);
  expect(result.archived).toBe(0);
  const looseFiles = (await readdir(paths.notesDir)).filter((f) => f.endsWith(".json"));
  expect(looseFiles.length).toBe(2);
});

test("compaction deletes the files it read, including conflict copies", async () => {
  const paths = await tempStore();
  const old = record("OLDX", "2026-06-01T08:00:00-06:00", "original");
  await writeNote(paths, old);
  await writeFile(join(paths.notesDir, "OLDX conflict 2.json"), serializeNote(old), "utf-8");
  await writeFile(
    join(paths.notesDir, "WEIRD NAME.json"),
    serializeNote(record("OLDY", "2026-06-02T08:00:00-06:00", "odd filename")),
    "utf-8",
  );

  const result = await compactNotes(paths, NOW);
  expect(result.archived).toBe(2);

  const looseFiles = (await readdir(paths.notesDir)).filter((f) => f.endsWith(".json"));
  expect(looseFiles).toEqual([]);
  const all = await readAllNotes(paths);
  expect(all.map((n) => n.id)).toEqual(["OLDX", "OLDY"]);

  const again = await compactNotes(paths, NOW);
  expect(again.archived).toBe(0);
});

test("compaction skips unreadable archives instead of truncating them", async () => {
  const paths = await tempStore();
  await writeFile(
    paths.archiveFile(2026),
    `${serializeNote(record("SAFE", "2026-01-01T08:00:00-07:00", "already archived"))}\n`,
    "utf-8",
  );
  await chmod(paths.archiveFile(2026), 0o000);
  await writeNote(paths, record("OLDLOOSE2", "2026-06-01T08:00:00-06:00", "wants archiving"));

  try {
    const result = await compactNotes(paths, NOW);
    expect(result.archived).toBe(0);
    expect(result.skippedYears).toEqual([2026]);
  } finally {
    await chmod(paths.archiveFile(2026), 0o644);
  }
  const archiveText = await readFile(paths.archiveFile(2026), "utf-8");
  expect(archiveText).toContain("SAFE");
  const looseFiles = (await readdir(paths.notesDir)).filter((f) => f.endsWith(".json"));
  expect(looseFiles).toEqual(["OLDLOOSE2.json"]);
});

test("doctor flags unreadable note directories", async () => {
  const { runDoctor } = await import("./doctor.ts");
  const base = await mkdtemp(join(tmpdir(), "c2-notes-test-"));
  const paths = pathsFor(join(base, "store"));
  await mkdir(paths.root, { recursive: true });
  await writeFile(paths.notesDir, "a file where a directory should be", "utf-8");
  const report = await runDoctor(paths);
  expect(report.issues.some((i) => i.startsWith("notes/: unreadable"))).toBe(true);
});

test("doctor tolerates loose/archive duplicates but flags malformed records", async () => {
  const { runDoctor } = await import("./doctor.ts");
  const paths = await tempStore();
  const dup = record("DUP", "2026-06-01T08:00:00-06:00", "both places");
  await writeNote(paths, dup);
  await writeFile(paths.archiveFile(2026), `${serializeNote(dup)}\n`, "utf-8");

  const clean = await runDoctor(paths);
  expect(clean.issues).toEqual([]);

  await writeFile(
    paths.archiveFile(2027),
    `${JSON.stringify({ id: "X", date: "2027-01-01T00:00:00-07:00" })}\n`,
    "utf-8",
  );
  await writeFile(join(paths.notesDir, "BADSHAPE.json"), '{"id":"BADSHAPE"}', "utf-8");
  const dirty = await runDoctor(paths);
  expect(dirty.issues).toContain("notes/archive/2027.jsonl: line 1 malformed note record");
  expect(dirty.issues).toContain("notes/BADSHAPE.json: malformed note record");
});

test("a note both loose and archived reads once", async () => {
  const paths = await tempStore();
  const n = record("DUP", "2026-06-01T08:00:00-06:00", "loose wins");
  await writeNote(paths, n);
  await writeFile(
    paths.archiveFile(2026),
    `${serializeNote({ ...n, body: "archived copy" })}\n`,
    "utf-8",
  );
  const all = await readAllNotes(paths);
  expect(all.length).toBe(1);
  expect(all[0]!.body).toBe("loose wins");
});
