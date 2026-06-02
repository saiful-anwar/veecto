package expand

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SupportedExts maps file extensions that are accepted during directory scanning.
var SupportedExts = map[string]bool{
	".txt": true, ".md": true, ".pdf": true,
	".html": true, ".htm": true,
	".docx": true, ".epub": true,
	".latex": true, ".rst": true, ".org": true,
}

// Inputs expands raw CLI arguments into a flat, deduplicated, sorted list of file paths.
// Supports glob patterns, directories (scanned for SupportedExts), URLs (passed through),
// and stdin ("-"). Non-glob file paths are validated to exist.
func Inputs(raw []string) ([]string, error) {
	var expanded []string
	seen := make(map[string]bool)

	for _, r := range raw {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		if r == "-" {
			expanded = append(expanded, "-")
			continue
		}
		if strings.HasPrefix(r, "http://") || strings.HasPrefix(r, "https://") {
			if !seen[r] {
				expanded = append(expanded, r)
				seen[r] = true
			}
			continue
		}

		matches, err := filepath.Glob(r)
		if err != nil {
			return nil, fmt.Errorf("glob %s: %w", r, err)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("no files match: %s", r)
		}

		for _, m := range matches {
			if seen[m] {
				continue
			}
			seen[m] = true

			info, err := os.Stat(m)
			if err != nil {
				return nil, fmt.Errorf("stat %s: %w", m, err)
			}
			if info.IsDir() {
				dirFiles, err := listDir(m)
				if err != nil {
					return nil, err
				}
				for _, df := range dirFiles {
					if !seen[df] {
						expanded = append(expanded, df)
						seen[df] = true
					}
				}
			} else {
				expanded = append(expanded, m)
			}
		}
	}

	sort.Strings(expanded)
	return expanded, nil
}

// listDir returns all supported files in a directory (non-recursive).
func listDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if SupportedExts[ext] {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no supported files in directory: %s", dir)
	}
	return files, nil
}
