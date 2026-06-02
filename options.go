package veecto

import "github.com/saiful-anwar/veecto/internal/core"

// DefaultConfig returns a Config with sensible defaults (recursive chunking, OpenAI provider, etc.).
func DefaultConfig() Config { return core.DefaultConfig() }

// LoadConfig reads a YAML config file, resolves ${ENV_VAR} placeholders, applies defaults,
// and validates the result.
func LoadConfig(path string) (Config, error) { return core.LoadConfig(path) }

// FindConfig searches for a config file at the given path, or auto-discovers it from
// ./ → ~/.config/veecto/ → /etc/veecto/. Returns empty string if none is found (no error).
func FindConfig(path string) (string, error) { return core.FindConfig(path) }

// DefaultProgressFn returns a ProgressFunc that prints progress to stderr.
// On error it prints "! <input>: <err>"; on success "+ <input> (<type>): <n> chunks in <dur>".
func DefaultProgressFn() ProgressFunc { return core.DefaultProgressFn() }

// SilentProgressFn returns a no-op ProgressFunc.
func SilentProgressFn() ProgressFunc { return core.SilentProgressFn() }
