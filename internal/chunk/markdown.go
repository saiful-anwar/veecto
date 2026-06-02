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

// splitMarkdownSections splits text at heading lines (#, ##, etc.).
func splitMarkdownSections(text string) []string {
	var sections []string
	lines := strings.Split(text, "\n")
	var buf strings.Builder

	for _, line := range lines {
		if isHeading(line) && buf.Len() > 0 {
			sections = append(sections, strings.TrimSpace(buf.String()))
			buf.Reset()
		}
		buf.WriteString(line)
		buf.WriteRune('\n')
	}

	if buf.Len() > 0 {
		sections = append(sections, strings.TrimSpace(buf.String()))
	}
	return sections
}

func isHeading(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	for _, r := range trimmed {
		if r == '#' {
			continue
		}
		if r == ' ' {
			return true
		}
		return false
	}
	return false
}

func (c *Markdown) buildChunks(sections []string) []core.Chunk {
	var chunks []core.Chunk
	var buf strings.Builder
	start := 0
	charPos := 0

	for _, s := range sections {
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
		buf.WriteRune('\n')
		charPos += len([]rune(s)) + 1
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
