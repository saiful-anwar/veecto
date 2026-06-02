package chunk

import (
	"github.com/saiful-anwar/veecto/internal/core"
)

// New creates a core.Chunker based on the strategy in cfg.
func New(cfg core.Config) core.Chunker {
	ascii := cfg.Chunking.AsciiNormalize
	switch cfg.Chunking.Strategy {
	case "fixed":
		return &Fixed{Size: cfg.Chunking.Size, Overlap: cfg.Chunking.Overlap, AsciiNormalize: ascii}
	case "sentence":
		return &Sentence{Size: cfg.Chunking.Size, Overlap: cfg.Chunking.Overlap, AsciiNormalize: ascii}
	case "markdown":
		return &Markdown{Size: cfg.Chunking.Size, Overlap: cfg.Chunking.Overlap, AsciiNormalize: ascii}
	default:
		return &Recursive{Size: cfg.Chunking.Size, Overlap: cfg.Chunking.Overlap, AsciiNormalize: ascii}
	}
}
