package veecto

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/saiful-anwar/veecto/internal/core"
)

// Version is the build version of the library. Set at build time via ldflags.
var Version = "dev"

// Process ingests a single input (file path, URL, or "-" for stdin) through the full
// pipeline: resolve input → read → chunk → embed → build Document.
func Process(ctx context.Context, input string, cfg Config) (*Document, error) {
	startAt := time.Now().UTC()

	var text string
	var fr FileResult
	localPath, cleanup, err := ResolveInput(ctx, input)
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}

	if localPath == "-" {
		fr, err = ReadStdin()
	} else {
		// Check raw file size before reading.
		info, errStat := os.Stat(localPath)
		if errStat != nil {
			return nil, fmt.Errorf("stat: %w", errStat)
		}
		if info.Size() > cfg.Pipeline.MaxFileSize {
			return nil, fmt.Errorf("file too large: %d bytes (max: %d)",
				info.Size(), cfg.Pipeline.MaxFileSize)
		}

		fr, err = ReadFile(localPath)
	}
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	text = fr.Text

	if cfg.Pipeline.MaxTextSize > 0 && int64(len([]rune(text))) > cfg.Pipeline.MaxTextSize {
		return nil, fmt.Errorf("text too large after conversion: %d runes (max: %d)",
			len([]rune(text)), cfg.Pipeline.MaxTextSize)
	}

	chunker := NewChunker(cfg)
	chunks, err := chunker.Chunk(text)
	if err != nil {
		return nil, fmt.Errorf("chunk: %w", err)
	}

	embedder, err := NewEmbedder(cfg)
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

	doc := buildDocument(input, fr, chunks, embedder, startAt)
	return doc, nil
}

// ProcessDefault is a convenience wrapper that uses DefaultConfig.
func ProcessDefault(ctx context.Context, input string) (*Document, error) {
	return Process(ctx, input, DefaultConfig())
}

// ProcessFile is a convenience wrapper around Process for file paths.
func ProcessFile(ctx context.Context, path string, cfg Config) (*Document, error) {
	return Process(ctx, path, cfg)
}

// ProcessURL is a convenience wrapper around Process for HTTP/HTTPS URLs.
func ProcessURL(ctx context.Context, rawURL string, cfg Config) (*Document, error) {
	return Process(ctx, rawURL, cfg)
}

// ProcessAll processes multiple inputs concurrently, respecting the concurrency
// limit in Config.Pipeline.Concurrency. On the first error all pending work is
// cancelled and the error is returned.
func ProcessAll(ctx context.Context, inputs []string, cfg Config) ([]*Document, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	docs := make([]*Document, len(inputs))
	type idxErr struct {
		idx int
		err error
	}
	errCh := make(chan idxErr, len(inputs))

	var wg sync.WaitGroup
	sem := make(chan struct{}, cfg.Pipeline.Concurrency)

	for i, input := range inputs {
		wg.Add(1)
		go func(i int, input string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				errCh <- idxErr{i, ctx.Err()}
				return
			}
			defer func() { <-sem }()

			doc, err := Process(ctx, input, cfg)
			if err != nil {
				errCh <- idxErr{i, fmt.Errorf("%s: %w", input, err)}
				cancel()
				return
			}
			docs[i] = doc
			errCh <- idxErr{i, nil}
		}(i, input)
	}

	wg.Wait()
	close(errCh)

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

// Run processes all inputs through the pipeline, reporting progress via the
// OnProgress callback for each input.
func (p *Pipeline) Run(ctx context.Context, inputs ...string) ([]*Document, error) {
	if len(inputs) == 0 {
		return nil, nil
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	docs := make([]*Document, len(inputs))
	var mu sync.Mutex

	var wg sync.WaitGroup
	sem := make(chan struct{}, p.config.Pipeline.Concurrency)
	errCh := make(chan error, len(inputs))

	for i, input := range inputs {
		wg.Add(1)
		go func(i int, input string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				errCh <- fmt.Errorf("%s: %w", input, ctx.Err())
				return
			}
			defer func() { <-sem }()

			doc, pr := p.ProcessOneWithResult(ctx, input)
			if pr.Error != nil {
				errCh <- fmt.Errorf("%s: %w", input, pr.Error)
				cancel()
				return
			}

			mu.Lock()
			docs[i] = doc
			mu.Unlock()
		}(i, input)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return nil, err
		}
	}

	return docs, nil
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
func buildDocument(input string, fr FileResult, chunks []Chunk, embedder Embedder, startAt time.Time) *Document {
	finishedAt := time.Now().UTC()

	source := input
	if input == "-" {
		source = "stdin"
	} else {
		source = filepath.Base(input)
	}

	hashInput := input + fr.Hash
	hash := sha256.Sum256([]byte(hashInput))
	stem := strings.TrimSuffix(source, filepath.Ext(source))
	if stem == "" {
		stem = "file"
	}
	docID := fmt.Sprintf("%s_%x", stem, hash[:8])

	dim := embedder.Dimension()
	if dim == 0 && len(chunks) > 0 && len(chunks[0].Vector) > 0 {
		dim = len(chunks[0].Vector)
	}

	for i := range chunks {
		chunks[i].ChunkID = fmt.Sprintf("%s_%d", docID, i)
	}

	return &Document{
		DocID: docID,
		Metadata: Metadata{
			Source:    source,
			FileType:  fr.FileType,
			FileSize:  fr.Size,
			FileHash:  fr.Hash,
			CreatedAt: startAt,
		},
		Embedding: EmbeddingMetadata{
			Provider:  embedder.Provider(),
			Model:     embedder.Model(),
			Dimension: dim,
			Version:   "v1",
		},
		StartAt:    startAt,
		FinishedAt: finishedAt,
		TotalChunk: len(chunks),
		Chunks:     chunks,
	}
}
