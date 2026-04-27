package storage

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"github.com/richhaase/c2/internal/config"
	"github.com/richhaase/c2/internal/model"
)

func HasStrokeData(workoutID int) (bool, error) {
	info, err := os.Stat(strokeDataPath(workoutID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return !info.IsDir(), nil
}

func WriteStrokeData(workoutID int, strokes []model.StrokeData) error {
	return writeStrokeDataPath(strokeDataPath(workoutID), strokes)
}

func writeStrokeDataPath(path string, strokes []model.StrokeData) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(file)
	for _, stroke := range strokes {
		data, err := json.Marshal(stroke)
		if err != nil {
			_ = file.Close()
			return err
		}
		if _, err := writer.Write(data); err != nil {
			_ = file.Close()
			return err
		}
		if err := writer.WriteByte('\n'); err != nil {
			_ = file.Close()
			return err
		}
	}
	if err := writer.Flush(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func ReadStrokeData(workoutID int) ([]model.StrokeData, error) {
	return readStrokeDataPath(strokeDataPath(workoutID))
}

func readStrokeDataPath(path string) ([]model.StrokeData, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []model.StrokeData{}, nil
		}
		return nil, err
	}
	var strokes []model.StrokeData
	reader := bufio.NewReader(file)
	lineNumber := 0
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			lineNumber++
			stroke, err := parseStrokeLine(path, lineNumber, line)
			if err != nil {
				_ = file.Close()
				return nil, err
			}
			if stroke != nil {
				strokes = append(strokes, *stroke)
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			_ = file.Close()
			return nil, err
		}
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	return strokes, nil
}

func parseStrokeLine(path string, lineNumber int, line []byte) (*model.StrokeData, error) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return nil, nil
	}
	var stroke model.StrokeData
	if err := json.Unmarshal(line, &stroke); err != nil {
		return nil, fmt.Errorf("%s line %d: %w", path, lineNumber, err)
	}
	return &stroke, nil
}

func strokeDataPath(workoutID int) string {
	return filepath.Join(config.StrokesDir(), strconv.Itoa(workoutID)+".jsonl")
}
