import { mkdir, readFile, writeFile } from "node:fs/promises";
import { homedir } from "node:os";
import { join } from "node:path";

export interface Config {
  api: { base_url: string; token: string };
  sync: { last_sync?: string; machine_type: string };
  goal: { target_meters: number; start_date: string; end_date: string };
  display: { date_format: string };
}

export function defaultConfig(): Config {
  return {
    api: { base_url: "https://log.concept2.com", token: "" },
    sync: { machine_type: "rower" },
    goal: { target_meters: 1_000_000, start_date: "", end_date: "" },
    display: { date_format: "%m/%d" },
  };
}

export function configDir(): string {
  return join(homedir(), ".config", "c2");
}

export function dataDir(): string {
  return join(configDir(), "data");
}

export async function ensureDirs(): Promise<void> {
  await mkdir(join(dataDir(), "strokes"), { recursive: true });
}

export async function loadConfig(): Promise<Config> {
  const path = join(configDir(), "config.json");
  const defaults = defaultConfig();
  try {
    const text = await readFile(path, "utf-8");
    const parsed = JSON.parse(text) as Partial<Config>;
    return {
      api: { ...defaults.api, ...parsed.api },
      sync: { ...defaults.sync, ...parsed.sync },
      goal: { ...defaults.goal, ...parsed.goal },
      display: { ...defaults.display, ...parsed.display },
    };
  } catch (err: unknown) {
    if ((err as NodeJS.ErrnoException).code === "ENOENT") {
      return defaults;
    }
    throw err;
  }
}

export async function saveConfig(cfg: Config): Promise<void> {
  const path = join(configDir(), "config.json");
  await mkdir(configDir(), { recursive: true });
  const text = JSON.stringify(cfg, null, 2);
  await writeFile(path, `${text}\n`, "utf-8");
}

export function parseGoalDate(s: string): Date {
  const d = new Date(`${s}T00:00:00`);
  if (Number.isNaN(d.getTime())) throw new Error(`Invalid date: ${s}`);
  return d;
}
