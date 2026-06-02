package chunk

import (
	"testing"

	"github.com/saiful-anwar/veecto/internal/core"
)

func TestFixedChunker(t *testing.T) {
	c := &Fixed{Size: 10, Overlap: 2}
	chunks, err := c.Chunk("hello world this is a test")
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	for _, ch := range chunks {
		if ch.Text == "" {
			t.Error("empty chunk text")
		}
		if ch.TextClean == "" {
			t.Error("empty cleaned text")
		}
		if ch.TokenCount <= 0 {
			t.Error("token count should be > 0")
		}
	}
}

func TestFixedChunkerEmpty(t *testing.T) {
	c := &Fixed{Size: 10, Overlap: 2}
	_, err := c.Chunk("")
	if err == nil {
		t.Error("expected error on empty text")
	}
}

func TestFixedChunkerZeroChunks(t *testing.T) {
	c := &Fixed{Size: 100, Overlap: 0}
	_, err := c.Chunk("   \n\n  \t  ")
	if err == nil {
		t.Error("expected error on whitespace-only text")
	}
}

func TestFixedChunkerLargeOverlap(t *testing.T) {
	c := &Fixed{Size: 10, Overlap: 15}
	chunks, err := c.Chunk("hello world this is a test document")
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
}

func TestRecursiveChunker(t *testing.T) {
	c := &Recursive{Size: 20, Overlap: 5}
	text := "Paragraph one. This is the first paragraph.\n\nParagraph two. This is the second.\n\nParagraph three."
	chunks, err := c.Chunk(text)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	for _, ch := range chunks {
		if ch.Text == "" {
			t.Error("empty chunk text")
		}
		if ch.CharEnd <= ch.CharStart {
			t.Error("char_end should be > char_start")
		}
	}
}

func TestRecursiveChunkerEmpty(t *testing.T) {
	c := &Recursive{Size: 20, Overlap: 5}
	_, err := c.Chunk("")
	if err == nil {
		t.Error("expected error on empty text")
	}
}

func TestSentenceChunker(t *testing.T) {
	c := &Sentence{Size: 100, Overlap: 0}
	text := "First sentence here. Second sentence here! Third sentence? Fourth is long enough to test boundaries of chunk."
	chunks, err := c.Chunk(text)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	for _, ch := range chunks {
		if ch.Text == "" {
			t.Error("empty chunk text")
		}
	}
}

func TestSentenceChunkerEmpty(t *testing.T) {
	c := &Sentence{Size: 100, Overlap: 0}
	_, err := c.Chunk("")
	if err == nil {
		t.Error("expected error on empty text")
	}
}

func TestSentenceChunkerShortText(t *testing.T) {
	c := &Sentence{Size: 100, Overlap: 0}
	chunks, err := c.Chunk("Short text.")
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
}

func TestMarkdownChunker(t *testing.T) {
	c := &Markdown{Size: 200, Overlap: 0}
	text := `# Title

Introduction paragraph.

## Section One

Content under section one.

## Section Two

More content here.

### Subsection

Deep content.`
	chunks, err := c.Chunk(text)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
}

func TestMarkdownChunkerNoHeadings(t *testing.T) {
	c := &Markdown{Size: 200, Overlap: 0}
	chunks, err := c.Chunk("Just a plain paragraph without any markdown headings. But it should still produce chunks.")
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
}

func TestCleanText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  hello   world  ", "hello world"},
		{"\n\nhello\n\nworld\n\n", "hello\n\nworld"},
		{"hello\tworld", "hello\tworld"},
		{"hello\x00world", "helloworld"},
		{"", ""},
	}
	for _, tt := range tests {
		got := cleanText(tt.input, false)
		if got != tt.expected {
			t.Errorf("cleanText(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCleanTextASCII(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"café", "cafe"},
		{"façade", "facade"},
		{"你好世界", ""},
		{"hello world", "hello world"},
	}
	for _, tt := range tests {
		got := cleanText(tt.input, true)
		if got != tt.expected {
			t.Errorf("cleanText(%q, true) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestApproxTokenCount(t *testing.T) {
	if n := approxTokenCount("hello world"); n != 3 {
		t.Errorf("expected ~3 tokens for 'hello world', got %d", n)
	}
	if n := approxTokenCount(""); n != 0 {
		t.Errorf("expected 0 tokens for empty string, got %d", n)
	}
}

func TestNewChunker(t *testing.T) {
	cfg := core.DefaultConfig()
	cfg.Chunking.Strategy = "recursive"
	if _, ok := New(cfg).(*Recursive); !ok {
		t.Error("expected Recursive")
	}
	cfg.Chunking.Strategy = "fixed"
	if _, ok := New(cfg).(*Fixed); !ok {
		t.Error("expected Fixed")
	}
	cfg.Chunking.Strategy = "sentence"
	if _, ok := New(cfg).(*Sentence); !ok {
		t.Error("expected Sentence")
	}
	cfg.Chunking.Strategy = "markdown"
	if _, ok := New(cfg).(*Markdown); !ok {
		t.Error("expected Markdown")
	}
}
