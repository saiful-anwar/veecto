package ingest

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nawodyaishan/pdf2md-tui/pkg/domain"
	"github.com/nawodyaishan/pdf2md-tui/pkg/repository/pdf"
	"github.com/nawodyaishan/pdf2md-tui/pkg/repository/storage"
	"github.com/nawodyaishan/pdf2md-tui/pkg/service"
	"github.com/saiful-anwar/word2md"
)

// CheckDeps verifies that system dependencies are available.
// Since veecto uses pure Go libraries for document extraction,
// no external binaries are required.
func CheckDeps() error {
	return nil
}

// Result holds the extracted text content and metadata from a file read.
type Result struct {
	Text     string
	FileType string
	Size     int64
	Hash     string
}

// ReadFile reads a file and returns its text content and metadata.
// PDFs use pdf2md-tui; DOCX files use word2md; txt and md are read directly.
func ReadFile(path string) (Result, error) {
	ext := strings.ToLower(filepath.Ext(path))

	info, err := os.Stat(path)
	if err != nil {
		return Result{}, fmt.Errorf("stat: %w", err)
	}

	switch ext {
	case ".txt":
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, fmt.Errorf("read: %w", err)
		}
		return Result{Text: string(data), FileType: "text", Size: info.Size(), Hash: hash(data)}, nil

	case ".md":
		data, err := os.ReadFile(path)
		if err != nil {
			return Result{}, fmt.Errorf("read: %w", err)
		}
		return Result{Text: string(data), FileType: "markdown", Size: info.Size(), Hash: hash(data)}, nil

	case ".pdf":
		text, err := readPDF(path)
		if err != nil {
			return Result{}, err
		}
		return Result{Text: text, FileType: "pdf", Size: info.Size(), Hash: hash([]byte(text))}, nil

	case ".docx":
		text, err := readDocx(path)
		if err != nil {
			return Result{}, err
		}
		return Result{Text: text, FileType: "docx", Size: info.Size(), Hash: hash([]byte(text))}, nil

	default:
		return Result{}, fmt.Errorf("unsupported file type: %s", ext)
	}
}

// ReadStdin reads all data from standard input.
func ReadStdin() (Result, error) {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return Result{}, fmt.Errorf("read stdin: %w", err)
	}
	return Result{Text: string(data), FileType: "text", Size: int64(len(data)), Hash: hash(data)}, nil
}

func hash(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))[:16]
}

// readPDF converts a PDF to plain text using pdf2md-tui.
func readPDF(path string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "veecto-pdf-*")
	if err != nil {
		return "", fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := domain.NewConfig()
	cfg.StripNoise = true
	cfg.ExtractImages = false

	parser := pdf.NewParser()
	store := storage.NewStorage()
	conv := service.NewConverterService(cfg, store, parser, nil)

	res := conv.Convert(path, tmpDir)
	switch res.Status {
	case domain.StatusError:
		return "", fmt.Errorf("pdf2md: %w", res.Err)
	case domain.StatusIgnored:
		if res.Err != nil {
			return "", fmt.Errorf("pdf skipped: %w", res.Err)
		}
		return "", fmt.Errorf("pdf skipped: scanned or image-only document")
	}

	md, err := os.ReadFile(res.OutputPath)
	if err != nil {
		return "", fmt.Errorf("read markdown output: %w", err)
	}

	return stripMarkdown(string(md)), nil
}

// readDocx converts a DOCX file to plain text using word2md.
func readDocx(path string) (string, error) {
	res, err := word2md.ConvertFile(context.Background(), path,
		word2md.WithHeadingDetection(true),
		word2md.WithListDetection(true),
		word2md.WithInlineFormatting(true),
		word2md.WithHyperlinks(true),
	)
	if err != nil {
		return "", fmt.Errorf("word2md: %w", err)
	}

	return stripMarkdown(res.Markdown), nil
}

var (
	reCodeFence     = regexp.MustCompile("```[^`]*```")
	reInlineCode    = regexp.MustCompile("`[^`]+`")
	reImage         = regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`)
	reLink          = regexp.MustCompile(`\[([^\]]*)\]\([^)]+\)`)
	reHeader        = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	reBold          = regexp.MustCompile(`\*\*(.*?)\*\*`)
	reItalic        = regexp.MustCompile(`\*(.*?)\*`)
	reUndBold       = regexp.MustCompile(`__(.*?)__`)
	reUndItalic     = regexp.MustCompile(`_(.*?)_`)
	reTableRow      = regexp.MustCompile(`(?m)^\|.*\|$`)
	reTableSep      = regexp.MustCompile(`(?m)^\|[\s\-:|]+\|$`)
	reHR            = regexp.MustCompile(`(?m)^[-*_]{3,}\s*$`)
	reBlockquote    = regexp.MustCompile(`(?m)^>\s?`)
	reUnorderedList = regexp.MustCompile(`(?m)^[\s]*[-*+]\s+`)
	reOrderedList   = regexp.MustCompile(`(?m)^[\s]*\d+\.\s+`)
	reBlankLines    = regexp.MustCompile(`\n{3,}`)
)

// stripMarkdown removes Markdown formatting, returning plain text.
func stripMarkdown(md string) string {
	md = reCodeFence.ReplaceAllString(md, "")
	md = reInlineCode.ReplaceAllString(md, "")
	md = reImage.ReplaceAllString(md, "$1")
	md = reLink.ReplaceAllString(md, "$1")
	md = reHeader.ReplaceAllString(md, "")
	md = reBold.ReplaceAllString(md, "$1")
	md = reItalic.ReplaceAllString(md, "$1")
	md = reUndBold.ReplaceAllString(md, "$1")
	md = reUndItalic.ReplaceAllString(md, "$1")
	md = reTableRow.ReplaceAllString(md, "")
	md = reTableSep.ReplaceAllString(md, "")
	md = reHR.ReplaceAllString(md, "")
	md = reBlockquote.ReplaceAllString(md, "")
	md = reUnorderedList.ReplaceAllString(md, "")
	md = reOrderedList.ReplaceAllString(md, "")
	md = reBlankLines.ReplaceAllString(md, "\n\n")
	return strings.TrimSpace(md)
}
