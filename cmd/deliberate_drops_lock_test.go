package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Phase 198 plan 09 (SHOW-05, roadmap criterion 5) -- the three display
// choices the team deliberately left out of the classic v5.4.0 experience
// stay out, and the one line they deliberately kept stays a statement, not a
// question. See 198-CONTEXT.md's "Carried forward" section and
// 198-RESEARCH.md Q10 (the confirmed-clean scan this file extends into a
// permanent lock rather than a one-time grep).

// deliberateDropsScanTargets returns every shipped Go source file (excluding
// _test.go, which is where this scan's own tool-name/phrase strings live)
// plus every wrapper markdown file under .claude/commands and
// .opencode/commands -- the two surfaces SHOW-05's drops could quietly
// return through.
func deliberateDropsScanTargets(t *testing.T) []string {
	t.Helper()
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	var files []string

	goFiles, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob cmd/*.go: %v", err)
	}
	for _, f := range goFiles {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		files = append(files, f)
	}

	for _, dir := range []string{
		filepath.Join(root, ".claude", "commands"),
		filepath.Join(root, ".opencode", "commands"),
	} {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if strings.HasSuffix(path, ".md") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}

	return files
}

// deliberateDropsMultiplexerPhrases are the tool name and phrase forms the
// first drop (no tmux / no second-terminal-window dependency, SEE-03) scans
// for. Built here, inside the test file the scan itself excludes, so the
// scan can never trip on its own vocabulary.
func deliberateDropsMultiplexerPhrases() []string {
	tmux := "tm" + "ux"
	return []string{
		tmux,
		"terminal multiplexer",
		"second terminal window",
		"another terminal window",
		"new terminal window",
		"separate terminal window",
		"split pane",
	}
}

// deliberateDropsForbiddenBoxCorners are the box-drawing corner code points
// (light, double, and heavy) that together close a bordered rectangle --
// every one of them EXCEPT light "└" (U+2514), which is explicitly permitted
// below: it is the nested-detail marker SEE-12's house style requires
// ("└── detail"), a single corner glyph used as a list-item prefix, never
// paired with the other three corners to enclose a section.
func deliberateDropsForbiddenBoxCorners() map[rune]string {
	return map[rune]string{
		'┌': "┌ light down-and-right corner",
		'┐': "┐ light down-and-left corner",
		'┘': "┘ light up-and-left corner",
		'╔': "╔ double down-and-right corner",
		'╗': "╗ double down-and-left corner",
		'╚': "╚ double up-and-right corner",
		'╝': "╝ double up-and-left corner",
		'┏': "┏ heavy down-and-right corner",
		'┓': "┓ heavy down-and-left corner",
		'┗': "┗ heavy up-and-right corner",
		'┛': "┛ heavy up-and-left corner",
	}
}

// deliberateDropsBoxCornerFileAllowlist is the shrink-only exception list for
// the box-corner scan, mirroring TestHumanDisplaysUseHeadedSectionsNotMachineTables's
// established file-level allowlist pattern (cmd/display_house_style_test.go)
// rather than inventing a second convention.
func deliberateDropsBoxCornerFileAllowlist() map[string]string {
	return map[string]string{
		"codex_visuals.go": "aetherWordmark / renderAetherWordmark: a decorative ASCII-art splash logo spelling out block letters with box-drawing glyphs -- not a bordered frame around a section of content, and unrelated to the SEE-checkpoint-style boxed table SHOW-05 retires.",
	}
}

// TestDeliberatelyDroppedDisplayChoicesStayDropped locks two of SHOW-05's
// three deliberate drops: no terminal-multiplexer/second-terminal-window
// dependency (SEE-03), and no boxed/bordered section frame (SEE-12) other
// than the explicitly-kept nested-detail marker.
func TestDeliberatelyDroppedDisplayChoicesStayDropped(t *testing.T) {
	files := deliberateDropsScanTargets(t)
	phrases := deliberateDropsMultiplexerPhrases()
	forbiddenCorners := deliberateDropsForbiddenBoxCorners()
	cornerAllowlist := deliberateDropsBoxCornerFileAllowlist()

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		lower := strings.ToLower(string(data))
		for _, phrase := range phrases {
			if strings.Contains(lower, phrase) {
				t.Errorf("%s references %q -- SEE-03 retired the tmux/second-terminal-window watch experience; live progress goes through emitVisualProgress/writeVisualOutput in the one terminal the owner is already in", file, phrase)
			}
		}

		if reason, ok := cornerAllowlist[filepath.Base(file)]; ok {
			_ = reason // documented exception, not scanned for box corners
			continue
		}
		for _, r := range string(data) {
			if name, forbidden := forbiddenCorners[r]; forbidden {
				t.Errorf("%s contains %s (%U) -- a boxed/bordered section frame, which SHOW-05 retires. The nested-detail marker \"└──\" (U+2514, light up-and-right) is the only box-drawing corner this house style permits; that is a different character from the one found here.", file, name, r)
			}
		}
	}
}

// TestSafeToClearLineStaysAStatement locks SHOW-05's third drop: the retired
// "clear context now?" question stays retired, and the kept
// "safe to clear your context now" line remains a statement, never a
// question. Extends (does not duplicate) TestHandoffConfirmedBeforeClearGuidance
// (cmd/display_house_style_test.go), which only checks presence/absence of
// the "Handoff saved" claim, not question-vs-statement form or a
// cross-file scan for the retired phrasing.
func TestSafeToClearLineStaysAStatement(t *testing.T) {
	saveGlobals(t)
	tmpDir := t.TempDir()
	t.Setenv("AETHER_ROOT", tmpDir)
	t.Setenv("AETHER_PLATFORM", "")
	store = nil
	t.Setenv("COLONY_DATA_DIR", "")

	platforms := []string{"claude", "codex", "opencode"}

	// Branch 1: no handoff confirmed on disk -- must never claim safety, and
	// its own clearing sentence must never be a question (it is an
	// instruction: run this first). The guidance appends a further sentence
	// after it ("Run `aether status` first."), so the check is against the
	// clearing sentence's own terminator, not the whole return value's
	// trailing character.
	var withoutHandoff string
	for _, platform := range platforms {
		out := renderContextClearGuidanceForPlatform(platform)
		if strings.Contains(out, "safe to clear") {
			t.Errorf("platform %q: no-handoff guidance falsely claims safety: %q", platform, out)
		}
		if sentence := deliberateDropsClearingSentence(t, out); strings.HasSuffix(sentence, "?") {
			t.Errorf("platform %q: no-handoff guidance's clearing sentence is a question, not a statement: %q", platform, sentence)
		}
		withoutHandoff = out
	}
	if strings.TrimSpace(withoutHandoff) == "" {
		t.Fatalf("renderContextClearGuidanceForPlatform returned nothing for the no-handoff branch")
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, ".aether"), 0755); err != nil {
		t.Fatalf("mkdir .aether: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".aether", "HANDOFF.md"), []byte("# handoff"), 0644); err != nil {
		t.Fatalf("write handoff: %v", err)
	}

	// Branch 2: handoff confirmed on disk -- the KEPT statement, asserted on
	// the function's real return value, not a text search elsewhere. Same
	// sentence-boundary reasoning: a further sentence
	// ("Run `aether resume` to restore.") follows the clearing claim.
	var withHandoff string
	for _, platform := range platforms {
		out := renderContextClearGuidanceForPlatform(platform)
		if !strings.Contains(out, "safe to clear") {
			t.Fatalf("platform %q: handoff-exists guidance does not confirm safety: %q", platform, out)
		}
		if sentence := deliberateDropsClearingSentence(t, out); strings.HasSuffix(sentence, "?") {
			t.Errorf("platform %q: the kept \"safe to clear\" line has regressed into a question: %q", platform, sentence)
		}
		withHandoff = out
	}

	clearingWording := deliberateDropsDeriveClearingWording(t, withHandoff)

	// Now scan every source and wrapper file for the SAME wording, derived
	// above from the function's own return value rather than hand-typed
	// here, so a future reword of the kept line cannot make this scan go
	// stale: no line carrying that wording may end in a question mark,
	// anywhere in the scanned sources or wrapper prose.
	lowerWording := strings.ToLower(clearingWording)
	for _, file := range deliberateDropsScanTargets(t) {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		for i, line := range strings.Split(string(data), "\n") {
			if !strings.Contains(strings.ToLower(line), lowerWording) {
				continue
			}
			if strings.HasSuffix(strings.TrimSpace(line), "?") {
				t.Errorf("%s:%d carries the clearing wording %q as a question, not a statement: %s", file, i+1, clearingWording, strings.TrimSpace(line))
			}
		}
	}
}

// deliberateDropsClearingSentence extracts the single sentence containing
// "clear" from a real renderContextClearGuidanceForPlatform return value.
// Both branches append a further sentence after their clearing claim ("Run
// `aether status` first." / "Run `aether resume` to restore."), so checking
// the WHOLE return value's trailing character would miss a clearing sentence
// that itself regressed into a question but happened to have more text
// appended after it -- the sentence boundary (the next '.', '?', or the
// start/end of the string) is what a "statement, not a question" claim must
// be checked against.
func deliberateDropsClearingSentence(t *testing.T, guidance string) string {
	t.Helper()
	lower := strings.ToLower(guidance)
	idx := strings.Index(lower, "clear")
	if idx == -1 {
		t.Fatalf("could not find a clearing sentence: %q contains no \"clear\"", guidance)
	}
	start := idx
	for start > 0 {
		c := guidance[start-1]
		if c == '.' || c == '?' || c == '\n' {
			break
		}
		start--
	}
	end := idx
	for end < len(guidance) {
		c := guidance[end]
		end++
		if c == '.' || c == '?' {
			break
		}
	}
	return strings.TrimSpace(guidance[start:end])
}

// deliberateDropsDeriveClearingWording extracts the "clear ... context" core
// phrase from a real return value of renderContextClearGuidanceForPlatform,
// rather than a literal typed into this test -- so a future reword of the
// kept line updates what this scan looks for automatically instead of
// silently going stale.
func deliberateDropsDeriveClearingWording(t *testing.T, guidance string) string {
	t.Helper()
	lower := strings.ToLower(guidance)
	clearIdx := strings.Index(lower, "clear")
	if clearIdx == -1 {
		t.Fatalf("could not derive the clearing wording: %q contains no \"clear\"", guidance)
	}
	rest := lower[clearIdx:]
	contextIdx := strings.Index(rest, "context")
	if contextIdx == -1 {
		t.Fatalf("could not derive the clearing wording: no \"context\" found after \"clear\" in %q", guidance)
	}
	end := clearIdx + contextIdx + len("context")
	phrase := strings.TrimSpace(guidance[clearIdx:end])
	if phrase == "" {
		t.Fatalf("derived an empty clearing phrase from %q", guidance)
	}
	return phrase
}
