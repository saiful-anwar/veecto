package veecto

import (
	"github.com/saiful-anwar/veecto/internal/chunk"
	"github.com/saiful-anwar/veecto/internal/core"
	"github.com/saiful-anwar/veecto/internal/embed"
	"github.com/saiful-anwar/veecto/internal/output"
)

// Public type aliases — all re-exported from internal/core so external consumers
// import them as veecto.Document, veecto.Config, etc.
type (
	Document          = core.Document
	Chunk             = core.Chunk
	Metadata          = core.Metadata
	EmbeddingMetadata = core.EmbeddingMetadata
	Progress          = core.Progress
	ProgressFunc      = core.ProgressFunc
	Config            = core.Config
	Chunker           = core.Chunker
	Embedder          = core.Embedder
	Writer            = core.Writer
)

// NewChunker creates a Chunker based on the strategy in cfg (recursive, fixed, sentence, markdown).
func NewChunker(cfg Config) Chunker { return chunk.New(cfg) }

// NewEmbedder creates an Embedder based on the provider in cfg (openai, ollama, gemini, http)
// and wraps it with retry middleware when cfg.Embedding.Retries > 0.
func NewEmbedder(cfg Config) (Embedder, error) { return embed.New(cfg) }

// NewWriter creates a JSONL or pretty-JSON Writer backed by a file at path.
// Deprecated: use NewWriterFormat.
func NewWriter(path string, pretty bool) (Writer, error) { return output.New(path, pretty) }

// NewWriterFormat creates a Writer using an explicit format name
// ("jsonl", "pretty", or "json-array").
func NewWriterFormat(path string, format string) (Writer, error) { return output.NewFormat(path, format) }

// MultiWriter fans out Write/Close to multiple underlying writers.
func MultiWriter(writers ...Writer) Writer { return output.Multi(writers...) }
