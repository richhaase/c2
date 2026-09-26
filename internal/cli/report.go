package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/atomicfile"
	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/envelope"
	"github.com/richhaase/c2/internal/goals"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/report"
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
		legacy    bool
	)

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate HTML progress report and open in browser",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, p, workouts, err := loadWorkouts(cmd)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if len(workouts) == 0 && !asData && legacy {
				fmt.Fprintln(out, "No workouts found. Run `c2 sync` first.")
				return nil
			}

			weeks, err := weekCount(cmd, weeksFlag)
			if err != nil {
				return err
			}
			var html string
			if legacy {
				cfg, err = legacyGoalConfig(cfg, p)
				if err != nil {
					return err
				}
				result, err := report.Build(cfg, p, workouts, cfg.Now(), weeks)
				if err != nil {
					return err
				}
				if asData {
					return envelope.Print(out, "c2.report.v1", result.Payload)
				}
				html = result.HTML
			} else {
				result, err := report.BuildOverview(cfg, p, workouts, cfg.Now(), weeks)
				if err != nil {
					return err
				}
				if asData {
					return envelope.Print(out, "c2.report.v2", result)
				}
				html, err = report.RenderOverview(result)
				if err != nil {
					return err
				}
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
			if err := atomicfile.Write(outPath, []byte(html), 0o644); err != nil {
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
	cmd.Flags().BoolVar(&asData, "json", false, "emit the report content as JSON instead of HTML")
	cmd.Flags().BoolVar(&legacy, "legacy", false, "use the original single-goal report and c2.report.v1 schema")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "don't open in browser")
	return cmd
}

func legacyGoalConfig(cfg config.Config, p paths.DataPaths) (config.Config, error) {
	c, err := goals.Read(p)
	if err != nil {
		return cfg, err
	}
	if c != nil {
		cfg.Goal = config.GoalConfig{}
		for _, g := range c.Goals {
			if g.ID == "legacy" && !g.Archived && g.Kind == "volume" && g.Equipment == "all" && g.Effort == "workout" {
				cfg.Goal = config.GoalConfig{TargetMeters: int(g.Target), StartDate: g.From, EndDate: g.To}
			}
		}
	}
	if cfg.Goal.StartDate == "" || cfg.Goal.EndDate == "" {
		return cfg, fmt.Errorf("No dated legacy goal; use the default multi-goal output.")
	}
	return cfg, nil
}
