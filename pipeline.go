package veecto

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/saiful-anwar/veecto/internal/core"
)

// Version is the build version of the library. Set at build time via ldflags.
var Version = "dev"

// Process ingests a single input (file path, URL, or "-" for stdin) through the full
// pipeline: resolve input → read → chunk → embed → build Document. When cfg is omitted,
// DefaultConfig is used.
func Process(ctx context.Context, input string, cfg ...Config) (*Document, error) {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}

	localPath, cleanup, err := ResolveInput(input)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	var fr FileResult
	if localPath == "-" {
		fr, err = ReadStdin()
	} else {
		fr, err = ReadFile(localPath)
	}
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	if int64(len([]rune(fr.Text))) > c.Pipeline.MaxFileSize {
		return nil, fmt.Errorf("file too large after conversion: %d bytes (max: %d)",
			len([]rune(fr.Text)), c.Pipeline.MaxFileSize)
	}

	chunker := NewChunker(c)
	chunks, err := chunker.Chunk(fr.Text)
	if err != nil {
		return nil, fmt.Errorf("chunk: %w", err)
	}

	embedder, err := NewEmbedder(c)
	if err != nil {
		return nil, fmt.Errorf("embedder: %w", err)
	}

	texts := extractTexts(chunks)
	vectors, err := embedder.Embed(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}

	if len(vectors) != len(chunks) {
		return nil, fmt.Errorf("mismatch: %d chunks but %d vectors from %s %s",
			len(chunks), len(vectors), embedder.Provider(), embedder.Model())
	}

	for i := range chunks {
		chunks[i].Vector = vectors[i]
	}

	doc := buildDocument(input, fr, chunks, embedder)
	return doc, nil
}

// ProcessFile is a convenience wrapper around Process for file paths.
func ProcessFile(ctx context.Context, path string, cfg ...Config) (*Document, error) {
	return Process(ctx, path, cfg...)
}

// ProcessURL is a convenience wrapper around Process for HTTP/HTTPS URLs.
func ProcessURL(ctx context.Context, rawURL string, cfg ...Config) (*Document, error) {
	return Process(ctx, rawURL, cfg...)
}

// ProcessAll processes multiple inputs concurrently, respecting the concurrency
// limit in Config.Pipeline.Concurrency. The first error causes an immediate return.
func ProcessAll(ctx context.Context, inputs []string, cfg ...Config) ([]*Document, error) {
	c := DefaultConfig()
	if len(cfg) > 0 {
		c = cfg[0]
	}

	if len(inputs) == 0 {
		return nil, nil
	}

	docs := make([]*Document, len(inputs))
	type idxErr struct {
		idx int
		err error
	}
	errCh := make(chan idxErr, len(inputs))

	var wg sync.WaitGroup
	sem := make(chan struct{}, c.Pipeline.Concurrency)

	for i, input := range inputs {
		wg.Add(1)
		go func(i int, input string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			doc, err := Process(ctx, input, c)
			if err != nil {
				errCh <- idxErr{i, fmt.Errorf("%s: %w", input, err)}
				return
			}
			docs[i] = doc
			errCh <- idxErr{i, nil}
		}(i, input)
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	for ie := range errCh {
		if ie.err != nil {
			return nil, ie.err
		}
	}

	return docs, nil
}

// Pipeline is a reusable processing pipeline with configurable concurrency and progress reporting.
type Pipeline struct {
	config     core.Config
	onProgress core.ProgressFunc
}

// NewPipeline creates a Pipeline with the given config. Progress reporting is silent by default;
// call OnProgress to enable it.
func NewPipeline(cfg Config) *Pipeline {
	return &Pipeline{config: cfg, onProgress: core.SilentProgressFn()}
}

// OnProgress sets the progress callback. Pass nil to silence progress.
func (p *Pipeline) OnProgress(fn ProgressFunc) {
	if fn != nil {
		p.onProgress = fn
	} else {
		p.onProgress = core.SilentProgressFn()
	}
}

// Run processes all inputs sequentially through the pipeline.
func (p *Pipeline) Run(ctx context.Context, inputs ...string) ([]*Document, error) {
	return ProcessAll(ctx, inputs, p.config)
}

// ProcessOne processes a single input and reports progress via the callback.
func (p *Pipeline) ProcessOne(ctx context.Context, input string) (*Document, error) {
	start := time.Now()
	doc, err := Process(ctx, input, p.config)
	p.onProgress(Progress{
		Input:    input,
		Duration: time.Since(start),
		Error:    err,
	})
	return doc, err
}

// ProcessOneWithResult processes a single input and returns both the Document and Progress info,
// allowing the caller to inspect results even on partial success.
func (p *Pipeline) ProcessOneWithResult(ctx context.Context, input string) (*Document, Progress) {
	start := time.Now()
	doc, err := Process(ctx, input, p.config)
	pr := Progress{
		Input:    input,
		Duration: time.Since(start),
		Error:    err,
	}
	if doc != nil {
		pr.FileType = doc.Metadata.FileType
		pr.ChunkCount = doc.TotalChunk
	}
	return doc, pr
}

// Config returns a copy of the pipeline's configuration.
func (p *Pipeline) Config() core.Config {
	return p.config
}

// extractTexts collects the best available text (TextClean > Text) for embedding.
func extractTexts(chunks []Chunk) []string {
	texts := make([]string, len(chunks))
	for i := range chunks {
		if chunks[i].TextClean != "" {
			texts[i] = chunks[i].TextClean
		} else {
			texts[i] = chunks[i].Text
		}
	}
	return texts
}

// buildDocument assembles a Document from processed data.
func buildDocument(input string, fr FileResult, chunks []Chunk, embedder Embedder) *Document {
	now := time.Now().UTC()

	source := input
	if !strings.HasPrefix(source, "http://") && !strings.HasPrefix(source, "https://") {
		source = filepath.Base(input)
	}

	hashInput := input + fr.Hash + now.String()
	hash := sha256.Sum256([]byte(hashInput))
	docID := fmt.Sprintf("%s_%x",
		strings.TrimSuffix(source, filepath.Ext(source)),
		hash[:8],
	)

	dim := embedder.Dimension()
	if dim == 0 && len(chunks) > 0 && len(chunks[0].Vector) > 0 {
		dim = len(chunks[0].Vector)
	}

	return &Document{
		DocID: docID,
		Metadata: Metadata{
			Source:    source,
			FileType:  fr.FileType,
			FileSize:  fr.Size,
			FileHash:  fr.Hash,
			CreatedAt: now,
		},
		Embedding: EmbeddingMetadata{
			Provider:  embedder.Provider(),
			Model:     embedder.Model(),
			Dimension: dim,
			Version:   "v1",
		},
		StartAt:    now,
		FinishedAt: now,
		TotalChunk: len(chunks),
		Chunks:     chunks,
	}
}
