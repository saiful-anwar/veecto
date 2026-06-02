package chunk

import (
	"fmt"

	"github.com/saiful-anwar/veecto/internal/core"
)

// Fixed splits text into chunks of a fixed character size with optional overlap.
type Fixed struct {
	Size           int
	Overlap        int
	AsciiNormalize bool
}

// Chunk implements core.Chunker.
func (c *Fixed) Chunk(text string) ([]core.Chunk, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}

	runes := []rune(text)
	total := len(runes)
	if total == 0 {
		return nil, fmt.Errorf("empty text")
	}

	step := c.Size - c.Overlap
	if step <= 0 {
		step = c.Size
	}

	var chunks []core.Chunk
	for pos := 0; pos < total; {
		end := pos + c.Size
		if end > total {
			end = total
		}

		chunkText := string(runes[pos:end])
		clean := cleanText(chunkText, c.AsciiNormalize)
		if clean != "" {
			chunks = append(chunks, core.Chunk{
				Index:      len(chunks),
				Text:       chunkText,
				TextClean:  clean,
				TokenCount: approxTokenCount(chunkText),
				CharStart:  pos,
				CharEnd:    end,
			})
		}
		pos += step
	}

	if len(chunks) == 0 {
		return nil, fmt.Errorf("no chunks produced")
	}
	return chunks, nil
}
