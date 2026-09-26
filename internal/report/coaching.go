package report

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/richhaase/c2/internal/documents"
	"github.com/richhaase/c2/internal/notes"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/stats"
)

const (
	recentNoteDays      = 14
	maxRecentNotes      = 20
	planExcerptMaxChars = 1500
)

var htmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

func esc(s string) string {
	return htmlEscaper.Replace(s)
}

type Narrative struct {
	Date string `json:"date"`
	Text string `json:"text"`
}

type coachingContent struct {
	narrative   *Narrative
	notes       []notes.Record
	planExcerpt *string
}

func utf16Len(s string) int {
	return len(utf16.Encode([]rune(s)))
}

func utf16Slice(s string, n int) string {
	units := utf16.Encode([]rune(s))
	if n >= len(units) {
		return s
	}
	return string(utf16.Decode(units[:n]))
}

func splitPlanSections(plan string) []string {
	var out []string
	start := 0
	for i := 0; i+3 < len(plan); i++ {
		if plan[i] == '\n' && plan[i+1] == '#' && plan[i+2] == '#' && plan[i+3] == ' ' {
			out = append(out, plan[start:i])
			start = i + 1
		}
	}
	return append(out, plan[start:])
}

var mdHeadingLine = regexp.MustCompile(`^#{1,4}\s`)

func planSectionIsSubstantive(section string) bool {
	for _, line := range strings.Split(section, "\n") {
		t := strings.TrimSpace(line)
		if t != "" && !mdHeadingLine.MatchString(t) && t != "---" {
			return true
		}
	}
	return false
}

func gatherCoaching(p paths.DataPaths, now time.Time) (coachingContent, error) {
	var out coachingContent

	sinceKey := stats.LocalYMD(now.AddDate(0, 0, -recentNoteDays))
	allNotes, err := notes.ReadAll(p)
	if err != nil {
		return out, err
	}
	recent := notes.Apply(allNotes, notes.Filter{Since: sinceKey})
	if len(recent) > maxRecentNotes {
		recent = recent[len(recent)-maxRecentNotes:]
	}
	out.notes = recent

	dates, err := documents.ListNarratives(p)
	if err != nil {
		return out, err
	}
	if len(dates) > 0 {
		latest := dates[len(dates)-1]
		text, ok, err := documents.Read(p.NarrativeFile(latest))
		if err != nil {
			return out, err
		}
		if ok && strings.TrimSpace(text) != "" {
			out.narrative = &Narrative{Date: latest, Text: text}
		}
	}

	plan, ok, err := documents.Read(p.Plan)
	if err != nil {
		return out, err
	}
	if ok && strings.TrimSpace(plan) != "" {
		sections := splitPlanSections(plan)
		end := 1
		excerpt := strings.TrimSpace(sections[0])
		for !planSectionIsSubstantive(excerpt) && end < len(sections) {
			excerpt = excerpt + "\n\n" + strings.TrimSpace(sections[end])
			end++
		}
		if utf16Len(excerpt) > planExcerptMaxChars {
			excerpt = utf16Slice(excerpt, planExcerptMaxChars) + "…"
		} else if len(sections) > end {
			excerpt = excerpt + "\n\n_(full plan: `c2 plan show`)_"
		}
		out.planExcerpt = &excerpt
	}

	return out, nil
}

var (
	mdHeading = regexp.MustCompile(`^(#{1,4})\s+(.*)$`)
	mdItem    = regexp.MustCompile(`^[-*]\s+(.*)$`)
)

func mdLite(text string) string {
	var blocks []string
	var paragraph []string
	var list []string

	flushParagraph := func() {
		if len(paragraph) > 0 {
			blocks = append(blocks, "<p>"+strings.Join(paragraph, " ")+"</p>")
			paragraph = nil
		}
	}
	flushList := func() {
		if len(list) > 0 {
			var items strings.Builder
			for _, i := range list {
				items.WriteString("<li>" + i + "</li>")
			}
			blocks = append(blocks, "<ul>"+items.String()+"</ul>")
			list = nil
		}
	}

	for _, rawLine := range strings.Split(text, "\n") {
		line := esc(strings.TrimSpace(rawLine))
		if line == "" {
			flushParagraph()
			flushList()
			continue
		}
		if m := mdHeading.FindStringSubmatch(line); m != nil {
			flushParagraph()
			flushList()
			level := "h4"
			if len(m[1]) <= 2 {
				level = "h3"
			}
			blocks = append(blocks, "<"+level+">"+m[2]+"</"+level+">")
			continue
		}
		if m := mdItem.FindStringSubmatch(line); m != nil {
			flushParagraph()
			list = append(list, m[1])
			continue
		}
		flushList()
		paragraph = append(paragraph, line)
	}
	flushParagraph()
	flushList()
	return strings.Join(blocks, "\n")
}

func buildNarrativeSection(narrative Narrative) string {
	return `<div class="section">
  <h2>Coach's Report &mdash; ` + esc(narrative.Date) + `</h2>
  <div class="prose">
` + mdLite(narrative.Text) + `
  </div>
</div>`
}

func buildNotesSection(records []notes.Record) string {
	rows := make([]string, 0, len(records))
	for _, n := range records {
		workout := ""
		if n.WorkoutID != nil {
			workout = " &middot; workout " + strconv.FormatInt(*n.WorkoutID, 10)
		}
		rows = append(rows, `  <div class="note-row">
    <div class="note-meta">`+esc(n.Date[:10])+` &middot; `+esc(n.Type)+` (`+esc(n.Author)+`)`+workout+`</div>
    <div class="note-body">`+esc(n.Body)+`</div>
  </div>`)
	}
	return `<div class="section">
  <h2>Recent Notes</h2>
` + strings.Join(rows, "\n") + `
</div>`
}

func buildPlanSection(excerpt string) string {
	return `<div class="section">
  <h2>Training Plan</h2>
  <div class="prose">
` + mdLite(excerpt) + `
  </div>
</div>`
}
