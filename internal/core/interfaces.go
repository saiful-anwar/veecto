package core

import "context"

// Chunker splits text into Chunks.
type Chunker interface {
	Chunk(text string) ([]Chunk, error)
}

// Embedder converts text segments into float32 vector embeddings.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Provider() string
	Model() string
	Dimension() int
}

// Writer persists Documents to an output sink.
type Writer interface {
	Write(doc *Document) error
	Close() error
}
