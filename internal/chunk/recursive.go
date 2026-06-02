package chunk

import (
	"fmt"
	"strings"

	"github.com/saiful-anwar/veecto/internal/core"
)

// recSeperators is the priority-ordered list of split points for recursive chunking.
var recSeperators = []string{"\n\n", "\n", ". ", ", ", " "}

// Recursive splits text by trying increasingly granular separators
// (\n\n → \n → .  → ,  → space) until chunks fit within Size.
type Recursive struct {
	Size    int
	Overlap int
}

// Chunk implements core.Chunker.
func (c *Recursive) Chunk(text string) ([]core.Chunk, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}

	runes := []rune(text)
	if len(runes) == 0 {
		return nil, fmt.Errorf("empty text")
	}

	chunks := c.split(runes, 0, c.Size)
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks produced")
	}
	return chunks, nil
}

func (c *Recursive) split(runes []rune, start, targetSize int) []core.Chunk {
	if start >= len(runes) {
		return nil
	}

	end := start + targetSize
	if end > len(runes) {
		end = len(runes)
	}

	segment := string(runes[start:end])
	isLast := end >= len(runes)
	isSmall := len([]rune(segment)) <= c.Size/2

	if isLast || isSmall {
		clean := cleanText(segment)
		if clean == "" {
			return nil
		}
		return []core.Chunk{{
			Index:      0,
			Text:       segment,
			TextClean:  clean,
			TokenCount: approxTokenCount(segment),
			CharStart:  start,
			CharEnd:    end,
		}}
	}

	splitAt := findSplitBack(runes[start:end], recSeperators)
	if splitAt <= 0 {
		splitAt = end - start
	}

	actualEnd := start + splitAt
	chunkText := string(runes[start:actualEnd])
	clean := cleanText(chunkText)

	var result []core.Chunk
	if clean != "" {
		result = append(result, core.Chunk{
			Index:      0,
			Text:       chunkText,
			TextClean:  clean,
			TokenCount: approxTokenCount(chunkText),
			CharStart:  start,
			CharEnd:    actualEnd,
		})
	}

	overlapStart := actualEnd - c.Overlap
	if overlapStart < start {
		overlapStart = start
	}

	rest := c.split(runes, overlapStart, c.Size)
	for i := range rest {
		rest[i].Index = len(result) + i
		result = append(result, rest[i])
	}
	return result
}

// findSplitBack scans text right-to-left for one of the separators and returns the best split position.
func findSplitBack(runes []rune, seps []string) int {
	text := string(runes)
	best := 0
	for _, sep := range seps {
		if idx := strings.LastIndex(text, sep); idx > 0 {
			candidate := idx + len(sep)
			if candidate > best {
				best = candidate
			}
		}
	}
	return best
}
