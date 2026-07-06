import { expect, test } from "bun:test";
import { chmod, mkdir, mkdtemp } from "node:fs/promises";
import { homedir, tmpdir } from "node:os";
import { join } from "node:path";
import { canonicalRoot, expandTilde, pathsFor } from "./paths.ts";

test("expandTilde resolves home shorthand", () => {
  expect(expandTilde("~")).toBe(homedir());
  expect(expandTilde("~/Documents/kb")).toBe(join(homedir(), "Documents", "kb"));
  expect(expandTilde("/absolute/path")).toBe("/absolute/path");
});

test("pathsFor derives every path from the root", () => {
  const p = pathsFor("/tmp/c2-store");
  expect(p.root).toBe("/tmp/c2-store");
  expect(p.meta).toBe("/tmp/c2-store/meta.json");
  expect(p.workouts).toBe("/tmp/c2-store/workouts.jsonl");
  expect(p.strokeFile(42)).toBe("/tmp/c2-store/strokes/42.jsonl");
  expect(p.archiveFile(2026)).toBe("/tmp/c2-store/notes/archive/2026.jsonl");
  expect(p.narrativeFile("2026-07-05")).toBe("/tmp/c2-store/reports/2026-07-05.md");
});

test("pathsFor expands tilde roots", () => {
  const p = pathsFor("~/kb/c2");
  expect(p.root).toBe(join(homedir(), "kb", "c2"));
});

test("canonicalRoot keeps missing tail components", async () => {
  const base = await mkdtemp(join(tmpdir(), "c2-canon-"));
  const target = join(base, "a", "b", "c");
  const out = await canonicalRoot(target);
  expect(out.endsWith(join("a", "b", "c"))).toBe(true);
});

test("canonicalRoot preserves the full path when an ancestor is unreadable", async () => {
  const base = await mkdtemp(join(tmpdir(), "c2-canon-eacces-"));
  const locked = join(base, "locked");
  await mkdir(locked);
  await chmod(locked, 0o000);
  try {
    const target = join(locked, "x", "y");
    const out = await canonicalRoot(target);
    expect(out.endsWith(join("locked", "x", "y"))).toBe(true);
  } finally {
    await chmod(locked, 0o755);
  }
});
