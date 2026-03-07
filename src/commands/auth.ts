import type { Command } from "commander";
import { loadConfig, defaultConfig, ensureDirs, saveConfig } from "../config.ts";
import { C2Client } from "../api/client.ts";

export function registerAuth(program: Command): void {
  program
    .command("auth <token>")
    .description("Save access token and verify")
    .action(async (token: string) => {
      let cfg;
      try {
        cfg = await loadConfig();
      } catch {
        cfg = defaultConfig();
      }
      cfg.api.token = token;

      await ensureDirs();
      await saveConfig(cfg);
      console.log("Token saved.");

      const client = C2Client.fromConfig(cfg);
      const user = await client.getUser();
      console.log(`Authenticated as: ${user.username} (ID: ${user.id})`);
    });
}
