package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/atomicfile"
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

type documentPayload struct {
	Name    string `json:"name"`
	Date    string `json:"date,omitempty"`
	Content string `json:"content"`
}

func documentText(content string) string {
	if !strings.HasSuffix(content, "\n") {
		return content + "\n"
	}
	return content
}

func writeDocument(path, content string) error {
	return atomicfile.Write(path, []byte(documentText(content)), 0o644)
}

func newDocCmd(name, short string, pathOf func(paths.DataPaths) string) *cobra.Command {
	doc := &cobra.Command{Use: name, Short: short}
	var input contentInput
	var showJSON, setJSON bool

	show := &cobra.Command{
		Use:   "show",
		Short: "Print the " + name,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, p, err := loadReadableStore(cmd)
			if err != nil {
				return err
			}
			content, ok, err := documents.Read(pathOf(p))
			if err != nil {
				return err
			}
			if !ok {
				return reportf(cmd, "No %s recorded yet. Set one with `c2 %s set <file|->`.", name, name)
			}
			if showJSON {
				return envelope.Print(cmd.OutOrStdout(), "c2.document.v1", documentPayload{Name: name, Content: content})
			}
			fmt.Fprint(cmd.OutOrStdout(), content)
			return nil
		},
	}
	show.Flags().BoolVar(&showJSON, "json", false, "output document as JSON")
	doc.AddCommand(show)

	set := &cobra.Command{
		Use:     "set [file]",
		Short:   "Replace the " + name + " from a file or stdin",
		Example: "  c2 " + name + " set --file " + name + ".md\n  c2 " + name + " set --file - --json\n  c2 " + name + " show --json",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := input.read(cmd, args, true)
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
			if setJSON {
				return envelope.Print(cmd.OutOrStdout(), "c2.document.v1", documentPayload{Name: name, Content: documentText(content)})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s updated (%d chars).\n", name, len(content))
			return nil
		},
	}
	input.flags(set)
	set.Flags().BoolVar(&setJSON, "json", false, "output saved document as JSON")
	doc.AddCommand(set)

	return doc
}

func newNarrativeCmd() *cobra.Command {
	narrative := &cobra.Command{
		Use:   "narrative",
		Short: "Dated coaching report narratives",
	}

	var input contentInput
	var addJSON, showJSON bool
	add := &cobra.Command{
		Use:     "add <date> [file]",
		Short:   "Save the narrative for a date (YYYY-MM-DD) from a file or stdin",
		Example: "  c2 narrative add 2026-09-26 --file report.md\n  c2 narrative add 2026-09-26 --file - --json\n  c2 narrative show --json",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			date := args[0]
			if !models.IsValidYMD(date) {
				return reportf(cmd, "Error: invalid date %q (expected YYYY-MM-DD).", date)
			}
			content, err := input.read(cmd, args[1:], true)
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
			if addJSON {
				return envelope.Print(cmd.OutOrStdout(), "c2.document.v1", documentPayload{Name: "narrative", Date: date, Content: documentText(content)})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Narrative saved for %s.\n", date)
			return nil
		},
	}
	input.flags(add)
	add.Flags().BoolVar(&addJSON, "json", false, "output saved narrative as JSON")
	narrative.AddCommand(add)

	show := &cobra.Command{
		Use:   "show [date]",
		Short: "Print the narrative for a date (latest if omitted)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, p, err := loadReadableStore(cmd)
			if err != nil {
				return err
			}
			target := ""
			if len(args) > 0 {
				target = args[0]
				if !models.IsValidYMD(target) {
					return reportf(cmd, "Error: invalid date %q (expected YYYY-MM-DD).", target)
				}
			} else {
				dates, err := documents.ListNarratives(p)
				if err != nil {
					return err
				}
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
			if showJSON {
				return envelope.Print(cmd.OutOrStdout(), "c2.document.v1", documentPayload{Name: "narrative", Date: target, Content: content})
			}
			fmt.Fprint(cmd.OutOrStdout(), content)
			return nil
		},
	}
	show.Flags().BoolVar(&showJSON, "json", false, "output narrative as JSON")
	narrative.AddCommand(show)

	var asJSON bool
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List narrative dates",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, p, err := loadReadableStore(cmd)
			if err != nil {
				return err
			}
			dates, err := documents.ListNarratives(p)
			if err != nil {
				return err
			}
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
