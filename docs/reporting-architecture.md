# Reporting direction

Keep Go for the CLI and data core, and render the new multi-goal overview
with the standard HTML template package. This decision addresses the actual
next report: multiple independent goals, explicit activity periods and
freshness, coaching text, and expandable evidence in a file opened directly
by the reader. It does not require a running application or server.

The overview is assembled once as structured data. JSON serializes that
model and HTML renders it; presentation does not independently evaluate
achievements. The old renderer remains available for explicit legacy output.
Templates escape user-authored goal and coaching text contextually.

The report package keeps the shared work in `data.go` (activity summaries,
recent workouts, and split selection) and `coaching.go` (coaching content and
lightweight Markdown). `overview.go` builds and renders the multi-goal view
directly from that data. `report.go` assembles the legacy result, with its HTML
generation and styles isolated in `legacy.go` and `legacy_style.go`. The two
formats retain their own date-window rules and JSON field order; sharing data
helpers does not make their historical semantics interchangeable.

The implemented overview provides a concrete check that these requirements
fit the existing runtime without another dependency or installation step.
Template editing still happens in Go source, but presentation is separated
from calculation. Splitting or generating assets is an engineering option
when it provides value; this change avoids adding a frontend build pipeline
just to split files.

Node remains a reasonable alternative if a concrete future report requires
Node-side capabilities or a single-language authoring workflow materially
reduces maintenance. Browser interaction alone does not establish that:
frontend assets can be bundled at build time and included in a Go program's
output. Conversely, Node single-executable packaging means a Node solution
need not require readers to install a separate runtime.

For a richer future report, evaluate one presentation slice against the
same report model and compare authoring effort, offline file behavior,
distribution, and consistency with CLI output. Try a frontend bundle or an
isolated Node renderer if the actual requirements justify it before deciding
to port the core. No performance or authoring-speed benchmark was performed
for this decision.

References: [Go HTML templates](https://pkg.go.dev/html/template),
[esbuild browser bundles](https://esbuild.github.io/getting-started/#bundling-for-the-browser),
[Node single-executable applications](https://nodejs.org/api/single-executable-applications.html).
