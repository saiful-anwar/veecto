package chunk

import (
	"fmt"
	"strings"

	"github.com/saiful-anwar/veecto/internal/core"
)

// Markdown splits text into chunks by Markdown heading boundaries.
type Markdown struct {
	Size           int
	Overlap        int
	AsciiNormalize bool
}

// Chunk implements core.Chunker.
func (c *Markdown) Chunk(text string) ([]core.Chunk, error) {
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}

	sections := splitMarkdownSections(text)
	if len(sections) == 0 {
		return nil, fmt.Errorf("no markdown sections found")
	}

	return c.buildChunks(sections), nil
}

// isFenceStart returns true if the line starts a fenced code block (``` or ~~~).
func isFenceStart(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	// Must start with ``` or ~~~, optionally followed by a language identifier.
	first := true
	for _, r := range trimmed {
		if r == '`' || r == '~' {
			first = false
			continue
		}
		// Once we hit a non-backtick/tilde, if we've seen at least 3, it's a fence start.
		return !first
	}
	// Line is all backticks/tildes.
	return len(trimmed) >= 3
}

// isATXHeading checks if a line is an ATX heading (# heading).
func isATXHeading(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	hashCount := 0
	for _, r := range trimmed {
		if r == '#' {
			hashCount++
			if hashCount > 6 {
				return false
			}
			continue
		}
		if r == ' ' && hashCount > 0 {
			return true
		}
		return false
	}
	return false
}

// isSetextUnderline checks if a line is a Setext heading underline (=== or ---).
func isSetextUnderline(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	firstChar := trimmed[0]
	if firstChar != '=' && firstChar != '-' {
		return false
	}
	for _, r := range trimmed {
		if r != rune(firstChar) {
			return false
		}
	}
	return len(trimmed) >= 3
}

// splitMarkdownSections splits text at heading lines and paragraphs (after `` ` or ~~~ fences).
func splitMarkdownSections(text string) []string {
	var sections []string
	lines := strings.Split(text, "\n")
	var buf strings.Builder
	inFence := false
	prevLineWasText := false

	for _, line := range lines {
		// Toggle fenced code block state.
		if isFenceStart(line) {
			inFence = !inFence
		}

		if inFence {
			buf.WriteString(line)
			buf.WriteRune('\n')
			prevLineWasText = false
			continue
		}

		// Setext heading: previous line was text, this line is === or ---.
		if prevLineWasText && isSetextUnderline(line) {
			current := strings.TrimSuffix(buf.String(), "\n")
			buf.Reset()
			buf.WriteString(current)
			buf.WriteString(line)
			buf.WriteRune('\n')
			prevLineWasText = false
			continue
		}

		if isATXHeading(line) && buf.Len() > 0 {
			sections = append(sections, strings.TrimSpace(buf.String()))
			buf.Reset()
		}

		buf.WriteString(line)
		buf.WriteRune('\n')

		prevLineWasText = !isATXHeading(line) && !isSetextUnderline(line) && !isFenceStart(line) && strings.TrimSpace(line) != ""
	}

	if buf.Len() > 0 {
		sections = append(sections, strings.TrimSpace(buf.String()))
	}
	return sections
}

func (c *Markdown) buildChunks(sections []string) []core.Chunk {
	var chunks []core.Chunk
	var buf strings.Builder
	bufStart := 0
	charPos := 0

	for _, s := range sections {
		sLen := len([]rune(s)) + 1 // +1 for the appended \n
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
		buf.WriteRune('\n')
		charPos += len([]rune(s)) + 1
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
