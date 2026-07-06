import { beforeAll, expect, test } from "bun:test";
import { mkdir, mkdtemp, readFile, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { isAbsolute, join, sep } from "node:path";

let home: string;

function run(
  args: string[],
  opts: { cwd?: string; home?: string } = {},
): { code: number; stdout: string; stderr: string } {
  const proc = Bun.spawnSync({
    cmd: ["bun", join(import.meta.dir, "index.ts"), ...args],
    env: { ...process.env, HOME: opts.home ?? home },
    cwd: opts.cwd,
    stdout: "pipe",
    stderr: "pipe",
  });
  return {
    code: proc.exitCode,
    stdout: proc.stdout.toString(),
    stderr: proc.stderr.toString(),
  };
}

beforeAll(async () => {
  home = await mkdtemp(join(tmpdir(), "c2-cli-test-"));
  const dataDir = join(home, ".config", "c2", "data");
  await mkdir(join(dataDir, "strokes"), { recursive: true });
  await writeFile(
    join(home, ".config", "c2", "config.json"),
    JSON.stringify({
      api: { base_url: "https://log.concept2.com", token: "tok" },
      goal: { target_meters: 1_000_000, start_date: "2026-01-01", end_date: "2026-12-31" },
    }),
    "utf-8",
  );
  const workouts = [
    {
      id: 1,
      user_id: 1,
      date: "2026-07-01 08:00:00",
      distance: 8000,
      type: "rower",
      time: 12000,
      time_formatted: "20:00.0",
      stroke_rate: 24,
      heart_rate: { average: 140 },
      comments: "felt strong",
    },
    {
      id: 2,
      user_id: 1,
      date: "2026-06-28 08:00:00",
      distance: 6000,
      type: "rower",
      time: 9000,
      time_formatted: "15:00.0",
    },
  ];
  await writeFile(
    join(dataDir, "workouts.jsonl"),
    `${workouts.map((w) => JSON.stringify(w)).join("\n")}\n`,
    "utf-8",
  );
});

test("bare c2 prints help and exits 0", () => {
  const r = run([]);
  expect(r.code).toBe(0);
  expect(r.stdout).toContain("Usage: c2");
  expect(r.stdout).not.toContain("Report written");
});

test("unknown command errors cleanly", () => {
  const r = run(["statsu"]);
  expect(r.code).toBe(1);
  expect(r.stderr).toContain("unknown command");
  expect(r.stderr).not.toContain("report");
});

test("status --json emits versioned envelope", () => {
  const r = run(["status", "--json"]);
  expect(r.code).toBe(0);
  const parsed = JSON.parse(r.stdout);
  expect(parsed.schema).toBe("c2.status.v1");
  expect(parsed.data.goal.totalMeters).toBe(14000);
  expect(parsed.data.this_week).toBeDefined();
  expect(parsed.data.recent_weeks.length).toBe(4);
});

test("log --json emits workout rows", () => {
  const r = run(["log", "--json"]);
  expect(r.code).toBe(0);
  const parsed = JSON.parse(r.stdout);
  expect(parsed.schema).toBe("c2.log.v1");
  expect(parsed.data.count).toBe(2);
  expect(parsed.data.workouts[0].id).toBe(1);
  expect(parsed.data.workouts[0].pace_500m).toBe("1:15.0");
  expect(parsed.data.workouts[0].comments).toBe("felt strong");
});

test("log human output shows comments", () => {
  const r = run(["log"]);
  expect(r.code).toBe(0);
  expect(r.stdout).toContain("— felt strong");
});

test("log date filters work and reject garbage", () => {
  const filtered = run(["log", "--from", "2026-07-01", "--json"]);
  expect(JSON.parse(filtered.stdout).data.count).toBe(1);

  const bad = run(["log", "--from", "not-a-date"]);
  expect(bad.code).toBe(1);
  expect(bad.stderr).toContain("invalid --from date");
});

test("trend --json emits week summaries", () => {
  const r = run(["trend", "--json", "-w", "3"]);
  expect(r.code).toBe(0);
  const parsed = JSON.parse(r.stdout);
  expect(parsed.schema).toBe("c2.trend.v1");
  expect(parsed.data.weeks.length).toBe(3);
  const withData = parsed.data.weeks.filter((w: { meters: number }) => w.meters > 0);
  expect(withData.length).toBe(2);
});

test("export rejects invalid dates but allows empty ranges", () => {
  const bad = run(["export", "--from", "not-a-date"]);
  expect(bad.code).toBe(1);
  expect(bad.stderr).toContain("invalid --from date");

  const rollover = run(["export", "--from", "2026-02-31"]);
  expect(rollover.code).toBe(1);
  expect(rollover.stderr).toContain("invalid --from date");

  const empty = run(["export", "--from", "2030-01-01", "-f", "json"]);
  expect(empty.code).toBe(0);
  expect(empty.stderr).toContain("No workouts match");
  expect(JSON.parse(empty.stdout)).toEqual([]);

  const emptyCSV = run(["export", "--from", "2030-01-01"]);
  expect(emptyCSV.code).toBe(0);
  expect(emptyCSV.stdout.trim().split("\n").length).toBe(1);
  expect(emptyCSV.stdout).toContain("id,date,distance");
});

test("--json emits envelopes even on an empty store", async () => {
  const home2 = await mkdtemp(join(tmpdir(), "c2-cli-empty-"));
  await mkdir(join(home2, ".config", "c2", "data", "strokes"), { recursive: true });
  await writeFile(
    join(home2, ".config", "c2", "config.json"),
    JSON.stringify({
      goal: { target_meters: 1_000_000, start_date: "2026-01-01", end_date: "2026-12-31" },
    }),
    "utf-8",
  );

  const status = run(["status", "--json"], { home: home2 });
  expect(status.code).toBe(0);
  const statusParsed = JSON.parse(status.stdout);
  expect(statusParsed.schema).toBe("c2.status.v1");
  expect(statusParsed.data.goal.totalMeters).toBe(0);

  const log = run(["log", "--json"], { home: home2 });
  expect(JSON.parse(log.stdout).data.count).toBe(0);

  const trend = run(["trend", "--json", "-w", "2"], { home: home2 });
  expect(JSON.parse(trend.stdout).data.weeks.length).toBe(2);
});

test("export json remains a raw array for legacy consumers", () => {
  const r = run(["export", "-f", "json"]);
  expect(r.code).toBe(0);
  const parsed = JSON.parse(r.stdout);
  expect(Array.isArray(parsed)).toBe(true);
  expect(parsed.length).toBe(2);
});

test("data info reports the store", () => {
  const r = run(["data", "info", "--json"]);
  expect(r.code).toBe(0);
  const parsed = JSON.parse(r.stdout);
  expect(parsed.schema).toBe("c2.data.info.v1");
  expect(parsed.data.workouts).toBe(2);
  expect(parsed.data.first_date).toBe("2026-06-28");
  expect(parsed.data.root).toBe(join(home, ".config", "c2", "data"));
});

test("data move relocates the store and updates config", async () => {
  const target = join(home, "kb-data");
  const r = run(["data", "move", target]);
  expect(r.code).toBe(0);
  expect(r.stdout).toContain("Config updated");

  const info = run(["data", "info", "--json"]);
  const parsed = JSON.parse(info.stdout);
  expect(parsed.data.root.endsWith(`${sep}kb-data`)).toBe(true);
  expect(parsed.data.workouts).toBe(2);

  const back = run(["log", "--json"]);
  expect(JSON.parse(back.stdout).data.count).toBe(2);
});

test("data move persists an absolute path for relative targets", async () => {
  const r = run(["data", "move", "rel-store"], { cwd: home });
  expect(r.code).toBe(0);

  const cfg = JSON.parse(await readFile(join(home, ".config", "c2", "config.json"), "utf-8"));
  expect(isAbsolute(cfg.data_dir)).toBe(true);
  expect(cfg.data_dir.endsWith(`${sep}rel-store`)).toBe(true);

  const elsewhere = run(["log", "--json"], { cwd: tmpdir() });
  expect(JSON.parse(elsewhere.stdout).data.count).toBe(2);
});

test("data move refuses targets nested in the store", () => {
  const r = run(["data", "move", join(home, "rel-store", "sub")]);
  expect(r.code).toBe(1);
  expect(r.stderr).toContain("inside the current data directory");
});
