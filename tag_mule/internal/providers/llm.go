package providers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/doro098/tag_mule/internal/config"
)

type LLMTagger struct {
	provider Provider
}

func NewLLMTagger(provider Provider) *LLMTagger {
	return &LLMTagger{provider: provider}
}

func (t *LLMTagger) Tag(text string, existingTags []string, cfg config.SourceConfig) ([]string, error) {
	var sb strings.Builder
	sb.WriteString(cfg.SystemPrompt)
	sb.WriteString("\n\nTexto: ")
	sb.WriteString(text)
	if len(existingTags) > 0 {
		sb.WriteString("\nEtiquetas existentes: ")
		sb.WriteString(strings.Join(existingTags, ", "))
	}
	sb.WriteString("\n\nResponde SOLO un JSON array de strings, sin explicaciones.")

	raw, err := t.provider.Generate(sb.String(), cfg)
	if err != nil {
		return nil, fmt.Errorf("llm generate: %w", err)
	}

	raw = strings.TrimSpace(raw)
	// Try to extract JSON array from response
	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("llm response is not a JSON array: %s", raw)
	}

	var tags []string
	if err := json.Unmarshal([]byte(raw[start:end+1]), &tags); err != nil {
		return nil, fmt.Errorf("llm parse tags: %w, raw: %s", err, raw)
	}

	// Filter out existing tags
	filtered := make([]string, 0, len(tags))
	existing := make(map[string]bool, len(existingTags))
	for _, t := range existingTags {
		existing[strings.ToLower(t)] = true
	}
	for _, t := range tags {
		if !existing[strings.ToLower(t)] {
			filtered = append(filtered, t)
		}
	}

	// Limit to max_tags
	if len(filtered) > cfg.MaxTags {
		filtered = filtered[:cfg.MaxTags]
	}

	return filtered, nil
}
