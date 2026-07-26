package notes

import (
	"bytes"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/richhaase/c2/internal/atomicfile"
	"github.com/richhaase/c2/internal/jsonx"
	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/paths"
)

var Types = []string{"subjective", "observation", "lesson"}

var Authors = []string{"athlete", "coach"}

const compactAgeDays = 7

const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

var crockfordEncoding = base32.NewEncoding(crockford).WithPadding(base32.NoPadding)

type Record struct {
	ID        string   `json:"id"`
	Date      string   `json:"date"`
	Type      string   `json:"type"`
	WorkoutID *int64   `json:"workout_id,omitempty"`
	Tags      []string `json:"tags,omitempty"`
	Body      string   `json:"body"`
	Author    string   `json:"author"`
}

func ULID(now time.Time) (string, error) {
	ms := now.UnixMilli()
	if ms < 0 || ms >= 1<<48 {
		return "", fmt.Errorf("note timestamp is outside the ULID range")
	}
	timePart := make([]byte, 10)
	for i := 9; i >= 0; i-- {
		timePart[i] = crockford[ms%32]
		ms /= 32
	}
	entropy := make([]byte, 10)
	if _, err := rand.Read(entropy); err != nil {
		return "", err
	}
	return string(timePart) + crockfordEncoding.EncodeToString(entropy), nil
}

func LocalISO(t time.Time) string {
	return t.Format("2006-01-02T15:04:05-07:00")
}

func Serialize(n Record) (string, error) {
	out, err := jsonx.Compact(n)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

var dateShapePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T.*(?:Z|[+-]\d{2}:\d{2})$`)

var noteTimeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04Z07:00",
}

func ParseDate(s string) (time.Time, bool) {
	for _, layout := range noteTimeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func IsShaped(n Record) bool {
	if n.ID == "" || n.Date == "" || n.Body == "" {
		return false
	}
	if !slices.Contains(Types, n.Type) || !slices.Contains(Authors, n.Author) {
		return false
	}
	if !dateShapePattern.MatchString(n.Date) {
		return false
	}
	if len(n.Date) < 10 || !models.IsValidYMD(n.Date[:10]) {
		return false
	}
	if _, ok := ParseDate(n.Date); !ok {
		return false
	}
	return true
}

func Parse(raw []byte) (Record, bool) {
	var n Record
	if err := json.Unmarshal(raw, &n); err != nil {
		return Record{}, false
	}
	if !IsShaped(n) {
		return Record{}, false
	}
	return n, true
}

func Compare(a, b Record) int {
	at, _ := ParseDate(a.Date)
	bt, _ := ParseDate(b.Date)
	if !at.Equal(bt) {
		if at.Before(bt) {
			return -1
		}
		return 1
	}
	return strings.Compare(a.ID, b.ID)
}

func Write(p paths.DataPaths, n Record) error {
	if err := os.MkdirAll(p.NotesDir, 0o755); err != nil {
		return err
	}
	serialized, err := Serialize(n)
	if err != nil {
		return err
	}
	return atomicfile.Write(filepath.Join(p.NotesDir, n.ID+".json"), []byte(serialized+"\n"), 0o644)
}

type looseEntry struct {
	note Record
	file string
}

func readLooseEntries(p paths.DataPaths) ([]looseEntry, error) {
	files, err := os.ReadDir(p.NotesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	names := make([]string, 0, len(files))
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".json") {
			names = append(names, f.Name())
		}
	}
	sort.Strings(names)

	var entries []looseEntry
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(p.NotesDir, name))
		if err != nil {
			return nil, err
		}
		if n, ok := Parse(data); ok {
			entries = append(entries, looseEntry{note: n, file: name})
		}
	}
	return entries, nil
}

func readArchived(p paths.DataPaths) (map[string]Record, error) {
	out := map[string]Record{}
	files, err := os.ReadDir(p.ArchiveDir)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".jsonl") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(p.ArchiveDir, f.Name()))
		if err != nil {
			return nil, err
		}
		for line := range bytes.SplitSeq(data, []byte("\n")) {
			if len(bytes.TrimSpace(line)) == 0 {
				continue
			}
			if n, ok := Parse(line); ok {
				if _, exists := out[n.ID]; !exists {
					out[n.ID] = n
				}
			}
		}
	}
	return out, nil
}

func ReadAll(p paths.DataPaths) ([]Record, error) {
	merged, err := readArchived(p)
	if err != nil {
		return nil, err
	}
	loose, err := readLooseEntries(p)
	if err != nil {
		return nil, err
	}
	for _, e := range loose {
		merged[e.note.ID] = e.note
	}
	out := make([]Record, 0, len(merged))
	for _, n := range merged {
		out = append(out, n)
	}
	slices.SortFunc(out, Compare)
	return out, nil
}

type Filter struct {
	Type      string
	Since     string
	WorkoutID *int64
}

func Apply(records []Record, f Filter) []Record {
	out := make([]Record, 0, len(records))
	for _, n := range records {
		if f.Type != "" && n.Type != f.Type {
			continue
		}
		if f.Since != "" && n.Date[:10] < f.Since {
			continue
		}
		if f.WorkoutID != nil && (n.WorkoutID == nil || *n.WorkoutID != *f.WorkoutID) {
			continue
		}
		out = append(out, n)
	}
	return out
}

type archiveYear struct {
	notes         []Record
	safeToRewrite bool
}

func readArchiveYear(p paths.DataPaths, year int) archiveYear {
	data, err := os.ReadFile(p.ArchiveFile(year))
	if err != nil {
		return archiveYear{safeToRewrite: os.IsNotExist(err)}
	}
	seen := map[string]bool{}
	var out []Record
	for line := range bytes.SplitSeq(data, []byte("\n")) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		n, ok := Parse(line)
		if !ok {
			return archiveYear{safeToRewrite: false}
		}
		if seen[n.ID] {
			return archiveYear{safeToRewrite: false}
		}
		seen[n.ID] = true
		out = append(out, n)
	}
	return archiveYear{notes: out, safeToRewrite: true}
}

type CompactResult struct {
	Archived     int
	Years        []int
	SkippedYears []int
}

func Compact(p paths.DataPaths, now time.Time) (CompactResult, error) {
	cutoff := now.AddDate(0, 0, -compactAgeDays)

	entries, err := readLooseEntries(p)
	if err != nil {
		return CompactResult{}, err
	}
	deduped := map[string]Record{}
	contentByID := map[string]string{}
	divergent := map[string]bool{}
	for _, e := range entries {
		content, err := Serialize(e.note)
		if err != nil {
			return CompactResult{}, err
		}
		if prior, ok := contentByID[e.note.ID]; ok && prior != content {
			divergent[e.note.ID] = true
		}
		contentByID[e.note.ID] = content
		deduped[e.note.ID] = e.note
	}

	var eligible []Record
	for _, n := range deduped {
		if divergent[n.ID] {
			continue
		}
		t, ok := ParseDate(n.Date)
		if !ok || !t.Before(cutoff) {
			continue
		}
		eligible = append(eligible, n)
	}
	if len(eligible) == 0 {
		return CompactResult{}, nil
	}

	byYear := map[int][]Record{}
	for _, n := range eligible {
		year, err := strconv.Atoi(n.Date[:4])
		if err != nil {
			continue
		}
		byYear[year] = append(byYear[year], n)
	}

	if err := os.MkdirAll(p.ArchiveDir, 0o755); err != nil {
		return CompactResult{}, err
	}

	result := CompactResult{}
	years := make([]int, 0, len(byYear))
	for year := range byYear {
		years = append(years, year)
	}
	sort.Ints(years)

	for _, year := range years {
		batch := byYear[year]
		existing := readArchiveYear(p, year)
		if !existing.safeToRewrite {
			result.SkippedYears = append(result.SkippedYears, year)
			continue
		}
		merged := map[string]Record{}
		for _, n := range existing.notes {
			merged[n.ID] = n
		}
		for _, n := range batch {
			merged[n.ID] = n
		}
		combined := make([]Record, 0, len(merged))
		for _, n := range merged {
			combined = append(combined, n)
		}
		slices.SortFunc(combined, Compare)

		var buf bytes.Buffer
		for _, n := range combined {
			serialized, err := Serialize(n)
			if err != nil {
				return CompactResult{}, err
			}
			buf.WriteString(serialized)
			buf.WriteByte('\n')
		}
		if err := atomicfile.Write(p.ArchiveFile(year), buf.Bytes(), 0o644); err != nil {
			result.SkippedYears = append(result.SkippedYears, year)
			continue
		}

		archivedIDs := map[string]bool{}
		for _, n := range batch {
			archivedIDs[n.ID] = true
		}
		for _, e := range entries {
			if archivedIDs[e.note.ID] {
				os.Remove(filepath.Join(p.NotesDir, e.file))
			}
		}
		result.Archived += len(batch)
		result.Years = append(result.Years, year)
	}

	sort.Ints(result.Years)
	sort.Ints(result.SkippedYears)
	return result, nil
}
