package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/storage"
	"github.com/richhaase/c2/internal/store"
)

const maxWeeks = 520

func reportf(cmd *cobra.Command, format string, args ...any) error {
	fmt.Fprintf(cmd.ErrOrStderr(), format+"\n", args...)
	return errReported
}

func loadStore() (config.Config, paths.DataPaths, error) {
	cfg, err := config.Load()
	if err != nil {
		return cfg, paths.DataPaths{}, err
	}
	return cfg, paths.For(cfg.DataDir), nil
}

func loadWorkouts(cmd *cobra.Command) (config.Config, paths.DataPaths, []models.Workout, error) {
	cfg, p, err := loadStore()
	if err != nil {
		return cfg, p, nil, err
	}
	if err := store.RejectForeign(p, warner(cmd)); err != nil {
		return cfg, p, nil, reportf(cmd, "%v", err)
	}
	workouts, err := storage.ReadWorkouts(p)
	return cfg, p, workouts, err
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

func weekCount(cmd *cobra.Command, raw string) (int, error) {
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > maxWeeks {
		return 0, reportf(cmd, "Error: --weeks must be between 1 and %d.", maxWeeks)
	}
	return n, nil
}
