package chunk

import (
	"math"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// cleanText normalizes whitespace, strips control characters (except \n \t \r),
// optionally NFKD-decomposes and strips combining marks + non-ASCII,
// and trims leading/trailing whitespace.
func cleanText(text string, asciiNormalize bool) string {
	if asciiNormalize {
		t := transform.Chain(norm.NFKD, runes.Remove(runes.In(unicode.Mn)))
		result, _, _ := transform.String(t, text)
		text = result
	}

	var b strings.Builder
	b.Grow(len(text))
	prevSpace := false

	for _, r := range text {
		if r == '\n' || r == '\t' || r == '\r' {
			b.WriteRune(r)
			prevSpace = false
			continue
		}
		if unicode.IsControl(r) {
			continue
		}
		if asciiNormalize && r > 127 {
			continue
		}
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}

	return strings.TrimSpace(b.String())
}

// approxTokenCount estimates token count as ceil(len(runes) / 4).
func approxTokenCount(text string) int {
	return int(math.Ceil(float64(len([]rune(text))) / 4.0))
}
