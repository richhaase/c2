import type { Command } from "commander";
import { C2Client } from "../api/client.ts";
import { loadConfig } from "../config.ts";
import { initStore } from "../data.ts";
import type { Workout } from "../models.ts";
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
      const now = new Date();
      await initStore(paths, now);

      const client = C2Client.fromConfig(cfg);
      const meta = await readMeta(paths);
      const from = meta?.last_sync ?? cfg.sync.last_sync ?? "";

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

      const total = await workoutCount(paths);
      console.log(`Total workouts: ${total}`);
    });
}
