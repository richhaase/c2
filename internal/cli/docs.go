package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/documents"
	"github.com/richhaase/c2/internal/envelope"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/store"
)

type narrativesPayload struct {
	Count int      `json:"count"`
	Dates []string `json:"dates"`
}

func readContent(cmd *cobra.Command, source string) (string, error) {
	if source != "" && source != "-" {
		data, err := os.ReadFile(source)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	data, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func writeDocument(path, content string) error {
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func newDocCmd(name, short string, pathOf func(paths.DataPaths) string) *cobra.Command {
	doc := &cobra.Command{Use: name, Short: short}

	doc.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Print the " + name,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, p, err := loadStore()
			if err != nil {
				return err
			}
			if err := store.RejectForeign(p, warner(cmd)); err != nil {
				return reportf(cmd, "%v", err)
			}
			content, ok, err := documents.Read(pathOf(p))
			if err != nil {
				return err
			}
			if !ok {
				return reportf(cmd, "No %s recorded yet. Set one with `c2 %s set <file|->`.", name, name)
			}
			fmt.Fprint(cmd.OutOrStdout(), content)
			return nil
		},
	})

	doc.AddCommand(&cobra.Command{
		Use:   "set [file]",
		Short: "Replace the " + name + " from a file or stdin",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			source := ""
			if len(args) > 0 {
				source = args[0]
			}
			content, err := readContent(cmd, source)
			if err != nil {
				return err
			}
			if strings.TrimSpace(content) == "" {
				return reportf(cmd, "Error: refusing to save an empty %s.", name)
			}
			_, p, err := loadStore()
			if err != nil {
				return err
			}
			if err := store.EnsureForWrite(p, time.Now(), warner(cmd)); err != nil {
				return reportf(cmd, "%v", err)
			}
			if err := writeDocument(pathOf(p), content); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s updated (%d chars).\n", name, len(content))
			return nil
		},
	})

	return doc
}

func newNarrativeCmd() *cobra.Command {
	narrative := &cobra.Command{
		Use:   "narrative",
		Short: "Dated coaching report narratives",
	}

	narrative.AddCommand(&cobra.Command{
		Use:   "add <date> [file]",
		Short: "Save the narrative for a date (YYYY-MM-DD) from a file or stdin",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			date := args[0]
			if !models.IsValidYMD(date) {
				return reportf(cmd, "Error: invalid date %q (expected YYYY-MM-DD).", date)
			}
			source := ""
			if len(args) > 1 {
				source = args[1]
			}
			content, err := readContent(cmd, source)
			if err != nil {
				return err
			}
			if strings.TrimSpace(content) == "" {
				return reportf(cmd, "Error: refusing to save an empty narrative.")
			}
			_, p, err := loadStore()
			if err != nil {
				return err
			}
			if err := store.EnsureForWrite(p, time.Now(), warner(cmd)); err != nil {
				return reportf(cmd, "%v", err)
			}
			if err := writeDocument(p.NarrativeFile(date), content); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Narrative saved for %s.\n", date)
			return nil
		},
	})

	narrative.AddCommand(&cobra.Command{
		Use:   "show [date]",
		Short: "Print the narrative for a date (latest if omitted)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, p, err := loadStore()
			if err != nil {
				return err
			}
			if err := store.RejectForeign(p, warner(cmd)); err != nil {
				return reportf(cmd, "%v", err)
			}
			target := ""
			if len(args) > 0 {
				target = args[0]
				if !models.IsValidYMD(target) {
					return reportf(cmd, "Error: invalid date %q (expected YYYY-MM-DD).", target)
				}
			} else {
				dates := documents.ListNarratives(p)
				if len(dates) == 0 {
					return reportf(cmd, "No narratives recorded yet.")
				}
				target = dates[len(dates)-1]
			}
			content, ok, err := documents.Read(p.NarrativeFile(target))
			if err != nil {
				return err
			}
			if !ok {
				return reportf(cmd, "No narrative for %s.", target)
			}
			fmt.Fprint(cmd.OutOrStdout(), content)
			return nil
		},
	})

	var asJSON bool
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List narrative dates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, p, err := loadStore()
			if err != nil {
				return err
			}
			if err := store.RejectForeign(p, warner(cmd)); err != nil {
				return reportf(cmd, "%v", err)
			}
			dates := documents.ListNarratives(p)
			out := cmd.OutOrStdout()
			if asJSON {
				if dates == nil {
					dates = []string{}
				}
				return envelope.Print(out, "c2.narratives.v1", narrativesPayload{
					Count: len(dates),
					Dates: dates,
				})
			}
			if len(dates) == 0 {
				fmt.Fprintln(out, "No narratives recorded yet.")
				return nil
			}
			for _, d := range dates {
				fmt.Fprintln(out, d)
			}
			return nil
		},
	}
	listCmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	narrative.AddCommand(listCmd)

	return narrative
}
