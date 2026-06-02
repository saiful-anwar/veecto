package core

import "time"

// Document is the top-level output of the ingestion pipeline for a single input.
type Document struct {
	DocID      string            `json:"doc_id"`
	Metadata   Metadata          `json:"metadata"`
	Embedding  EmbeddingMetadata `json:"embedding"`
	StartAt    time.Time         `json:"startAt"`
	FinishedAt time.Time         `json:"finishedAt"`
	TotalChunk int               `json:"totalChunk"`
	Chunks     []Chunk           `json:"chunks"`
}

// Metadata holds provenance information about the source file.
type Metadata struct {
	Source    string    `json:"source"`
	FileType  string    `json:"file_type,omitempty"`
	FileSize  int64     `json:"file_size,omitempty"`
	FileHash  string    `json:"file_hash,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// EmbeddingMetadata describes the embedding model used.
type EmbeddingMetadata struct {
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Dimension int    `json:"dimension"`
	Version   string `json:"version,omitempty"`
}

// Chunk is a single text segment with optional cleaning and vector embedding.
type Chunk struct {
	ChunkID    string    `json:"chunk_id,omitempty"`
	Index      int       `json:"index"`
	Text       string    `json:"text"`
	TextClean  string    `json:"text_clean,omitempty"`
	Vector     []float32 `json:"vector,omitempty"`
	TokenCount int       `json:"token_count,omitempty"`
	CharStart  int       `json:"char_start,omitempty"`
	CharEnd    int       `json:"char_end,omitempty"`
}

// Progress is reported to ProgressFunc during pipeline execution.
type Progress struct {
	Input      string
	FileType   string
	ChunkCount int
	Duration   time.Duration
	Error      error
}

// ProgressFunc is a callback type for pipeline progress reporting.
type ProgressFunc func(Progress)
