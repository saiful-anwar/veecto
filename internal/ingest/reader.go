package ingest

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// extFormatMap maps file extensions to pandoc --from format identifiers.
var extFormatMap = map[string]string{
	".pdf":   "pdf",
	".html":  "html",
	".htm":   "html",
	".docx":  "docx",
	".epub":  "epub",
	".latex": "latex",
	".rst":   "rst",
	".org":   "org",
}

// CheckDeps verifies that pandoc and pdftohtml are installed and available on PATH.
func CheckDeps() error {
	var missing []string
	for _, name := range []string{"pandoc", "pdftohtml"} {
		if _, err := exec.LookPath(name); err != nil {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing dependencies: %s (brew install pandoc poppler)", strings.Join(missing, ", "))
	}
	return nil
}

func checkExec(name string) error {
	if _, err := exec.LookPath(name); err != nil {
		return fmt.Errorf("%s not found in $PATH", name)
	}
	return nil
}

// Result holds the extracted text content and metadata from a file read.
type Result struct {
	Text     string
	FileType string
	Size     int64
	Hash     string
}

// ReadFile reads a file and returns its text content and metadata. PDFs use
// pdftohtml → pandoc pipeline; other formats use pandoc directly.
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

	default:
		if fromFmt, ok := extFormatMap[ext]; ok {
			text, err := readViaPandoc(path, fromFmt)
			if err != nil {
				return Result{}, err
			}
			return Result{Text: text, FileType: fromFmt, Size: info.Size(), Hash: hash([]byte(text))}, nil
		}
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

// readViaPandoc converts a file to plain text using pandoc.
func readViaPandoc(path, fromFmt string) (string, error) {
	if err := checkExec("pandoc"); err != nil {
		return "", err
	}

	var buf, stderrBuf bytes.Buffer
	cmd := exec.Command("pandoc", path, "--from", fromFmt, "--to", "plain", "--wrap", "none")
	cmd.Stdout = &buf
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pandoc: %w\n%s", err, strings.TrimSpace(stderrBuf.String()))
	}
	return buf.String(), nil
}

// readPDF converts a PDF to plain text via pdftohtml → pandoc.
func readPDF(path string) (string, error) {
	if err := checkExec("pdftohtml"); err != nil {
		return "", err
	}
	if err := checkExec("pandoc"); err != nil {
		return "", err
	}

	tmpDir, err := os.MkdirTemp("", "veecto-pdf-*")
	if err != nil {
		return "", fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	htmlFile := filepath.Join(tmpDir, "output.html")
	var stderrBuf bytes.Buffer
	cmd := exec.Command("pdftohtml", "-c", "-noframes", path, htmlFile)
	cmd.Stderr = &stderrBuf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("pdftohtml: %w\n%s", err, strings.TrimSpace(stderrBuf.String()))
	}

	return readViaPandoc(htmlFile, "html")
}
