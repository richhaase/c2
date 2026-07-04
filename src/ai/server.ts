import { spawn } from "node:child_process";
import { C2Client } from "../api/client.ts";
import { buildReportHTML } from "../commands/report.ts";
import type { Config } from "../config.ts";
import type { Workout } from "../models.ts";
import { runTurn } from "./agent.ts";
import { type ChatMessage, OpenRouterClient } from "./client.ts";
import { appendNote, readNotes, readProfile } from "./memory.ts";
import { chatPanel } from "./panel.ts";
import { buildSystemPrompt } from "./prompt.ts";
import { buildTools } from "./tools.ts";

export interface ServeOptions {
  cfg: Config;
  workouts: Workout[];
  weeks: number;
  now: Date;
  port: number;
  openBrowser: boolean;
}

function openInBrowser(url: string): void {
  const [cmd, args] =
    process.platform === "darwin"
      ? ["open", [url]]
      : process.platform === "win32"
        ? ["cmd", ["/c", "start", "", url]]
        : ["xdg-open", [url]];
  const child = spawn(cmd as string, args as string[], { stdio: "ignore", detached: true });
  child.on("error", (err) => console.error(`Could not open browser: ${err.message}`));
  child.unref();
}

function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

export async function serveCoachReport(opts: ServeOptions): Promise<void> {
  const { cfg, workouts, weeks, now } = opts;
  if (!cfg.ai.api_key) {
    console.error("AI coaching is not configured. Run `c2 setup` to add an OpenRouter key.");
    process.exit(1);
  }

  const client = new OpenRouterClient(cfg.ai.base_url, cfg.ai.api_key, cfg.ai.model);
  const api = C2Client.fromConfig(cfg);
  const profile = await readProfile();
  const notes = await readNotes(50);

  let events: string[] = [];
  const { defs, dispatch } = buildTools({ cfg, workouts, api, now: () => new Date() }, (m) =>
    events.push(m),
  );

  const surface = `The athlete is viewing an HTML progress report covering the last ${weeks} weeks alongside this chat. Answer questions about what that report shows, and go deeper with tools when useful.`;
  const messages: ChatMessage[] = [
    { role: "system", content: buildSystemPrompt({ cfg, workouts, profile, notes, now, surface }) },
  ];

  const sessionToken = crypto.randomUUID();
  const reportHTML = buildReportHTML(cfg, workouts, weeks, now);
  const page = reportHTML.replace("</body>", `${chatPanel(sessionToken)}\n</body>`);

  let chatLock: Promise<unknown> = Promise.resolve();

  function serialized<T>(work: () => Promise<T>): Promise<T> {
    const run = chatLock.then(work);
    chatLock = run.catch(() => undefined);
    return run;
  }

  function handleChat(message: string): Promise<Response> {
    return serialized(async () => {
      events = [];
      messages.push({ role: "user", content: message });
      try {
        const reply = await runTurn(messages, { client, tools: defs, dispatch });
        return json({ reply, events });
      } catch (err) {
        return json({ error: (err as Error).message, events });
      }
    });
  }

  function handleNoteSave(note: string): Promise<Response> {
    return serialized(async () => {
      await appendNote(note, new Date());
      messages.push({
        role: "system",
        content: `The athlete just saved a note to coach memory: ${note}`,
      });
      return json({ saved: true });
    });
  }

  const server = Bun.serve({
    hostname: cfg.ai.bind_address,
    port: opts.port,
    idleTimeout: 0,
    async fetch(req, srv) {
      const url = new URL(req.url);
      const allowedHosts = new Set([
        `127.0.0.1:${srv.port}`,
        `localhost:${srv.port}`,
        `[::1]:${srv.port}`,
        `${cfg.ai.bind_address}:${srv.port}`,
      ]);
      if (!allowedHosts.has(url.host)) {
        return json({ error: "invalid host" }, 403);
      }
      if (req.method === "GET" && url.pathname === "/") {
        return new Response(page, { headers: { "Content-Type": "text/html; charset=utf-8" } });
      }
      if (url.pathname.startsWith("/api/") && req.headers.get("x-c2-token") !== sessionToken) {
        return json({ error: "invalid session token" }, 403);
      }
      if (req.method === "POST" && url.pathname === "/api/chat") {
        const body = (await req.json().catch(() => ({}))) as { message?: string };
        const message = (body.message ?? "").trim();
        if (!message) return json({ error: "empty message" }, 400);
        return handleChat(message);
      }
      if (req.method === "GET" && url.pathname === "/api/notes") {
        return json({ notes: await readNotes(200) });
      }
      if (req.method === "POST" && url.pathname === "/api/notes") {
        const body = (await req.json().catch(() => ({}))) as { note?: string };
        const note = (body.note ?? "").trim();
        if (!note) return json({ error: "empty note" }, 400);
        return handleNoteSave(note);
      }
      return new Response("Not found", { status: 404 });
    },
  });

  const displayHost = cfg.ai.bind_address === "0.0.0.0" ? "localhost" : cfg.ai.bind_address;
  const displayURL = `http://${displayHost}:${server.port}`;
  console.log(`c2 coach report — ${cfg.ai.model}`);
  console.log(`Serving at ${displayURL}  (Ctrl-C to stop)`);
  if (opts.openBrowser) openInBrowser(displayURL);
}
