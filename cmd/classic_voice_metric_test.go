package cmd

// Phase "Classic Visual Voice" plan 01 (tracer task) -- the reference
// density measurement.
//
// The reference figure this file derives is computed from six Classic
// display blocks extracted verbatim from commits 3a5b81c2 ("Open Chambers",
// 2026-02-15) and 794fadfb (2026-02-08), never typed in as a literal. The
// glyph set the measurement checks against is read from the production
// tables (casteEmojiMap, casteLabelMap via casteEmoji, and voiceGlyphMap) so
// a table edit cannot silently desync this test from what the runtime
// actually renders. See the classic-visual-voice phase RESEARCH document,
// section 2, and the "Reference Density Metric Proposal" section, for the
// full derivation this file implements.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"testing"
	"unicode"
)

// ---------------------------------------------------------------------------
// The closed glyph set -- read from the production tables, never re-typed.
// ---------------------------------------------------------------------------

// voiceGlyphSet builds the closed glyph set a rendered line's leading glyph
// must belong to for the line to count as "led": the union of casteEmojiMap
// (read through the casteEmoji accessor so an operator override is honoured
// exactly like production), and voiceGlyphMap (read through voiceGlyph for
// the same reason). No glyph literal is re-typed here -- every value comes
// from calling the production functions over the production tables' own key
// sets.
func voiceGlyphSet() map[string]bool {
	set := make(map[string]bool)
	for caste := range casteEmojiMap {
		if glyph := casteEmoji(caste); glyph != "" {
			set[glyph] = true
		}
	}
	for caste := range casteLabelMap {
		if glyph := casteEmoji(caste); glyph != "" {
			set[glyph] = true
		}
	}
	for kind := range voiceGlyphMap {
		if glyph := voiceGlyph(kind); glyph != "" {
			set[glyph] = true
		}
	}
	return set
}

// sortedGlyphsLongestFirst returns the glyph set as a slice ordered by byte
// length, descending, so a prefix match tries the longest candidate first --
// this is what makes a glyph carrying a variation selector (e.g. "👁️", which
// is the eye rune plus U+FE0F) match correctly instead of a shorter,
// unintentionally-overlapping candidate winning first.
func sortedGlyphsLongestFirst(set map[string]bool) []string {
	glyphs := make([]string, 0, len(set))
	for glyph := range set {
		glyphs = append(glyphs, glyph)
	}
	sort.Slice(glyphs, func(i, j int) bool {
		if len(glyphs[i]) != len(glyphs[j]) {
			return len(glyphs[i]) > len(glyphs[j])
		}
		return glyphs[i] < glyphs[j]
	})
	return glyphs
}

// ---------------------------------------------------------------------------
// Content-line extraction -- banner, rule, stage-marker, fence and
// ASCII-art lines are not content; everything else is.
// ---------------------------------------------------------------------------

var (
	voiceBannerLineRe      = regexp.MustCompile(`^━━ .+ ━━$`)
	voiceStageMarkerLineRe = regexp.MustCompile(`^── .+ ──$`)
	// voiceLeadingMarkerRe strips exactly one leading list-or-tree marker
	// (a dash, a bullet, a numbered prefix, or a box-drawing branch) before
	// the glyph check, so a glyph sitting behind a list marker still counts
	// as glyph-led.
	voiceLeadingMarkerRe = regexp.MustCompile(`^(?:[-•]\s+|(?:├──|└──)\s*|\d+[.)]\s+)`)
	// voiceRuleRuneSet is the set of runes a pure rule/divider line is made
	// of, once any flanking glyphs have been stripped.
	voiceRuleRuneSet = "━═─_-="
)

// stripFlankingVoiceGlyphs repeatedly removes a leading or trailing glyph
// token (from the closed glyph set, longest match first) and the whitespace
// beside it, so a rule line like "═══...═══ 🔨🐜🏗️🐜🔨" -- Classic's
// glyph-flanked banner rule -- reduces to its rule-rune core.
func stripFlankingVoiceGlyphs(line string, glyphsLongestFirst []string) string {
	for i := 0; i < 32; i++ {
		trimmed := strings.TrimSpace(line)
		changed := false
		for _, g := range glyphsLongestFirst {
			if strings.HasPrefix(trimmed, g) {
				trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, g))
				changed = true
			}
			if strings.HasSuffix(trimmed, g) {
				trimmed = strings.TrimSpace(strings.TrimSuffix(trimmed, g))
				changed = true
			}
		}
		line = trimmed
		if !changed {
			break
		}
	}
	return line
}

// isVoiceRuleLine reports whether line, once any flanking glyphs are
// stripped, is made only of horizontal-rule runes.
func isVoiceRuleLine(line string, glyphsLongestFirst []string) bool {
	core := stripFlankingVoiceGlyphs(line, glyphsLongestFirst)
	core = strings.ReplaceAll(core, " ", "")
	if core == "" {
		return false
	}
	for _, r := range core {
		if !strings.ContainsRune(voiceRuleRuneSet, r) {
			return false
		}
	}
	return true
}

// isVoiceSpacedTitleLine reports whether line is a letter-spaced ALL-CAPS
// banner title -- the Classic house style ("A N T   C O U N C I L",
// "P H A S E   {id}   C O M P L E T E") as well as the current renderBanner
// title text on its own line. Every whitespace-delimited field must be
// either exactly one rune or a `{placeholder}` token.
func isVoiceSpacedTitleLine(line string) bool {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return false
	}
	for _, f := range fields {
		if strings.HasPrefix(f, "{") && strings.HasSuffix(f, "}") {
			continue
		}
		if len([]rune(f)) == 1 {
			continue
		}
		return false
	}
	return true
}

// hasVoiceLetterOrDigit reports whether line contains at least one letter or
// digit rune -- the box-drawing wordmark and the anthill ASCII art contain
// neither, which is what excludes them without a bespoke "is this art"
// detector.
func hasVoiceLetterOrDigit(line string) bool {
	for _, r := range line {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

// voiceContentLines strips ANSI, drops blank lines, and drops every line
// that is a banner, a rule, a stage marker, a markdown fence, or has no
// letter or digit at all (ASCII art). Everything left is a content line.
func voiceContentLines(rendered string) []string {
	glyphs := sortedGlyphsLongestFirst(voiceGlyphSet())
	var content []string
	for _, raw := range strings.Split(stripANSI(rendered), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if voiceBannerLineRe.MatchString(line) {
			continue
		}
		if voiceStageMarkerLineRe.MatchString(line) {
			continue
		}
		if strings.HasPrefix(line, "```") {
			continue
		}
		if isVoiceSpacedTitleLine(line) {
			continue
		}
		if isVoiceRuleLine(line, glyphs) {
			continue
		}
		if !hasVoiceLetterOrDigit(line) {
			continue
		}
		content = append(content, line)
	}
	return content
}

// voiceLineIsLed reports whether line, after stripping leading whitespace and
// then one leading list-or-tree marker, begins with a glyph from the closed
// set -- tested longest-first so a glyph carrying a variation selector
// matches correctly.
func voiceLineIsLed(line string, glyphsLongestFirst []string) bool {
	line = strings.TrimLeft(line, " \t")
	line = voiceLeadingMarkerRe.ReplaceAllString(line, "")
	for _, g := range glyphsLongestFirst {
		if strings.HasPrefix(line, g) {
			return true
		}
	}
	return false
}

// voiceDensity returns the led-line count, the content-line count and the
// ratio for one rendered screen.
func voiceDensity(rendered string) (led, total int, ratio float64) {
	glyphs := sortedGlyphsLongestFirst(voiceGlyphSet())
	lines := voiceContentLines(rendered)
	total = len(lines)
	for _, line := range lines {
		if voiceLineIsLed(line, glyphs) {
			led++
		}
	}
	if total == 0 {
		return led, total, 0
	}
	return led, total, float64(led) / float64(total)
}

// ---------------------------------------------------------------------------
// The extracted reference fixture -- digest-tied to the reference commits.
// ---------------------------------------------------------------------------

type classicVoiceManifest struct {
	Blocks []classicVoiceBlock `json:"blocks"`
}

type classicVoiceBlock struct {
	Name        string `json:"name"`
	Commit      string `json:"commit"`
	Path        string `json:"path"`
	StartMarker string `json:"start_marker"`
	EndMarker   string `json:"end_marker"`
	SHA256      string `json:"sha256"`
}

const classicVoiceManifestPath = "testdata/classic-voice/manifest.json"

func loadClassicVoiceManifest(t *testing.T) classicVoiceManifest {
	t.Helper()
	data, err := os.ReadFile(classicVoiceManifestPath)
	if err != nil {
		t.Fatalf("read %s: %v", classicVoiceManifestPath, err)
	}
	var manifest classicVoiceManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse %s: %v", classicVoiceManifestPath, err)
	}
	return manifest
}

// extractClassicVoiceBlock re-derives a block's content directly from the
// named git object: the first line containing StartMarker through the first
// following line containing EndMarker, inclusive.
func extractClassicVoiceBlock(block classicVoiceBlock) (string, error) {
	out, err := exec.Command("git", "show", block.Commit+":"+block.Path).Output()
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(out), "\n")
	start := -1
	for i, line := range lines {
		if strings.Contains(line, block.StartMarker) {
			start = i
			break
		}
	}
	if start == -1 {
		return "", nil
	}
	end := -1
	for i := start; i < len(lines); i++ {
		if strings.Contains(lines[i], block.EndMarker) {
			end = i
			break
		}
	}
	if end == -1 {
		return "", nil
	}
	return strings.Join(lines[start:end+1], "\n") + "\n", nil
}

// classicVoiceBlockPath is the on-disk fixture file for one manifest block.
func classicVoiceBlockPath(name string) string {
	return "testdata/classic-voice/" + name + ".txt"
}

type classicVoiceReferenceBlock struct {
	Name     string
	Rendered string
}

// classicReferenceBlocks loads every fixture block named in the manifest
// from disk, failing the test rather than silently skipping when the
// registry is empty or a file is missing.
func classicReferenceBlocks(t *testing.T) []classicVoiceReferenceBlock {
	t.Helper()
	manifest := loadClassicVoiceManifest(t)
	if len(manifest.Blocks) == 0 {
		t.Fatalf("%s names no reference blocks at all", classicVoiceManifestPath)
	}
	blocks := make([]classicVoiceReferenceBlock, 0, len(manifest.Blocks))
	for _, b := range manifest.Blocks {
		data, err := os.ReadFile(classicVoiceBlockPath(b.Name))
		if err != nil {
			t.Fatalf("read reference block %q: %v", b.Name, err)
		}
		blocks = append(blocks, classicVoiceReferenceBlock{Name: b.Name, Rendered: string(data)})
	}
	return blocks
}

// classicReferenceDensityOverBlocks reduces a set of reference blocks to the
// one house figure: total led lines divided by total content lines, summed
// across every block, never averaged per-block-first (a block with many
// lines should weigh more than a block with few).
func classicReferenceDensityOverBlocks(blocks []classicVoiceReferenceBlock) float64 {
	var led, total int
	for _, b := range blocks {
		l, c, _ := voiceDensity(b.Rendered)
		led += l
		total += c
	}
	if total == 0 {
		return 0
	}
	return float64(led) / float64(total)
}

// classicReferenceDensity loads every fixture block and computes the one
// derived house figure the plan's success criteria hold every voiced screen
// to.
func classicReferenceDensity(t *testing.T) float64 {
	t.Helper()
	return classicReferenceDensityOverBlocks(classicReferenceBlocks(t))
}

// TestClassicReferenceFixtureMatchesTheReferenceCommit re-extracts every
// manifest row directly from its named git object and fails on digest
// mismatch, so the checked-in fixture cannot silently drift from the
// reference commit it claims to be. When the repository has no such object
// (a shallow clone, for instance) the row is skipped with an explicit
// message naming the missing object -- never silently passed.
func TestClassicReferenceFixtureMatchesTheReferenceCommit(t *testing.T) {
	manifest := loadClassicVoiceManifest(t)
	if len(manifest.Blocks) < 6 {
		t.Fatalf("manifest names %d blocks, want at least 6", len(manifest.Blocks))
	}
	sawCommit794fadfb := false
	for _, block := range manifest.Blocks {
		block := block
		t.Run(block.Name, func(t *testing.T) {
			if _, err := exec.Command("git", "cat-file", "-e", block.Commit).CombinedOutput(); err != nil {
				t.Skipf("git object %s is not present in this checkout -- cannot verify %s against it", block.Commit, block.Name)
			}
			if strings.HasPrefix(block.Commit, "794fadfb") {
				sawCommit794fadfb = true
			}
			extracted, err := extractClassicVoiceBlock(block)
			if err != nil {
				t.Fatalf("re-extract %s from %s:%s: %v", block.Name, block.Commit, block.Path, err)
			}
			if extracted == "" {
				t.Fatalf("re-extraction of %s found neither the start marker %q nor the end marker %q in %s:%s",
					block.Name, block.StartMarker, block.EndMarker, block.Commit, block.Path)
			}
			sum := sha256.Sum256([]byte(extracted))
			gotDigest := hex.EncodeToString(sum[:])
			if len(block.SHA256) != 64 {
				t.Fatalf("manifest row %q has a %d-character sha256, want 64", block.Name, len(block.SHA256))
			}
			if gotDigest != block.SHA256 {
				t.Errorf("%s has drifted from %s:%s -- re-extracted digest %s, manifest records %s",
					block.Name, block.Commit, block.Path, gotDigest, block.SHA256)
			}
			onDisk, err := os.ReadFile(classicVoiceBlockPath(block.Name))
			if err != nil {
				t.Fatalf("read checked-in fixture for %s: %v", block.Name, err)
			}
			if string(onDisk) != extracted {
				t.Errorf("checked-in fixture for %s does not match a fresh re-extraction from %s:%s",
					block.Name, block.Commit, block.Path)
			}
		})
	}
	if !sawCommit794fadfb {
		t.Errorf("manifest names no row from commit 794fadfb -- the plan requires at least one (the plan fixture)")
	}
}

// TestClassicReferenceDensityIsDerivedFromTheReferenceCommit asserts the
// figure is computed, bounded, contributed to by every named block, and
// genuinely sensitive to the fixture set -- never a typed-in literal.
func TestClassicReferenceDensityIsDerivedFromTheReferenceCommit(t *testing.T) {
	blocks := classicReferenceBlocks(t)
	if len(blocks) < 6 {
		t.Fatalf("only %d reference blocks contributed, want at least 6", len(blocks))
	}
	density := classicReferenceDensityOverBlocks(blocks)
	if density <= 0 || density >= 1 {
		t.Fatalf("classicReferenceDensity = %v, want a figure strictly between 0 and 1", density)
	}

	for i := range blocks {
		without := make([]classicVoiceReferenceBlock, 0, len(blocks)-1)
		without = append(without, blocks[:i]...)
		without = append(without, blocks[i+1:]...)
		if len(without) == 0 {
			continue
		}
		reduced := classicReferenceDensityOverBlocks(without)
		if reduced == density {
			t.Errorf("removing block %q left the reference figure unchanged at %v -- it is not deriving from every fixture file",
				blocks[i].Name, density)
		}
	}
}

// TestVoiceDensityCanFail is the guard on the guard: a planted flat screen
// must measure below the reference figure, and a planted decorated screen
// must measure at or above it -- proving the measurement can fail in both
// directions, not only pass.
func TestVoiceDensityCanFail(t *testing.T) {
	reference := classicReferenceDensity(t)

	flat := "Goal: Ship the billing rewrite\n" +
		"Phase: 2 of 4\n" +
		"Tasks: 3 of 10 complete\n" +
		"Status: in progress\n" +
		"State: EXECUTING\n"
	led, total, ratio := voiceDensity(flat)
	if total == 0 {
		t.Fatalf("the planted flat screen produced no content lines to measure")
	}
	if ratio >= reference {
		t.Errorf("the planted flat screen (led=%d total=%d ratio=%v) did not measure below the reference figure %v:\n%s",
			led, total, ratio, reference, flat)
	}

	decorated := voiceLine("goal", "Goal: Ship the billing rewrite") + "\n" +
		voiceLine("phase", "Phase: 2 of 4") + "\n" +
		voiceLine("task", "Tasks: 3 of 10 complete") + "\n" +
		voiceLine("status", "Status: in progress") + "\n" +
		voiceLine("done", "State: EXECUTING") + "\n"
	led, total, ratio = voiceDensity(decorated)
	if total == 0 {
		t.Fatalf("the planted decorated screen produced no content lines to measure")
	}
	if ratio < reference {
		t.Errorf("the planted decorated screen (led=%d total=%d ratio=%v) did not measure at or above the reference figure %v:\n%s",
			led, total, ratio, reference, decorated)
	}
}
