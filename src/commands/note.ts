import type { Command } from "commander";
import { loadConfig } from "../config.ts";
import { ensureStoreForWrite } from "../data.ts";
import { printJSON } from "../envelope.ts";
import { isValidYMD } from "../models.ts";
import {
  filterNotes,
  localISO,
  NOTE_AUTHORS,
  NOTE_TYPES,
  type NoteAuthor,
  type NoteRecord,
  type NoteType,
  readAllNotes,
  ulid,
  writeNote,
} from "../notes.ts";
import { dataPaths } from "../paths.ts";
import { readWorkouts } from "../storage.ts";
import { resolveWorkout } from "./show.ts";

async function readBody(bodyArg: string | undefined): Promise<string> {
  if (bodyArg != null && bodyArg !== "-") return bodyArg.trim();
  const text = await new Response(Bun.stdin.stream()).text();
  return text.trim();
}

export function parseNoteDate(raw: string): string | null {
  const prefix = /^(\d{4}-\d{2}-\d{2})/.exec(raw);
  if (prefix != null && !isValidYMD(prefix[1]!)) return null;
  const d = isValidYMD(raw) ? new Date(`${raw}T12:00:00`) : new Date(raw);
  if (Number.isNaN(d.getTime())) return null;
  return localISO(d);
}

function noteLine(n: NoteRecord): string {
  const workout = n.workout_id != null ? ` w:${n.workout_id}` : "";
  const tags = n.tags && n.tags.length > 0 ? ` #${n.tags.join(" #")}` : "";
  return `${n.date.slice(0, 10)}  [${n.type}/${n.author}]${workout}${tags}  ${n.body}`;
}

export function registerNote(program: Command): void {
  const note = program.command("note").description("Coaching notes and subjective reports");

  note
    .command("add [body]")
    .description("Record a note (body as argument, or '-' / omitted to read stdin)")
    .option("--type <type>", "subjective, observation, or lesson", "observation")
    .option("--workout <id>", "link to a workout id or 'last'")
    .option("--tags <tags>", "comma-separated tags")
    .option("--author <author>", "athlete or coach", "athlete")
    .option("--date <date>", "backdate the note (YYYY-MM-DD or ISO timestamp)")
    .action(
      async (
        bodyArg: string | undefined,
        opts: { type: string; workout?: string; tags?: string; author: string; date?: string },
      ) => {
        if (!(NOTE_TYPES as readonly string[]).includes(opts.type)) {
          console.error(`Error: --type must be one of ${NOTE_TYPES.join(", ")}.`);
          process.exit(1);
        }
        if (!(NOTE_AUTHORS as readonly string[]).includes(opts.author)) {
          console.error(`Error: --author must be one of ${NOTE_AUTHORS.join(", ")}.`);
          process.exit(1);
        }

        const body = await readBody(bodyArg);
        if (body === "") {
          console.error("Error: note body is empty.");
          process.exit(1);
        }

        const cfg = await loadConfig();
        const paths = dataPaths(cfg);
        const now = new Date();
        const storeError = await ensureStoreForWrite(paths, now);
        if (storeError != null) {
          console.error(storeError);
          process.exit(1);
        }

        let workoutId: number | undefined;
        if (opts.workout) {
          const workouts = await readWorkouts(paths);
          const w = resolveWorkout(workouts, opts.workout);
          if (w == null) {
            console.error(`Error: no workout matching "${opts.workout}".`);
            process.exit(1);
          }
          workoutId = w.id;
        }

        let date = localISO(now);
        if (opts.date) {
          const parsed = parseNoteDate(opts.date);
          if (parsed == null) {
            console.error(`Error: invalid --date "${opts.date}".`);
            process.exit(1);
          }
          date = parsed;
        }

        const record: NoteRecord = {
          id: ulid(now),
          date,
          type: opts.type as NoteType,
          workout_id: workoutId,
          tags: opts.tags
            ? opts.tags
                .split(",")
                .map((t) => t.trim())
                .filter((t) => t !== "")
            : undefined,
          body,
          author: opts.author as NoteAuthor,
        };
        await writeNote(paths, record);
        console.log(record.id);
      },
    );

  note
    .command("list")
    .description("List notes, newest last")
    .option("--type <type>", "filter by type")
    .option("--since <date>", "only notes on or after date (YYYY-MM-DD)")
    .option("--workout <id>", "only notes linked to a workout id")
    .option("-n, --count <n>", "show only the most recent n notes")
    .option("--json", "output as JSON")
    .action(
      async (opts: {
        type?: string;
        since?: string;
        workout?: string;
        count?: string;
        json?: boolean;
      }) => {
        if (opts.since && !isValidYMD(opts.since)) {
          console.error(`Error: invalid --since date "${opts.since}" (expected YYYY-MM-DD).`);
          process.exit(1);
        }
        const cfg = await loadConfig();
        const paths = dataPaths(cfg);
        let notes = filterNotes(await readAllNotes(paths), {
          type: opts.type,
          since: opts.since,
          workoutId: opts.workout != null ? Number(opts.workout) : undefined,
        });
        if (opts.count != null) {
          const n = parseInt(opts.count, 10);
          if (Number.isNaN(n) || n < 1) {
            console.error("Error: --count must be a positive integer.");
            process.exit(1);
          }
          notes = notes.slice(-n);
        }

        if (opts.json) {
          printJSON("c2.notes.v1", { count: notes.length, notes });
          return;
        }
        if (notes.length === 0) {
          console.log("No notes found.");
          return;
        }
        for (const n of notes) {
          console.log(noteLine(n));
        }
      },
    );

  note
    .command("show <id>")
    .description("Show one note in full")
    .option("--json", "output as JSON")
    .action(async (id: string, opts: { json?: boolean }) => {
      const cfg = await loadConfig();
      const paths = dataPaths(cfg);
      const match = (await readAllNotes(paths)).find((n) => n.id === id);
      if (match == null) {
        console.error(`No note with id ${id}.`);
        process.exit(1);
      }
      if (opts.json) {
        printJSON("c2.note.v1", match);
        return;
      }
      console.log(`Id: ${match.id}`);
      console.log(`Date: ${match.date}`);
      console.log(`Type: ${match.type} (${match.author})`);
      if (match.workout_id != null) console.log(`Workout: ${match.workout_id}`);
      if (match.tags && match.tags.length > 0) console.log(`Tags: ${match.tags.join(", ")}`);
      console.log();
      console.log(match.body);
    });
}
