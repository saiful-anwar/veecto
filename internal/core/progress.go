package core

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var progressMu sync.Mutex

// DefaultProgressFn returns a ProgressFunc that prints to stderr:
// "! <input>: <err>" on error, "+ <input> (<type>): <n> chunks in <dur>" on success.
func DefaultProgressFn() ProgressFunc {
	return func(p Progress) {
		progressMu.Lock()
		defer progressMu.Unlock()
		if p.Error != nil {
			fmt.Fprintf(os.Stderr, "! %s: %v\n", p.Input, p.Error)
			return
		}
		fmt.Fprintf(os.Stderr, "+ %s (%s): %d chunks in %s\n",
			p.Input, p.FileType, p.ChunkCount, p.Duration.Round(time.Millisecond))
	}
}

// SilentProgressFn returns a no-op ProgressFunc.
func SilentProgressFn() ProgressFunc {
	return func(Progress) {}
}
