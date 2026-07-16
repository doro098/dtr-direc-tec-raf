package providers

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/doro098/tag_mule/internal/config"
	"github.com/doro098/tag_mule/internal/storage"
)

type EmbeddingTagger struct {
	provider Provider
	db       *storage.SQLite
	source   string
}

func NewEmbeddingTagger(provider Provider, db *storage.SQLite, source string) *EmbeddingTagger {
	return &EmbeddingTagger{provider: provider, db: db, source: source}
}

type categoryScore struct {
	Category string
	Score    float64
}

func (t *EmbeddingTagger) Tag(text string, existingTags []string, cfg config.SourceConfig) ([]string, error) {
	textEmbedding, err := t.provider.Embed(text, cfg.Model)
	if err != nil {
		return nil, fmt.Errorf("embed text: %w", err)
	}

	type catEmb struct {
		category  string
		embedding []float64
	}

	categories := make([]catEmb, 0, len(cfg.Categories))
	for _, cat := range cfg.Categories {
		emb, err := t.db.GetCategoryEmbedding(t.source, cat)
		if err != nil {
			// Cache miss - compute and save
			emb, err = t.provider.Embed(cat, cfg.Model)
			if err != nil {
				continue
			}
			t.db.SaveCategoryEmbedding(t.source, cat, emb)
		}
		categories = append(categories, catEmb{category: cat, embedding: emb})
	}

	var scores []categoryScore
	for _, ce := range categories {
		sim := cosineSimilarity(textEmbedding, ce.embedding)
		if sim >= cfg.Threshold {
			scores = append(scores, categoryScore{Category: ce.category, Score: sim})
		}
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	existing := make(map[string]bool, len(existingTags))
	for _, t := range existingTags {
		existing[strings.ToLower(t)] = true
	}

	result := make([]string, 0, cfg.MaxTags)
	for _, s := range scores {
		if !existing[strings.ToLower(s.Category)] {
			result = append(result, s.Category)
			if len(result) >= cfg.MaxTags {
				break
			}
		}
	}

	return result, nil
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
