package notes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/richhaase/c2/internal/atomicfile"
	"github.com/richhaase/c2/internal/paths"
)

type storedNote struct {
	record Record
	raw    []byte
}

type editableNote struct {
	record  Record
	path    string
	data    []byte
	archive bool
	records []storedNote
}

func locateForEdit(p paths.DataPaths, id string) (editableNote, error) {
	var found *editableNote
	for _, dir := range []struct {
		path, suffix string
		archive      bool
	}{{p.NotesDir, ".json", false}, {p.ArchiveDir, ".jsonl", true}} {
		entries, err := os.ReadDir(dir.path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return editableNote{}, err
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), dir.suffix) {
				continue
			}
			path := filepath.Join(dir.path, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				return editableNote{}, err
			}
			lines := [][]byte{data}
			if dir.archive {
				lines = bytes.Split(data, []byte("\n"))
			}
			var records []storedNote
			var match *Record
			for _, line := range lines {
				if dir.archive && len(bytes.TrimSpace(line)) == 0 {
					continue
				}
				n, ok := Parse(line)
				if !ok {
					return editableNote{}, fmt.Errorf("Cannot edit notes: malformed record in %s. Run `c2 data doctor`.", path)
				}
				records = append(records, storedNote{record: n, raw: line})
				if n.ID != id {
					continue
				}
				if found != nil || match != nil {
					return editableNote{}, fmt.Errorf("Multiple copies of note %s; reconcile them before editing.", id)
				}
				decoder := json.NewDecoder(bytes.NewReader(line))
				decoder.DisallowUnknownFields()
				if err := decoder.Decode(&n); err != nil {
					return editableNote{}, fmt.Errorf("Note %s has unsupported fields; update C2 before editing: %w", id, err)
				}
				match = &n
			}
			if match != nil {
				found = &editableNote{record: *match, path: path, data: data, archive: dir.archive, records: records}
			}
		}
	}
	if found == nil {
		return editableNote{}, fmt.Errorf("No note with id %s. Use `c2 note list` to find its ID.", id)
	}
	return *found, nil
}

func ReadForEdit(p paths.DataPaths, id string) (Record, error) {
	found, err := locateForEdit(p, id)
	return found.record, err
}

func Update(p paths.DataPaths, before, after Record) (bool, error) {
	if before.ID != after.ID || !IsShaped(after) || strings.TrimSpace(after.Body) == "" {
		return false, fmt.Errorf("Invalid correction: preserve the note ID and provide valid fields and nonempty text.")
	}
	found, err := locateForEdit(p, before.ID)
	if err != nil {
		return false, err
	}
	expected, err := Serialize(before)
	if err != nil {
		return false, err
	}
	current, err := Serialize(found.record)
	if err != nil {
		return false, err
	}
	if current != expected {
		return false, fmt.Errorf("Note %s changed while being edited; read it again before retrying.", before.ID)
	}
	replacement, err := Serialize(after)
	if err != nil {
		return false, err
	}
	if replacement == current {
		return false, nil
	}
	data := []byte(replacement + "\n")
	if found.archive {
		for i, entry := range found.records {
			if entry.record.ID == after.ID {
				found.records[i] = storedNote{record: after, raw: []byte(replacement)}
			}
		}
		slices.SortFunc(found.records, func(a, b storedNote) int { return Compare(a.record, b.record) })
		var buf bytes.Buffer
		for _, entry := range found.records {
			buf.Write(entry.raw)
			buf.WriteByte('\n')
		}
		data = buf.Bytes()
	}
	latest, err := os.ReadFile(found.path)
	if err != nil {
		return false, err
	}
	if !bytes.Equal(latest, found.data) {
		return false, fmt.Errorf("Note storage changed while saving; read note %s again before retrying.", before.ID)
	}
	if err := atomicfile.Write(found.path, data, 0o644); err != nil {
		return false, err
	}
	return true, nil
}
