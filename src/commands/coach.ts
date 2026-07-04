import type { Command } from "commander";
import { runTurn } from "../ai/agent.ts";
import { type ChatMessage, OpenRouterClient } from "../ai/client.ts";
import { readNotes, readProfile } from "../ai/memory.ts";
import { buildSystemPrompt } from "../ai/prompt.ts";
import { buildTools } from "../ai/tools.ts";
import { C2Client } from "../api/client.ts";
import { loadConfig } from "../config.ts";
import { readWorkouts } from "../storage.ts";

export function registerCoach(program: Command): void {
  program
    .command("coach")
    .description("Start an interactive AI coaching session over your rowing data")
    .action(async () => {
      const cfg = await loadConfig();
      if (!cfg.ai.api_key) {
        console.error("AI coaching is not configured. Run `c2 setup` to add an OpenRouter key.");
        process.exit(1);
      }
      if (!cfg.goal.start_date || !cfg.goal.end_date) {
        console.error("Goal dates not configured. Run `c2 setup` first.");
        process.exit(1);
      }

      const workouts = await readWorkouts();
      if (workouts.length === 0) {
        console.log("No workouts found. Run `c2 sync` first.");
        return;
      }

      const model = cfg.ai.model;
      const client = new OpenRouterClient(cfg.ai.base_url, cfg.ai.api_key, model);
      const api = C2Client.fromConfig(cfg);
      const now = new Date();

      const profile = await readProfile();
      const notes = await readNotes(50);
      const { defs, dispatch } = buildTools({ cfg, workouts, api, now }, (msg) =>
        console.log(`  · ${msg}`),
      );

      const messages: ChatMessage[] = [
        { role: "system", content: buildSystemPrompt({ cfg, workouts, profile, notes, now }) },
      ];

      console.log(`c2 coach — ${model}`);
      console.log(
        "Ask about your training, goals, or a specific piece. Ctrl-D or 'exit' to quit.\n",
      );

      for (;;) {
        const input = prompt("you> ");
        if (input == null) {
          console.log();
          break;
        }
        const trimmed = input.trim();
        if (trimmed === "") continue;
        if (trimmed === "exit" || trimmed === "quit") break;

        messages.push({ role: "user", content: trimmed });
        try {
          const reply = await runTurn(messages, { client, tools: defs, dispatch });
          console.log(`\ncoach> ${reply}\n`);
        } catch (err) {
          console.error(`\nError: ${(err as Error).message}\n`);
        }
      }
    });
}
