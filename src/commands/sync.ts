import type { Command } from "commander";
import type { Workout } from "../models.ts";
import { loadConfig, ensureDirs, saveConfig } from "../config.ts";
import { C2Client } from "../api/client.ts";
import {
  appendWorkouts,
  workoutCount,
  hasStrokeData,
  writeStrokeData,
} from "../storage.ts";

async function syncStrokes(
  client: C2Client,
  workouts: Workout[],
): Promise<number> {
  let count = 0;
  for (const w of workouts) {
    if (!w.stroke_data || (await hasStrokeData(w.id))) continue;
    try {
      const strokes = await client.getStrokes(w.id);
      if (strokes.length > 0) {
        await writeStrokeData(w.id, strokes);
        count++;
      }
    } catch (err) {
      console.error(
        `Warning: failed to fetch strokes for workout ${w.id}: ${err}`,
      );
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
      await ensureDirs();

      const client = C2Client.fromConfig(cfg);
      const from = cfg.sync.last_sync ?? "";

      if (from) {
        console.log(`Syncing workouts since ${from}...`);
      } else {
        console.log("First sync \u2014 pulling all workouts...");
      }

      const workouts = await client.getAllResults(from, "");
      const written = await appendWorkouts(workouts);
      console.log(`Fetched ${workouts.length} workouts, ${written} new.`);

      const strokeCount = await syncStrokes(client, workouts);
      if (strokeCount > 0) {
        console.log(`Fetched stroke data for ${strokeCount} workouts.`);
      }

      cfg.sync.last_sync = new Date().toISOString().replace(/\.\d{3}Z$/, "Z");
      await saveConfig(cfg);

      const total = await workoutCount();
      console.log(`Total workouts: ${total}`);
    });
}
