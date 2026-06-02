package output

import (
	"encoding/json"
	"os"

	"github.com/saiful-anwar/veecto/internal/core"
)

// jsonlWriter writes Documents as newline-delimited JSON.
type jsonlWriter struct {
	file *os.File
	enc  *json.Encoder
}

func (w *jsonlWriter) Write(doc *core.Document) error {
	if w.enc == nil {
		w.enc = json.NewEncoder(w.file)
	}
	return w.enc.Encode(doc)
}

func (w *jsonlWriter) Close() error {
	return w.file.Close()
}
