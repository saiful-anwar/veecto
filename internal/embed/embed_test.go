package embed

import (
	"context"
	"testing"

	"github.com/saiful-anwar/veecto/internal/core"
)

type mockEmbedder struct {
	provider  string
	model     string
	dimension int
	vecs      [][]float32
	err       error
}

func (m *mockEmbedder) Embed(_ context.Context, _ []string) ([][]float32, error) {
	return m.vecs, m.err
}
func (m *mockEmbedder) Provider() string { return m.provider }
func (m *mockEmbedder) Model() string    { return m.model }
func (m *mockEmbedder) Dimension() int   { return m.dimension }

func TestEmbedBatch(t *testing.T) {
	var calls [][]string
	call := func(ctx context.Context, texts []string) ([][]float32, error) {
		calls = append(calls, texts)
		result := make([][]float32, len(texts))
		for i := range texts {
			result[i] = []float32{float32(i)}
		}
		return result, nil
	}

	texts := []string{"a", "b", "c", "d", "e"}
	results, err := embedBatch(context.Background(), texts, 2, call)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 5 {
		t.Errorf("expected 5 results, got %d", len(results))
	}
	if len(calls) != 3 {
		t.Errorf("expected 3 batches, got %d", len(calls))
	}

}

func TestEmbedBatchEmpty(t *testing.T) {
	call := func(ctx context.Context, texts []string) ([][]float32, error) {
		return nil, nil
	}
	results, err := embedBatch(context.Background(), nil, 32, call)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestFloat64To32(t *testing.T) {
	src := []float64{1.0, 2.5, 3.14159}
	dst := float64To32(src)
	if len(dst) != 3 {
		t.Fatalf("expected 3, got %d", len(dst))
	}
	if dst[0] != 1.0 || dst[1] != 2.5 {
		t.Errorf("conversion mismatch")
	}
}

func TestRetryEmbedder(t *testing.T) {
	inner := &mockEmbedder{
		provider:  "mock",
		model:     "test",
		dimension: 4,
	}

	retry := Retry(inner, 2, 0)
	if retry.Provider() != "mock" {
		t.Errorf("expected mock, got %s", retry.Provider())
	}
	if retry.Model() != "test" {
		t.Errorf("expected test, got %s", retry.Model())
	}
}

type mockEmbedderErr struct {
	mockEmbedder
	failAttempts int
	attempts     int
}

func (m *mockEmbedderErr) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	m.attempts++
	if m.attempts <= m.failAttempts {
		return nil, errTestFail
	}
	return [][]float32{{0.1, 0.2}}, nil
}

var errTestFail = &testError{}

type testError struct{}

func (e *testError) Error() string { return "test error" }

func TestRetryEmbedderSuccess(t *testing.T) {
	inner := &mockEmbedderErr{failAttempts: 2}
	retry := Retry(inner, 3, 0)
	vecs, err := retry.Embed(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatalf("expected success after retries: %v", err)
	}
	if len(vecs) != 1 {
		t.Errorf("expected 1 vector, got %d", len(vecs))
	}
}

func TestRetryEmbedderFailure(t *testing.T) {
	inner := &mockEmbedderErr{failAttempts: 5}
	retry := Retry(inner, 2, 0)
	_, err := retry.Embed(context.Background(), []string{"hello"})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
}

func TestNewEmbedder(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.Embedding.Provider = "openai"
	_, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	cfg.Embedding.Provider = "unknown"
	_, err = New(cfg)
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}
