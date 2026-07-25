package doctor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/richhaase/c2/internal/notes"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/storage"
)

type Report struct {
	Issues       []string
	CheckedFiles int
}

var strokeFields = []string{"t", "d", "p", "spm", "hr"}

type checker struct {
	report *Report
}

func (c *checker) issue(format string, args ...any) {
	c.report.Issues = append(c.report.Issues, fmt.Sprintf(format, args...))
}

func (c *checker) readOrNil(path, label string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		if !storage.IsMissing(err) {
			c.issue("%s: unreadable (%v)", label, err)
		}
		return nil
	}
	return data
}

func (c *checker) listDir(dir, label string) []os.DirEntry {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !storage.IsMissing(err) {
			c.issue("%s: unreadable (%v)", label, err)
		}
		return nil
	}
	return entries
}

func isStrokeShaped(raw []byte) bool {
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return false
	}
	for _, key := range strokeFields {
		value, present := parsed[key]
		if !present || value == nil {
			continue
		}
		if _, ok := value.(float64); !ok {
			return false
		}
	}
	return true
}

func nonEmptyLines(data []byte) [][]byte {
	var out [][]byte
	for line := range bytes.SplitSeq(data, []byte("\n")) {
		out = append(out, line)
	}
	return out
}

func Run(p paths.DataPaths) Report {
	report := Report{Issues: []string{}}
	c := &checker{report: &report}

	c.checkMeta(p)
	c.checkWorkouts(p)
	c.checkStrokes(p)
	c.checkLooseNotes(p)
	c.checkArchives(p)

	return report
}

func (c *checker) checkMeta(p paths.DataPaths) {
	data := c.readOrNil(p.Meta, "meta.json")
	if data == nil {
		return
	}
	c.report.CheckedFiles++
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		c.issue("meta.json: not valid JSON")
		return
	}
	if _, ok := parsed["schema_version"].(float64); !ok {
		c.issue("meta.json: missing numeric schema_version")
	}
}

func (c *checker) checkWorkouts(p paths.DataPaths) {
	data := c.readOrNil(p.Workouts, "workouts.jsonl")
	if data == nil {
		return
	}
	c.report.CheckedFiles++
	for i, line := range nonEmptyLines(data) {
		lineNo := i + 1
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var parsed map[string]any
		if err := json.Unmarshal(line, &parsed); err != nil {
			c.issue("workouts.jsonl: line %d is not valid JSON", lineNo)
			continue
		}
		_, idOK := parsed["id"].(float64)
		_, dateOK := parsed["date"].(string)
		_, distanceOK := parsed["distance"].(float64)
		_, timeOK := parsed["time"].(float64)
		if !idOK || !dateOK || !distanceOK || !timeOK {
			c.issue("workouts.jsonl: line %d malformed workout record", lineNo)
		}
	}
}

func (c *checker) checkStrokes(p paths.DataPaths) {
	for _, entry := range c.listDir(p.StrokesDir, "strokes/") {
		name := entry.Name()
		if !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		label := "strokes/" + name
		data := c.readOrNil(filepath.Join(p.StrokesDir, name), label)
		if data == nil {
			continue
		}
		c.report.CheckedFiles++
		for i, line := range nonEmptyLines(data) {
			lineNo := i + 1
			if len(bytes.TrimSpace(line)) == 0 {
				continue
			}
			if !json.Valid(line) {
				c.issue("%s: line %d is not valid JSON", label, lineNo)
				continue
			}
			if !isStrokeShaped(line) {
				c.issue("%s: line %d malformed stroke record", label, lineNo)
			}
		}
	}
}

func (c *checker) checkLooseNotes(p paths.DataPaths) {
	type seenNote struct {
		content string
		file    string
	}
	looseContent := map[string]seenNote{}
	divergent := map[string]bool{}

	for _, entry := range c.listDir(p.NotesDir, "notes/") {
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		label := "notes/" + name
		data := c.readOrNil(filepath.Join(p.NotesDir, name), label)
		if data == nil {
			continue
		}
		c.report.CheckedFiles++
		if !json.Valid(data) {
			c.issue("%s: not valid JSON", label)
			continue
		}
		note, ok := notes.Parse(data)
		if !ok {
			c.issue("%s: malformed note record", label)
			continue
		}
		content := notes.Serialize(note)
		if prior, exists := looseContent[note.ID]; exists && prior.content != content && !divergent[note.ID] {
			divergent[note.ID] = true
			c.issue("notes: divergent copies of note %s (%s, %s); reconcile before they compact", note.ID, prior.file, name)
		}
		looseContent[note.ID] = seenNote{content: content, file: name}
	}
}

func (c *checker) checkArchives(p paths.DataPaths) {
	archiveIDs := map[string]bool{}

	for _, entry := range c.listDir(p.ArchiveDir, "notes/archive/") {
		name := entry.Name()
		if !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		label := "notes/archive/" + name
		data := c.readOrNil(filepath.Join(p.ArchiveDir, name), label)
		if data == nil {
			continue
		}
		c.report.CheckedFiles++

		hasPrev := false
		var prevMs int64
		prevID := ""
		for i, line := range nonEmptyLines(data) {
			lineNo := i + 1
			if len(bytes.TrimSpace(line)) == 0 {
				continue
			}
			if !json.Valid(line) {
				c.issue("%s: line %d is not valid JSON", label, lineNo)
				continue
			}
			note, ok := notes.Parse(line)
			if !ok {
				c.issue("%s: line %d malformed note record", label, lineNo)
				continue
			}
			t, _ := notes.ParseDate(note.Date)
			ms := t.UnixMilli()
			if hasPrev && (ms < prevMs || (ms == prevMs && note.ID < prevID)) {
				c.issue("%s: line %d out of (date, id) order", label, lineNo)
			}
			hasPrev = true
			prevMs = ms
			prevID = note.ID
			if archiveIDs[note.ID] {
				c.issue("%s: duplicate note id %s", label, note.ID)
			}
			archiveIDs[note.ID] = true
		}
	}
}
