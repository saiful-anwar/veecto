package embed

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/saiful-anwar/veecto/internal/core"
)

// httpError wraps an HTTP response error with status code and body.
type httpError struct {
	code int
	body string
}

func (e *httpError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.code, e.body)
}

// retryableStatus returns true for status codes that should be retried.
func retryableStatus(code int) bool {
	switch code {
	case 408, 425, 429, 500, 502, 503, 504:
		return true
	default:
		return code >= 500
	}
}

var retryAfterRE = regexp.MustCompile(`(?i)Retry-After:\s*(\d+)`)

// retry wraps a core.Embedder with exponential backoff retry logic.
type retry struct {
	inner     core.Embedder
	maxRetry  int
	baseDelay time.Duration
}

// Retry wraps e with exponential backoff. baseDelay is doubled on each retry.
// Each delay includes ±25 % jitter. Only retries on retryable errors
// (network timeouts, HTTP 408/425/429/500/502/503/504).
func Retry(e core.Embedder, maxRetries int, baseDelay time.Duration) core.Embedder {
	return &retry{inner: e, maxRetry: maxRetries, baseDelay: baseDelay}
}

func (r *retry) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	var lastErr error
	for attempt := 0; attempt <= r.maxRetry; attempt++ {
		if attempt > 0 {
			delay := r.backoff(attempt, lastErr)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
		vecs, err := r.inner.Embed(ctx, texts)
		if err == nil {
			return vecs, nil
		}
		if !isRetryable(err) {
			return nil, err
		}
		lastErr = err
	}
	return nil, fmt.Errorf("embed after %d retries: %w", r.maxRetry, lastErr)
}

// isRetryable returns true if the error can be retried.
func isRetryable(err error) bool {
	// Network errors (timeouts, connection refused, DNS failures).
	var netErr net.Error
	if ok := asNetError(err, &netErr); ok {
		return true
	}
	// HTTP status code.
	var he *httpError
	if ok := asHTTPError(err, &he); ok {
		return retryableStatus(he.code)
	}
	// Generic errors are not retried (bad request, auth failure, etc.).
	return false
}

func asNetError(err error, target *net.Error) bool {
	if err == nil {
		return false
	}
	for err != nil {
		if e, ok := err.(net.Error); ok {
			*target = e
			return true
		}
		err = unwrap(err)
	}
	return false
}

func asHTTPError(err error, target **httpError) bool {
	if err == nil {
		return false
	}
	for err != nil {
		if e, ok := err.(*httpError); ok {
			*target = e
			return true
		}
		err = unwrap(err)
	}
	return false
}

func unwrap(err error) error {
	type unwrapper interface {
		Unwrap() error
	}
	u, ok := err.(unwrapper)
	if !ok {
		return nil
	}
	return u.Unwrap()
}

// backoff computes the delay for the given attempt and error.
// Uses exponential backoff with ±25 % jitter. On 429, attempts
// to extract Retry-After from the error body.
func (r *retry) backoff(attempt int, lastErr error) time.Duration {
	// Try Retry-After header from HTTP error body.
	var he *httpError
	if asHTTPError(lastErr, &he) && he.code == 429 {
		if d := parseRetryAfter(he.body); d > 0 {
			return d
		}
	}

	delay := r.baseDelay * (1 << (attempt - 1))
	return jitter(delay, 0.25)
}

// parseRetryAfter looks for Retry-After: <seconds> in an HTTP response body
// or headers. Headers are not available in httpError (only body), but the
// client-side error may include them. Falls back to body scanning.
func parseRetryAfter(body string) time.Duration {
	// Try header-style lines in body.
	for _, line := range strings.Split(body, "\n") {
		m := retryAfterRE.FindStringSubmatch(line)
		if len(m) >= 2 {
			if sec, err := strconv.Atoi(m[1]); err == nil && sec > 0 && sec <= 120 {
				return time.Duration(sec) * time.Second
			}
		}
	}
	return 0
}

func (r *retry) Provider() string { return r.inner.Provider() }
func (r *retry) Model() string    { return r.inner.Model() }
func (r *retry) Dimension() int   { return r.inner.Dimension() }

// jitter adds ±fraction random jitter to d.
func jitter(d time.Duration, fraction float64) time.Duration {
	if fraction <= 0 {
		return d
	}
	maxJitter := time.Duration(float64(d) * fraction)
	if maxJitter == 0 {
		return d
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(2*maxJitter+1)))
	if err != nil {
		return d
	}
	return d - maxJitter + time.Duration(n.Int64())
}
