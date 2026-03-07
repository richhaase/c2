import type { Command } from "commander";
import {
  loadConfig,
  defaultConfig,
  ensureDirs,
  saveConfig,
  configDir,
  parseGoalDate,
} from "../config.ts";
import { formatMeters } from "../display.ts";
import { C2Client } from "../api/client.ts";
import { join } from "node:path";

function promptValue(label: string, current: string): string {
  const display = current ? ` [${current}]` : "";
  const input = prompt(`${label}${display}:`);
  if (input === null) {
    console.log("\nSetup cancelled.");
    process.exit(0);
  }
  return input.trim() || current;
}

export function registerSetup(program: Command): void {
  program
    .command("setup")
    .description("Configure token, goal, and preferences")
    .action(async () => {
      let cfg;
      try {
        cfg = await loadConfig();
      } catch (err) {
        console.error(`Warning: could not load existing config: ${(err as Error).message}`);
        console.error("Starting from defaults.");
        cfg = defaultConfig();
      }

      console.log("Concept2 CLI Setup\n");

      // Token
      const token = promptValue(
        "API token (from log.concept2.com)",
        cfg.api.token,
      );
      cfg.api.token = token;

      // Goal target
      const targetDisplay = formatMeters(cfg.goal.target_meters);
      const targetInput = promptValue("Goal target meters", targetDisplay);
      const parsed = parseInt(targetInput.replace(/,/g, ""), 10);
      if (!isNaN(parsed) && parsed > 0) {
        cfg.goal.target_meters = parsed;
      }

      // Goal dates
      const startInput = promptValue("Goal start date (YYYY-MM-DD)", cfg.goal.start_date);
      try {
        parseGoalDate(startInput);
        cfg.goal.start_date = startInput;
      } catch {
        console.log(`Invalid date "${startInput}", keeping previous value.`);
      }

      const endInput = promptValue("Goal end date (YYYY-MM-DD)", cfg.goal.end_date);
      try {
        parseGoalDate(endInput);
        cfg.goal.end_date = endInput;
      } catch {
        console.log(`Invalid date "${endInput}", keeping previous value.`);
      }

      // Save
      await ensureDirs();
      await saveConfig(cfg);
      console.log(`\nConfig written to ${join(configDir(), "config.toml")}`);

      if (!cfg.goal.start_date || !cfg.goal.end_date) {
        console.log("\nNote: Goal dates not set. Commands like `c2 status` require start/end dates.");
      }

      // Verify token
      if (cfg.api.token) {
        console.log("Verifying token...");
        try {
          const client = C2Client.fromConfig(cfg);
          const user = await client.getUser();
          console.log(`Authenticated as: ${user.username} (ID: ${user.id})`);
        } catch (err) {
          console.error(
            `Warning: could not verify token: ${(err as Error).message}`,
          );
        }
      }
    });
}
