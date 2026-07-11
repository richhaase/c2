import { beforeAll, expect, test } from "bun:test";
import { mkdir, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import { isAbsolute, join, sep } from "node:path";

let home: string;

function ymd(d: Date): string {
  const mm = String(d.getMonth() + 1).padStart(2, "0");
  const dd = String(d.getDate()).padStart(2, "0");
  return `${d.getFullYear()}-${mm}-${dd}`;
}

function daysFromNow(n: number): Date {
  const d = new Date();
  d.setDate(d.getDate() + n);
  return d;
}

const RECENT = ymd(daysFromNow(-2));
const OLDER = ymd(daysFromNow(-9));
const FUTURE = ymd(daysFromNow(365));

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
      goal: {
        target_meters: 1_000_000,
        start_date: ymd(daysFromNow(-180)),
        end_date: ymd(daysFromNow(180)),
      },
    }),
    "utf-8",
  );
  const workouts = [
    {
      id: 1,
      user_id: 1,
      date: `${RECENT} 08:00:00`,
      distance: 8000,
      type: "rower",
      time: 12000,
      time_formatted: "20:00.0",
      stroke_rate: 24,
      heart_rate: { average: 140 },
      comments: "felt strong",
      workout: {
        targets: { pace: 1500 },
        splits: [
          {
            type: "distance",
            time: 6100,
            distance: 4000,
            stroke_rate: 24,
            heart_rate: { average: 135, max: 142 },
          },
          {
            type: "distance",
            time: 5900,
            distance: 4000,
            stroke_rate: 25,
            heart_rate: { average: 144, max: 150 },
          },
        ],
      },
    },
    {
      id: 2,
      user_id: 1,
      date: `${OLDER} 08:00:00`,
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
  await writeFile(
    join(dataDir, "strokes", "1.jsonl"),
    `${JSON.stringify({ t: 100, d: 500, p: 1500, spm: 24, hr: 130 })}\n${JSON.stringify({ t: 200, d: 1000, p: 1480, spm: 25, hr: 140 })}\n`,
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
  const filtered = run(["log", "--from", RECENT, "--json"]);
  expect(JSON.parse(filtered.stdout).data.count).toBe(1);

  const bad = run(["log", "--from", "not-a-date"]);
  expect(bad.code).toBe(1);
  expect(bad.stderr).toContain("invalid --from date");

  const none = run(["log", "--from", FUTURE]);
  expect(none.code).toBe(0);
  expect(none.stdout).toContain("No workouts match");
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

test("show resolves last and ids, renders splits and strokes", () => {
  const last = run(["show", "last"]);
  expect(last.code).toBe(0);
  expect(last.stdout).toContain("Id: 1");
  expect(last.stdout).toContain("Splits (negative):");
  expect(last.stdout).toContain("Stroke data: 2 samples");
  expect(last.stdout).toContain("Comments: felt strong");
  expect(last.stdout).toContain("Target pace: 2:30.0/500m");

  const byId = run(["show", "2"]);
  expect(byId.code).toBe(0);
  expect(byId.stdout).toContain("Id: 2");
  expect(byId.stdout).not.toContain("Splits");

  const missing = run(["show", "999"]);
  expect(missing.code).toBe(1);
  expect(missing.stderr).toContain("No workout with id 999");
});

test("show --json emits full detail envelope", () => {
  const r = run(["show", "last", "--json"]);
  expect(r.code).toBe(0);
  const parsed = JSON.parse(r.stdout);
  expect(parsed.schema).toBe("c2.show.v1");
  expect(parsed.data.workout.id).toBe(1);
  expect(parsed.data.splits.length).toBe(2);
  expect(parsed.data.splits[0].pace_500m).toBe("1:16.3");
  expect(parsed.data.split_shape).toBe("negative");
  expect(parsed.data.stroke_summary.samples).toBe(2);
  expect(parsed.data.target_pace_500m_seconds).toBe(150);
  expect(parsed.data.raw.time).toBe(12000);
  expect(parsed.data.raw.workout.targets.pace).toBe(1500);
  expect(parsed.data.raw.workout.splits.length).toBe(2);
});

test("stats weekly/goal/splits/hr-pace emit versioned envelopes", () => {
  const weekly = run(["stats", "weekly", "-w", "3", "--json"]);
  expect(weekly.code).toBe(0);
  const weeklyParsed = JSON.parse(weekly.stdout);
  expect(weeklyParsed.schema).toBe("c2.stats.weekly.v1");
  expect(weeklyParsed.data.weeks.length).toBe(3);

  const goal = run(["stats", "goal", "--json"]);
  expect(goal.code).toBe(0);
  const goalParsed = JSON.parse(goal.stdout);
  expect(goalParsed.schema).toBe("c2.stats.goal.v1");
  expect(goalParsed.data.goal.totalMeters).toBe(14000);
  expect(goalParsed.data.projection.projected_total_meters).toBeGreaterThan(0);
  expect(goalParsed.data.this_week).toBeDefined();

  const splits = run(["stats", "splits", "1", "--json"]);
  expect(splits.code).toBe(0);
  const splitsParsed = JSON.parse(splits.stdout);
  expect(splitsParsed.schema).toBe("c2.stats.splits.v1");
  expect(splitsParsed.data.split_shape).toBe("negative");
  expect(splitsParsed.data.splits.length).toBe(2);

  const hr = run(["stats", "hr-pace", "-w", "4", "--json"]);
  expect(hr.code).toBe(0);
  const hrParsed = JSON.parse(hr.stdout);
  expect(hrParsed.schema).toBe("c2.stats.hr-pace.v1");
  expect(Array.isArray(hrParsed.data.bands)).toBe(true);
});

test("stats splits handles workouts without split data", () => {
  const r = run(["stats", "splits", "2"]);
  expect(r.code).toBe(0);
  expect(r.stdout).toContain("no split data");
});

test("note add/list/show round-trip through the CLI", () => {
  const added = run([
    "note",
    "add",
    "--type",
    "subjective",
    "--workout",
    "last",
    "--tags",
    "hr,ramp",
    "felt slow early, opened up late",
  ]);
  expect(added.code).toBe(0);
  const noteId = added.stdout.trim();
  expect(noteId.length).toBe(26);

  const list = run(["note", "list", "--json"]);
  const parsed = JSON.parse(list.stdout);
  expect(parsed.schema).toBe("c2.notes.v1");
  expect(parsed.data.count).toBe(1);
  expect(parsed.data.notes[0].workout_id).toBe(1);
  expect(parsed.data.notes[0].tags).toEqual(["hr", "ramp"]);

  const shown = run(["note", "show", noteId]);
  expect(shown.code).toBe(0);
  expect(shown.stdout).toContain("felt slow early");

  const filtered = run(["note", "list", "--type", "lesson"]);
  expect(filtered.stdout).toContain("No notes found");

  const badType = run(["note", "add", "--type", "vibes", "x"]);
  expect(badType.code).toBe(1);
  expect(badType.stderr).toContain("--type must be one of");

  const rollover = run(["note", "add", "--date", "2026-02-31", "x"]);
  expect(rollover.code).toBe(1);
  expect(rollover.stderr).toContain('invalid --date "2026-02-31"');

  const partial = run(["note", "add", "--date", "2026-07", "x"]);
  expect(partial.code).toBe(1);
  expect(partial.stderr).toContain('invalid --date "2026-07"');

  const badWorkout = run(["note", "list", "--workout", "banana"]);
  expect(badWorkout.code).toBe(1);
  expect(badWorkout.stderr).toContain('invalid --workout id "banana"');

  const badFilter = run(["note", "list", "--type", "vibes"]);
  expect(badFilter.code).toBe(1);
  expect(badFilter.stderr).toContain("--type must be one of");
});

test("failed workout links leave no store behind", async () => {
  const home8 = await mkdtemp(join(tmpdir(), "c2-cli-noworkout-"));
  await mkdir(join(home8, ".config", "c2"), { recursive: true });
  await writeFile(join(home8, ".config", "c2", "config.json"), JSON.stringify({}), "utf-8");

  const bad = run(["note", "add", "--workout", "999", "orphan note"], { home: home8 });
  expect(bad.code).toBe(1);
  expect(bad.stderr).toContain('no workout matching "999"');

  const info = run(["data", "info"], { home: home8 });
  expect(info.code).toBe(1);
  expect(info.stderr).toContain("No data store");
});

test("coaching reads reject foreign directories", async () => {
  const home9 = await mkdtemp(join(tmpdir(), "c2-cli-foreignread-"));
  await mkdir(join(home9, ".config", "c2"), { recursive: true });
  const foreignDir = join(home9, "someones-docs");
  await mkdir(foreignDir);
  await writeFile(join(foreignDir, "plan.md"), "# someone else's plan\n", "utf-8");
  await writeFile(join(foreignDir, "novel.docx"), "chapter one", "utf-8");
  await writeFile(
    join(home9, ".config", "c2", "config.json"),
    JSON.stringify({ data_dir: foreignDir }),
    "utf-8",
  );

  const planShow = run(["plan", "show"], { home: home9 });
  expect(planShow.code).toBe(1);
  expect(planShow.stderr).toContain("not a c2 data store");
  expect(planShow.stdout).not.toContain("someone else's plan");

  const noteList = run(["note", "list"], { home: home9 });
  expect(noteList.code).toBe(1);
  expect(noteList.stderr).toContain("not a c2 data store");

  await writeFile(join(foreignDir, "workouts.jsonl"), "{ not json at all\n", "utf-8");
  const linked = run(["note", "add", "--workout", "1", "x"], { home: home9 });
  expect(linked.code).toBe(1);
  expect(linked.stderr).toContain("not a c2 data store");
});

test("invalid --date rejects before any store side effects", async () => {
  const home7 = await mkdtemp(join(tmpdir(), "c2-cli-nodate-"));
  await mkdir(join(home7, ".config", "c2"), { recursive: true });
  await writeFile(join(home7, ".config", "c2", "config.json"), JSON.stringify({}), "utf-8");

  const bad = run(["note", "add", "--date", "2026-02-31", "x"], { home: home7 });
  expect(bad.code).toBe(1);
  expect(bad.stderr).toContain('invalid --date "2026-02-31"');

  const info = run(["data", "info"], { home: home7 });
  expect(info.code).toBe(1);
  expect(info.stderr).toContain("No data store");
});

test("first coaching write initializes a proper store", async () => {
  const home5 = await mkdtemp(join(tmpdir(), "c2-cli-first-write-"));
  await mkdir(join(home5, ".config", "c2"), { recursive: true });
  await writeFile(join(home5, ".config", "c2", "config.json"), JSON.stringify({}), "utf-8");

  const planFile = join(home5, "p.md");
  await writeFile(planFile, "# Plan\n", "utf-8");
  expect(run(["plan", "set", planFile], { home: home5 }).code).toBe(0);

  const info = run(["data", "info", "--json"], { home: home5 });
  expect(info.code).toBe(0);
  const parsed = JSON.parse(info.stdout);
  expect(parsed.data.state).toBe("store");
  expect(parsed.data.schema_version).toBe(1);

  expect(run(["note", "add", "first note ever"], { home: home5 }).code).toBe(0);
  expect(run(["data", "doctor"], { home: home5 }).code).toBe(0);
});

test("note add links into show output", () => {
  const shown = run(["show", "1"]);
  expect(shown.stdout).toContain("Notes:");
  expect(shown.stdout).toContain("felt slow early");

  const json = run(["show", "1", "--json"]);
  expect(JSON.parse(json.stdout).data.notes.length).toBe(1);
});

test("backdated notes and compaction via data compact", async () => {
  const old = run([
    "note",
    "add",
    "--date",
    ymd(daysFromNow(-30)),
    "--type",
    "lesson",
    "--author",
    "coach",
    "old lesson to archive",
  ]);
  expect(old.code).toBe(0);

  const compact = run(["data", "compact"]);
  expect(compact.code).toBe(0);
  expect(compact.stdout).toContain("Compacted 1 note");

  const list = run(["note", "list", "--json"]);
  expect(JSON.parse(list.stdout).data.count).toBe(2);

  const again = run(["data", "compact"]);
  expect(again.stdout).toContain("Nothing to compact");

  const doctor = run(["data", "doctor"]);
  expect(doctor.code).toBe(0);
  expect(doctor.stdout).toContain("no problems found");

  const info = run(["data", "info", "--json"]);
  expect(JSON.parse(info.stdout).data.notes).toBe(2);
});

test("plan, playbook, and narrative round-trip", async () => {
  const planFile = join(home, "plan-src.md");
  await writeFile(planFile, "# Plan\nEvery other day, 6K.\n", "utf-8");
  expect(run(["plan", "set", planFile]).code).toBe(0);
  const plan = run(["plan", "show"]);
  expect(plan.code).toBe(0);
  expect(plan.stdout).toContain("Every other day");

  const missing = run(["playbook", "show"]);
  expect(missing.code).toBe(1);
  expect(missing.stderr).toContain("No playbook recorded");

  const narrFile = join(home, "narr.md");
  await writeFile(narrFile, "Solid week.\n", "utf-8");
  expect(run(["narrative", "add", RECENT, narrFile]).code).toBe(0);
  expect(run(["narrative", "show"]).stdout).toContain("Solid week");
  expect(run(["narrative", "show", RECENT]).stdout).toContain("Solid week");
  const narrList = run(["narrative", "list", "--json"]);
  expect(JSON.parse(narrList.stdout).data.dates).toEqual([RECENT]);

  const badDate = run(["narrative", "add", "not-a-date", narrFile]);
  expect(badDate.code).toBe(1);
});

test("data doctor reports corruption", async () => {
  const info = run(["data", "info", "--json"]);
  const root = JSON.parse(info.stdout).data.root;
  await writeFile(join(root, "notes", "ZZBAD.json"), "{ nope", "utf-8");
  const doctor = run(["data", "doctor"]);
  expect(doctor.code).toBe(1);
  expect(doctor.stderr).toContain("ZZBAD.json: not valid JSON");
  await rm(join(root, "notes", "ZZBAD.json"));
  expect(run(["data", "doctor"]).code).toBe(0);
});

test("export rejects invalid dates but allows empty ranges", () => {
  const bad = run(["export", "--from", "not-a-date"]);
  expect(bad.code).toBe(1);
  expect(bad.stderr).toContain("invalid --from date");

  const rollover = run(["export", "--from", "2026-02-31"]);
  expect(rollover.code).toBe(1);
  expect(rollover.stderr).toContain("invalid --from date");

  const empty = run(["export", "--from", FUTURE, "-f", "json"]);
  expect(empty.code).toBe(0);
  expect(empty.stderr).toContain("No workouts match");
  expect(JSON.parse(empty.stdout)).toEqual([]);

  const emptyCSV = run(["export", "--from", FUTURE]);
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
      goal: {
        target_meters: 1_000_000,
        start_date: ymd(daysFromNow(-180)),
        end_date: ymd(daysFromNow(180)),
      },
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
  expect(parsed.data.first_date).toBe(OLDER);
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

test("foreign data_dir gets clean errors, not raw failures", async () => {
  const home3 = await mkdtemp(join(tmpdir(), "c2-cli-foreign-"));
  await mkdir(join(home3, ".config", "c2"), { recursive: true });
  const filePath = join(home3, "not-a-dir");
  await writeFile(filePath, "regular file", "utf-8");
  await writeFile(
    join(home3, ".config", "c2", "config.json"),
    JSON.stringify({ data_dir: filePath, api: { token: "tok" } }),
    "utf-8",
  );

  const info = run(["data", "info"], { home: home3 });
  expect(info.code).toBe(1);
  expect(info.stderr).toContain("not a c2 data store");

  const move = run(["data", "move", join(home3, "elsewhere")], { home: home3 });
  expect(move.code).toBe(1);
  expect(move.stderr).toContain("not a c2 data store");

  const sync = run(["sync"], { home: home3 });
  expect(sync.code).toBe(1);
  expect(sync.stderr).toContain("not a c2 data store");

  const noteAdd = run(["note", "add", "should not land here"], { home: home3 });
  expect(noteAdd.code).toBe(1);
  expect(noteAdd.stderr).toContain("not a c2 data store");

  const compact = run(["data", "compact"], { home: home3 });
  expect(compact.code).toBe(1);
  expect(compact.stderr).toContain("nothing to compact");

  await writeFile(
    join(home3, ".config", "c2", "config.json"),
    JSON.stringify({ data_dir: join(filePath, "nested"), api: { token: "tok" } }),
    "utf-8",
  );
  const nested = run(["data", "info"], { home: home3 });
  expect(nested.code).toBe(1);
  expect(nested.stderr).toContain("not a c2 data store");

  const readThrough = run(["log"], { home: home3 });
  expect(readThrough.code).toBe(0);
  expect(readThrough.stdout).toContain("No workouts found");

  const emptyDir = join(home3, "empty-dir");
  await mkdir(emptyDir);
  await writeFile(
    join(home3, ".config", "c2", "config.json"),
    JSON.stringify({ data_dir: emptyDir, api: { token: "tok" } }),
    "utf-8",
  );
  const emptyInfo = run(["data", "info"], { home: home3 });
  expect(emptyInfo.code).toBe(1);
  expect(emptyInfo.stderr).toContain("empty directory");

  const emptyDoctor = run(["data", "doctor"], { home: home3 });
  expect(emptyDoctor.code).toBe(1);
  expect(emptyDoctor.stderr).toContain("No data store");

  await writeFile(
    join(home3, ".config", "c2", "config.json"),
    JSON.stringify({ data_dir: 123, api: { token: "tok" } }),
    "utf-8",
  );
  const badType = run(["data", "info"], { home: home3 });
  expect(badType.code).toBe(1);
  expect(badType.stderr).toContain(join(".config", "c2", "data"));
});
