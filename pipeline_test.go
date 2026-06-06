package veecto

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type testEmbedder struct {
	name      string
	model     string
	dimension int
}

func (e *testEmbedder) Embed(_ context.Context, texts []string) ([][]float32, error) {
	vecs := make([][]float32, len(texts))
	for i := range texts {
		vecs[i] = []float32{0.1, 0.2, 0.3}
	}
	return vecs, nil
}
func (e *testEmbedder) Provider() string { return e.name }
func (e *testEmbedder) Model() string    { return e.model }
func (e *testEmbedder) Dimension() int   { return e.dimension }

// Test config that uses a mock HTTP embedder.
func testConfig(t *testing.T, svr *httptest.Server) Config {
	t.Helper()
	cfg := DefaultConfig()
	cfg.Embedding.Provider = "http"
	cfg.Embedding.HTTP.Endpoint = svr.URL
	cfg.Embedding.Retries = 0
	cfg.Embedding.BatchSize = 2
	cfg.Pipeline.Concurrency = 2
	return cfg
}

func startTestEmbedServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		resp := struct {
			Embeddings [][]float64 `json:"embeddings"`
		}{
			Embeddings: make([][]float64, len(req.Input)),
		}
		for i := range req.Input {
			resp.Embeddings[i] = []float64{0.1, 0.2, 0.3}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestProcessDefault(t *testing.T) {
	svr := startTestEmbedServer(t)
	defer svr.Close()

	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "Hello world. This is a test document for veecto."
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	doc, err := Process(ctx, path, testConfig(t, svr))
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if doc.DocID == "" {
		t.Error("expected non-empty DocID")
	}
	if doc.TotalChunk == 0 {
		t.Error("expected at least one chunk")
	}
	if doc.Metadata.Source != "test.txt" {
		t.Errorf("expected source 'test.txt', got %q", doc.Metadata.Source)
	}
	if !doc.StartAt.Before(doc.FinishedAt) && !doc.StartAt.Equal(doc.FinishedAt) {
		t.Errorf("StartAt (%v) should be <= FinishedAt (%v)", doc.StartAt, doc.FinishedAt)
	}
}

func TestProcessStdin(t *testing.T) {
	svr := startTestEmbedServer(t)
	defer svr.Close()

	content := "Hello stdin"
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	done := make(chan struct{})
	go func() {
		w.Write([]byte(content))
		w.Close()
		close(done)
	}()

	ctx := context.Background()
	doc, err := Process(ctx, "-", testConfig(t, svr))
	if err != nil {
		t.Fatalf("Process(stdin) failed: %v", err)
	}
	<-done

	if doc.Metadata.Source != "stdin" {
		t.Errorf("expected source 'stdin', got %q", doc.Metadata.Source)
	}
	if doc.TotalChunk == 0 {
		t.Error("expected at least one chunk")
	}
}

func TestProcessAllCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := ProcessAll(ctx, []string{"nonexistent/input.txt"}, DefaultConfig())
	if err == nil {
		t.Error("expected error on cancelled context")
	}
}

func TestProcessAllConcurrent(t *testing.T) {
	svr := startTestEmbedServer(t)
	defer svr.Close()

	dir := t.TempDir()
	paths := make([]string, 3)
	for i := range paths {
		paths[i] = filepath.Join(dir, fmt.Sprintf("file%d.txt", i))
		os.WriteFile(paths[i], []byte(fmt.Sprintf("Content of file %d.", i)), 0644)
	}

	ctx := context.Background()
	cfg := testConfig(t, svr)
	cfg.Pipeline.Concurrency = 2

	docs, err := ProcessAll(ctx, paths, cfg)
	if err != nil {
		t.Fatalf("ProcessAll failed: %v", err)
	}
	if len(docs) != 3 {
		t.Fatalf("expected 3 docs, got %d", len(docs))
	}
}

func TestProcessAllError(t *testing.T) {
	svr := startTestEmbedServer(t)
	defer svr.Close()

	dir := t.TempDir()
	good := filepath.Join(dir, "good.txt")
	os.WriteFile(good, []byte("hello"), 0644)

	ctx := context.Background()
	cfg := testConfig(t, svr)

	// One good input, one non-existent.
	_, err := ProcessAll(ctx, []string{good, "/nonexistent/input.txt"}, cfg)
	if err == nil {
		t.Error("expected error for non-existent input")
	}
}

func TestExtractTexts(t *testing.T) {
	chunks := []Chunk{
		{Text: "raw text", TextClean: ""},
		{Text: "raw 2", TextClean: "clean 2"},
	}
	texts := extractTexts(chunks)
	if len(texts) != 2 {
		t.Fatalf("expected 2 texts, got %d", len(texts))
	}
	if texts[0] != "raw text" {
		t.Errorf("expected 'raw text', got %q", texts[0])
	}
	if texts[1] != "clean 2" {
		t.Errorf("expected 'clean 2', got %q", texts[1])
	}
}

func TestBuildDocument(t *testing.T) {
	fr := FileResult{
		Text:     "test",
		FileType: "text",
		Size:     4,
		Hash:     "abcdef",
	}
	chunks := []Chunk{
		{Index: 0, Text: "chunk one", TokenCount: 3},
		{Index: 1, Text: "chunk two", TokenCount: 3},
	}
	embedder := &testEmbedder{name: "test", model: "m1", dimension: 3}

	doc := buildDocument("test.txt", fr, chunks, embedder, mustParseTime(t, "2020-01-01T00:00:00Z"))
	if doc == nil {
		t.Fatal("expected non-nil document")
	}
	if doc.DocID == "" {
		t.Error("expected non-empty DocID")
	}
	if !strings.HasPrefix(doc.DocID, "test_") {
		t.Errorf("expected DocID prefix 'test_', got %q", doc.DocID)
	}
	if doc.TotalChunk != 2 {
		t.Errorf("expected 2 chunks, got %d", doc.TotalChunk)
	}
	if doc.Chunks[0].ChunkID == "" {
		t.Error("expected non-empty ChunkID")
	}
	if doc.Chunks[0].ChunkID != doc.DocID+"_0" {
		t.Errorf("expected ChunkID %q, got %q", doc.DocID+"_0", doc.Chunks[0].ChunkID)
	}
	if doc.Chunks[1].ChunkID != doc.DocID+"_1" {
		t.Errorf("expected ChunkID %q, got %q", doc.DocID+"_1", doc.Chunks[1].ChunkID)
	}
	if doc.StartAt.Format("2006-01-02") != "2020-01-01" {
		t.Errorf("expected StartAt 2020-01-01, got %v", doc.StartAt)
	}
	if !doc.FinishedAt.After(doc.StartAt) && !doc.FinishedAt.Equal(doc.StartAt) {
		t.Errorf("FinishedAt (%v) should be >= StartAt (%v)", doc.FinishedAt, doc.StartAt)
	}
}

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatal(err)
	}
	return ts
}
