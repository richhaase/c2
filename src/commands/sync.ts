import type { Command } from "commander";
import { C2Client } from "../api/client.ts";
import { loadConfig } from "../config.ts";
import { initStore, inspectDataDir } from "../data.ts";
import type { Workout } from "../models.ts";
import { compactNotes } from "../notes.ts";
import type { DataPaths } from "../paths.ts";
import { dataPaths } from "../paths.ts";
import {
  appendWorkouts,
  hasStrokeData,
  readMeta,
  SCHEMA_VERSION,
  workoutCount,
  writeMeta,
  writeStrokeData,
} from "../storage.ts";

async function syncStrokes(
  client: C2Client,
  paths: DataPaths,
  workouts: Workout[],
): Promise<number> {
  let count = 0;
  let failures = 0;
  for (const w of workouts) {
    if (!w.stroke_data || (await hasStrokeData(paths, w.id))) continue;
    try {
      const strokes = await client.getStrokes(w.id);
      if (strokes.length > 0) {
        await writeStrokeData(paths, w.id, strokes);
        count++;
      }
    } catch (err) {
      failures++;
      console.error(
        `Warning: failed to fetch strokes for workout ${w.id}: ${(err as Error).message}`,
      );
      if (failures >= 3) {
        console.error("Too many failures, skipping remaining stroke data.");
        break;
      }
    }
  }
  return count;
}

export function registerSync(program: Command): void {
  program
    .command("sync")
    .description("Pull new workouts from the API")
    .action(async () => {
      const cfg = await loadConfig();
      if (!cfg.api.token) {
        console.error("No API token configured. Run `c2 setup` first.");
        process.exit(1);
      }
      const paths = dataPaths(cfg);
      const inspection = await inspectDataDir(paths);
      if (inspection.state === "foreign") {
        console.error(
          `${paths.root} exists but is not a c2 data store. Fix data_dir via \`c2 setup\`.`,
        );
        process.exit(1);
      }
      if (!inspection.writable) {
        console.error(`Cannot write to ${paths.root}.`);
        process.exit(1);
      }
      const now = new Date();
      await initStore(paths, now);

      const client = C2Client.fromConfig(cfg);
      const meta = await readMeta(paths);
      let from = meta?.last_sync ?? "";
      if (!from && cfg.sync.last_sync && (await workoutCount(paths)) > 0) {
        from = cfg.sync.last_sync;
      }

      if (from) {
        console.log(`Syncing workouts since ${from}...`);
      } else {
        console.log("First sync — pulling all workouts...");
      }

      const workouts = await client.getAllResults(from, "");
      const written = await appendWorkouts(paths, workouts);
      console.log(`Fetched ${workouts.length} workouts, ${written} new.`);

      const strokeCount = await syncStrokes(client, paths, workouts);
      if (strokeCount > 0) {
        console.log(`Fetched stroke data for ${strokeCount} workouts.`);
      }

      await writeMeta(paths, {
        schema_version: meta?.schema_version ?? SCHEMA_VERSION,
        created: meta?.created ?? now.toISOString(),
        last_sync: new Date().toISOString().replace(/\.\d{3}Z$/, "Z"),
      });

      const compacted = await compactNotes(paths, now);
      for (const year of compacted.skippedYears) {
        console.error(
          `Warning: notes/archive/${year}.jsonl has corrupt lines; left untouched (run \`c2 data doctor\`).`,
        );
      }
      if (compacted.archived > 0) {
        console.log(
          `Compacted ${compacted.archived} note${compacted.archived === 1 ? "" : "s"} into ${compacted.years.map((y) => `${y}.jsonl`).join(", ")}.`,
        );
      }

      const total = await workoutCount(paths);
      console.log(`Total workouts: ${total}`);
    });
}
