package output

import (
	"encoding/json"
	"os"

	"github.com/saiful-anwar/veecto/internal/core"
)

// prettyWriter writes Documents as indented JSON.
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
