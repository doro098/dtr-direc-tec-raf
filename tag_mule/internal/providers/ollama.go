package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type OllamaClient struct {
	BaseURL string
	client  *http.Client
}

func NewOllama(baseURL string) *OllamaClient {
	return &OllamaClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{},
	}
}

type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func (o *OllamaClient) Generate(prompt string, _ SourceCfg) (string, error) {
	model := _cfg.Model
	if model == "" {
		model = "qwen2.5:3b"
	}

	reqBody := ollamaGenerateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false,
	}
	data, _ := json.Marshal(reqBody)

	resp, err := o.client.Post(o.BaseURL+"/api/generate", "application/json", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("ollama generate: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("ollama read body: %w", err)
	}

	var result ollamaGenerateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("ollama parse response: %w", err)
	}

	return result.Response, nil
}

type ollamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaEmbedResponse struct {
	Embedding []float64 `json:"embedding"`
}

func (o *OllamaClient) Embed(text string, model string) ([]float64, error) {
	if model == "" {
		model = "nomic-embed-text"
	}

	reqBody := ollamaEmbedRequest{
		Model:  model,
		Prompt: text,
	}
	data, _ := json.Marshal(reqBody)

	resp, err := o.client.Post(o.BaseURL+"/api/embeddings", "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("ollama embed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ollama read embeddings: %w", err)
	}

	var result ollamaEmbedResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("ollama parse embeddings: %w", err)
	}

	return result.Embedding, nil
}
