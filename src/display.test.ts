import { describe, expect, test } from "bun:test";
import {
  formatMeters,
  formatPercent,
  formatMetersPerWeek,
  sparkBar,
  trendArrow,
  paceArrow,
} from "./display.ts";

describe("formatMeters", () => {
  test.each([
    [0, "0"],
    [500, "500"],
    [1000, "1,000"],
    [12345, "12,345"],
    [1000000, "1,000,000"],
  ])("formatMeters(%i) = %s", (input, expected) => {
    expect(formatMeters(input)).toBe(expected);
  });
});

describe("formatPercent", () => {
  test.each([
    [0, "0.0%"],
    [0.5, "50.0%"],
    [1.0, "100.0%"],
    [0.1234, "12.3%"],
    [0.131, "13.1%"],
  ])("formatPercent(%f) = %s", (input, expected) => {
    expect(formatPercent(input)).toBe(expected);
  });
});

describe("formatMetersPerWeek", () => {
  test("formats with unit", () => {
    expect(formatMetersPerWeek(20212)).toBe("20,212m/week");
  });
});

describe("sparkBar", () => {
  test("returns empty string when max is 0", () => {
    expect(sparkBar(100, 0)).toBe("");
  });

  test("returns full bar for max value", () => {
    const bar = sparkBar(100, 100);
    expect(bar).toBe("\u2588".repeat(20));
  });

  test("returns half bar for 50%", () => {
    const bar = sparkBar(50, 100);
    expect(bar.length).toBe(20);
    expect(bar).toBe("\u2588".repeat(10) + "\u2591".repeat(10));
  });
});

describe("trendArrow", () => {
  test("returns space when prev is 0", () => {
    expect(trendArrow(0, 100)).toBe(" ");
  });

  test("returns up arrow for increase", () => {
    expect(trendArrow(100, 110)).toBe("\u2191");
  });

  test("returns down arrow for decrease", () => {
    expect(trendArrow(100, 90)).toBe("\u2193");
  });

  test("returns right arrow for stable", () => {
    expect(trendArrow(100, 101)).toBe("\u2192");
  });
});

describe("paceArrow", () => {
  test("reversed: lower pace shows up arrow (improvement)", () => {
    expect(paceArrow(180, 170)).toBe("\u2191");
  });

  test("reversed: higher pace shows down arrow", () => {
    expect(paceArrow(170, 180)).toBe("\u2193");
  });
});
