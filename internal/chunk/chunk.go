package chunk

import (
	"github.com/saiful-anwar/veecto/internal/core"
)

// New creates a core.Chunker based on the strategy in cfg.
func New(cfg core.Config) core.Chunker {
	switch cfg.Chunking.Strategy {
	case "fixed":
		return &Fixed{Size: cfg.Chunking.Size, Overlap: cfg.Chunking.Overlap}
	case "sentence":
		return &Sentence{Size: cfg.Chunking.Size, Overlap: cfg.Chunking.Overlap}
	case "markdown":
		return &Markdown{Size: cfg.Chunking.Size, Overlap: cfg.Chunking.Overlap}
	default:
		return &Recursive{Size: cfg.Chunking.Size, Overlap: cfg.Chunking.Overlap}
	}
}
