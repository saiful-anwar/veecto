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

// gemini implements core.Embedder for the Google Gemini embeddings API.
type gemini struct {
	apiKey    string
	model     string
	batchSize int
	baseURL   string
	client    *http.Client

	mu        sync.Mutex
	dimension int
}

func newGemini(apiKey, model string, batchSize int, baseURL, bearerToken string, client *http.Client) core.Embedder {
	if model == "" {
		model = "text-embedding-004"
	}
	if batchSize <= 0 {
		batchSize = 32
	}
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	token := apiKey
	if bearerToken != "" {
		token = bearerToken
	}
	return &gemini{apiKey: token, model: model, batchSize: batchSize, baseURL: baseURL, client: client}
}

type geminiContent struct {
	Parts []struct {
		Text string `json:"text"`
	} `json:"parts"`
}

type geminiRequest struct {
	Content geminiContent `json:"content"`
}

type geminiResponse struct {
	Embedding struct {
		Values []float64 `json:"values"`
	} `json:"embedding"`
}

func (e *gemini) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return embedBatch(ctx, texts, e.batchSize, func(ctx context.Context, batch []string) ([][]float32, error) {
		results := make([][]float32, 0, len(batch))
		for _, text := range batch {
			vec, err := e.call(ctx, text)
			if err != nil {
				return nil, err
			}
			results = append(results, vec)
		}
		return results, nil
	})
}

func (e *gemini) call(ctx context.Context, text string) ([]float32, error) {
	url := fmt.Sprintf("%s/models/%s:embedContent", e.baseURL, e.model)

	body := geminiRequest{
		Content: geminiContent{
			Parts: []struct {
				Text string `json:"text"`
			}{{Text: text}},
		},
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gemini read: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, &httpError{code: resp.StatusCode, body: string(raw)}
	}

	var res geminiResponse
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("gemini decode: %w", err)
	}

	vec := float64To32(res.Embedding.Values)

	e.mu.Lock()
	if e.dimension == 0 {
		e.dimension = len(vec)
	}
	e.mu.Unlock()

	return vec, nil
}

func (e *gemini) Provider() string { return "gemini" }
func (e *gemini) Model() string    { return e.model }
func (e *gemini) Dimension() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.dimension
}
