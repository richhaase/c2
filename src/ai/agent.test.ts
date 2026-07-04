import { expect, test } from "bun:test";
import type { C2Client } from "../api/client.ts";
import { defaultConfig } from "../config.ts";
import type { Workout } from "../models.ts";
import { runTurn } from "./agent.ts";
import type { ChatMessage, OpenRouterClient } from "./client.ts";
import { buildTools } from "./tools.ts";

function fixtureWorkout(id: number, date: string, distance: number): Workout {
  return {
    id,
    user_id: 1,
    date,
    distance,
    type: "rower",
    time: 12000,
    time_formatted: "20:00.0",
    stroke_rate: 24,
    heart_rate: { average: 140 },
  };
}

function scriptedClient(script: ChatMessage[]): OpenRouterClient {
  let i = 0;
  return {
    chat: async (_messages: ChatMessage[], _tools: unknown) => {
      const next = script[i];
      i++;
      if (!next) throw new Error("scripted client exhausted");
      return next;
    },
  } as unknown as OpenRouterClient;
}

const cfg = {
  ...defaultConfig(),
  goal: { target_meters: 1_000_000, start_date: "2026-01-01", end_date: "2026-12-31" },
};
const now = new Date("2026-07-04T12:00:00Z");
const workouts = [
  fixtureWorkout(1, "2026-07-01 08:00:00", 8000),
  fixtureWorkout(2, "2026-06-28 08:00:00", 6000),
];

test("agent runs a tool call then returns final text", async () => {
  const seen: string[] = [];
  const api = {} as C2Client;
  const { defs, dispatch } = buildTools({ cfg, workouts, api, now }, (m) => seen.push(m));

  const client = scriptedClient([
    {
      role: "assistant",
      content: null,
      tool_calls: [
        { id: "c1", type: "function", function: { name: "goal_progress", arguments: "{}" } },
      ],
    },
    { role: "assistant", content: "You have logged 14,000 m so far." },
  ]);

  const messages: ChatMessage[] = [
    { role: "system", content: "sys" },
    { role: "user", content: "how am I doing?" },
  ];
  const reply = await runTurn(messages, { client, tools: defs, dispatch });

  expect(reply).toBe("You have logged 14,000 m so far.");
  expect(seen).toContain("checking goal progress");
  const toolMsg = messages.find((m) => m.role === "tool");
  expect(toolMsg).toBeDefined();
  expect(JSON.parse(toolMsg!.content as string).totalMeters).toBe(14000);
});

test("list_workouts tool filters and compacts", async () => {
  const api = {} as C2Client;
  const { dispatch } = buildTools({ cfg, workouts, api, now }, () => {});
  const raw = await dispatch("list_workouts", JSON.stringify({ from: "2026-07-01" }));
  const parsed = JSON.parse(raw);
  expect(parsed.count).toBe(1);
  expect(parsed.workouts[0].id).toBe(1);
  expect(parsed.workouts[0].pace_500m).toBe("1:15.0");
});

test("agent stops at the step limit without looping forever", async () => {
  const api = {} as C2Client;
  const { defs, dispatch } = buildTools({ cfg, workouts, api, now }, () => {});
  const loopingCall: ChatMessage = {
    role: "assistant",
    content: null,
    tool_calls: [
      { id: "c", type: "function", function: { name: "goal_progress", arguments: "{}" } },
    ],
  };
  const client = scriptedClient(Array.from({ length: 20 }, () => loopingCall));
  const messages: ChatMessage[] = [{ role: "system", content: "sys" }];
  const reply = await runTurn(messages, { client, tools: defs, dispatch });
  expect(reply).toContain("tool-call limit");
});
