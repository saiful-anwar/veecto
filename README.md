# veecto

**veecto** is a RAG ingestion library and CLI that converts documents into vectorized chunks — ready for embedding databases.

```
Input (file/URL/glob/dir/stdin) → Ingest → Chunk → Embed → JSONL output
```

## Features

- **Multi-format input** — `.txt`, `.md`, `.pdf`, `.html`, `.docx`, `.epub`, `.rst`, `.org`, `.latex`
- **Flexible input** — file paths, globs (`*.md`), directories, URLs (`https://...`), stdin (`-`)
- **URL auto-download** — pass any HTTP/HTTPS URL, it's fetched automatically (500MB limit, 5min timeout)
- **PDF via pdftohtml + pandoc** — layout-aware text extraction (not raw `pdftotext`)
- **4 chunking strategies**:
  - `recursive` — splits by `\n\n` → `\n` → `. ` → `, ` → ` ` (best for general text)
  - `fixed` — character-aligned with configurable overlap
  - `sentence` — preserves sentence boundaries (`.` / `!` / `?`)
  - `markdown` — splits by `#` / `##` / `###` headings
- **4 pluggable embedders** — OpenAI, Ollama, Gemini, HTTP (bring your own)
- **Retry + exponential backoff** — auto-retries on transient API failures (configurable)
- **Concurrent processing** — process multiple files in parallel (configurable concurrency)
- **Text cleaning** — Unicode NFC, control char removal, whitespace collapse
- **Metadata enrichment** — source, file type, file size, SHA-256 hash, timestamps
- **JSONL output** — streaming, vector-DB-friendly (one `Document` per line)
- **Pretty JSON** — `--pretty` flag for human-readable output
- **Config search paths** — `./veecto.yaml` → `~/.config/veecto/veecto.yaml` → `/etc/veecto/veecto.yaml`
- **Dual-mode** — import as a Go library **or** use as a CLI binary
- **Validation** — `veecto validate` checks config + system dependencies

## Install

### CLI binary

```bash
go install github.com/saiful-anwar/veecto/cmd/veecto@latest
```

Requires [pandoc](https://pandoc.org) and [poppler](https://poppler.freedesktop.org) (for PDF):

```bash
# macOS
brew install pandoc poppler

# Ubuntu/Debian
sudo apt install pandoc poppler-utils

# Windows
# poppler: https://github.com/oschwartz10612/poppler-windows/releases
# pandoc: https://pandoc.org/installing.html
```

### Go library

```bash
go get github.com/saiful-anwar/veecto
```

## Quick Start (CLI)

```bash
# Ingest a local PDF
veecto ingest doc.pdf -o output.jsonl

# Ingest a URL (auto-downloads)
veecto ingest https://example.com/article.html -e ollama -o output.jsonl

# Ingest all markdown files + a directory
veecto ingest *.md ./docs/ -o corpus.jsonl

# Ingest from stdin
cat article.txt | veecto ingest - -o output.jsonl

# Use sentence-based chunking with Gemini
veecto ingest doc.pdf --chunk-strategy sentence -e gemini --model gmeini-embedding-001

# Validate setup
veecto validate
```

### CLI flags

```
-c, --config string       Config file path
-o, --output string       Output file path (default: output.jsonl)
    --pretty              Pretty-print JSON output
    --chunk-strategy      Chunking strategy (recursive|fixed|sentence|markdown)
    --chunk-size int      Chunk size in chars (default: 512)
    --chunk-overlap int   Chunk overlap in chars (default: 50)
-e, --embedder string     Embedder provider (openai|ollama|gemini|http)
    --model string        Embedding model name
    --batch-size int      Embedding batch size (default: 32)
    --concurrency int     Max concurrent files to process (default: 4)
    --retries int         Max retries for embedding API calls (default: 3)
-v, --verbose             Verbose output
```

## Configuration

Config is auto-discovered: `./veecto.yaml` → `~/.config/veecto/veecto.yaml` → `/etc/veecto/veecto.yaml`.
Or specify with `--config`.

```yaml
pipeline:
  output: "output.jsonl"
  concurrency: 4
  max_file_size: 524288000

chunking:
  strategy: "recursive"     # recursive|fixed|sentence|markdown
  size: 512
  overlap: 50

embedding:
  provider: "gemini"        # openai|ollama|gemini|http
  batch_size: 32
  retries: 3                # retry count with exponential backoff

  openai:
    model: "text-embedding-3-small"
    api_key: "${OPENAI_API_KEY}"

  ollama:
    endpoint: "http://localhost:11434"
    model: "nomic-embed-text"

  gemini:
    model: "gemini-embedding-001"
    api_key: "${GEMINI_API_KEY}"

  http:
    endpoint: "http://localhost:8080/embed"
```

## Library Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/saiful-anwar/veecto"
)

func main() {
    ctx := context.Background()

    // One-shot processing
    doc, err := veecto.Process(ctx, "doc.pdf")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("Processed %s: %d chunks\n", doc.DocID, doc.TotalChunk)

    // With custom config
    cfg := veecto.DefaultConfig()
    cfg.Chunking.Strategy = "markdown"
    cfg.Embedding.Provider = "ollama"

    doc, err = veecto.Process(ctx, "article.md", cfg)
    if err != nil {
        log.Fatal(err)
    }

    // Reusable pipeline with progress
    pipe := veecto.NewPipeline(cfg)
    pipe.OnProgress(func(p veecto.Progress) {
        fmt.Printf("%s: %d chunks in %v\n", p.Input, p.ChunkCount, p.Duration)
    })
    docs, err := pipe.Run(ctx, "file1.txt", "file2.md")

    // Write output
    writer, _ := veecto.NewWriter("output.jsonl", false)
    defer writer.Close()
    for _, d := range docs {
        writer.Write(d)
    }
}
```

### Custom Embedder

```go
type MyEmbedder struct{}

func (e *MyEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
    // your custom embedding logic here
}
func (e *MyEmbedder) Provider() string { return "myprovider" }
func (e *MyEmbedder) Model() string    { return "my-model" }
func (e *MyEmbedder) Dimension() int   { return 768 }
```

## Supported Inputs

| Input | Method |
|-------|--------|
| `.txt`, `.md` | Direct read |
| `.pdf` | `pdftohtml -c -noframes` → `pandoc --to plain` |
| `.html`, `.htm`, `.docx`, `.epub`, `.rst`, `.org`, `.latex` | `pandoc` |
| URL (`http://`, `https://`) | Auto-download (500MB limit, 5min timeout) |
| stdin (`-`) | Read from pipe |
| Globs (`*.md`) | Shell-style glob expansion |
| Directories (`./docs/`) | Recursively discover supported files |

## Output Format

[JSONL](https://jsonlines.org) by default. Use `--pretty` for indented JSON.

```jsonl
{"doc_id":"doc_d41d8cd9","metadata":{"source":"doc.pdf","file_type":"pdf","file_size":102400,"file_hash":"a1b2c3d4e5f6"},"embedding":{"provider":"ollama","model":"nomic-embed-text","dimension":768,"version":"v1"},"startAt":"2026-06-02T12:00:00Z","finishedAt":"2026-06-02T12:00:10Z","totalChunk":2,"chunks":[{"index":0,"text":"RAG systems...","text_clean":"RAG systems...","vector":[0.021,-0.334,0.992],"token_count":42,"char_start":0,"char_end":120}]}
```

## Chunking Strategies

### `recursive` (default)

Tries increasingly granular separators in priority order until chunks fit within `size`:
`\n\n` → `\n` → `. ` → `, ` → ` `. At each level, scans right-to-left for the best split point. Falls back to a hard character cut when no separator is found. Applies `overlap` between adjacent chunks.

**Best for:** General prose, articles, mixed-content documents.

### `fixed`

Blindly splits every `size` characters with configurable `overlap`. No awareness of paragraphs, sentences, or word boundaries.

**Best for:** Code files, log data, or any content where structural boundaries don't matter.

### `sentence`

Splits text at sentence-ending punctuation (`.` / `!` / `?` / `\n`), then merges consecutive sentences until they exceed `size`. Every chunk starts and ends at a sentence boundary.

**Best for:** Articles, research papers, documentation — where breaking mid-sentence would lose meaning.

### `markdown`

Splits at Markdown heading lines (`#`, `##`, `###`, etc.), then merges adjacent sections until they exceed `size`. Preserves heading hierarchy — chunks never split mid-section.

**Best for:** Wikis, API docs, README files — where sections are natural semantic units.

## Text Cleaning

The `text_clean` field applies these rules in order:

1. **Unicode NFC normalization**
2. **Strip ASCII control characters** (except `\n`, `\t`, `\r`)
3. **Collapse whitespace** — runs of spaces/tabs → single space
4. **Trim leading/trailing whitespace**
5. **Drop zero-length chunks** after cleaning

## Embedding Providers

| Provider | Default Model | Auth | Retry |
|----------|--------------|------|-------|
| **openai** | `text-embedding-3-small` | `api_key` (env `OPENAI_API_KEY`) | Yes |
| **ollama** | `nomic-embed-text` | None (local) | Yes (configurable) |
| **gemini** | `gemini-embedding-001` | `api_key` (env `GEMINI_API_KEY`) | Yes |
| **http** | custom | Configurable endpoint | Yes |

## Tests

```bash
go test -v -count=1 ./...
```

## Project Structure

```
veecto/
├── cmd/veecto/main.go    # CLI binary entry point
├── pipeline.go           # Process, ProcessAll, Pipeline
├── types.go              # Type aliases + factory functions (NewChunker, NewEmbedder, NewWriter)
├── options.go            # DefaultConfig, LoadConfig, FindConfig, ProgressFn wrappers
├── reader.go             # ReadFile, ReadStdin, ResolveInput, CheckDeps, ExpandInputs
├── internal/             # Implementation details (not exported)
│   ├── chunk/            # 4 chunker strategies (Recursive, Fixed, Sentence, Markdown)
│   ├── embed/            # 4 providers (OpenAI, Ollama, Gemini, HTTP) + retry middleware
│   ├── output/           # JSONL, Pretty, Multi writers
│   ├── ingest/           # File/URL reading, pandoc/pdftohtml wrappers
│   ├── expand/           # Glob, directory, stdin expansion
│   └── core/             # Shared types, interfaces, Config, Progress
├── go.mod / go.sum
└── README.md
```
