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

// httpEmbed implements core.Embedder for a custom HTTP embeddings endpoint.
type httpEmbed struct {
	endpoint  string
	batchSize int
}

func newHTTP(endpoint string, batchSize int) core.Embedder {
	if endpoint == "" {
		endpoint = "http://localhost:8080/embed"
	}
	if batchSize <= 0 {
		batchSize = 32
	}
	return &httpEmbed{endpoint: endpoint, batchSize: batchSize}
}

type httpRequest struct {
	Input []string `json:"input"`
}

type httpResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
	Data       []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

func (e *httpEmbed) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return embedBatch(ctx, texts, e.batchSize, e.call)
}

func (e *httpEmbed) call(ctx context.Context, texts []string) ([][]float32, error) {
	body := httpRequest{Input: texts}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http embedder: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("http embedder read: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http embedder: HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var res httpResponse
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("http embedder decode: %w", err)
	}

	if len(res.Embeddings) > 0 {
		result := make([][]float32, len(res.Embeddings))
		for i, emb := range res.Embeddings {
			result[i] = float64To32(emb)
		}
		return result, nil
	}
	if len(res.Data) > 0 {
		result := make([][]float32, len(res.Data))
		for i, d := range res.Data {
			result[i] = float64To32(d.Embedding)
		}
		return result, nil
	}

	return nil, fmt.Errorf("http embedder: no embeddings in response")
}

func (e *httpEmbed) Provider() string { return "http" }
func (e *httpEmbed) Model() string    { return "custom" }
func (e *httpEmbed) Dimension() int   { return 0 }
