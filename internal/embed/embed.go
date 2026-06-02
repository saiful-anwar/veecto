package embed

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/saiful-anwar/veecto/internal/core"
)

// New creates a core.Embedder from the given config. The embedder is wrapped with
// retry middleware when cfg.Embedding.Retries > 0.
func New(cfg core.Config) (core.Embedder, error) {
	var e core.Embedder
	switch cfg.Embedding.Provider {
	case "openai":
		e = newOpenAI(cfg.Embedding.OpenAI.APIKey, cfg.Embedding.OpenAI.Model, cfg.Embedding.BatchSize)
	case "ollama":
		e = newOllama(cfg.Embedding.Ollama.Endpoint, cfg.Embedding.Ollama.Model, cfg.Embedding.BatchSize)
	case "gemini":
		e = newGemini(cfg.Embedding.Gemini.APIKey, cfg.Embedding.Gemini.Model, cfg.Embedding.BatchSize)
	case "http":
		e = newHTTP(cfg.Embedding.HTTP.Endpoint, cfg.Embedding.BatchSize)
	default:
		return nil, fmt.Errorf("unknown embedder provider: %s", cfg.Embedding.Provider)
	}
	if cfg.Embedding.Retries > 0 {
		e = Retry(e, cfg.Embedding.Retries, 1)
	}
	return e, nil
}

// httpClient is the shared HTTP client used by all embedder providers.
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// embedBatch splits texts into batches and calls fn for each batch, respecting ctx cancellation.
func embedBatch(ctx context.Context, texts []string, batchSize int, call func(context.Context, []string) ([][]float32, error)) ([][]float32, error) {
	var results [][]float32
	for i := 0; i < len(texts); i += batchSize {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		vecs, err := call(ctx, texts[i:end])
		if err != nil {
			return nil, fmt.Errorf("batch %d: %w", i/batchSize, err)
		}
		results = append(results, vecs...)
	}
	return results, nil
}

// float64To32 converts a []float64 slice to []float32.
func float64To32(src []float64) []float32 {
	dst := make([]float32, len(src))
	for i, v := range src {
		dst[i] = float32(v)
	}
	return dst
}
