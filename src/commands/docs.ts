import { mkdir, readdir, readFile, writeFile } from "node:fs/promises";
import { dirname } from "node:path";
import type { Command } from "commander";
import { loadConfig } from "../config.ts";
import { printJSON } from "../envelope.ts";
import { isValidYMD } from "../models.ts";
import type { DataPaths } from "../paths.ts";
import { dataPaths } from "../paths.ts";

async function readContent(source: string | undefined): Promise<string> {
  if (source != null && source !== "-") {
    return readFile(source, "utf-8");
  }
  return new Response(Bun.stdin.stream()).text();
}

async function readIfExists(path: string): Promise<string | null> {
  try {
    return await readFile(path, "utf-8");
  } catch (err: unknown) {
    const code = (err as NodeJS.ErrnoException).code;
    if (code === "ENOENT" || code === "ENOTDIR") return null;
    throw err;
  }
}

function registerDoc(
  program: Command,
  name: "plan" | "playbook",
  description: string,
  pathOf: (paths: DataPaths) => string,
): void {
  const doc = program.command(name).description(description);

  doc
    .command("show")
    .description(`Print the ${name}`)
    .action(async () => {
      const cfg = await loadConfig();
      const content = await readIfExists(pathOf(dataPaths(cfg)));
      if (content == null) {
        console.error(`No ${name} recorded yet. Set one with \`c2 ${name} set <file|->\`.`);
        process.exit(1);
      }
      process.stdout.write(content);
    });

  doc
    .command("set [file]")
    .description(`Replace the ${name} from a file or stdin`)
    .action(async (file: string | undefined) => {
      const content = await readContent(file);
      if (content.trim() === "") {
        console.error(`Error: refusing to save an empty ${name}.`);
        process.exit(1);
      }
      const cfg = await loadConfig();
      const target = pathOf(dataPaths(cfg));
      await mkdir(dirname(target), { recursive: true });
      await writeFile(target, content.endsWith("\n") ? content : `${content}\n`, "utf-8");
      console.log(`${name} updated (${content.length} chars).`);
    });
}

export function registerDocs(program: Command): void {
  registerDoc(program, "plan", "Training plan (managed document)", (p) => p.plan);
  registerDoc(
    program,
    "playbook",
    "Coaching knowledge playbook (managed document)",
    (p) => p.playbook,
  );

  const narrative = program.command("narrative").description("Dated coaching report narratives");

  narrative
    .command("add <date> [file]")
    .description("Save the narrative for a date (YYYY-MM-DD) from a file or stdin")
    .action(async (date: string, file: string | undefined) => {
      if (!isValidYMD(date)) {
        console.error(`Error: invalid date "${date}" (expected YYYY-MM-DD).`);
        process.exit(1);
      }
      const content = await readContent(file);
      if (content.trim() === "") {
        console.error("Error: refusing to save an empty narrative.");
        process.exit(1);
      }
      const cfg = await loadConfig();
      const paths = dataPaths(cfg);
      await mkdir(paths.reportsDir, { recursive: true });
      await writeFile(
        paths.narrativeFile(date),
        content.endsWith("\n") ? content : `${content}\n`,
        "utf-8",
      );
      console.log(`Narrative saved for ${date}.`);
    });

  narrative
    .command("show [date]")
    .description("Print the narrative for a date (latest if omitted)")
    .action(async (date: string | undefined) => {
      const cfg = await loadConfig();
      const paths = dataPaths(cfg);
      let target = date;
      if (target != null && !isValidYMD(target)) {
        console.error(`Error: invalid date "${target}" (expected YYYY-MM-DD).`);
        process.exit(1);
      }
      if (target == null) {
        const dates = await listNarratives(paths);
        target = dates[dates.length - 1];
        if (target == null) {
          console.error("No narratives recorded yet.");
          process.exit(1);
        }
      }
      const content = await readIfExists(paths.narrativeFile(target));
      if (content == null) {
        console.error(`No narrative for ${target}.`);
        process.exit(1);
      }
      process.stdout.write(content);
    });

  narrative
    .command("list")
    .description("List narrative dates")
    .option("--json", "output as JSON")
    .action(async (opts: { json?: boolean }) => {
      const cfg = await loadConfig();
      const dates = await listNarratives(dataPaths(cfg));
      if (opts.json) {
        printJSON("c2.narratives.v1", { count: dates.length, dates });
        return;
      }
      if (dates.length === 0) {
        console.log("No narratives recorded yet.");
        return;
      }
      for (const d of dates) {
        console.log(d);
      }
    });
}

async function listNarratives(paths: DataPaths): Promise<string[]> {
  try {
    return (await readdir(paths.reportsDir))
      .filter((f) => f.endsWith(".md"))
      .map((f) => f.slice(0, -3))
      .filter((d) => isValidYMD(d))
      .sort();
  } catch {
    return [];
  }
}
