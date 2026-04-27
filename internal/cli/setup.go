package cli

import (
	"bufio"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/display"
	"github.com/spf13/cobra"
)

func newSetupCommand(version string, deps Dependencies) *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Configure token, goal, and preferences",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := deps.LoadConfig()
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not load existing config: %v\n", err)
				fmt.Fprintln(cmd.ErrOrStderr(), "Starting from defaults.")
				cfg = config.Default()
			}

			reader := bufio.NewReader(deps.Stdin)
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "Concept2 CLI Setup")
			fmt.Fprintln(out)

			token, err := promptValue(reader, out, "API token (from log.concept2.com)", cfg.API.Token, true)
			if err != nil {
				return err
			}
			cfg.API.Token = token

			target, err := promptValue(reader, out, "Goal target meters", display.FormatMeters(cfg.Goal.TargetMeters), false)
			if err != nil {
				return err
			}
			parsed, parseErr := strconv.Atoi(strings.ReplaceAll(target, ",", ""))
			if parseErr == nil && parsed > 0 {
				cfg.Goal.TargetMeters = parsed
			}

			start, err := promptValue(reader, out, "Goal start date (YYYY-MM-DD)", cfg.Goal.StartDate, false)
			if err != nil {
				return err
			}
			if _, err := config.ParseGoalDate(start); err == nil {
				cfg.Goal.StartDate = start
			} else if start != "" {
				fmt.Fprintf(out, "Invalid date %q, keeping previous value.\n", start)
			}

			end, err := promptValue(reader, out, "Goal end date (YYYY-MM-DD)", cfg.Goal.EndDate, false)
			if err != nil {
				return err
			}
			if _, err := config.ParseGoalDate(end); err == nil {
				cfg.Goal.EndDate = end
			} else if end != "" {
				fmt.Fprintf(out, "Invalid date %q, keeping previous value.\n", end)
			}

			if err := deps.EnsureDirs(); err != nil {
				return err
			}
			if err := deps.SaveConfig(cfg); err != nil {
				return err
			}
			fmt.Fprintf(out, "\nConfig written to %s\n", filepath.Join(config.Dir(), "config.json"))

			if cfg.Goal.StartDate == "" || cfg.Goal.EndDate == "" {
				fmt.Fprintln(out, "\nNote: Goal dates not set. Commands like `c2 status` require start/end dates.")
			}

			if cfg.API.Token != "" {
				fmt.Fprintln(out, "Verifying token...")
				user, err := deps.VerifyUser(cmd.Context(), cfg, version)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not verify token: %v\n", err)
				} else {
					fmt.Fprintf(out, "Authenticated as: %s (ID: %d)\n", user.Username, user.ID)
				}
			}
			return nil
		},
	}
}

func promptValue(reader *bufio.Reader, out interface{ Write([]byte) (int, error) }, label string, current string, mask bool) (string, error) {
	displayValue := current
	if mask {
		displayValue = maskToken(current)
	}
	if displayValue != "" {
		fmt.Fprintf(out, "%s [%s]: ", label, displayValue)
	} else {
		fmt.Fprintf(out, "%s: ", label)
	}
	input, err := reader.ReadString('\n')
	if err != nil && len(input) == 0 {
		return "", err
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return current, nil
	}
	return input, nil
}

func maskToken(token string) string {
	if len(token) <= 4 {
		return token
	}
	return strings.Repeat("*", len(token)-4) + token[len(token)-4:]
}
