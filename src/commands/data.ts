import { sep } from "node:path";
import type { Command } from "commander";
import { loadConfig, saveConfig } from "../config.ts";
import { inspectDataDir, moveStore, storeSummary } from "../data.ts";
import { formatMeters } from "../display.ts";
import { runDoctor } from "../doctor.ts";
import { printJSON } from "../envelope.ts";
import { compactNotes } from "../notes.ts";
import { canonicalRoot, dataPaths, pathsFor } from "../paths.ts";

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
      if (inspection.state === "foreign") {
        console.error(
          `${paths.root} exists but is not a c2 data store. Fix data_dir via \`c2 setup\`.`,
        );
        process.exit(1);
      }
      if (inspection.state === "empty") {
        console.error(
          `${paths.root} is an empty directory, not yet a data store. Run \`c2 sync\` to initialize it.`,
        );
        process.exit(1);
      }

      const summary = await storeSummary(paths);
      const legacyStore = summary.schemaVersion == null && summary.workouts > 0;
      const lastSync = summary.lastSync ?? (legacyStore ? (cfg.sync.last_sync ?? null) : null);
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
    .command("compact")
    .description("Archive notes older than 7 days into yearly files")
    .action(async () => {
      const cfg = await loadConfig();
      const paths = dataPaths(cfg);
      const inspection = await inspectDataDir(paths);
      if (inspection.state !== "store") {
        console.error(`${paths.root} is not a c2 data store; nothing to compact.`);
        process.exit(1);
      }
      if (!inspection.writable) {
        console.error(`Cannot write to ${paths.root}.`);
        process.exit(1);
      }
      const result = await compactNotes(paths, new Date());
      for (const year of result.skippedYears) {
        console.error(
          `Warning: notes/archive/${year}.jsonl could not be safely rewritten; notes left loose (run \`c2 data doctor\`).`,
        );
      }
      if (result.archived > 0) {
        console.log(
          `Compacted ${result.archived} note${result.archived === 1 ? "" : "s"} into ${result.years.map((y) => `notes/archive/${y}.jsonl`).join(", ")}.`,
        );
      } else if (result.skippedYears.length === 0) {
        console.log("Nothing to compact.");
      }
      if (result.skippedYears.length > 0) {
        process.exit(1);
      }
    });

  data
    .command("doctor")
    .description("Validate the data store and report problems")
    .action(async () => {
      const cfg = await loadConfig();
      const paths = dataPaths(cfg);
      const inspection = await inspectDataDir(paths);
      if (inspection.state !== "store") {
        console.error(`No data store at ${paths.root}.`);
        process.exit(1);
      }
      const report = await runDoctor(paths);
      if (report.issues.length === 0) {
        console.log(`OK — ${report.checkedFiles} files checked, no problems found.`);
        return;
      }
      console.error(
        `${report.issues.length} problem${report.issues.length === 1 ? "" : "s"} found:`,
      );
      for (const issue of report.issues) {
        console.error(`  - ${issue}`);
      }
      process.exit(1);
    });

  data
    .command("move <dir>")
    .description("Relocate the data store and update config")
    .action(async (dir: string) => {
      const cfg = await loadConfig();
      const from = pathsFor(await canonicalRoot(cfg.data_dir));
      const source = await inspectDataDir(from);
      if (source.state !== "store") {
        console.error(`${from.root} is not a c2 data store; nothing to move.`);
        process.exit(1);
      }

      const to = pathsFor(await canonicalRoot(dir));
      if (to.root === from.root) {
        console.error("Target is the current data directory.");
        process.exit(1);
      }
      if (to.root.startsWith(from.root + sep) || from.root.startsWith(to.root + sep)) {
        console.error("Target must not be inside the current data directory (or contain it).");
        process.exit(1);
      }

      let copied: { files: number; bytes: number };
      try {
        copied = await moveStore(from, to);
      } catch (err) {
        console.error(`Error: ${(err as Error).message}`);
        process.exit(1);
      }
      try {
        cfg.data_dir = to.root;
        await saveConfig(cfg);
      } catch (err) {
        console.error(`Error: copy completed but config update failed: ${(err as Error).message}`);
        console.error(
          `Set data_dir to ${to.root} in ~/.config/c2/config.json manually, or remove ${to.root} and retry.`,
        );
        process.exit(1);
      }
      console.log(
        `Copied ${copied.files} files (${formatMeters(copied.bytes)} bytes) to ${to.root}`,
      );
      console.log(`Config updated: data_dir = ${to.root}`);
      console.log(`Old data left at ${from.root} — remove it manually once satisfied.`);
    });
}
