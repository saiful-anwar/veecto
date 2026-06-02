package chunk

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/saiful-anwar/veecto/internal/core"
)

// Sentence splits text into chunks that preserve sentence boundaries.
type Sentence struct {
	Size           int
	Overlap        int
	AsciiNormalize bool
}

// Chunk implements core.Chunker.
func (c *Sentence) Chunk(text string) ([]core.Chunk, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}

	sentences := splitSentences(text)
	if len(sentences) == 0 {
		return nil, fmt.Errorf("no sentences found")
	}

	return c.buildChunks(sentences), nil
}

// splitSentences splits text at sentence boundaries (. ! ? \n).
func splitSentences(text string) []string {
	var sentences []string
	var buf strings.Builder

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		buf.WriteRune(runes[i])

		if isSentenceEnd(runes[i]) {
			if i+1 >= len(runes) || unicode.IsUpper(runes[i+1]) || runes[i+1] == '\n' {
				sentences = append(sentences, buf.String())
				buf.Reset()
			}
		}
	}

	if buf.Len() > 0 {
		sentences = append(sentences, buf.String())
	}
	return sentences
}

func isSentenceEnd(r rune) bool {
	return r == '.' || r == '!' || r == '?' || r == '\n'
}

func (c *Sentence) buildChunks(sentences []string) []core.Chunk {
	var chunks []core.Chunk
	var buf strings.Builder
	start := 0
	charPos := 0

	for _, s := range sentences {
		if buf.Len() > 0 && len([]rune(buf.String()+s)) > c.Size {
			text := strings.TrimSpace(buf.String())
			if text != "" {
				clean := cleanText(text, c.AsciiNormalize)
				end := charPos
				chunks = append(chunks, core.Chunk{
					Index:      len(chunks),
					Text:       text,
					TextClean:  clean,
					TokenCount: approxTokenCount(text),
					CharStart:  start,
					CharEnd:    end,
				})
				start = charPos
			}
			buf.Reset()
		}
		buf.WriteString(s)
		charPos += len([]rune(s))
	}

	remainder := strings.TrimSpace(buf.String())
	if remainder != "" {
		clean := cleanText(remainder, c.AsciiNormalize)
		chunks = append(chunks, core.Chunk{
			Index:      len(chunks),
			Text:       remainder,
			TextClean:  clean,
			TokenCount: approxTokenCount(remainder),
			CharStart:  start,
			CharEnd:    charPos,
		})
	}

	if len(chunks) == 0 {
		return nil
	}
	return chunks
}
