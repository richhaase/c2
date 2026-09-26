package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/envelope"
	"github.com/richhaase/c2/internal/goals"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/notes"
	"github.com/richhaase/c2/internal/store"
)

type goalsPayload struct {
	Goals []goals.Progress `json:"goals"`
}

func newGoalCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "goal", Short: "Manage personal goals and inspect qualifying evidence"}
	cmd.Example = "  c2 goal list\n  c2 goal add 'Comfortable pace' --kind pace --target 2:30 --timezone America/Denver\n  c2 goal show <id> --json\n  c2 goal update <id> --target 2:25 --json\n  c2 goal archive <id> --json"
	cmd.AddCommand(newGoalListCmd(), newGoalWriteCmd(false), newGoalWriteCmd(true), newGoalArchiveCmd())
	return cmd
}

func goalValue(kind string, value float64) string {
	if kind == "pace" {
		return models.FormatSeconds(value) + "/500m"
	}
	return models.ToFixed(value, 0) + " m"
}

func printGoals(cmd *cobra.Command, results []goals.Progress) {
	out := cmd.OutOrStdout()
	if len(results) == 0 {
		fmt.Fprintln(out, "No active goals. Create one with `c2 goal add`.")
	}
	for _, p := range results {
		g := p.Goal
		state := "in progress"
		if p.Achieved {
			state = "achieved on " + p.AchievedOn
		}
		if g.Archived {
			state = "archived; " + state
		}
		fmt.Fprintf(out, "%s  %s — %s\n", g.ID, g.Name, state)
		from, to := g.From, g.To
		if from == "" {
			from = "available history"
		}
		if to == "" {
			to = "no deadline"
		}
		fmt.Fprintf(out, "  %s → %s; %s; %s efforts\n", from, to, g.Equipment, g.Effort)
		value := "no qualifying evidence"
		if p.Value != nil {
			value = goalValue(g.Kind, *p.Value)
		}
		fmt.Fprintf(out, "  %s; target %s\n", value, goalValue(g.Kind, g.Target))
		if g.Kind == "pace" {
			fmt.Fprintln(out, "  Evidence basis: whole-workout average (interval records exclude rests).")
		}
		if g.MinDistance > 0 {
			fmt.Fprintf(out, "  Minimum effort distance: %d m\n", g.MinDistance)
		}
		if p.Evidence != nil {
			fmt.Fprintf(out, "  Workout %d: %s, %d m, %s\n", p.Evidence.WorkoutID, p.Evidence.Date, p.Evidence.Distance, p.Evidence.Continuity)
		}
		if p.UnknownEfforts > 0 {
			fmt.Fprintf(out, "  %d workouts lack evidence of continuity and were excluded.\n", p.UnknownEfforts)
		}
	}
}

func newGoalListCmd() *cobra.Command {
	var asJSON, all bool
	cmd := &cobra.Command{Use: "list [id]", Aliases: []string{"show"}, Short: "Show goals, progress, and workout evidence", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		cfg, p, workouts, err := loadWorkouts(cmd)
		if err != nil {
			return err
		}
		items, loc, err := goals.Load(p, cfg)
		if err != nil {
			return err
		}
		results := make([]goals.Progress, 0)
		for _, g := range items {
			if len(args) == 1 && g.ID != args[0] {
				continue
			}
			if len(args) == 0 && g.Archived && !all {
				continue
			}
			results = append(results, goals.Evaluate(g, workouts, cfg.Now().In(loc)))
		}
		if len(args) == 1 && len(results) == 0 {
			return fmt.Errorf("No goal with ID %s.", args[0])
		}
		if asJSON {
			return envelope.Print(cmd.OutOrStdout(), "c2.goal.v1", goalsPayload{Goals: results})
		}
		printGoals(cmd, results)
		return nil
	}}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	cmd.Flags().BoolVar(&all, "all", false, "include archived goals")
	return cmd
}

func parseGoalTarget(kind, value string) (float64, error) {
	if kind == "pace" && strings.Contains(value, ":") {
		parts := strings.Split(value, ":")
		if len(parts) != 2 {
			return 0, fmt.Errorf("Pace must be seconds or m:ss per 500m.")
		}
		minutes, e1 := strconv.Atoi(parts[0])
		seconds, e2 := strconv.ParseFloat(parts[1], 64)
		if e1 != nil || e2 != nil || minutes < 0 || seconds < 0 || seconds >= 60 {
			return 0, fmt.Errorf("Pace must be seconds or m:ss per 500m.")
		}
		return float64(minutes)*60 + seconds, nil
	}
	return strconv.ParseFloat(value, 64)
}

func newGoalWriteCmd(update bool) *cobra.Command {
	var name, kind, target, from, to, equipment, effort, zone string
	var minDistance int
	var archived, asJSON bool
	use, short := "add <name>", "Create a personal goal"
	if update {
		use, short = "update <id>", "Revise a goal; progress is recalculated from recorded history"
	}
	cmd := &cobra.Command{Use: use, Short: short, Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		cfg, p, workouts, err := loadWorkouts(cmd)
		if err != nil {
			return err
		}
		c, err := goals.Read(p)
		if err != nil {
			return err
		}
		if c == nil {
			created, err := goals.NewCollection(cfg, zone, workouts)
			if err != nil {
				return err
			}
			c = &created
		}
		if zone != "" && zone != c.Timezone {
			return fmt.Errorf("This store uses %s; a goal change cannot change its timezone.", c.Timezone)
		}
		g := goals.Goal{Name: args[0], Kind: kind, Equipment: equipment, Effort: effort}
		index := -1
		if update {
			for i, item := range c.Goals {
				if item.ID == args[0] {
					g = item
					index = i
					break
				}
			}
			if index < 0 {
				return fmt.Errorf("No goal with ID %s.", args[0])
			}
		} else {
			id, err := notes.ULID(cfg.Now())
			if err != nil {
				return err
			}
			g.ID = id
			if kind == "distance" && !cmd.Flags().Changed("effort") {
				g.Effort = "continuous"
			}
		}
		if cmd.Flags().Changed("name") {
			g.Name = name
		}
		if cmd.Flags().Changed("kind") {
			g.Kind = kind
		}
		if !update || cmd.Flags().Changed("target") {
			g.Target, err = parseGoalTarget(g.Kind, target)
			if err != nil {
				return err
			}
		}
		if cmd.Flags().Changed("from") {
			g.From = from
		}
		if cmd.Flags().Changed("to") {
			g.To = to
		}
		if cmd.Flags().Changed("equipment") {
			g.Equipment = equipment
		}
		if cmd.Flags().Changed("effort") {
			g.Effort = effort
		}
		if cmd.Flags().Changed("min-distance") {
			g.MinDistance = minDistance
		}
		if cmd.Flags().Changed("archived") {
			g.Archived = archived
		}
		if err := goals.Validate(g); err != nil {
			return err
		}
		if index < 0 {
			c.Goals = append(c.Goals, g)
		} else {
			c.Goals[index] = g
		}
		if err := store.EnsureForWrite(p, cfg.Now(), warner(cmd)); err != nil {
			return err
		}
		if err := goals.Write(p, *c); err != nil {
			return err
		}
		if asJSON {
			return envelope.Print(cmd.OutOrStdout(), "c2.goal.saved.v1", g)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Saved goal %s: %s\n", g.ID, g.Name)
		return nil
	}}
	cmd.Flags().StringVar(&name, "name", "", "goal name")
	cmd.Flags().StringVar(&kind, "kind", "volume", "volume, distance, or pace")
	cmd.Flags().StringVar(&target, "target", "", "meters, or seconds/m:ss per 500m for pace")
	cmd.Flags().StringVar(&from, "from", "", "inclusive start date; empty means available history")
	cmd.Flags().StringVar(&to, "to", "", "inclusive end date; empty means no deadline")
	cmd.Flags().StringVar(&equipment, "equipment", "rower", "Concept2 equipment type, or all")
	cmd.Flags().StringVar(&effort, "effort", "workout", "workout or continuous (distance goals default to continuous)")
	cmd.Flags().IntVar(&minDistance, "min-distance", 0, "minimum workout meters for a pace goal")
	cmd.Flags().StringVar(&zone, "timezone", "", "analysis timezone when first preparing this store")
	cmd.Flags().BoolVar(&archived, "archived", false, "retire or restore this goal")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output saved goal as JSON")
	return cmd
}

func newGoalArchiveCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{Use: "archive <id>", Short: "Retire a goal without deleting it", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		cfg, p, err := loadStore()
		if err != nil {
			return err
		}
		c, err := goals.Read(p)
		if err != nil {
			return err
		}
		if c == nil {
			return fmt.Errorf("Prepare the store with `c2 data prepare --timezone <zone>` first.")
		}
		for i, g := range c.Goals {
			if g.ID == args[0] {
				if err := store.EnsureForWrite(p, cfg.Now(), warner(cmd)); err != nil {
					return err
				}
				c.Goals[i].Archived = true
				if err := goals.Write(p, *c); err != nil {
					return err
				}
				if asJSON {
					return envelope.Print(cmd.OutOrStdout(), "c2.goal.saved.v1", c.Goals[i])
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Archived %s at %s.\n", g.Name, cfg.Now().Format("2006-01-02"))
				return nil
			}
		}
		return fmt.Errorf("No goal with ID %s.", args[0])
	}}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output saved goal as JSON")
	return cmd
}
