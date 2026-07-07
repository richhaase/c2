import { expect, test } from "bun:test";
import { mkdir, mkdtemp, readdir, readFile, writeFile } from "node:fs/promises";
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
