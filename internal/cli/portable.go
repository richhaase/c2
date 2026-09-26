package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/doctor"
	"github.com/richhaase/c2/internal/goals"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/storage"
	"github.com/richhaase/c2/internal/store"
)

func validatePortableStore(p paths.DataPaths, cmd *cobra.Command) error {
	inspection, err := store.Inspect(p, warner(cmd))
	if err != nil {
		return err
	}
	if inspection.State != store.StateStore {
		return fmt.Errorf("No existing C2 store at %s.", p.Root)
	}
	if err := store.CheckSchema(p, warner(cmd)); err != nil {
		return err
	}
	r := doctor.Run(p)
	if len(r.Issues) > 0 {
		return fmt.Errorf("Store validation failed:\n%s", strings.Join(r.Issues, "\n"))
	}
	return nil
}

func newDataPrepareCmd() *cobra.Command {
	var zone string
	cmd := &cobra.Command{Use: "prepare", Short: "Make goals and calendar settings portable before copying this store", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		cfg, p, workouts, err := loadWorkouts(cmd)
		if err != nil {
			return err
		}
		if err := validatePortableStore(p, cmd); err != nil {
			return err
		}
		c, err := goals.Read(p)
		if err != nil {
			return err
		}
		if c != nil {
			if zone != "" && zone != c.Timezone {
				return fmt.Errorf("This store already uses %s; preparing it cannot change calendar semantics.", c.Timezone)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Store already prepared: %s, %d goals.\n", c.Timezone, len(c.Goals))
			return nil
		}
		created, err := goals.NewCollection(cfg, zone, workouts)
		if err != nil {
			return err
		}
		if err := goals.Write(p, created); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Prepared %s with %d goals and timezone %s. Copy the entire folder while C2 is idle.\n", p.Root, len(created.Goals), created.Timezone)
		return nil
	}}
	cmd.Flags().StringVar(&zone, "timezone", "", "analysis timezone, for example America/Denver or UTC")
	return cmd
}

func newDataUseCmd() *cobra.Command {
	return &cobra.Command{Use: "use <dir>", Short: "Adopt a prepared store without changing its goals or requiring a token", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		p := paths.For(paths.CanonicalRoot(args[0]))
		if err := validatePortableStore(p, cmd); err != nil {
			return err
		}
		c, err := goals.Read(p)
		if err != nil {
			return err
		}
		if c == nil {
			return fmt.Errorf("Prepare the store on its source machine with `c2 data prepare --timezone <zone>` before adopting it.")
		}
		workouts, err := storage.ReadWorkouts(p)
		if err != nil {
			return err
		}
		if c.AccountID != 0 {
			if err := goals.CheckAccount(c, workouts, c.AccountID); err != nil {
				return err
			}
		}
		cfg.DataDir = p.Root
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Using %s (%s, %d goals). Credentials remain local.\n", p.Root, c.Timezone, len(c.Goals))
		return nil
	}}
}
