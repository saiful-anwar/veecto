package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/saiful-anwar/veecto/internal/core"
)

// openAI implements core.Embedder for the OpenAI embeddings API.
type openAI struct {
	apiKey    string
	model     string
	batchSize int
	baseURL   string
	client    *http.Client

	mu        sync.Mutex
	dimension int
}

func newOpenAI(apiKey, model string, batchSize int, baseURL, bearerToken string, client *http.Client) core.Embedder {
	if model == "" {
		model = "text-embedding-3-small"
	}
	if batchSize <= 0 {
		batchSize = 32
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	token := apiKey
	if bearerToken != "" {
		token = bearerToken
	}
	return &openAI{apiKey: token, model: model, batchSize: batchSize, baseURL: baseURL, client: client}
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/embeddings", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openai read: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, &httpError{code: resp.StatusCode, body: string(raw)}
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

	e.mu.Lock()
	if e.dimension == 0 && len(result) > 0 {
		e.dimension = len(result[0])
	}
	e.mu.Unlock()

	return result, nil
}

func (e *openAI) Provider() string { return "openai" }
func (e *openAI) Model() string    { return e.model }
func (e *openAI) Dimension() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.dimension
}
