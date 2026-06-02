package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saiful-anwar/veecto/internal/core"
)

func TestOutputJSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.jsonl")

	w, err := New(path, false)
	if err != nil {
		t.Fatal(err)
	}

	doc := &core.Document{
		DocID:      "test_doc",
		TotalChunk: 1,
	}
	if err := w.Write(doc); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "test_doc") {
		t.Error("output should contain doc_id")
	}
}

func TestOutputPretty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")

	w, err := New(path, true)
	if err != nil {
		t.Fatal(err)
	}

	doc := &core.Document{
		DocID:      "test_doc",
		TotalChunk: 2,
		Chunks: []core.Chunk{
			{Index: 0, Text: "hello", TokenCount: 1},
			{Index: 1, Text: "world", TokenCount: 1},
		},
	}
	if err := w.Write(doc); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "\n  ") {
		t.Error("pretty output should have indentation")
	}
}

func TestOutputEmptyPath(t *testing.T) {
	_, err := New("", false)
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestMultiWriter(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "out1.jsonl")
	p2 := filepath.Join(dir, "out2.jsonl")

	w1, _ := New(p1, false)
	w2, _ := New(p2, false)
	mw := Multi(w1, w2)

	doc := &core.Document{DocID: "multi_test", TotalChunk: 0}
	if err := mw.Write(doc); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{p1, p2} {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "multi_test") {
			t.Errorf("%s should contain doc_id", p)
		}
	}
}
