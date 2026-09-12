package cmd

import (
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// visualsPathOverride can be set by tests to point loadVisualsConfig at a
// specific file. When empty the default "colony/ceremony/visuals.md" is used.
var visualsPathOverride string

var (
	loadedVisuals *visualsConfig
	visualsOnce   sync.Once
)

type visualsConfig struct {
	VisualsVersion  string              `yaml:"visuals_version"`
	CasteEmojiMap   map[string]string   `yaml:"caste_emoji_map"`
	CasteColorMap   map[string]string   `yaml:"caste_color_map"`
	CasteLabelMap   map[string]string   `yaml:"caste_label_map"`
	CommandEmojiMap map[string]string   `yaml:"command_emoji_map"`
	VoiceGlyphMap   map[string]string   `yaml:"voice_glyph_map"`
	CastePrefixes   map[string][]string `yaml:"caste_prefixes"`
	DefaultPrefixes []string            `yaml:"default_prefixes"`
	AetherWordmark  string              `yaml:"aether_wordmark"`
	VisualDivider   string              `yaml:"visual_divider"`
}

// loadVisualsConfig reads colony/ceremony/visuals.md once per process and
// caches the result.  If the file is missing, unreadable, or unparseable it
// returns nil so callers fall back to hardcoded defaults.
//
// colony/ceremony/visuals.md was deleted in Phase 191 (dead-wood criterion
// 2): it was a CWD-relative, dev-checkout-only loader, and every value it
// ever defined was already exactly what cmd/codex_visuals.go's compiled
// defaults produce -- see cmd/codex_visuals_test.go's
// TestVisualsConfigFoldedDefaultsMatchOriginalFile and
// TestVisualsConfigOriginalFileHadPreexistingParseDefect for the proof.
// This function is kept as a permanently-fallback path (D-04,
// 191-CONTEXT.md): visualsPathOverride still lets a future colony/ file, or
// a test, supply values again without any caller needing to change.
func loadVisualsConfig() *visualsConfig {
	visualsOnce.Do(func() {
		path := visualsPathOverride
		if path == "" {
			path = "colony/ceremony/visuals.md"
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return
		}
		front := extractVisualsYAMLFrontmatter(string(data))
		if front == "" {
			return
		}
		var cfg visualsConfig
		if err := yaml.Unmarshal([]byte(front), &cfg); err != nil {
			return
		}
		loadedVisuals = &cfg
	})
	return loadedVisuals
}

// resetVisualsCache forces the next call to loadVisualsConfig() to re-read
// the file.  Used by tests that need to switch between different visuals.md
// contents.
func resetVisualsCache() {
	loadedVisuals = nil
	visualsOnce = sync.Once{}
}

func extractVisualsYAMLFrontmatter(content string) string {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return ""
	}
	content = strings.TrimPrefix(content, "---")
	content = strings.TrimPrefix(content, "\n")
	content = strings.TrimPrefix(content, "\r\n")
	idx := strings.Index(content, "\n---")
	if idx == -1 {
		return ""
	}
	return strings.TrimSpace(content[:idx])
}

// fileCasteEmoji returns the emoji for a caste from the loaded config.
func fileCasteEmoji(caste string) (string, bool) {
	cfg := loadVisualsConfig()
	if cfg == nil {
		return "", false
	}
	v, ok := cfg.CasteEmojiMap[caste]
	return v, ok
}

// fileCasteColor returns the ANSI color code for a caste from the loaded config.
func fileCasteColor(caste string) (string, bool) {
	cfg := loadVisualsConfig()
	if cfg == nil {
		return "", false
	}
	v, ok := cfg.CasteColorMap[caste]
	return v, ok
}

// fileCasteLabel returns the human-readable label for a caste from the loaded config.
func fileCasteLabel(caste string) (string, bool) {
	cfg := loadVisualsConfig()
	if cfg == nil {
		return "", false
	}
	v, ok := cfg.CasteLabelMap[caste]
	return v, ok
}

// fileCommandEmoji returns the emoji for a command from the loaded config.
func fileCommandEmoji(command string) (string, bool) {
	cfg := loadVisualsConfig()
	if cfg == nil {
		return "", false
	}
	v, ok := cfg.CommandEmojiMap[command]
	return v, ok
}

// fileCastePrefixes returns the name-prefix list for a caste from the loaded config.
func fileCastePrefixes(caste string) ([]string, bool) {
	cfg := loadVisualsConfig()
	if cfg == nil {
		return nil, false
	}
	v, ok := cfg.CastePrefixes[caste]
	return v, ok
}

// fileDefaultPrefixes returns the fallback prefix list from the loaded config.
func fileDefaultPrefixes() ([]string, bool) {
	cfg := loadVisualsConfig()
	if cfg == nil || len(cfg.DefaultPrefixes) == 0 {
		return nil, false
	}
	return cfg.DefaultPrefixes, true
}

// fileAetherWordmark returns the ASCII wordmark from the loaded config.
func fileAetherWordmark() (string, bool) {
	cfg := loadVisualsConfig()
	if cfg == nil {
		return "", false
	}
	return cfg.AetherWordmark, true
}

// fileVisualDivider returns the divider string from the loaded config.
func fileVisualDivider() (string, bool) {
	cfg := loadVisualsConfig()
	if cfg == nil {
		return "", false
	}
	return cfg.VisualDivider, true
}
