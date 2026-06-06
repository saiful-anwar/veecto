package output

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/saiful-anwar/veecto/internal/core"
)

// New creates a core.Writer backed by a file at path. When pretty is true
// the output is indented JSONL (one Document per line); otherwise plain JSONL.
// Deprecated: use NewFormat with an explicit format constant.
func New(path string, pretty bool) (core.Writer, error) {
	format := FormatJSONL
	if pretty {
		format = FormatPretty
	}
	return NewFormat(path, format)
}

// NewFormat creates a core.Writer backed by a file at path using the given
// format (FormatJSONL, FormatPretty, or FormatJSONArray).
func NewFormat(path string, format string) (core.Writer, error) {
	if path == "" {
		return nil, fmt.Errorf("output path required")
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return nil, fmt.Errorf("create output dir: %w", err)
		}
	}
	// #nosec G304 -- path is a user-provided output path from CLI flags/config.
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create output: %w", err)
	}
	switch format {
	case FormatJSONArray:
		return &arrayWriter{file: f, first: true}, nil
	case FormatPretty:
		return &prettyWriter{file: f}, nil
	default:
		return &jsonlWriter{file: f}, nil
	}
}
