package output

import (
	"fmt"
	"strings"

	"github.com/saiful-anwar/veecto/internal/core"
)

// multi fans out Write/Close to multiple underlying writers.
type multi struct {
	writers []core.Writer
}

// Multi creates a core.Writer that delegates to all provided writers.
func Multi(writers ...core.Writer) core.Writer {
	return &multi{writers: writers}
}

func (w *multi) Write(doc *core.Document) error {
	for _, wr := range w.writers {
		if err := wr.Write(doc); err != nil {
			return err
		}
	}
	return nil
}

func (w *multi) Close() error {
	var errs []string
	for _, wr := range w.writers {
		if err := wr.Close(); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close errors: %s", strings.Join(errs, "; "))
	}
	return nil
}
