package cli

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/display"
	"github.com/richhaase/c2/internal/doctor"
	"github.com/richhaase/c2/internal/envelope"
	"github.com/richhaase/c2/internal/notes"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/store"
)

type dataInfoPayload struct {
	Root          string         `json:"root"`
	State         store.DirState `json:"state"`
	Writable      bool           `json:"writable"`
	SchemaVersion *int           `json:"schema_version"`
	LastSync      *string        `json:"last_sync"`
	Workouts      int            `json:"workouts"`
	FirstDate     *string        `json:"first_date"`
	LastDate      *string        `json:"last_date"`
	StrokeFiles   int            `json:"stroke_files"`
	Notes         int            `json:"notes"`
}

func warner(cmd *cobra.Command) func(string) {
	return func(msg string) {
		fmt.Fprintln(cmd.ErrOrStderr(), msg)
	}
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func newDataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "data",
		Short: "Manage the c2 data store",
	}
	cmd.AddCommand(newDataInfoCmd(), newDataCompactCmd(), newDataDoctorCmd(), newDataMoveCmd())
	return cmd
}

func newDataInfoCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show data store location and contents",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, p, err := loadStore()
			if err != nil {
				return err
			}
			warn := warner(cmd)
			inspection, err := store.Inspect(p, warn)
			if err != nil {
				return err
			}
			switch inspection.State {
			case store.StateMissing:
				return reportf(cmd, "No data store at %s. Run `c2 setup` or `c2 sync` first.", p.Root)
			case store.StateForeign:
				return reportf(cmd, "%s exists but is not a c2 data store. Fix data_dir via `c2 setup`.", p.Root)
			case store.StateEmpty:
				return reportf(cmd, "%s is an empty directory, not yet a data store. Run `c2 sync` to initialize it.", p.Root)
			}

			summary, err := store.Summarize(p, warn)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if asJSON {
				return envelope.Print(out, "c2.data.info.v1", dataInfoPayload{
					Root:          p.Root,
					State:         inspection.State,
					Writable:      inspection.Writable,
					SchemaVersion: summary.SchemaVersion,
					LastSync:      nilIfEmpty(summary.LastSync),
					Workouts:      summary.Workouts,
					FirstDate:     nilIfEmpty(summary.FirstDate),
					LastDate:      nilIfEmpty(summary.LastDate),
					StrokeFiles:   summary.StrokeFiles,
					Notes:         summary.Notes,
				})
			}

			fmt.Fprintf(out, "Data store: %s\n", p.Root)
			schema := "(unknown — meta.json missing or corrupt)"
			if summary.SchemaVersion != nil {
				schema = fmt.Sprintf("%d", *summary.SchemaVersion)
			}
			fmt.Fprintf(out, "Schema version: %s\n", schema)
			lastSync := "never"
			if summary.LastSync != "" {
				lastSync = summary.LastSync
			}
			fmt.Fprintf(out, "Last sync: %s\n", lastSync)
			dates := ""
			if summary.FirstDate != "" {
				dates = fmt.Sprintf(" (%s → %s)", summary.FirstDate, summary.LastDate)
			}
			fmt.Fprintf(out, "Workouts: %s%s\n", display.FormatMeters(summary.Workouts), dates)
			fmt.Fprintf(out, "Stroke files: %s\n", display.FormatMeters(summary.StrokeFiles))
			fmt.Fprintf(out, "Notes: %s\n", display.FormatMeters(summary.Notes))
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}

func newDataCompactCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "compact",
		Short: "Archive notes older than 7 days into yearly files",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, p, err := loadStore()
			if err != nil {
				return err
			}
			warn := warner(cmd)
			inspection, err := store.Inspect(p, warn)
			if err != nil {
				return err
			}
			if inspection.State != store.StateStore {
				return reportf(cmd, "%s is not a c2 data store; nothing to compact.", p.Root)
			}
			if !inspection.Writable {
				return reportf(cmd, "Cannot write to %s.", p.Root)
			}
			result, err := notes.Compact(p, time.Now())
			if err != nil {
				return err
			}
			for _, year := range result.SkippedYears {
				fmt.Fprintf(cmd.ErrOrStderr(),
					"Warning: notes/archive/%d.jsonl could not be safely rewritten; notes left loose (run `c2 data doctor`).\n", year)
			}
			out := cmd.OutOrStdout()
			if result.Archived > 0 {
				names := make([]string, 0, len(result.Years))
				for _, y := range result.Years {
					names = append(names, fmt.Sprintf("notes/archive/%d.jsonl", y))
				}
				fmt.Fprintf(out, "Compacted %d note%s into %s.\n",
					result.Archived, plural(result.Archived), strings.Join(names, ", "))
			} else if len(result.SkippedYears) == 0 {
				fmt.Fprintln(out, "Nothing to compact.")
			}
			if len(result.SkippedYears) > 0 {
				return errReported
			}
			return nil
		},
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func newDataDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Validate the data store and report problems",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, p, err := loadStore()
			if err != nil {
				return err
			}
			inspection, err := store.Inspect(p, warner(cmd))
			if err != nil {
				return err
			}
			if inspection.State != store.StateStore {
				return reportf(cmd, "No data store at %s.", p.Root)
			}
			report := doctor.Run(p)
			out := cmd.OutOrStdout()
			if len(report.Issues) == 0 {
				fmt.Fprintf(out, "OK — %d files checked, no problems found.\n", report.CheckedFiles)
				return nil
			}
			errOut := cmd.ErrOrStderr()
			fmt.Fprintf(errOut, "%d problem%s found:\n", len(report.Issues), plural(len(report.Issues)))
			for _, issue := range report.Issues {
				fmt.Fprintf(errOut, "  - %s\n", issue)
			}
			return errReported
		},
	}
}

func newDataMoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "move <dir>",
		Short: "Relocate the data store and update config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			warn := warner(cmd)
			from := paths.For(paths.CanonicalRoot(cfg.DataDir))
			source, err := store.Inspect(from, warn)
			if err != nil {
				return err
			}
			if source.State != store.StateStore {
				return reportf(cmd, "%s is not a c2 data store; nothing to move.", from.Root)
			}

			to := paths.For(paths.CanonicalRoot(args[0]))
			if to.Root == from.Root {
				return reportf(cmd, "Target is the current data directory.")
			}
			sep := string(filepath.Separator)
			if strings.HasPrefix(to.Root, from.Root+sep) || strings.HasPrefix(from.Root, to.Root+sep) {
				return reportf(cmd, "Target must not be inside the current data directory (or contain it).")
			}

			copied, err := store.Move(from, to, warn)
			if err != nil {
				return reportf(cmd, "Error: %v", err)
			}
			cfg.DataDir = to.Root
			if err := config.Save(cfg); err != nil {
				errOut := cmd.ErrOrStderr()
				fmt.Fprintf(errOut, "Error: copy completed but config update failed: %v\n", err)
				fmt.Fprintf(errOut, "Set data_dir to %s in ~/.config/c2/config.json manually, or remove %s and retry.\n", to.Root, to.Root)
				return errReported
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Copied %d files (%s bytes) to %s\n", copied.Files, display.FormatMeters(int(copied.Bytes)), to.Root)
			fmt.Fprintf(out, "Config updated: data_dir = %s\n", to.Root)
			fmt.Fprintf(out, "Old data left at %s — remove it manually once satisfied.\n", from.Root)
			return nil
		},
	}
}
