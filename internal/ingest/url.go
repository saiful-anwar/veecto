package ingest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxDownloadSize = 500 * 1024 * 1024

// downloadClient is the HTTP client used for URL downloads.
var downloadClient = &http.Client{Timeout: 5 * time.Minute}

// ResolveInput resolves a raw input string to a local file path.
// URLs are downloaded to a temp file (cleanup func removes it).
// Stdin ("-") is returned as-is. Directories return an error.
func ResolveInput(raw string) (localPath string, cleanup func(), err error) {
	if raw == "-" {
		return raw, func() {}, nil
	}

	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		path, err := fetchURL(raw)
		if err != nil {
			return "", nil, err
		}
		return path, func() { os.Remove(path) }, nil
	}

	info, err := os.Stat(raw)
	if err != nil {
		return "", nil, fmt.Errorf("input not found: %s", raw)
	}
	if info.IsDir() {
		return "", nil, fmt.Errorf("unexpected directory: %s (use a file path or glob pattern)", raw)
	}
	return raw, func() {}, nil
}

// fetchURL downloads a URL to a temp file and returns its path.
func fetchURL(rawURL string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("request: %w", err)
	}

	resp, err := downloadClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d downloading %s", resp.StatusCode, rawURL)
	}

	ext := detectExt(rawURL, resp.Header.Get("Content-Type"))
	if ext == "" {
		return "", fmt.Errorf("unknown content type: %s", resp.Header.Get("Content-Type"))
	}

	tmp, err := os.CreateTemp("", "veecto-url-*"+ext)
	if err != nil {
		return "", fmt.Errorf("temp file: %w", err)
	}

	src := io.LimitReader(resp.Body, maxDownloadSize)
	_, err = io.Copy(tmp, src)
	tmp.Close()
	if err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("save download: %w", err)
	}

	return tmp.Name(), nil
}

// detectExt determines the file extension from a URL path or Content-Type header.
func detectExt(rawURL, contentType string) string {
	parsed, err := url.Parse(rawURL)
	if err == nil {
		if ext := filepath.Ext(parsed.Path); ext != "" {
			return strings.ToLower(ext)
		}
	}

	switch {
	case strings.Contains(contentType, "application/pdf"):
		return ".pdf"
	case strings.Contains(contentType, "text/html"):
		return ".html"
	case strings.Contains(contentType, "text/plain"):
		return ".txt"
	case strings.Contains(contentType, "text/markdown"):
		return ".md"
	case strings.Contains(contentType, "application/epub"):
		return ".epub"
	default:
		return ""
	}
}
