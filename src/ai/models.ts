export interface ModelChoice {
  id: string;
  note: string;
}

export const RECOMMENDED_MODELS: ModelChoice[] = [
  {
    id: "openai/gpt-5.4-mini",
    note: "Recommended. Strongest sequential tool-calling in its tier (tau2-bench 93.4%), first-party serving, works with c2 as-is. ~$0.10/session. Weak spot: recall degrades past ~64k context.",
  },
  {
    id: "google/gemini-3.5-flash",
    note: "The reliability ceiling: best agentic scores, 1M context for stroke-heavy sessions, ~2x the cost. Needs reasoning round-trip support — c2 passes messages back verbatim, but this pairing is unverified.",
  },
  {
    id: "qwen/qwen3.7-plus",
    note: "Budget pick, ~5x cheaper. Has an open tool-calling bug that only affects streaming (c2 is non-streaming today); coaching-persona quality unproven. Choose when cost matters most.",
  },
];

export const DEFAULT_MODEL = RECOMMENDED_MODELS[0]!.id;

export function resolveModelChoice(input: string, current: string): string {
  const trimmed = input.trim();
  if (trimmed === "") return current;
  if (/^\d+$/.test(trimmed)) {
    const idx = parseInt(trimmed, 10);
    if (idx >= 1 && idx <= RECOMMENDED_MODELS.length) {
      return RECOMMENDED_MODELS[idx - 1]!.id;
    }
    return current;
  }
  return trimmed;
}
