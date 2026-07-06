import type { Command } from "commander";
import { loadConfig, saveConfig } from "../config.ts";
import { inspectDataDir, moveStore, storeSummary } from "../data.ts";
import { formatMeters } from "../display.ts";
import { printJSON } from "../envelope.ts";
import { dataPaths, pathsFor } from "../paths.ts";

export function registerData(program: Command): void {
  const data = program.command("data").description("Manage the c2 data store");

  data
    .command("info")
    .description("Show data store location and contents")
    .option("--json", "output as JSON")
    .action(async (opts: { json?: boolean }) => {
      const cfg = await loadConfig();
      const paths = dataPaths(cfg);
      const inspection = await inspectDataDir(paths);

      if (inspection.state === "missing") {
        console.error(`No data store at ${paths.root}. Run \`c2 setup\` or \`c2 sync\` first.`);
        process.exit(1);
      }

      const summary = await storeSummary(paths);
      const lastSync = summary.lastSync ?? cfg.sync.last_sync ?? null;
      if (opts.json) {
        printJSON("c2.data.info.v1", {
          root: paths.root,
          state: inspection.state,
          writable: inspection.writable,
          schema_version: summary.schemaVersion,
          last_sync: lastSync,
          workouts: summary.workouts,
          first_date: summary.firstDate || null,
          last_date: summary.lastDate || null,
          stroke_files: summary.strokeFiles,
          notes: summary.notes,
        });
        return;
      }

      console.log(`Data store: ${paths.root}`);
      console.log(`Schema version: ${summary.schemaVersion ?? "(no meta.json — legacy store)"}`);
      console.log(`Last sync: ${lastSync ?? "never"}`);
      console.log(
        `Workouts: ${formatMeters(summary.workouts)}${summary.firstDate ? ` (${summary.firstDate} → ${summary.lastDate})` : ""}`,
      );
      console.log(`Stroke files: ${formatMeters(summary.strokeFiles)}`);
      console.log(`Notes: ${formatMeters(summary.notes)}`);
    });

  data
    .command("move <dir>")
    .description("Relocate the data store and update config")
    .action(async (dir: string) => {
      const cfg = await loadConfig();
      const from = dataPaths(cfg);
      const source = await inspectDataDir(from);
      if (source.state === "missing") {
        console.error(`No data store at ${from.root}; nothing to move.`);
        process.exit(1);
      }

      const to = pathsFor(dir);
      if (to.root === from.root) {
        console.error("Target is the current data directory.");
        process.exit(1);
      }

      try {
        const copied = await moveStore(from, to);
        cfg.data_dir = dir;
        await saveConfig(cfg);
        console.log(
          `Copied ${copied.files} files (${formatMeters(copied.bytes)} bytes) to ${to.root}`,
        );
        console.log(`Config updated: data_dir = ${dir}`);
        console.log(`Old data left at ${from.root} — remove it manually once satisfied.`);
      } catch (err) {
        console.error(`Error: ${(err as Error).message}`);
        process.exit(1);
      }
    });
}
