package providers

import "github.com/doro098/tag_mule/internal/config"

type Provider interface {
	Generate(prompt string, cfg config.SourceConfig) (string, error)
	Embed(text string, model string) ([]float64, error)
}

type Tagger interface {
	Tag(text string, existingTags []string, cfg config.SourceConfig) ([]string, error)
}
