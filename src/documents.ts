import { readdir, readFile } from "node:fs/promises";
import { isValidYMD } from "./models.ts";
import type { DataPaths } from "./paths.ts";

export async function readDocument(path: string): Promise<string | null> {
  try {
    return await readFile(path, "utf-8");
  } catch (err: unknown) {
    const code = (err as NodeJS.ErrnoException).code;
    if (code === "ENOENT" || code === "ENOTDIR") return null;
    throw err;
  }
}

export async function listNarratives(paths: DataPaths): Promise<string[]> {
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
