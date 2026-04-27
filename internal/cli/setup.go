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
				if _, writeErr := fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not load existing config: %v\n", err); writeErr != nil {
					return writeErr
				}
				if _, writeErr := fmt.Fprintln(cmd.ErrOrStderr(), "Starting from defaults."); writeErr != nil {
					return writeErr
				}
				cfg = config.Default()
			}

			reader := bufio.NewReader(deps.Stdin)
			out := cmd.OutOrStdout()
			if _, err := fmt.Fprintln(out, "Concept2 CLI Setup"); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}

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
				if _, err := fmt.Fprintf(out, "Invalid date %q, keeping previous value.\n", start); err != nil {
					return err
				}
			}

			end, err := promptValue(reader, out, "Goal end date (YYYY-MM-DD)", cfg.Goal.EndDate, false)
			if err != nil {
				return err
			}
			if _, err := config.ParseGoalDate(end); err == nil {
				cfg.Goal.EndDate = end
			} else if end != "" {
				if _, err := fmt.Fprintf(out, "Invalid date %q, keeping previous value.\n", end); err != nil {
					return err
				}
			}

			if err := deps.EnsureDirs(); err != nil {
				return err
			}
			if err := deps.SaveConfig(cfg); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "\nConfig written to %s\n", filepath.Join(config.Dir(), "config.json")); err != nil {
				return err
			}

			if cfg.Goal.StartDate == "" || cfg.Goal.EndDate == "" {
				if _, err := fmt.Fprintln(out, "\nNote: Goal dates not set. Commands like `c2 status` require start/end dates."); err != nil {
					return err
				}
			}

			if cfg.API.Token != "" {
				if _, err := fmt.Fprintln(out, "Verifying token..."); err != nil {
					return err
				}
				user, err := deps.VerifyUser(cmd.Context(), cfg, version)
				if err != nil {
					if _, writeErr := fmt.Fprintf(cmd.ErrOrStderr(), "Warning: could not verify token: %v\n", err); writeErr != nil {
						return writeErr
					}
				} else {
					if _, err := fmt.Fprintf(out, "Authenticated as: %s (ID: %d)\n", user.Username, user.ID); err != nil {
						return err
					}
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
		if _, err := fmt.Fprintf(out, "%s [%s]: ", label, displayValue); err != nil {
			return "", err
		}
	} else {
		if _, err := fmt.Fprintf(out, "%s: ", label); err != nil {
			return "", err
		}
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
