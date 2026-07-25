package cli

import (
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/richhaase/c2/internal/envelope"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/notes"
	"github.com/richhaase/c2/internal/storage"
	"github.com/richhaase/c2/internal/store"
)

type notesPayload struct {
	Count int            `json:"count"`
	Notes []notes.Record `json:"notes"`
}

var noteDatePattern = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})([T ].+)?$`)

var offsetSuffixPattern = regexp.MustCompile(`(?:[+-]\d{2}:\d{2}|Z)$`)

func parseNoteDate(raw string) (string, bool) {
	m := noteDatePattern.FindStringSubmatch(raw)
	if m == nil || !models.IsValidYMD(m[1]) {
		return "", false
	}
	if offsetSuffixPattern.MatchString(raw) {
		normalized := strings.Replace(raw, " ", "T", 1)
		if strings.HasSuffix(normalized, "Z") {
			normalized = normalized[:len(normalized)-1] + "+00:00"
		}
		if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\S+$`).MatchString(normalized) {
			return "", false
		}
		if _, ok := notes.ParseDate(normalized); !ok {
			return "", false
		}
		return normalized, true
	}
	if m[2] == "" {
		t := models.ParseLocal(raw + "T12:00:00")
		if t.IsZero() {
			return "", false
		}
		return notes.LocalISO(t), true
	}
	t := models.ParseLocal(raw)
	if t.IsZero() {
		return "", false
	}
	return notes.LocalISO(t), true
}

func readBody(cmd *cobra.Command, arg string) (string, error) {
	if arg != "" && arg != "-" {
		return strings.TrimSpace(arg), nil
	}
	data, err := io.ReadAll(cmd.InOrStdin())
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func noteLine(n notes.Record) string {
	workout := ""
	if n.WorkoutID != nil {
		workout = fmt.Sprintf(" w:%d", *n.WorkoutID)
	}
	tags := ""
	if len(n.Tags) > 0 {
		tags = " #" + strings.Join(n.Tags, " #")
	}
	return fmt.Sprintf("%s  [%s/%s]%s%s  %s", n.Date[:10], n.Type, n.Author, workout, tags, n.Body)
}

func newNoteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "note",
		Short: "Coaching notes and subjective reports",
	}
	cmd.AddCommand(newNoteAddCmd(), newNoteListCmd(), newNoteShowCmd())
	return cmd
}

func newNoteAddCmd() *cobra.Command {
	var (
		noteType string
		workout  string
		tags     string
		author   string
		dateFlag string
	)
	cmd := &cobra.Command{
		Use:   "add [body]",
		Short: "Record a note (body as argument, or '-' / omitted to read stdin)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !slices.Contains(notes.Types, noteType) {
				return reportf(cmd, "Error: --type must be one of %s.", strings.Join(notes.Types, ", "))
			}
			if !slices.Contains(notes.Authors, author) {
				return reportf(cmd, "Error: --author must be one of %s.", strings.Join(notes.Authors, ", "))
			}

			backdate := ""
			if dateFlag != "" {
				parsed, ok := parseNoteDate(dateFlag)
				if !ok {
					return reportf(cmd, "Error: invalid --date %q.", dateFlag)
				}
				backdate = parsed
			}

			bodyArg := ""
			if len(args) > 0 {
				bodyArg = args[0]
			}
			body, err := readBody(cmd, bodyArg)
			if err != nil {
				return err
			}
			if body == "" {
				return reportf(cmd, "Error: note body is empty.")
			}

			_, p, err := loadStore()
			if err != nil {
				return err
			}
			now := time.Now()
			warn := warner(cmd)
			if err := store.RejectForeign(p, warn); err != nil {
				return reportf(cmd, "%v", err)
			}

			var workoutID *int64
			if workout != "" {
				workouts, err := storage.ReadWorkouts(p)
				if err != nil {
					return err
				}
				w := models.ResolveWorkout(workouts, workout)
				if w == nil {
					return reportf(cmd, "Error: no workout matching %q.", workout)
				}
				workoutID = &w.ID
			}

			if err := store.EnsureForWrite(p, now, warn); err != nil {
				return reportf(cmd, "%v", err)
			}

			date := backdate
			if date == "" {
				date = notes.LocalISO(now)
			}

			var tagList []string
			if tags != "" {
				for _, t := range strings.Split(tags, ",") {
					if trimmed := strings.TrimSpace(t); trimmed != "" {
						tagList = append(tagList, trimmed)
					}
				}
			}

			record := notes.Record{
				ID:        notes.ULID(now),
				Date:      date,
				Type:      noteType,
				WorkoutID: workoutID,
				Tags:      tagList,
				Body:      body,
				Author:    author,
			}
			if !notes.IsShaped(record) {
				return reportf(cmd, "Error: note would not round-trip; check --date and field values.")
			}
			if err := notes.Write(p, record); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), record.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&noteType, "type", "observation", "subjective, observation, or lesson")
	cmd.Flags().StringVar(&workout, "workout", "", "link to a workout id or 'last'")
	cmd.Flags().StringVar(&tags, "tags", "", "comma-separated tags")
	cmd.Flags().StringVar(&author, "author", "athlete", "athlete or coach")
	cmd.Flags().StringVar(&dateFlag, "date", "", "backdate the note (YYYY-MM-DD or ISO timestamp)")
	return cmd
}

func newNoteListCmd() *cobra.Command {
	var (
		noteType string
		since    string
		workout  string
		count    string
		asJSON   bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List notes, newest last",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if since != "" && !models.IsValidYMD(since) {
				return reportf(cmd, "Error: invalid --since date %q (expected YYYY-MM-DD).", since)
			}
			if noteType != "" && !slices.Contains(notes.Types, noteType) {
				return reportf(cmd, "Error: --type must be one of %s.", strings.Join(notes.Types, ", "))
			}
			var workoutID *int64
			if workout != "" {
				id, err := strconv.ParseInt(workout, 10, 64)
				if err != nil {
					return reportf(cmd, "Error: invalid --workout id %q.", workout)
				}
				workoutID = &id
			}

			_, p, err := loadStore()
			if err != nil {
				return err
			}
			if err := store.RejectForeign(p, warner(cmd)); err != nil {
				return reportf(cmd, "%v", err)
			}
			matched := notes.Apply(notes.ReadAll(p), notes.Filter{
				Type:      noteType,
				Since:     since,
				WorkoutID: workoutID,
			})
			if count != "" {
				n, err := positiveInt(cmd, count, "Error: --count must be a positive integer.")
				if err != nil {
					return err
				}
				if n < len(matched) {
					matched = matched[len(matched)-n:]
				}
			}

			out := cmd.OutOrStdout()
			if asJSON {
				if matched == nil {
					matched = []notes.Record{}
				}
				return envelope.Print(out, "c2.notes.v1", notesPayload{Count: len(matched), Notes: matched})
			}
			if len(matched) == 0 {
				fmt.Fprintln(out, "No notes found.")
				return nil
			}
			for _, n := range matched {
				fmt.Fprintln(out, noteLine(n))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&noteType, "type", "", "filter by type")
	cmd.Flags().StringVar(&since, "since", "", "only notes on or after date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&workout, "workout", "", "only notes linked to a workout id")
	cmd.Flags().StringVarP(&count, "count", "n", "", "show only the most recent n notes")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}

func newNoteShowCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "show <id>",
		Short: "Show one note in full",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, p, err := loadStore()
			if err != nil {
				return err
			}
			if err := store.RejectForeign(p, warner(cmd)); err != nil {
				return reportf(cmd, "%v", err)
			}
			var match *notes.Record
			for _, n := range notes.ReadAll(p) {
				if n.ID == args[0] {
					match = &n
					break
				}
			}
			if match == nil {
				return reportf(cmd, "No note with id %s.", args[0])
			}

			out := cmd.OutOrStdout()
			if asJSON {
				return envelope.Print(out, "c2.note.v1", match)
			}
			fmt.Fprintf(out, "Id: %s\n", match.ID)
			fmt.Fprintf(out, "Date: %s\n", match.Date)
			fmt.Fprintf(out, "Type: %s (%s)\n", match.Type, match.Author)
			if match.WorkoutID != nil {
				fmt.Fprintf(out, "Workout: %d\n", *match.WorkoutID)
			}
			if len(match.Tags) > 0 {
				fmt.Fprintf(out, "Tags: %s\n", strings.Join(match.Tags, ", "))
			}
			fmt.Fprintln(out)
			fmt.Fprintln(out, match.Body)
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}
