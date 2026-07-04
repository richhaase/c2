import { expect, test } from "bun:test";
import type { StrokeData } from "../models.ts";
import { parseNotes } from "./memory.ts";
import { DEFAULT_MODEL, RECOMMENDED_MODELS, resolveModelChoice } from "./models.ts";
import { allowedHostsFor } from "./server.ts";
import { downsampleStrokes } from "./tools.ts";

test("default model is the first recommendation", () => {
  expect(DEFAULT_MODEL).toBe(RECOMMENDED_MODELS[0]!.id);
  expect(RECOMMENDED_MODELS.length).toBeGreaterThanOrEqual(2);
});

test("numeric input picks from the menu", () => {
  expect(resolveModelChoice("1", "keep/me")).toBe(RECOMMENDED_MODELS[0]!.id);
  expect(resolveModelChoice(` ${RECOMMENDED_MODELS.length} `, "keep/me")).toBe(
    RECOMMENDED_MODELS[RECOMMENDED_MODELS.length - 1]!.id,
  );
});

test("out-of-range number keeps current", () => {
  expect(resolveModelChoice("0", "keep/me")).toBe("keep/me");
  expect(resolveModelChoice("99", "keep/me")).toBe("keep/me");
});

test("free text is a custom model id", () => {
  expect(resolveModelChoice("mistralai/mistral-medium", "keep/me")).toBe(
    "mistralai/mistral-medium",
  );
});

test("blank keeps current", () => {
  expect(resolveModelChoice("", "keep/me")).toBe("keep/me");
  expect(resolveModelChoice("   ", "keep/me")).toBe("keep/me");
});

function strokes(n: number): StrokeData[] {
  return Array.from({ length: n }, (_, i) => ({ t: i * 10, d: i * 5 }));
}

test("small stroke sets pass through untouched", () => {
  const s = strokes(200);
  expect(downsampleStrokes(s)).toBe(s);
});

test("corrupt notes lines are skipped, valid ones survive", () => {
  const text = [
    '{"date":"2026-07-01T12:00:00.000Z","note":"good one"}',
    '{"date":"2026-07-02T12:00:00.000Z","note":"tr',
    "not json at all",
    '{"unrelated":true}',
    '{"date":"2026-07-03T12:00:00.000Z","note":"also good"}',
  ].join("\n");
  const notes = parseNotes(text, 50);
  expect(notes.length).toBe(2);
  expect(notes[0]!.note).toBe("good one");
  expect(notes[1]!.note).toBe("also good");
  expect(parseNotes(text, 1).length).toBe(1);
  expect(parseNotes(text, 1)[0]!.note).toBe("also good");
});

test("loopback binds allow only local host forms", () => {
  const hosts = allowedHostsFor("127.0.0.1", 8080);
  expect(hosts.has("127.0.0.1:8080")).toBe(true);
  expect(hosts.has("localhost:8080")).toBe(true);
  expect(hosts.has("evil.example.com:8080")).toBe(false);
  expect(hosts.size).toBe(3);
});

test("wildcard binds allow the machine's interface addresses", () => {
  const hosts = allowedHostsFor("0.0.0.0", 8080);
  expect(hosts.has("127.0.0.1:8080")).toBe(true);
  expect(hosts.size).toBeGreaterThan(4 - 1);
  expect(hosts.has("evil.example.com:8080")).toBe(false);
  for (const h of hosts) {
    expect(h.endsWith(":8080")).toBe(true);
  }
});
test("large stroke sets downsample to the cap and keep endpoints", () => {
  const s = strokes(1000);
  const out = downsampleStrokes(s);
  expect(out.length).toBe(300);
  expect(out[0]).toBe(s[0]!);
  expect(out[out.length - 1]).toBe(s[999]!);
  const distances = out.map((x) => x.d ?? 0);
  expect([...distances].sort((a, b) => a - b)).toEqual(distances);
});
