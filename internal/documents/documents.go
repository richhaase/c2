package documents

import (
	"errors"
	"io/fs"
	"os"
	"sort"
	"strings"

	"github.com/richhaase/c2/internal/models"
	"github.com/richhaase/c2/internal/paths"
)

func Read(path string) (string, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	return string(data), true, nil
}

func ListNarratives(p paths.DataPaths) ([]string, error) {
	entries, err := os.ReadDir(p.ReportsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var dates []string
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		date := strings.TrimSuffix(name, ".md")
		if models.IsValidYMD(date) {
			dates = append(dates, date)
		}
	}
	sort.Strings(dates)
	return dates, nil
}
