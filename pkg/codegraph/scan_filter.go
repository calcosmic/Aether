package codegraph

import (
	"path/filepath"
	"strings"
)

// noiseDirs is the canonical set of directories to skip during repo scanning.
// All directory-walking code in Aether MUST use ShouldSkipDir() instead of
// maintaining local skip lists.
//
// Order: version control, Python, Node.js, Go, Rust, build artifacts,
// caches, Aether-managed, IDE/editor, temp.
//
// NOTE: bare "env" is intentionally excluded -- repos commonly have env/
// source directories (Pitfall 8).
var noiseDirs = map[string]bool{
	// Version control
	".git": true,
	// Python virtual environments and caches
	".venv":        true,
	"venv":         true,
	"__pycache__":  true,
	".mypy_cache":  true,
	".pytest_cache": true,
	".tox":         true,
	".ruff_cache":  true,
	".pytype":      true,
	"site-packages": true,
	// Node.js
	"node_modules": true,
	".next":        true,
	".nuxt":        true,
	".svelte-kit":  true,
	// Go
	"vendor": true,
	// Rust
	"target":  true,
	".cargo":  true,
	// General build/artifact dirs
	"dist":     true,
	"build":    true,
	"out":      true,
	"bin":      true,
	"coverage": true,
	// Caches and registries
	".cache":      true,
	".terraform":  true,
	".gradle":     true,
	// Aether-managed directories
	".aether":   true,
	".claude":   true,
	".codex":    true,
	".opencode": true,
	// IDE/editor
	".idea":   true,
	".vscode": true,
	// Temp
	"tmp":  true,
	"temp": true,
}

// noiseFileExts lists file extensions for files that should be skipped.
var noiseFileExts = map[string]bool{
	".pyc":   true, ".pyo":   true,           // Python bytecode
	".so":    true, ".dylib": true, ".dll": true, // Compiled libraries
	".map":   true,                             // Source maps
}

// compoundNoiseSuffixes are multi-part extensions to skip (checked before single ext).
var compoundNoiseSuffixes = []string{
	".min.js", ".min.css", ".bundle.js", ".d.ts",
}

// ShouldSkipDir returns true if a directory should be excluded from scanning.
func ShouldSkipDir(name string) bool {
	return noiseDirs[name]
}

// ShouldSkipFile returns true if a file should be excluded based on extension.
func ShouldSkipFile(name string) bool {
	lower := strings.ToLower(name)
	for _, suffix := range compoundNoiseSuffixes {
		if strings.HasSuffix(lower, suffix) {
			return true
		}
	}
	ext := filepath.Ext(lower)
	return noiseFileExts[ext]
}

// NoiseDirCount returns the number of directories in the skip list.
// Used by tests to detect divergence.
func NoiseDirCount() int {
	return len(noiseDirs)
}
