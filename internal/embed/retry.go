package embed

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/saiful-anwar/veecto/internal/core"
)

// retry wraps a core.Embedder with exponential backoff retry logic.
type retry struct {
	inner     core.Embedder
	maxRetry  int
	baseDelay time.Duration
}

// Retry wraps e with exponential backoff. baseDelay is doubled on each retry.
func Retry(e core.Embedder, maxRetries int, baseDelay time.Duration) core.Embedder {
	return &retry{inner: e, maxRetry: maxRetries, baseDelay: baseDelay}
}

func (r *retry) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	var lastErr error
	for attempt := 0; attempt <= r.maxRetry; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(r.baseDelay * time.Duration(math.Pow(2, float64(attempt-1)))):
			}
		}
		vecs, err := r.inner.Embed(ctx, texts)
		if err == nil {
			return vecs, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("embed after %d retries: %w", r.maxRetry, lastErr)
}

func (r *retry) Provider() string { return r.inner.Provider() }
func (r *retry) Model() string    { return r.inner.Model() }
func (r *retry) Dimension() int   { return r.inner.Dimension() }
