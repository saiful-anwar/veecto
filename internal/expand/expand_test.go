package expand

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInputsSingleFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "test.txt")
	os.WriteFile(p, []byte("hello"), 0644)

	expanded, err := Inputs([]string{p})
	if err != nil {
		t.Fatal(err)
	}
	if len(expanded) != 1 {
		t.Fatalf("expected 1, got %d", len(expanded))
	}
}

func TestInputsGlob(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644)
	os.WriteFile(filepath.Join(dir, "c.md"), []byte("c"), 0644)
	os.WriteFile(filepath.Join(dir, "d.pdf"), []byte("d"), 0644)

	expanded, err := Inputs([]string{filepath.Join(dir, "*.txt")})
	if err != nil {
		t.Fatal(err)
	}
	if len(expanded) != 2 {
		t.Fatalf("expected 2, got %d", len(expanded))
	}
}

func TestInputsDirectory(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.md"), []byte("b"), 0644)
	os.WriteFile(filepath.Join(dir, "c.pdf"), []byte("c"), 0644)
	os.WriteFile(filepath.Join(dir, "d.bin"), []byte("d"), 0644)

	expanded, err := Inputs([]string{dir})
	if err != nil {
		t.Fatal(err)
	}
	if len(expanded) != 3 {
		t.Fatalf("expected 3 (txt, md, pdf), got %d: %v", len(expanded), expanded)
	}
}

func TestInputsURL(t *testing.T) {
	expanded, err := Inputs([]string{"https://example.com/doc.pdf"})
	if err != nil {
		t.Fatal(err)
	}
	if len(expanded) != 1 {
		t.Fatalf("expected 1, got %d", len(expanded))
	}
	if expanded[0] != "https://example.com/doc.pdf" {
		t.Errorf("unexpected: %s", expanded[0])
	}
}

func TestInputsStdin(t *testing.T) {
	expanded, err := Inputs([]string{"-"})
	if err != nil {
		t.Fatal(err)
	}
	if len(expanded) != 1 || expanded[0] != "-" {
		t.Errorf("expected [\"-\"], got %v", expanded)
	}
}

func TestInputsNoMatch(t *testing.T) {
	_, err := Inputs([]string{"/nonexistent/glob_*.txt"})
	if err == nil {
		t.Error("expected error for no match")
	}
}

func TestListDir(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "test.md"), []byte("b"), 0644)

	files, err := listDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2, got %d", len(files))
	}
}
