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
// Supports glob patterns, directories (recursively scanned for SupportedExts), URLs (passed through),
// and stdin ("-").
func Inputs(raw []string) ([]string, error) {
	return InputsFiltered(raw, "", "", 0)
}

// InputsFiltered expands raw CLI arguments with include/exclude glob patterns and max depth.
// A non-empty include pattern restricts to files matching the glob; a non-empty exclude glob
// filters out matching files. Depth 0 means unlimited.
func InputsFiltered(raw []string, includeGlob, excludeGlob string, maxDepth int) ([]string, error) {
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
			// Malformed glob pattern — treat as literal path and let os.Stat validate.
			matches = []string{r}
		}
		if len(matches) == 0 {
			// No glob match — try as literal path.
			matches = []string{r}
		}

		for _, m := range matches {
			if seen[m] {
				continue
			}
			seen[m] = true

			info, err := os.Stat(m)
			if err != nil {
				return nil, fmt.Errorf("access %s: %w", m, err)
			}
			if info.IsDir() {
				dirFiles, err := walkDir(m, maxDepth)
				if err != nil {
					return nil, err
				}
				for _, df := range dirFiles {
					if !seen[df] {
						seen[df] = true
						if matchesFilter(df, includeGlob, excludeGlob) {
							expanded = append(expanded, df)
						}
					}
				}
			} else {
				if matchesFilter(m, includeGlob, excludeGlob) {
					expanded = append(expanded, m)
				}
			}
		}
	}

	sort.Strings(expanded)
	return expanded, nil
}

// matchesFilter returns true if path matches includeGlob (if set) and does not
// match excludeGlob (if set). Empty patterns are treated as "match all".
func matchesFilter(path, includeGlob, excludeGlob string) bool {
	if includeGlob != "" {
		match, _ := filepath.Match(includeGlob, filepath.Base(path))
		if !match {
			match, _ = filepath.Match(includeGlob, path)
		}
		if !match {
			return false
		}
	}
	if excludeGlob != "" {
		match, _ := filepath.Match(excludeGlob, filepath.Base(path))
		if !match {
			match, _ = filepath.Match(excludeGlob, path)
		}
		if match {
			return false
		}
	}
	return true
}

// walkDir recursively lists supported files in dir, respecting maxDepth (0 = unlimited).
func walkDir(dir string, maxDepth int) ([]string, error) {
	var files []string
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("abs: %w", err)
	}
	baseDepth := strings.Count(absDir, string(filepath.Separator))

	err = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Skip hidden directories unless it's the root.
			if strings.HasPrefix(d.Name(), ".") && path != dir {
				return filepath.SkipDir
			}
			// Check depth.
			if maxDepth > 0 {
				depth := strings.Count(path, string(filepath.Separator)) - baseDepth
				if depth > maxDepth {
					return filepath.SkipDir
				}
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if SupportedExts[ext] && !strings.HasPrefix(d.Name(), ".") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", dir, err)
	}

	return files, nil
}
