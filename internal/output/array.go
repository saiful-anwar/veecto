package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/saiful-anwar/veecto/internal/core"
)

// Output format constants supported by NewFormat.
const (
	FormatJSONL     = "jsonl"
	FormatPretty    = "pretty"
	FormatJSONArray = "json-array"
)

// arrayWriter writes Documents as a single valid JSON array.
type arrayWriter struct {
	file  *os.File
	first bool
}

func (w *arrayWriter) Write(doc *core.Document) error {
	if w.first {
		if _, err := w.file.Write([]byte("[\n")); err != nil {
			return err
		}
		w.first = false
	} else {
		if _, err := w.file.Write([]byte(",\n")); err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(doc, "  ", "  ")
	if err != nil {
		return err
	}
	if _, err := w.file.Write(append([]byte("  "), data...)); err != nil {
		return err
	}
	return nil
}

func (w *arrayWriter) Close() error {
	if !w.first {
		if _, err := w.file.Write([]byte("\n]\n")); err != nil {
			_ = w.file.Close()
			return err
		}
	}
	return w.file.Close()
}

// NewArray creates a Writer that produces a single valid JSON array.
func NewArray(path string) (core.Writer, error) {
	if path == "" {
		return nil, fmt.Errorf("output path required")
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create output dir: %w", err)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create output: %w", err)
	}
	return &arrayWriter{file: f, first: true}, nil
}
