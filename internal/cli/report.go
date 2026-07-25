package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/atomicfile"
	"github.com/richhaase/c2/internal/envelope"
	"github.com/richhaase/c2/internal/report"
	"github.com/richhaase/c2/internal/storage"
	"github.com/richhaase/c2/internal/store"
)

func openReport(cmd *cobra.Command, path string) {
	var child *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		child = exec.Command("open", path)
	case "windows":
		child = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	default:
		child = exec.Command("xdg-open", path)
	}
	if err := child.Start(); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Could not open report: %v\n", err)
		return
	}
	_ = child.Process.Release()
}

func newReportCmd() *cobra.Command {
	var (
		output    string
		weeksFlag string
		asData    bool
		noOpen    bool
	)

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate HTML progress report and open in browser",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, p, err := loadStore()
			if err != nil {
				return err
			}
			if cfg.Goal.StartDate == "" || cfg.Goal.EndDate == "" {
				return reportf(cmd, "Goal dates not configured. Run `c2 setup` to set start and end dates.")
			}
			if err := store.RejectForeign(p, warner(cmd)); err != nil {
				return reportf(cmd, "%v", err)
			}
			workouts, err := storage.ReadWorkouts(p)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if len(workouts) == 0 && !asData {
				fmt.Fprintln(out, "No workouts found. Run `c2 sync` first.")
				return nil
			}

			weeks, err := weekCount(cmd, weeksFlag)
			if err != nil {
				return err
			}
			result, err := report.Build(cfg, p, workouts, time.Now(), weeks)
			if err != nil {
				return err
			}
			if asData {
				return envelope.Print(out, "c2.report.v1", result.Payload)
			}

			outPath := ""
			if output != "" {
				outPath, err = filepath.Abs(output)
				if err != nil {
					return err
				}
			} else {
				dir, err := os.MkdirTemp("", "c2-report-")
				if err != nil {
					return err
				}
				outPath = filepath.Join(dir, "report.html")
			}
			if err := atomicfile.Write(outPath, []byte(result.HTML), 0o644); err != nil {
				return err
			}

			if noOpen {
				fmt.Fprintf(out, "Report written to: %s\n", outPath)
			} else {
				openReport(cmd, outPath)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "save to a specific file instead of a temp file")
	cmd.Flags().StringVarP(&weeksFlag, "weeks", "w", "12", "weeks of history to show")
	cmd.Flags().BoolVar(&asData, "data", false, "emit the report content as JSON instead of HTML")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "don't open in browser")
	return cmd
}
