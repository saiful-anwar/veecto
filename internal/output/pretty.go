package output

import (
	"encoding/json"
	"os"

	"github.com/saiful-anwar/veecto/internal/core"
)

// prettyWriter writes Documents as indented JSONL: one pretty-printed JSON
// object per line (separated by newlines). Not a valid JSON array.
type prettyWriter struct {
	file *os.File
}

func (w *prettyWriter) Write(doc *core.Document) error {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if _, err := w.file.Write(data); err != nil {
		return err
	}
	_, err = w.file.Write([]byte("\n"))
	return err
}

func (w *prettyWriter) Close() error {
	return w.file.Close()
}
