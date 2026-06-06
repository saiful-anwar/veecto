package veecto

import (
	"context"

	"github.com/saiful-anwar/veecto/internal/expand"
	"github.com/saiful-anwar/veecto/internal/ingest"
)

// FileResult holds the text content and metadata for a single file read.
type FileResult = ingest.Result

// ReadFile reads a file at the given path. Supports .txt, .md (direct read),
// .pdf (via pdf2md-tui), .docx (via word2md).
func ReadFile(path string) (FileResult, error) {
	return ingest.ReadFile(path)
}

// ReadStdin reads all data from os.Stdin.
func ReadStdin() (FileResult, error) {
	return ingest.ReadStdin()
}

// ResolveInput converts a raw input string into a local file path.
// URLs are downloaded to a temp file (which is cleaned up via the returned func).
// Context is used for cancellation of URL downloads.
func ResolveInput(ctx context.Context, raw string) (localPath string, cleanup func(), err error) {
	return ingest.ResolveInput(ctx, raw)
}

// CheckDeps verifies that system dependencies are available.
// Since veecto uses pure Go libraries for document extraction,
// no external binaries are required.
func CheckDeps() error {
	return ingest.CheckDeps()
}

// ExpandInputs expands glob patterns, directories, and URLs into a flat list of file paths.
// Results are deduplicated and sorted.
func ExpandInputs(raw []string) ([]string, error) {
	return expand.Inputs(raw)
}

// ExpandInputsFiltered expands inputs with include/exclude glob patterns and max depth.
func ExpandInputsFiltered(raw []string, include, exclude string, maxDepth int) ([]string, error) {
	return expand.InputsFiltered(raw, include, exclude, maxDepth)
}
