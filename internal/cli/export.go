package cli

import (
	"fmt"
	"sort"

	exporter "github.com/richhaase/c2/internal/export"
	"github.com/spf13/cobra"
)

func newExportCommand(deps Dependencies) *cobra.Command {
	var format string
	var from string
	var to string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export workouts to CSV or JSON",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			workouts, err := deps.ReadWorkouts()
			if err != nil {
				return err
			}
			if len(workouts) == 0 {
				return fmt.Errorf("no workouts found. Run `c2 sync` first")
			}

			workouts, err = exporter.FilterByDate(workouts, from, to)
			if err != nil {
				return err
			}
			if len(workouts) == 0 {
				return fmt.Errorf("no workouts match the specified date range")
			}
			sort.Slice(workouts, func(i, j int) bool {
				return workouts[i].Date < workouts[j].Date
			})

			var output string
			switch format {
			case "csv":
				output, err = exporter.CSV(workouts)
			case "json":
				output, err = exporter.JSON(workouts)
			case "jsonl":
				output, err = exporter.JSONL(workouts)
			default:
				return fmt.Errorf("unsupported format %q: must be csv, json, or jsonl", format)
			}
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), output)
			return err
		},
	}
	cmd.Flags().StringVarP(&format, "format", "f", "csv", "output format: csv, json, or jsonl")
	cmd.Flags().StringVar(&from, "from", "", "filter workouts from date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&to, "to", "", "filter workouts to date (YYYY-MM-DD)")
	return cmd
}
