package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/saiful-anwar/veecto/internal/core"
)

// ollama implements core.Embedder for Ollama's embeddings endpoint.
type ollama struct {
	endpoint  string
	model     string
	batchSize int
}

func newOllama(endpoint, model string, batchSize int) core.Embedder {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	if model == "" {
		model = "nomic-embed-text"
	}
	if batchSize <= 0 {
		batchSize = 32
	}
	return &ollama{endpoint: endpoint, model: model, batchSize: batchSize}
}

type ollamaRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type ollamaResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

func (e *ollama) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return embedBatch(ctx, texts, e.batchSize, e.call)
}

func (e *ollama) call(ctx context.Context, texts []string) ([][]float32, error) {
	body := ollamaRequest{Model: e.model, Input: texts}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.endpoint+"/api/embed", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ollama read: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("ollama: HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var res ollamaResponse
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("ollama decode: %w", err)
	}

	result := make([][]float32, len(res.Embeddings))
	for i, emb := range res.Embeddings {
		result[i] = float64To32(emb)
	}
	return result, nil
}

func (e *ollama) Provider() string { return "ollama" }
func (e *ollama) Model() string    { return e.model }
func (e *ollama) Dimension() int   { return 768 }
