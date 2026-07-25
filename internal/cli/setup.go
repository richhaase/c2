package cli

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/api"
	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/display"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/store"
)

type prompter struct {
	in  *bufio.Scanner
	out io.Writer
}

func newPrompter(cmd *cobra.Command) *prompter {
	return &prompter{
		in:  bufio.NewScanner(cmd.InOrStdin()),
		out: cmd.OutOrStdout(),
	}
}

func (p *prompter) line(label string) (string, bool) {
	fmt.Fprintf(p.out, "%s ", label)
	if !p.in.Scan() {
		return "", false
	}
	return strings.TrimSpace(p.in.Text()), true
}

func maskToken(token string) string {
	if len(token) <= 4 {
		return token
	}
	return strings.Repeat("·", len(token)-4) + token[len(token)-4:]
}

func (p *prompter) value(label, current string, mask bool) (string, bool) {
	hint := ""
	if current != "" {
		shown := current
		if mask {
			shown = maskToken(current)
		}
		hint = " [" + shown + "]"
	}
	input, ok := p.line(label + hint + ":")
	if !ok {
		return current, false
	}
	if input == "" {
		return current, true
	}
	return input, true
}

func (p *prompter) confirm(question string, defaultYes bool) (bool, bool) {
	suffix := "[y/N]"
	if defaultYes {
		suffix = "[Y/n]"
	}
	input, ok := p.line(question + " " + suffix)
	if !ok {
		return defaultYes, false
	}
	input = strings.ToLower(input)
	if input == "" {
		return defaultYes, true
	}
	return input == "y" || input == "yes", true
}

func chooseDataDir(cmd *cobra.Command, p *prompter, current string) (string, bool) {
	input, ok := p.value("Data directory", current, false)
	if !ok {
		return current, false
	}
	target := paths.For(input)
	warn := warner(cmd)
	inspection, err := store.Inspect(target, warn)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Cannot inspect %s: %v\n", target.Root, err)
		return current, true
	}

	out := cmd.OutOrStdout()
	if !inspection.Writable {
		fmt.Fprintf(cmd.ErrOrStderr(), "Cannot write to %s; keeping %s.\n", target.Root, current)
		return current, true
	}

	switch inspection.State {
	case store.StateMissing:
		create, ok := p.confirm(fmt.Sprintf("%s does not exist. Create it?", target.Root), true)
		if !ok {
			return current, false
		}
		if !create {
			fmt.Fprintf(out, "Keeping %s.\n", current)
			return current, true
		}
	case store.StateStore:
		summary, err := store.Summarize(target, warn)
		if err != nil {
			return current, true
		}
		if summary.Workouts > 0 {
			fmt.Fprintf(out, "Found existing c2 store: %d workouts (%s → %s), %d stroke files, %d notes.\n",
				summary.Workouts, summary.FirstDate, summary.LastDate, summary.StrokeFiles, summary.Notes)
		} else {
			fmt.Fprintln(out, "Found existing empty c2 store.")
		}
	case store.StateForeign:
		proceed, ok := p.confirm(fmt.Sprintf("%s is not empty and not a c2 store. Initialize a store here anyway?", target.Root), false)
		if !ok {
			return current, false
		}
		if !proceed {
			fmt.Fprintf(out, "Keeping %s.\n", current)
			return current, true
		}
	}

	if err := store.Init(target, time.Now(), warn); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "Could not initialize %s: %v\n", target.Root, err)
		return current, true
	}
	return paths.CanonicalRoot(target.Root), true
}

func newSetupCmd(b build) *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Configure token, goal, and preferences",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			errOut := cmd.ErrOrStderr()

			cfg, err := config.Load()
			if err != nil {
				fmt.Fprintf(errOut, "Warning: could not load existing config: %v\n", err)
				fmt.Fprintln(errOut, "Starting from defaults.")
				cfg = config.Default()
			}

			fmt.Fprintln(out, "Concept2 CLI Setup")
			fmt.Fprintln(out)

			p := newPrompter(cmd)

			token, ok := p.value("API token (from log.concept2.com)", cfg.API.Token, true)
			if !ok {
				return errReported
			}
			cfg.API.Token = token

			targetInput, ok := p.value("Goal target meters", display.FormatMeters(cfg.Goal.TargetMeters), false)
			if !ok {
				return errReported
			}
			if parsed, err := strconv.Atoi(strings.ReplaceAll(targetInput, ",", "")); err == nil && parsed > 0 {
				cfg.Goal.TargetMeters = parsed
			}

			startInput, ok := p.value("Goal start date (YYYY-MM-DD)", cfg.Goal.StartDate, false)
			if !ok {
				return errReported
			}
			if _, err := config.ParseGoalDate(startInput); err != nil {
				fmt.Fprintf(out, "Invalid date \"%s\", keeping previous value.\n", startInput)
			} else {
				cfg.Goal.StartDate = startInput
			}

			endInput, ok := p.value("Goal end date (YYYY-MM-DD)", cfg.Goal.EndDate, false)
			if !ok {
				return errReported
			}
			if _, err := config.ParseGoalDate(endInput); err != nil {
				fmt.Fprintf(out, "Invalid date \"%s\", keeping previous value.\n", endInput)
			} else {
				cfg.Goal.EndDate = endInput
			}

			dataDir, ok := chooseDataDir(cmd, p, cfg.DataDir)
			if !ok {
				return errReported
			}
			cfg.DataDir = dataDir

			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintln(out)
			fmt.Fprintf(out, "Config written to %s (mode 600)\n", config.File())
			fmt.Fprintf(out, "Data directory: %s\n", paths.For(cfg.DataDir).Root)

			if cfg.Goal.StartDate == "" || cfg.Goal.EndDate == "" {
				fmt.Fprintln(out)
				fmt.Fprintln(out, "Note: Goal dates not set. Commands like `c2 status` require start/end dates.")
			}

			if cfg.API.Token != "" {
				fmt.Fprintln(out, "Verifying token...")
				user, err := api.FromConfig(cfg, b.version).GetUser(cmd.Context())
				if err != nil {
					fmt.Fprintf(errOut, "Warning: could not verify token: %v\n", err)
				} else {
					fmt.Fprintf(out, "Authenticated as: %s (ID: %d)\n", user.Username, user.ID)
				}
			}
			return nil
		},
	}
}
