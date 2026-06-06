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

func isSentenceEnd(r rune) bool {
	return r == '.' || r == '!' || r == '?' || r == '。' || r == '！' || r == '？'
}

func isNewline(r rune) bool { return r == '\n' || r == '\r' }

// nextNonSpace returns the first non-space (non-newline) character after pos,
// and its index. Returns (0, len) when none is found.
func nextNonSpace(runes []rune, pos int) (rune, int) {
	for i := pos; i < len(runes); i++ {
		if !unicode.IsSpace(runes[i]) {
			return runes[i], i
		}
	}
	return 0, len(runes)
}

// splitSentences splits text at sentence boundaries.
//
// Rules:
//   - . ! ? 。！？ followed by an uppercase letter or digit → boundary.
//   - \n\n (one or more blank lines) → always a boundary.
//   - \n (single) followed by an uppercase letter → boundary.
//
// Terminating punctuation and trailing whitespace are included in the sentence.
func splitSentences(text string) []string {
	runes := []rune(text)
	if len(runes) == 0 {
		return nil
	}

	var sentences []string
	start := 0

	for i := 0; i < len(runes); i++ {
		r := runes[i]

		if isSentenceEnd(r) {
			// Special case: decimal numbers like "1.2" — digit before and after.
			if r == '.' && i > 0 && unicode.IsDigit(runes[i-1]) {
				nextRune, nextIdx := nextNonSpace(runes, i+1)
				if nextIdx < len(runes) && unicode.IsDigit(nextRune) {
					continue // decimal number, not a sentence boundary
				}
			}
			// Look ahead past whitespace for the next non-space character.
			nextRune, nextIdx := nextNonSpace(runes, i+1)
			// Always a boundary if we're at the end of text.
			if nextIdx >= len(runes) {
				sentences = append(sentences, string(runes[start:]))
				start = len(runes)
				break
			}
			// Only split if the next word starts with an uppercase letter or digit,
			// or if there's no whitespace and the next char is uppercase.
			if unicode.IsUpper(nextRune) || unicode.IsDigit(nextRune) ||
				(nextIdx == i+1 && unicode.IsUpper(nextRune)) {
				// Consume trailing whitespace but keep newlines.
				end := i + 1
				for end < len(runes) && (runes[end] == ' ' || runes[end] == '\t') {
					end++
				}
				sentences = append(sentences, string(runes[start:end]))
				start = end
				i = end - 1
			}
			continue
		}

		if isNewline(r) {
			// \n\n is always a boundary (paragraph break).
			nlCount := 0
			j := i
			for j < len(runes) && isNewline(runes[j]) {
				nlCount++
				j++
			}
			if nlCount >= 2 {
				// Paragraph break.
				end := j
				// Consume trailing horizontal whitespace.
				for end < len(runes) && (runes[end] == ' ' || runes[end] == '\t') {
					end++
				}
				sentences = append(sentences, string(runes[start:end]))
				start = end
				i = end - 1
				continue
			}
			// Single newline — check if next non-space is uppercase.
			nextRune, nextIdx := nextNonSpace(runes, j)
			if nextIdx >= len(runes) {
				sentences = append(sentences, string(runes[start:]))
				start = len(runes)
				break
			}
			if unicode.IsUpper(nextRune) {
				end := j
				for end < len(runes) && (runes[end] == ' ' || runes[end] == '\t') {
					end++
				}
				sentences = append(sentences, string(runes[start:end]))
				start = end
				i = end - 1
			}
		}
	}

	if start < len(runes) {
		sentences = append(sentences, string(runes[start:]))
	}

	return sentences
}

func (c *Sentence) buildChunks(sentences []string) []core.Chunk {
	var chunks []core.Chunk
	var buf strings.Builder
	bufStart := 0
	charPos := 0

	for _, s := range sentences {
		sLen := len([]rune(s))
		bufLen := len([]rune(buf.String()))

		if bufLen > 0 && bufLen+sLen > c.Size {
			text, cs := trimmedStart(buf.String(), bufStart)
			if text != "" {
				clean := cleanText(text, c.AsciiNormalize)
				chunks = append(chunks, core.Chunk{
					Index:      len(chunks),
					Text:       text,
					TextClean:  clean,
					TokenCount: approxTokenCount(text),
					CharStart:  cs,
					CharEnd:    charPos,
				})
				bufStart = charPos
			}
			buf.Reset()
		}
		buf.WriteString(s)
		charPos += sLen
	}

	remainder, cs := trimmedStart(buf.String(), bufStart)
	if remainder != "" {
		clean := cleanText(remainder, c.AsciiNormalize)
		chunks = append(chunks, core.Chunk{
			Index:      len(chunks),
			Text:       remainder,
			TextClean:  clean,
			TokenCount: approxTokenCount(remainder),
			CharStart:  cs,
			CharEnd:    charPos,
		})
	}

	if len(chunks) == 0 {
		return nil
	}
	return chunks
}

// trimmedStart trims whitespace and returns the adjusted char start.
func trimmedStart(s string, start int) (string, int) {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) == 0 {
		return "", 0
	}
	lead := len([]rune(s)) - len([]rune(strings.TrimLeft(s, " \t\n\r\u00a0")))
	return trimmed, start + lead
}
