package core

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Default configuration values used when fields are zero-valued in YAML config.
const (
	DefaultChunkSize       = 512
	DefaultChunkOverlap    = 50
	DefaultChunkStrategy   = "recursive"
	DefaultEmbedProvider   = "openai"
	DefaultBatchSize       = 32
	DefaultConcurrency     = 4
	DefaultMaxFileSize     = 500 * 1024 * 1024
	DefaultMaxDownloadSize = 500 * 1024 * 1024
	DefaultMaxTextSize     = 10 * 1024 * 1024
)

// Config defines the full set of configurable parameters for the ingestion pipeline.
type Config struct {
	Pipeline struct {
		Output      string `yaml:"output"`
		Concurrency int    `yaml:"concurrency"`
		MaxFileSize int64  `yaml:"max_file_size"`
		MaxTextSize int64  `yaml:"max_text_size"`
	} `yaml:"pipeline"`

	Chunking struct {
		Strategy       string `yaml:"strategy"`
		Size           int    `yaml:"size"`
		Overlap        int    `yaml:"overlap"`
		AsciiNormalize bool   `yaml:"ascii_normalize"`
	} `yaml:"chunking"`

	Embedding struct {
		Provider  string `yaml:"provider"`
		BatchSize int    `yaml:"batch_size"`
		Retries   int    `yaml:"retries"`
		Timeout   int    `yaml:"timeout"`
		OpenAI    struct {
			Model       string `yaml:"model"`
			APIKey      string `yaml:"api_key"`
			BaseURL     string `yaml:"base_url"`
			BearerToken string `yaml:"bearer_token"`
		} `yaml:"openai"`
		Ollama struct {
			Endpoint string `yaml:"endpoint"`
			Model    string `yaml:"model"`
		} `yaml:"ollama"`
		Gemini struct {
			Model       string `yaml:"model"`
			APIKey      string `yaml:"api_key"`
			BaseURL     string `yaml:"base_url"`
			BearerToken string `yaml:"bearer_token"`
		} `yaml:"gemini"`
		HTTP struct {
			Endpoint    string            `yaml:"endpoint"`
			BearerToken string            `yaml:"bearer_token"`
			Headers     map[string]string `yaml:"headers"`
		} `yaml:"http"`
	} `yaml:"embedding"`
}

// LoadConfig reads, parses, resolves env vars, applies defaults, and validates a YAML config file.
func LoadConfig(path string) (Config, error) {
	// #nosec G304 -- path is a user-provided config file path from CLI flags.
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	data = resolveEnvVars(data)

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	setDefaults(&cfg)

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	var cfg Config
	setDefaults(&cfg)
	return cfg
}

func setDefaults(cfg *Config) {
	if cfg.Chunking.Strategy == "" {
		cfg.Chunking.Strategy = DefaultChunkStrategy
	}
	if cfg.Chunking.Size <= 0 {
		cfg.Chunking.Size = DefaultChunkSize
	}
	if cfg.Chunking.Overlap <= 0 {
		cfg.Chunking.Overlap = DefaultChunkOverlap
	}
	if cfg.Embedding.Provider == "" {
		cfg.Embedding.Provider = DefaultEmbedProvider
	}
	if cfg.Embedding.BatchSize <= 0 {
		cfg.Embedding.BatchSize = DefaultBatchSize
	}
	if cfg.Embedding.Retries <= 0 {
		cfg.Embedding.Retries = 3
	}
	if cfg.Pipeline.Concurrency <= 0 {
		cfg.Pipeline.Concurrency = DefaultConcurrency
	}
	if cfg.Pipeline.MaxFileSize <= 0 {
		cfg.Pipeline.MaxFileSize = DefaultMaxFileSize
	}
	if cfg.Pipeline.MaxTextSize <= 0 {
		cfg.Pipeline.MaxTextSize = DefaultMaxTextSize
	}
	if cfg.Embedding.OpenAI.Model == "" {
		cfg.Embedding.OpenAI.Model = "text-embedding-3-small"
	}
	if cfg.Embedding.Ollama.Endpoint == "" {
		cfg.Embedding.Ollama.Endpoint = "http://localhost:11434"
	}
	if cfg.Embedding.Ollama.Model == "" {
		cfg.Embedding.Ollama.Model = "nomic-embed-text"
	}
	if cfg.Embedding.Gemini.Model == "" {
		cfg.Embedding.Gemini.Model = "text-embedding-004"
	}
	if cfg.Embedding.HTTP.Endpoint == "" {
		cfg.Embedding.HTTP.Endpoint = "http://localhost:8080/embed"
	}
}

var validStrategies = map[string]bool{
	"recursive": true,
	"fixed":     true,
	"sentence":  true,
	"markdown":  true,
}

var validProviders = map[string]bool{
	"openai": true,
	"ollama": true,
	"gemini": true,
	"http":   true,
}

// Validate checks the Config for invalid or conflicting values.
func (c Config) Validate() error {
	if c.Chunking.Size <= 0 {
		return fmt.Errorf("chunk size must be > 0")
	}
	if c.Chunking.Overlap < 0 {
		return fmt.Errorf("chunk overlap must be >= 0")
	}
	if c.Chunking.Overlap >= c.Chunking.Size {
		return fmt.Errorf("chunk overlap (%d) must be < size (%d)", c.Chunking.Overlap, c.Chunking.Size)
	}
	if !validStrategies[c.Chunking.Strategy] {
		return fmt.Errorf("unknown chunk strategy: %q (valid: recursive, fixed, sentence, markdown)", c.Chunking.Strategy)
	}
	if !validProviders[c.Embedding.Provider] {
		return fmt.Errorf("unknown embedder provider: %q (valid: openai, ollama, gemini, http)", c.Embedding.Provider)
	}
	if c.Embedding.BatchSize <= 0 {
		return fmt.Errorf("batch size must be > 0")
	}
	if c.Embedding.Retries < 0 {
		return fmt.Errorf("retries must be >= 0")
	}
	if c.Pipeline.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be > 0")
	}
	if c.Pipeline.MaxFileSize <= 0 {
		return fmt.Errorf("max_file_size must be > 0")
	}
	if c.Pipeline.MaxTextSize < 0 {
		return fmt.Errorf("max_text_size must be >= 0")
	}
	if c.Embedding.Provider == "openai" && c.Embedding.OpenAI.APIKey == "" {
		return fmt.Errorf("openai api_key required when provider=openai")
	}
	if c.Embedding.Provider == "gemini" && c.Embedding.Gemini.APIKey == "" {
		return fmt.Errorf("gemini api_key required when provider=gemini")
	}
	return nil
}

var envPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// resolveEnvVars replaces ${VAR} patterns with os.Getenv values. Unset vars are left as-is.
func resolveEnvVars(data []byte) []byte {
	return envPattern.ReplaceAllFunc(data, func(match []byte) []byte {
		name := string(match[2 : len(match)-1])
		if val := os.Getenv(name); val != "" {
			return []byte(val)
		}
		return match
	})
}

var configSearchPaths = []string{
	"veecto.yaml",
	"veecto.yml",
	".veecto.yaml",
}

// FindConfig searches for a config file. If path is non-empty it is checked directly.
// Otherwise the search order is: ./ → ~/.config/veecto/ → /etc/veecto/.
// Returns ("", nil) when no config is found.
func FindConfig(path string) (string, error) {
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("config not found: %s", path)
		}
		return path, nil
	}

	home, _ := os.UserHomeDir()
	searchDirs := []string{"."}
	if home != "" {
		searchDirs = append(searchDirs, filepath.Join(home, ".config", "veecto"))
	}
	searchDirs = append(searchDirs, "/etc/veecto")

	for _, dir := range searchDirs {
		for _, name := range configSearchPaths {
			full := filepath.Join(dir, name)
			if _, err := os.Stat(full); err == nil {
				return full, nil
			}
		}
	}

	return "", nil
}
