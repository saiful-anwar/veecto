package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Chunking.Strategy != "recursive" {
		t.Errorf("expected recursive, got %s", cfg.Chunking.Strategy)
	}
	if cfg.Chunking.Size != 512 {
		t.Errorf("expected 512, got %d", cfg.Chunking.Size)
	}
	if cfg.Chunking.Overlap != 50 {
		t.Errorf("expected 50, got %d", cfg.Chunking.Overlap)
	}
	if cfg.Embedding.Provider != "openai" {
		t.Errorf("expected openai, got %s", cfg.Embedding.Provider)
	}
	if cfg.Pipeline.Concurrency != 4 {
		t.Errorf("expected 4, got %d", cfg.Pipeline.Concurrency)
	}
	if cfg.Embedding.Retries != 3 {
		t.Errorf("expected 3, got %d", cfg.Embedding.Retries)
	}
}

func TestConfigValidate(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Embedding.Provider = "ollama"

	if err := cfg.Validate(); err != nil {
		t.Fatalf("config should be valid: %v", err)
	}

	cfg.Chunking.Size = 0
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for size=0")
	}
	cfg.Chunking.Size = 512

	cfg.Chunking.Overlap = 600
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for overlap >= size")
	}
	cfg.Chunking.Overlap = 50

	cfg.Chunking.Strategy = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid strategy")
	}
	cfg.Chunking.Strategy = "recursive"

	cfg.Embedding.Provider = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid provider")
	}
	cfg.Embedding.Provider = "openai"

	cfg.Embedding.BatchSize = 0
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for batch_size=0")
	}
	cfg.Embedding.BatchSize = 32
}

func TestResolveEnvVars(t *testing.T) {
	os.Setenv("TEST_VEECTO_KEY", "sk-test123")
	defer os.Unsetenv("TEST_VEECTO_KEY")

	input := []byte("key: ${TEST_VEECTO_KEY}")
	result := resolveEnvVars(input)
	expected := "key: sk-test123"
	if string(result) != expected {
		t.Errorf("got %q, want %q", string(result), expected)
	}
}

func TestResolveEnvVarsMissing(t *testing.T) {
	input := []byte("key: ${MISSING_VAR}")
	result := resolveEnvVars(input)
	if string(result) != string(input) {
		t.Errorf("should keep original for missing vars: got %q", string(result))
	}
}

func TestLoadConfigFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "veecto.yaml")
	content := []byte(`chunking:
  strategy: fixed
  size: 256
  overlap: 25
embedding:
  provider: ollama
  batch_size: 16
`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Chunking.Strategy != "fixed" {
		t.Errorf("expected fixed, got %s", cfg.Chunking.Strategy)
	}
	if cfg.Chunking.Size != 256 {
		t.Errorf("expected 256, got %d", cfg.Chunking.Size)
	}
	if cfg.Embedding.Provider != "ollama" {
		t.Errorf("expected ollama, got %s", cfg.Embedding.Provider)
	}
	if cfg.Embedding.BatchSize != 16 {
		t.Errorf("expected 16, got %d", cfg.Embedding.BatchSize)
	}
}

func TestLoadConfigDefaultsApplied(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "minimal.yaml")
	if err := os.WriteFile(path, []byte("pipeline:\n  output: test.jsonl\nembedding:\n  provider: ollama"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Pipeline.Concurrency != 4 {
		t.Errorf("expected concurrency default 4, got %d", cfg.Pipeline.Concurrency)
	}
	if cfg.Embedding.Retries != 3 {
		t.Errorf("expected retries default 3, got %d", cfg.Embedding.Retries)
	}
}

func TestFindConfig(t *testing.T) {
	dir := t.TempDir()
	origWd, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origWd)

	os.WriteFile("veecto.yaml", []byte("pipeline:\n  output: test.jsonl"), 0644)

	found, err := FindConfig("")
	if err != nil {
		t.Fatal(err)
	}
	if found == "" {
		t.Fatal("expected to find config")
	}
}

func TestFindConfigCustomPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom.yaml")
	os.WriteFile(path, []byte{}, 0644)

	found, err := FindConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if found != path {
		t.Errorf("expected %s, got %s", path, found)
	}
}

func TestFindConfigMissing(t *testing.T) {
	_, err := FindConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for missing config")
	}
}

func TestExpansionStrategyList(t *testing.T) {
	if !validStrategies["recursive"] {
		t.Error("recursive should be valid")
	}
	if !validStrategies["fixed"] {
		t.Error("fixed should be valid")
	}
	if !validStrategies["sentence"] {
		t.Error("sentence should be valid")
	}
	if !validStrategies["markdown"] {
		t.Error("markdown should be valid")
	}
	if validStrategies["invalid"] {
		t.Error("invalid should not be valid")
	}
}
