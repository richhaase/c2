# Correcting coaching notes

Find a note's ID, inspect it, and open its text in your editor:

```sh
c2 note list -n 5
c2 note show <id>
c2 note edit <id>
c2 note show <id>
```

Recent and archived notes use the same commands. An edit replaces the current
record and keeps its ID. Every field you did not explicitly change stays the
same, including the date, author, workout link, tags, and type. Use a new note
for a changed coaching judgment. Corrections have no revision history or undo.

## Editor behavior

C2 uses `VISUAL`, then `EDITOR`, then `vi`. For example:

```sh
export EDITOR='nano'
export VISUAL='code --wait'
```

Commands can contain arguments and quoted paths. C2 runs the executable directly;
shell expansions, pipelines, and redirects are not evaluated. GUI editors need
their wait option so C2 reads the file after you finish editing.

Only the note body appears in the editor. Save and exit to apply the correction.
Unchanged text makes no write. Empty or whitespace-only text is rejected, and an
editor error leaves the note unchanged. To cancel, quit without saving or exit
the editor with an error. The temporary file is private and removed afterward.

An interactive terminal is required to launch the editor. Scripts and agents
must supply text or metadata explicitly; C2 never launches an editor for them.

## Explicit corrections

```sh
c2 note edit <id> --body 'Corrected feedback' --json
c2 note edit <id> --file correction.md
cat correction.md | c2 note edit <id> --file - --json
c2 note edit <id> --tags technique,recovery --workout last
c2 note edit <id> --date 2026-09-26 --type lesson --author coach
c2 note edit <id> --workout '' --tags ''
```

Metadata-only corrections do not open an editor. An empty workout or tags option
clears that field. Explicit date corrections use the same date rules as note
creation: date-only values mean noon in the machine's local timezone; an ISO
timestamp with an offset expresses an exact instant.

`--body` and `--file` work for note creation and correction, `plan set`,
`playbook set`, and `narrative add`. Use `--file -` for standard input.
Supply exactly one text source. These explicit inputs preserve whitespace.
Managed documents retain their existing convention of adding a final newline
when absent. Note bodies keep exactly the supplied text.

Existing positional inputs still work: note arguments are text; document
arguments are file paths. Positional note text and implicit note stdin keep
their legacy trimming behavior. In a terminal, missing document or note-add
input produces guidance instead of waiting silently for standard input.

## Structured output and compatibility

| Command | JSON schema | Data |
| --- | --- | --- |
| `note add`, `note show` | `c2.note.v1` | Full note |
| `note edit` | `c2.note.edited.v1` | `changed` and full `note` |
| `note list` | `c2.notes.v1` | Count and notes |
| `plan set/show`, `playbook set/show`, `narrative add/show` | `c2.document.v1` | Name, narrative date when applicable, saved content |
| `goal add/update/archive` | `c2.goal.saved.v1` | Saved goal |
| `goal list/show` | `c2.goal.v1` | Goals, progress, and evidence |

Add `--json` to select these versioned envelopes. Editor diagnostics go to stderr,
so an interactive edit can still return a clean JSON receipt on stdout.
Successful document receipts include the final saved newline.

`note add` still prints only its ID by default, preserving existing scripts.
Readable note lists now include IDs; use JSON for stable parsing. Existing plain
document and goal confirmations retain their output.

## Storage and handoffs

Each correction atomically replaces one existing file. Archived corrections stay
in their original archive container, even when the corrected date changes year;
the date inside the record governs ordering and filtering. No loose shadow copy
is created. Reads, archiving, validation, and complete-store transfers support
this layout.

Before editing, C2 refuses malformed note storage, multiple copies of the target
ID, and unsupported fields on the target note. These errors require repairing
the store or upgrading C2 rather than silently dropping data. `c2 data doctor`
helps inspect storage problems. C2 also checks whether the note changed while
the editor was open and refuses a stale correction.

Keep one active writer, and finish cloud synchronization before switching
machines. These checks are not a distributed lock or cloud conflict resolver.
Use the complete-store transfer workflow in [personal goals](personal-goals.md)
to carry corrected notes together with goals and coaching documents.
