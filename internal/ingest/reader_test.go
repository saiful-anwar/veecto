package ingest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadPlainFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := "hello world"
	os.WriteFile(path, []byte(content), 0644)

	fr, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if fr.Text != content {
		t.Errorf("expected %q, got %q", content, fr.Text)
	}
	if fr.FileType != "text" {
		t.Errorf("expected text, got %s", fr.FileType)
	}
	if fr.Size != int64(len(content)) {
		t.Errorf("expected %d, got %d", len(content), fr.Size)
	}
	if fr.Hash == "" {
		t.Error("expected non-empty hash")
	}
}

func TestReadMarkdown(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	content := "# Hello\n\nWorld"
	os.WriteFile(path, []byte(content), 0644)

	fr, err := ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if fr.FileType != "markdown" {
		t.Errorf("expected markdown, got %s", fr.FileType)
	}
}

func TestReadUnsupportedType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.bin")
	os.WriteFile(path, []byte{0, 1, 2}, 0644)

	_, err := ReadFile(path)
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

func TestReadMissingFile(t *testing.T) {
	_, err := ReadFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestDetectExt(t *testing.T) {
	tests := []struct {
		url         string
		contentType string
		expected    string
	}{
		{"https://example.com/doc.pdf", "", ".pdf"},
		{"https://example.com/doc", "application/pdf", ".pdf"},
		{"https://example.com/doc", "text/html", ".html"},
		{"https://example.com/doc", "text/plain", ".txt"},
		{"https://example.com/doc.PDF", "", ".pdf"},
		{"https://example.com/doc", "application/octet-stream", ""},
	}
	for _, tt := range tests {
		ext := detectExt(tt.url, tt.contentType)
		if ext != tt.expected {
			t.Errorf("detectExt(%q, %q) = %q, want %q", tt.url, tt.contentType, ext, tt.expected)
		}
	}
}

func TestStripMarkdown(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "removes atx headers",
			input: "# Title\n\n## Subtitle\n\nBody",
			want:  "Title\n\nSubtitle\n\nBody",
		},
		{
			name:  "removes bold and italic",
			input: "**bold** and *italic* and __also__ and _that_",
			want:  "bold and italic and also and that",
		},
		{
			name:  "removes inline code",
			input: "Use `code` inline",
			want:  "Use  inline",
		},
		{
			name:  "removes links and images",
			input: "A [link](url) and ![image](img.png)",
			want:  "A link and image",
		},
		{
			name:  "removes tables",
			input: "| a | b |\n| --- | --- |\n| 1 | 2 |",
			want:  "",
		},
		{
			name:  "removes blockquotes",
			input: "> quoted text",
			want:  "quoted text",
		},
		{
			name:  "collapses multiple blank lines",
			input: "a\n\n\n\nb",
			want:  "a\n\nb",
		},
		{
			name:  "trims whitespace",
			input: "  \n\nhello\n\n  ",
			want:  "hello",
		},
		{
			name:  "preserves regular text",
			input: "Just plain text with no formatting",
			want:  "Just plain text with no formatting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripMarkdown(tt.input)
			if got != tt.want {
				t.Errorf("stripMarkdown(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestReadUnsupportedDocx(t *testing.T) {
	dir := t.TempDir()
	// .docx is now supported; a non-existent .docx file should give an open error
	path := filepath.Join(dir, "test.docx")
	_, err := ReadFile(path)
	if err == nil {
		t.Error("expected error for missing .docx file")
	}
}

func TestReadUnsupportedHtml(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.html")
	os.WriteFile(path, []byte("<p>hello</p>"), 0644)

	_, err := ReadFile(path)
	if err == nil {
		t.Error("expected error for unsupported .html file")
	}
	if !strings.Contains(err.Error(), "unsupported file type") {
		t.Errorf("expected 'unsupported file type' error, got: %v", err)
	}
}
