import { expect, test } from "bun:test";
import { mkdir, mkdtemp, readdir, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { initStore, inspectDataDir, moveStore, storeSummary } from "./data.ts";
import { pathsFor } from "./paths.ts";
import { readMeta, readWorkouts } from "./storage.ts";

const NOW = new Date("2026-07-05T12:00:00");

async function tempRoot(): Promise<string> {
  return mkdtemp(join(tmpdir(), "c2-data-test-"));
}

const WORKOUT_LINE = JSON.stringify({
  id: 1,
  user_id: 1,
  date: "2026-07-01 08:00:00",
  distance: 8000,
  type: "rower",
  time: 12000,
  time_formatted: "20:00.0",
});

test("inspect reports missing directory", async () => {
  const base = await tempRoot();
  const paths = pathsFor(join(base, "nope"));
  const insp = await inspectDataDir(paths);
  expect(insp.state).toBe("missing");
  expect(insp.writable).toBe(true);
});

test("inspect reports empty, store, legacy store, and foreign", async () => {
  const base = await tempRoot();

  const empty = pathsFor(join(base, "empty"));
  await mkdir(empty.root);
  expect((await inspectDataDir(empty)).state).toBe("empty");

  const store = pathsFor(join(base, "store"));
  await mkdir(store.root);
  await initStore(store, NOW);
  expect((await inspectDataDir(store)).state).toBe("store");

  const legacy = pathsFor(join(base, "legacy"));
  await mkdir(join(legacy.root, "strokes"), { recursive: true });
  await writeFile(legacy.workouts, `${WORKOUT_LINE}\n`, "utf-8");
  expect((await inspectDataDir(legacy)).state).toBe("store");

  const foreign = pathsFor(join(base, "foreign"));
  await mkdir(foreign.root);
  await writeFile(join(foreign.root, "novel.docx"), "chapter one", "utf-8");
  expect((await inspectDataDir(foreign)).state).toBe("foreign");
});

test("initStore creates directories and meta once", async () => {
  const base = await tempRoot();
  const paths = pathsFor(join(base, "init"));
  await mkdir(paths.root);
  await initStore(paths, NOW);

  const meta = await readMeta(paths);
  expect(meta?.schema_version).toBe(1);
  expect(meta?.created).toBe(NOW.toISOString());

  await initStore(paths, new Date("2027-01-01T00:00:00"));
  const again = await readMeta(paths);
  expect(again?.created).toBe(NOW.toISOString());
});

test("storeSummary counts contents", async () => {
  const base = await tempRoot();
  const paths = pathsFor(join(base, "sum"));
  await mkdir(paths.root);
  await initStore(paths, NOW);
  await writeFile(paths.workouts, `${WORKOUT_LINE}\n`, "utf-8");
  await writeFile(paths.strokeFile(1), '{"t":1}\n', "utf-8");
  await writeFile(join(paths.notesDir, "01A.json"), "{}", "utf-8");

  const summary = await storeSummary(paths);
  expect(summary.workouts).toBe(1);
  expect(summary.firstDate).toBe("2026-07-01");
  expect(summary.strokeFiles).toBe(1);
  expect(summary.notes).toBe(1);
  expect(summary.schemaVersion).toBe(1);
});

test("moveStore copies, verifies, and refuses non-empty targets", async () => {
  const base = await tempRoot();
  const from = pathsFor(join(base, "src"));
  await mkdir(from.root);
  await initStore(from, NOW);
  await writeFile(from.workouts, `${WORKOUT_LINE}\n`, "utf-8");
  await writeFile(from.strokeFile(1), '{"t":1}\n', "utf-8");

  const to = pathsFor(join(base, "dst"));
  const copied = await moveStore(from, to);
  expect(copied.files).toBeGreaterThanOrEqual(3);
  expect((await readWorkouts(to)).length).toBe(1);
  expect((await readdir(to.strokesDir)).length).toBe(1);

  const occupied = pathsFor(join(base, "occupied"));
  await mkdir(occupied.root);
  await writeFile(join(occupied.root, "file.txt"), "x", "utf-8");
  await expect(moveStore(from, occupied)).rejects.toThrow("not empty");
});
