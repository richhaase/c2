import { parse, stringify } from "smol-toml";
import { homedir } from "node:os";
import { join } from "node:path";
import { mkdir, readFile, writeFile } from "node:fs/promises";

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
    display: { date_format: "01/02" },
  };
}

export function configDir(): string {
  return join(homedir(), ".config", "c2cli");
}

export function dataDir(): string {
  return join(configDir(), "data");
}

export async function ensureDirs(): Promise<void> {
  await mkdir(join(dataDir(), "strokes"), { recursive: true });
}

export async function loadConfig(): Promise<Config> {
  const path = join(configDir(), "config.toml");
  const defaults = defaultConfig();
  try {
    const text = await readFile(path, "utf-8");
    const parsed = parse(text) as Record<string, Record<string, unknown>>;
    return {
      api: { ...defaults.api, ...(parsed["api"] as Partial<Config["api"]>) },
      sync: {
        ...defaults.sync,
        ...(parsed["sync"] as Partial<Config["sync"]>),
      },
      goal: {
        ...defaults.goal,
        ...(parsed["goal"] as Partial<Config["goal"]>),
      },
      display: {
        ...defaults.display,
        ...(parsed["display"] as Partial<Config["display"]>),
      },
    };
  } catch (err: unknown) {
    if ((err as NodeJS.ErrnoException).code === "ENOENT") {
      return defaults;
    }
    throw err;
  }
}

export async function saveConfig(cfg: Config): Promise<void> {
  const path = join(configDir(), "config.toml");
  await mkdir(configDir(), { recursive: true });
  const text = stringify(cfg as unknown as Record<string, unknown>);
  await writeFile(path, text, "utf-8");
}

export function parseGoalDate(s: string): Date {
  const d = new Date(s + "T00:00:00");
  if (isNaN(d.getTime())) throw new Error(`Invalid date: ${s}`);
  return d;
}
