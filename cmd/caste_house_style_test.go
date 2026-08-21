package cmd

import (
	"strings"
	"testing"
)

// TestCasteIdentityUsesHouseStyle pins the classic v5.4.0 double-emoji house
// style: every caste identity renders as "<glyph>🐜 <Label>" (e.g. 🔨🐜
// Builder), restored 2026-08-16 per the owner's richness-restoration decision
// (.planning/decisions/caste-emoji-house-style.md). The generic ant fallback
// stays a single 🐜 — 🐜🐜 was never the style.
func TestCasteIdentityUsesHouseStyle(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	for caste, emoji := range casteEmojiMap {
		identity := casteIdentity(caste)
		wantPrefix := emoji + "🐜 "
		if !strings.HasPrefix(identity, wantPrefix) {
			t.Errorf("casteIdentity(%q) = %q; want prefix %q (glyph + ant house style)", caste, identity, wantPrefix)
		}
	}

	unknown := casteIdentity("no-such-caste-xyz")
	if strings.Contains(unknown, "🐜🐜") {
		t.Errorf("fallback identity doubled the generic ant: %q", unknown)
	}
	if !strings.HasPrefix(unknown, "🐜 ") {
		t.Errorf("fallback identity = %q; want single generic ant prefix", unknown)
	}
}
