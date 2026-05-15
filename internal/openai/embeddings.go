package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type Embedder struct {
	APIKey     string
	Model      string
	Dimensions int
	Client     *http.Client
}

func (e Embedder) Embed(ctx context.Context, text string) ([]float64, error) {
	key := e.APIKey
	if key == "" {
		key = os.Getenv("OPENAI_API_KEY")
	}
	if key == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY or --openai-api-key is required for semantic search")
	}
	model := e.Model
	if model == "" {
		model = "text-embedding-3-small"
	}
	payload := map[string]any{"model": model, "input": text}
	if e.Dimensions > 0 {
		payload["dimensions"] = e.Dimensions
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	client := e.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai embeddings request failed: %s", resp.Status)
	}
	var parsed struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	if len(parsed.Data) == 0 {
		return nil, fmt.Errorf("openai embeddings response contained no embeddings")
	}
	return parsed.Data[0].Embedding, nil
}
