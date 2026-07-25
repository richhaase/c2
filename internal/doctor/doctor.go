package doctor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/notes"
	"github.com/richhaase/c2/internal/paths"
	"github.com/richhaase/c2/internal/storage"
)

type Report struct {
	Issues       []string
	CheckedFiles int
}

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

type strokeShape struct {
	T   json.RawMessage `json:"t"`
	D   json.RawMessage `json:"d"`
	P   json.RawMessage `json:"p"`
	SPM json.RawMessage `json:"spm"`
	HR  json.RawMessage `json:"hr"`
}

func isJSONObject(raw []byte) bool {
	trimmed := bytes.TrimSpace(raw)
	return len(trimmed) >= 2 && trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}'
}

func isStrokeShaped(raw []byte) bool {
	if !isJSONObject(raw) {
		return false
	}
	var parsed strokeShape
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return false
	}
	fields := []json.RawMessage{parsed.T, parsed.D, parsed.P, parsed.SPM, parsed.HR}
	for _, value := range fields {
		if len(value) == 0 {
			continue
		}
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return false
		}
		var number float64
		if err := json.Unmarshal(value, &number); err != nil {
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
	if !isJSONObject(data) {
		c.issue("meta.json: not valid JSON")
		return
	}
	var parsed struct {
		SchemaVersion json.RawMessage `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		c.issue("meta.json: not valid JSON")
		return
	}
	var version int
	if len(parsed.SchemaVersion) == 0 || json.Unmarshal(parsed.SchemaVersion, &version) != nil {
		c.issue("meta.json: missing numeric schema_version")
		return
	}
	if version != storage.SchemaVersion {
		c.issue("meta.json: unsupported schema_version %d (expected %d)", version, storage.SchemaVersion)
	}
}

func (c *checker) checkWorkouts(p paths.DataPaths) {
	data := c.readOrNil(p.Workouts, "workouts.jsonl")
	if data == nil {
		return
	}
	c.report.CheckedFiles++
	seen := map[int64]bool{}
	for i, line := range nonEmptyLines(data) {
		lineNo := i + 1
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if !json.Valid(line) {
			c.issue("workouts.jsonl: line %d is not valid JSON", lineNo)
			continue
		}
		var parsed struct {
			ID       *int64  `json:"id"`
			Date     *string `json:"date"`
			Distance *int    `json:"distance"`
			Time     *int    `json:"time"`
		}
		if !isJSONObject(line) || json.Unmarshal(line, &parsed) != nil ||
			parsed.ID == nil || parsed.Date == nil || parsed.Distance == nil || parsed.Time == nil ||
			models.ParseLocal(*parsed.Date).IsZero() {
			c.issue("workouts.jsonl: line %d malformed workout record", lineNo)
			continue
		}
		if seen[*parsed.ID] {
			c.issue("workouts.jsonl: line %d duplicate workout id %d", lineNo, *parsed.ID)
		}
		seen[*parsed.ID] = true
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
		content, err := notes.Serialize(note)
		if err != nil {
			c.issue("%s: malformed note record", label)
			continue
		}
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
