import type { ChatMessage, OpenRouterClient, ToolDef } from "./client.ts";

export interface AgentDeps {
  client: OpenRouterClient;
  tools: ToolDef[];
  dispatch: (name: string, args: string) => Promise<string>;
}

const MAX_STEPS = 12;

export async function runTurn(messages: ChatMessage[], deps: AgentDeps): Promise<string> {
  for (let step = 0; step < MAX_STEPS; step++) {
    const reply = await deps.client.chat(messages, deps.tools);
    messages.push(reply);

    if (!reply.tool_calls || reply.tool_calls.length === 0) {
      return reply.content ?? "";
    }

    for (const call of reply.tool_calls) {
      let result: string;
      try {
        result = await deps.dispatch(call.function.name, call.function.arguments);
      } catch (err) {
        result = JSON.stringify({ error: `tool failed: ${(err as Error).message}` });
      }
      messages.push({ role: "tool", tool_call_id: call.id, content: result });
    }
  }
  return "(coach reached the tool-call limit for this turn; ask again or narrow the question)";
}
