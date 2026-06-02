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

// openAI implements core.Embedder for the OpenAI embeddings API.
type openAI struct {
	apiKey    string
	model     string
	batchSize int
}

func newOpenAI(apiKey, model string, batchSize int) core.Embedder {
	if model == "" {
		model = "text-embedding-3-small"
	}
	if batchSize <= 0 {
		batchSize = 32
	}
	return &openAI{apiKey: apiKey, model: model, batchSize: batchSize}
}

func (e *openAI) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return embedBatch(ctx, texts, e.batchSize, e.call)
}

type openaiRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type openaiResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (e *openAI) call(ctx context.Context, texts []string) ([][]float32, error) {
	body := openaiRequest{Input: texts, Model: e.model}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/embeddings", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openai read: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("openai: HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var res openaiResponse
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("openai decode: %w", err)
	}
	if res.Error != nil {
		return nil, fmt.Errorf("openai: %s", res.Error.Message)
	}

	result := make([][]float32, len(res.Data))
	for i, d := range res.Data {
		result[i] = float64To32(d.Embedding)
	}
	return result, nil
}

func (e *openAI) Provider() string { return "openai" }
func (e *openAI) Model() string    { return e.model }
func (e *openAI) Dimension() int   { return 1536 }
