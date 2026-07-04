import pkg from "../../package.json";

export interface ToolCall {
  id: string;
  type: "function";
  function: { name: string; arguments: string };
}

export interface ChatMessage {
  role: "system" | "user" | "assistant" | "tool";
  content: string | null;
  tool_calls?: ToolCall[];
  tool_call_id?: string;
  name?: string;
}

export interface ToolDef {
  type: "function";
  function: {
    name: string;
    description: string;
    parameters: unknown;
  };
}

interface ChatResponse {
  choices?: { message: ChatMessage }[];
  error?: { message?: string };
}

export class OpenRouterClient {
  private baseURL: string;
  private apiKey: string;
  private model: string;

  constructor(baseURL: string, apiKey: string, model: string) {
    this.baseURL = baseURL.replace(/\/$/, "");
    this.apiKey = apiKey;
    this.model = model;
  }

  async chat(messages: ChatMessage[], tools: ToolDef[]): Promise<ChatMessage> {
    const resp = await fetch(`${this.baseURL}/chat/completions`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${this.apiKey}`,
        "Content-Type": "application/json",
        "User-Agent": `c2/${pkg.version}`,
        "X-Title": "c2 coach",
      },
      body: JSON.stringify({
        model: this.model,
        messages,
        tools: tools.length > 0 ? tools : undefined,
      }),
      signal: AbortSignal.timeout(120_000),
    });

    const text = await resp.text();
    if (!resp.ok) {
      throw new Error(`OpenRouter error (${resp.status}): ${text}`);
    }

    let parsed: ChatResponse;
    try {
      parsed = JSON.parse(text) as ChatResponse;
    } catch {
      throw new Error(`OpenRouter returned non-JSON response: ${text.slice(0, 200)}`);
    }

    if (parsed.error) {
      throw new Error(`OpenRouter error: ${parsed.error.message ?? "unknown"}`);
    }

    const message = parsed.choices?.[0]?.message;
    if (!message) {
      throw new Error("OpenRouter returned no message");
    }
    return message;
  }
}
