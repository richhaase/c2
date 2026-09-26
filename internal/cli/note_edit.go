package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/envelope"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/notes"
	"github.com/richhaase/c2/internal/storage"
	"github.com/richhaase/c2/internal/store"
)

type noteEditPayload struct {
	Changed bool         `json:"changed"`
	Note    notes.Record `json:"note"`
}

func newNoteEditCmd() *cobra.Command {
	var input contentInput
	var kind, author, date, workout, tags string
	var asJSON bool
	cmd := &cobra.Command{
		Use:     "edit <id> [body]",
		Short:   "Correct a recent or archived note; opens your editor when no changes are supplied",
		Example: "  c2 note list -n 5\n  c2 note edit <id>\n  c2 note edit <id> --body 'Corrected feedback' --json\n  c2 note edit <id> --file correction.md\n  c2 note edit <id> --workout last --tags technique,recovery\n  c2 note edit <id> --workout '' --tags ''",
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, p, err := loadStore()
			if err != nil {
				return err
			}
			if err := store.RejectForeign(p, warner(cmd)); err != nil {
				return err
			}
			if err := store.CheckSchema(p, warner(cmd)); err != nil {
				return err
			}
			before, err := notes.ReadForEdit(p, args[0])
			if err != nil {
				return err
			}
			after := before
			metadata := false
			for _, flag := range []string{"type", "author", "date", "workout", "tags"} {
				metadata = metadata || cmd.Flags().Changed(flag)
			}
			if cmd.Flags().Changed("type") {
				if !slices.Contains(notes.Types, kind) {
					return fmt.Errorf("--type must be one of %s.", strings.Join(notes.Types, ", "))
				}
				after.Type = kind
			}
			if cmd.Flags().Changed("author") {
				if !slices.Contains(notes.Authors, author) {
					return fmt.Errorf("--author must be one of %s.", strings.Join(notes.Authors, ", "))
				}
				after.Author = author
			}
			if cmd.Flags().Changed("date") {
				var ok bool
				after.Date, ok = parseNoteDate(date)
				if !ok {
					return fmt.Errorf("Invalid --date %q; use YYYY-MM-DD or an ISO timestamp.", date)
				}
			}
			if cmd.Flags().Changed("tags") {
				after.Tags = noteTags(tags)
			}
			if cmd.Flags().Changed("workout") {
				after.WorkoutID = nil
				if workout != "" {
					workouts, err := storage.ReadWorkouts(p)
					if err != nil {
						return err
					}
					w := models.ResolveWorkout(workouts, workout)
					if w == nil {
						return fmt.Errorf("No workout matching %q; use `c2 log` to find its ID.", workout)
					}
					after.WorkoutID = &w.ID
				}
			}
			if input.supplied(cmd, args[1:]) {
				after.Body, err = input.read(cmd, args[1:], false)
			} else if !metadata {
				if !interactiveInput(cmd) {
					return fmt.Errorf("No interactive terminal. Use --body, --file <path|->, or explicit metadata options to edit a note.")
				}
				after.Body, err = editInEditor(cmd, before.Body)
			}
			if err != nil {
				return err
			}
			changed, err := notes.Update(p, before, after)
			if err != nil {
				return err
			}
			if asJSON {
				return envelope.Print(cmd.OutOrStdout(), "c2.note.edited.v1", noteEditPayload{Changed: changed, Note: after})
			}
			if changed {
				fmt.Fprintf(cmd.OutOrStdout(), "Updated note %s. Inspect it with `c2 note show %s`.\n", after.ID, after.ID)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Note %s unchanged.\n", after.ID)
			}
			return nil
		},
	}
	input.flags(cmd)
	cmd.Flags().StringVar(&kind, "type", "", "correct the note type: subjective, observation, or lesson")
	cmd.Flags().StringVar(&author, "author", "", "correct attribution: athlete or coach")
	cmd.Flags().StringVar(&date, "date", "", "correct the date (YYYY-MM-DD or ISO timestamp)")
	cmd.Flags().StringVar(&workout, "workout", "", "workout ID or 'last'; empty clears the link")
	cmd.Flags().StringVar(&tags, "tags", "", "replace comma-separated tags; empty clears them")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output the saved note and whether it changed as JSON")
	return cmd
}

func noteTags(raw string) []string {
	var tags []string
	for _, tag := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(tag); trimmed != "" {
			tags = append(tags, trimmed)
		}
	}
	return tags
}
