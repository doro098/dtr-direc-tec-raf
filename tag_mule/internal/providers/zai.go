package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ZAIClient struct {
	BaseURL  string
	APIKey   string
	client   *http.Client
}

func NewZAI(baseURL, apiKey string) *ZAIClient {
	return &ZAIClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		client:  &http.Client{},
	}
}

type zaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type zaiChatRequest struct {
	Model    string       `json:"model"`
	Messages []zaiMessage `json:"messages"`
}

type zaiChoice struct {
	Message zaiMessage `json:"message"`
}

type zaiChatResponse struct {
	Choices []zaiChoice `json:"choices"`
}

func (z *ZAIClient) Generate(prompt string, _ SourceCfg) (string, error) {
	reqBody := zaiChatRequest{
		Model: "qwen2.5:3b",
		Messages: []zaiMessage{
			{Role: "system", Content: "Eres un clasificador de contenido."},
			{Role: "user", Content: prompt},
		},
	}
	data, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", z.BaseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+z.APIKey)

	resp, err := z.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("zai chat: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result zaiChatResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("zai parse: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("zai: no choices returned")
	}

	return result.Choices[0].Message.Content, nil
}

func (z *ZAIClient) Embed(text string, model string) ([]float64, error) {
	// Z.AI embeddings not implemented yet
	return nil, fmt.Errorf("zai embedding not supported")
}
