package output

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/saiful-anwar/veecto/internal/core"
)

// New creates a core.Writer backed by a file at path. When pretty is true
// the output is indented JSON; otherwise it is JSONL (one Document per line).
func New(path string, pretty bool) (core.Writer, error) {
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
	if pretty {
		return &prettyWriter{file: f}, nil
	}
	return &jsonlWriter{file: f}, nil
}
