import { expect, test } from "bun:test";
import { homedir } from "node:os";
import { join } from "node:path";
import { expandTilde, pathsFor } from "./paths.ts";

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
