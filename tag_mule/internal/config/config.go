package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type SourceConfig struct {
	CallbackURL  string   `yaml:"callback_url"`
	Method       string   `yaml:"method"`
	Provider     string   `yaml:"provider"`
	Model        string   `yaml:"model"`
	Temperature  float64  `yaml:"temperature,omitempty"`
	MaxTags      int      `yaml:"max_tags"`
	Threshold    float64  `yaml:"threshold,omitempty"`
	Categories   []string `yaml:"categories,omitempty"`
	SystemPrompt string   `yaml:"system_prompt,omitempty"`
}

type AppConfig struct {
	Sources map[string]SourceConfig `yaml:"sources"`
}

type Config struct {
	Port               string
	DBPath             string
	ConfigPath         string
	WorkerCount        int
	CallbackTimeout    time.Duration
	MaxAttempts        int
	OrphanTimeout      time.Duration
	OrphanCheckInterval time.Duration
	ShutdownTimeout    time.Duration
	OllamaURL          string
	ZaiAPIKey          string
	ZaiBaseURL         string
	LogLevel           string
	AppConfig          *AppConfig
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func Load() *Config {
	cfg := &Config{
		Port:                env("PORT", "8080"),
		DBPath:              env("DB_PATH", "/data/tagmule.db"),
		ConfigPath:          env("CONFIG_PATH", "/app/config.yaml"),
		WorkerCount:         3,
		CallbackTimeout:     30 * time.Second,
		MaxAttempts:         3,
		OrphanTimeout:       10 * time.Minute,
		OrphanCheckInterval: 2 * time.Minute,
		ShutdownTimeout:     30 * time.Second,
		OllamaURL:           env("OLLAMA_URL", "http://ollama:11434"),
		ZaiBaseURL:          env("ZAI_BASE_URL", "https://api.z.ai/v1"),
		ZaiAPIKey:           os.Getenv("ZAI_API_KEY"),
		LogLevel:            env("LOG_LEVEL", "info"),
	}

	if v := os.Getenv("WORKER_COUNT"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.WorkerCount)
	}

	appCfg, err := loadYAML(cfg.ConfigPath)
	if err != nil {
		logF("warn", "could not load config.yaml: %v", err)
		appCfg = &AppConfig{Sources: map[string]SourceConfig{}}
	}
	cfg.AppConfig = appCfg

	return cfg
}

func loadYAML(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg AppConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func logF(level, format string, args ...interface{}) {
	// Simple structured logging placeholder
}
