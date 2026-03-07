import { describe, expect, test } from "bun:test";
import { parseGoalDate, defaultConfig } from "./config.ts";

describe("parseGoalDate", () => {
  test("parses valid date string", () => {
    const d = parseGoalDate("2026-03-07");
    expect(d.getFullYear()).toBe(2026);
    expect(d.getMonth()).toBe(2); // March = 2
    expect(d.getDate()).toBe(7);
  });

  test("throws on empty string", () => {
    expect(() => parseGoalDate("")).toThrow("Invalid date");
  });

  test("throws on malformed string", () => {
    expect(() => parseGoalDate("not-a-date")).toThrow("Invalid date");
  });
});

describe("defaultConfig", () => {
  test("returns expected defaults", () => {
    const cfg = defaultConfig();
    expect(cfg.api.base_url).toBe("https://log.concept2.com");
    expect(cfg.api.token).toBe("");
    expect(cfg.sync.machine_type).toBe("rower");
    expect(cfg.goal.target_meters).toBe(1_000_000);
    expect(cfg.goal.start_date).toBe("");
    expect(cfg.goal.end_date).toBe("");
    expect(cfg.display.date_format).toBe("%m/%d");
  });
});
