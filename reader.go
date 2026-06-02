package veecto

import (
	"github.com/saiful-anwar/veecto/internal/expand"
	"github.com/saiful-anwar/veecto/internal/ingest"
)

// FileResult holds the text content and metadata for a single file read.
type FileResult = ingest.Result

// ReadFile reads a file at the given path. Supports .txt, .md (direct read),
// .pdf (via pdftohtml + pandoc), and other formats via pandoc.
func ReadFile(path string) (FileResult, error) {
	return ingest.ReadFile(path)
}

// ReadStdin reads all data from os.Stdin.
func ReadStdin() (FileResult, error) {
	return ingest.ReadStdin()
}

// ResolveInput converts a raw input string into a local file path.
// URLs are downloaded to a temp file (which is cleaned up via the returned func).
// Stdin ("-") is returned as-is. Directories return an error.
func ResolveInput(raw string) (localPath string, cleanup func(), err error) {
	return ingest.ResolveInput(raw)
}

// CheckDeps verifies that required external tools (pandoc, pdftohtml) are installed.
func CheckDeps() error {
	return ingest.CheckDeps()
}

// ExpandInputs expands glob patterns, directories, and URLs into a flat list of file paths.
// Results are deduplicated and sorted.
func ExpandInputs(raw []string) ([]string, error) {
	return expand.Inputs(raw)
}
