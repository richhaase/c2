package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/paths"
)

func reportf(cmd *cobra.Command, format string, args ...any) error {
	fmt.Fprintf(cmd.ErrOrStderr(), format+"\n", args...)
	return errReported
}

func configGoalEnd(raw string) (time.Time, error) {
	return config.ParseGoalDate(raw)
}

func loadStore() (config.Config, paths.DataPaths, error) {
	cfg, err := config.Load()
	if err != nil {
		return cfg, paths.DataPaths{}, err
	}
	return cfg, paths.For(cfg.DataDir), nil
}

func validateDateFlag(cmd *cobra.Command, flag, value string) error {
	if value == "" || models.IsValidYMD(value) {
		return nil
	}
	return reportf(cmd, "Error: invalid %s date %q (expected YYYY-MM-DD).", flag, value)
}

func positiveInt(cmd *cobra.Command, raw, message string) (int, error) {
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, reportf(cmd, "%s", message)
	}
	return n, nil
}
