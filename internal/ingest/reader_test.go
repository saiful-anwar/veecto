package ingest

import (
	"os"
	"path/filepath"
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
