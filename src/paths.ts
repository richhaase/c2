import { realpath } from "node:fs/promises";
import { homedir } from "node:os";
import { basename, dirname, join, resolve } from "node:path";
import type { Config } from "./config.ts";

export interface DataPaths {
  root: string;
  meta: string;
  workouts: string;
  strokesDir: string;
  strokeFile(id: number): string;
  notesDir: string;
  archiveDir: string;
  archiveFile(year: number): string;
  plan: string;
  playbook: string;
  reportsDir: string;
  narrativeFile(date: string): string;
}

export function expandTilde(p: string): string {
  if (p === "~") return homedir();
  if (p.startsWith("~/")) return join(homedir(), p.slice(2));
  return p;
}

export function pathsFor(root: string): DataPaths {
  const abs = resolve(expandTilde(root));
  return {
    root: abs,
    meta: join(abs, "meta.json"),
    workouts: join(abs, "workouts.jsonl"),
    strokesDir: join(abs, "strokes"),
    strokeFile: (id: number) => join(abs, "strokes", `${id}.jsonl`),
    notesDir: join(abs, "notes"),
    archiveDir: join(abs, "notes", "archive"),
    archiveFile: (year: number) => join(abs, "notes", "archive", `${year}.jsonl`),
    plan: join(abs, "plan.md"),
    playbook: join(abs, "playbook.md"),
    reportsDir: join(abs, "reports"),
    narrativeFile: (date: string) => join(abs, "reports", `${date}.md`),
  };
}

export function dataPaths(cfg: Config): DataPaths {
  return pathsFor(cfg.data_dir);
}

export async function canonicalRoot(p: string): Promise<string> {
  let base = resolve(expandTilde(p));
  const rest: string[] = [];
  for (;;) {
    try {
      const real = await realpath(base);
      return rest.length > 0 ? join(real, ...rest) : real;
    } catch (err: unknown) {
      if ((err as NodeJS.ErrnoException).code !== "ENOENT") {
        return rest.length > 0 ? join(base, ...rest) : base;
      }
      rest.unshift(basename(base));
      const parent = dirname(base);
      if (parent === base) return resolve(expandTilde(p));
      base = parent;
    }
  }
}
