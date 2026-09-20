package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/learn"
)

// These tests prove the consolidation pipeline's promotion output is
// reachable end-to-end: pipeline -> .aether/QUEEN.md -> readQUEENMd ->
// worker prompt section. Before this plan, PromoteInstinct wrote to
// .aether/data/QUEEN.md (a file colony-prime never reads) under a section
// (Instincts) that readQUEENMd never ingested. Both links are proven here,
// not merely asserted.

// TestConsolidationQueenPathIsTheFileColonyPrimeReads proves
// consolidationQueenPath() resolves to the same absolute path as
// localQueenPath() -- the file colony-prime actually reads -- and not a
// path under store.BasePath() (the store-relative dead file).
func TestConsolidationQueenPathIsTheFileColonyPrimeReads(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	store = s

	got := consolidationQueenPath()
	want := localQueenPath()
	if got != want {
		t.Fatalf("consolidationQueenPath() = %q, want %q (localQueenPath())", got, want)
	}

	expected := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	if got != expected {
		t.Fatalf("consolidationQueenPath() = %q, want %q", got, expected)
	}

	if strings.HasPrefix(got, s.BasePath()) {
		t.Fatalf("consolidationQueenPath() = %q must not live under store.BasePath() %q", got, s.BasePath())
	}
}

// legacyQueenDefaultContentFixture mirrors the pre-plan-162 queenDefaultContent
// shape: Wisdom, Patterns, Philosophies, Anti-Patterns, User Preferences, and
// Colony Charter -- with no Instincts section. It is hardcoded here rather
// than reusing the current queenDefaultContent constant because Task 1 also
// adds "## Instincts" to that constant (for newly created colonies); this
// fixture represents a repo whose local QUEEN.md predates that change and
// still needs the self-heal.
const legacyQueenDefaultContentFixture = `# QUEEN.md — Colony Wisdom Hub

## Wisdom
> Patterns and insights earned through colony work.

## Patterns
> Recurring solutions that worked.

## Philosophies
> Higher-level principles guiding decisions.

## Anti-Patterns
> Things to avoid.

## User Preferences
> Communication style and decision patterns.

## Colony Charter
> Colony name and goal.
`

// TestEnsureQueenInstinctsSectionAddsHeaderOnce proves ensureQueenInstinctsSection
// adds exactly one "## Instincts" header to a legacy-format local QUEEN.md,
// preserves every pre-existing line byte-for-byte, and is idempotent.
func TestEnsureQueenInstinctsSectionAddsHeaderOnce(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	if err := writeLocalQueenText(legacyQueenDefaultContentFixture); err != nil {
		t.Fatalf("seed local queen: %v", err)
	}
	before, err := loadLocalQueenText()
	if err != nil {
		t.Fatalf("load before: %v", err)
	}
	if strings.Contains(before, "## Instincts") {
		t.Fatalf("fixture already contains '## Instincts'; fixture is invalid for this test")
	}

	if err := ensureQueenInstinctsSection(); err != nil {
		t.Fatalf("ensureQueenInstinctsSection: %v", err)
	}
	after, err := loadLocalQueenText()
	if err != nil {
		t.Fatalf("load after: %v", err)
	}

	if !strings.HasPrefix(after, before) {
		t.Fatalf("pre-existing content was not preserved byte-for-byte.\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if count := strings.Count(after, "## Instincts"); count != 1 {
		t.Fatalf("expected exactly one '## Instincts' header, got %d in:\n%s", count, after)
	}

	// Idempotence: a second call adds nothing further.
	if err := ensureQueenInstinctsSection(); err != nil {
		t.Fatalf("ensureQueenInstinctsSection (2nd call): %v", err)
	}
	after2, err := loadLocalQueenText()
	if err != nil {
		t.Fatalf("load after second call: %v", err)
	}
	if after2 != after {
		t.Fatalf("second call was not idempotent.\nfirst call result:\n%s\nsecond call result:\n%s", after, after2)
	}
}

// TestEnsureQueenInstinctsSectionCreatesFileWhenAbsent proves the helper
// creates a local QUEEN.md containing both "## Wisdom" and "## Instincts"
// when no local QUEEN.md exists yet.
func TestEnsureQueenInstinctsSectionCreatesFileWhenAbsent(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	p := localQueenPath()
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatalf("expected no local QUEEN.md to exist yet at %s, stat err = %v", p, err)
	}

	if err := ensureQueenInstinctsSection(); err != nil {
		t.Fatalf("ensureQueenInstinctsSection: %v", err)
	}

	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("expected local QUEEN.md to be created at %s: %v", p, err)
	}
	text := string(data)
	if !strings.Contains(text, "## Wisdom") {
		t.Fatalf("expected created QUEEN.md to contain '## Wisdom':\n%s", text)
	}
	if !strings.Contains(text, "## Instincts") {
		t.Fatalf("expected created QUEEN.md to contain '## Instincts':\n%s", text)
	}
}

// seedQueenEligibleInstinct saves a single instincts.json entry that is
// QueenEligible per pkg/memory/consolidate.go (Confidence >= 0.75 and >= 3
// recorded applications), carrying a distinctive sentinel Action string so
// its presence downstream can be asserted unambiguously.
func seedQueenEligibleInstinct(t *testing.T, sentinel string) {
	t.Helper()
	applied := time.Now().UTC().Format(time.RFC3339)
	instincts := colony.InstinctsFile{
		Instincts: []colony.InstinctEntry{
			{
				ID:         "inst_sentinel_promotion",
				Domain:     "testing",
				Trigger:    "sentinel trigger condition",
				Action:     sentinel,
				TrustScore: 0.90,
				TrustTier:  "trusted",
				Confidence: 0.90,
				ApplicationHistory: []colony.InstinctApplicationEntry{
					{Timestamp: applied, Outcome: "helpful"},
					{Timestamp: applied, Outcome: "helpful"},
					{Timestamp: applied, Outcome: "helpful"},
				},
			},
		},
	}
	if err := store.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatalf("seed QueenEligible instinct: %v", err)
	}
}

// TestConsolidationPromotesIntoLocalQueen runs the real (non-dry-run)
// consolidation-phase-end path against a QueenEligible instinct and proves
// the sentinel Action text lands in <root>/.aether/QUEEN.md under
// "## Instincts", and that <root>/.aether/data/QUEEN.md (the old dead
// target) is never created.
func TestConsolidationPromotesIntoLocalQueen(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	store = s

	sentinel := "sentinel-action-zzyzx-promote-me-into-queen-md"
	seedQueenEligibleInstinct(t, sentinel)

	rootCmd.SetArgs([]string{"consolidation-phase-end"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("consolidation-phase-end failed: %v", err)
	}

	localQueen := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	data, err := os.ReadFile(localQueen)
	if err != nil {
		t.Fatalf("expected %s to exist after real consolidation run: %v", localQueen, err)
	}
	text := string(data)
	idx := strings.Index(text, "## Instincts")
	if idx == -1 {
		t.Fatalf("expected '## Instincts' section in %s:\n%s", localQueen, text)
	}
	if !strings.Contains(text[idx:], sentinel) {
		t.Fatalf("expected sentinel %q to appear under '## Instincts' in %s:\n%s", sentinel, localQueen, text)
	}

	deadFile := filepath.Join(s.BasePath(), "QUEEN.md")
	if _, err := os.Stat(deadFile); !os.IsNotExist(err) {
		t.Fatalf("%s must not exist (store-relative QUEEN.md is the dead target this plan retires), stat err = %v", deadFile, err)
	}
}

// TestPromotedInstinctReachesWorkerPrompt is the end-to-end reachability
// proof: pipeline -> .aether/QUEEN.md -> readQUEENMd -> colony-prime prompt
// section. Runs the real consolidation-phase-end path against a
// QueenEligible instinct, then -- critically -- clears instincts.json
// before assembling the prompt. The colony-prime "## Active Instincts"
// section reads instincts.json directly and unconditionally (T-162-09;
// this plan changes the QUEEN.md route, not that pre-existing exposure),
// so leaving instincts.json populated would let the sentinel reach the
// prompt through that unrelated path and make this test pass vacuously
// even if the QUEEN.md link were cut. Clearing it isolates the link this
// test exists to prove.
func TestPromotedInstinctReachesWorkerPrompt(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, _ := newTestStore(t)
	store = s

	sentinel := "sentinel-action-reaches-the-worker-prompt-qzplm"
	seedQueenEligibleInstinct(t, sentinel)

	rootCmd.SetArgs([]string{"consolidation-phase-end"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("consolidation-phase-end failed: %v", err)
	}

	// Isolate the QUEEN.md route from the unrelated instincts.json ->
	// "## Active Instincts" route by clearing instincts.json now that
	// promotion into QUEEN.md has already happened.
	if err := s.SaveJSON("instincts.json", colony.InstinctsFile{Instincts: []colony.InstinctEntry{}}); err != nil {
		t.Fatalf("clear instincts.json: %v", err)
	}

	output := buildColonyPrimeOutput(true)
	if strings.Contains(output.PromptSection, "## Active Instincts") {
		t.Fatalf("test isolation failed: '## Active Instincts' section still present after clearing instincts.json:\n%s", output.PromptSection)
	}
	if !strings.Contains(output.PromptSection, sentinel) {
		t.Fatalf("expected sentinel %q to reach buildColonyPrimeOutput(true).PromptSection via QUEEN.md, got:\n%s", sentinel, output.PromptSection)
	}
}

// TestConsolidationRefusesLinkedLocalQueenBoundary200 proves the injected
// repository writer revalidates the .aether boundary before touching QUEEN.md.
// A replaced boundary must fail closed instead of following the link and
// writing outside the repository.
func TestConsolidationRefusesLinkedLocalQueenBoundary200(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	store = s

	realAether := filepath.Join(tmpDir, ".aether-real")
	if err := os.Rename(filepath.Join(tmpDir, ".aether"), realAether); err != nil {
		t.Fatalf("move real .aether boundary: %v", err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(tmpDir, ".aether")); err != nil {
		t.Fatalf("replace .aether with symlink: %v", err)
	}

	writer := pipelineConfigForStore().QueenInstinctPromoter
	if writer == nil {
		t.Fatal("pipelineConfigForStore must inject a Queen instinct writer")
	}
	err := writer(context.Background(), colony.InstinctEntry{
		ID: "inst_linked_boundary", Domain: "testing", Trigger: "linked boundary",
		Action: "must-not-escape", Confidence: 0.90,
	}, "test-colony")
	if err == nil {
		t.Fatal("linked .aether boundary must be refused")
	}
	if _, statErr := os.Lstat(filepath.Join(outside, "QUEEN.md")); !os.IsNotExist(statErr) {
		t.Fatalf("writer escaped into linked boundary, stat err = %v", statErr)
	}
}

// TestConsolidationQueenWriteFailureIsNotPromoted200 proves cmd wiring keeps
// the package's truthful accounting when the contained target cannot be
// written. A directory at QUEEN.md is a deterministic write refusal.
func TestConsolidationQueenWriteFailureIsNotPromoted200(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	store = s
	seedQueenEligibleInstinct(t, "sentinel-must-not-be-counted-as-promoted")

	if err := os.Mkdir(filepath.Join(tmpDir, ".aether", "QUEEN.md"), 0755); err != nil {
		t.Fatalf("create unwritable Queen target: %v", err)
	}
	bus := events.NewBus(s, events.DefaultConfig())
	result, err := learn.NewPipeline(s, bus, pipelineConfigForStore()).RunConsolidation(context.Background())
	if err != nil {
		t.Fatalf("RunConsolidation returned top-level error: %v", err)
	}
	if len(result.QueenPromoted) != 0 {
		t.Fatalf("failed Queen write counted as promoted: %v", result.QueenPromoted)
	}
	foundWriteFailure := false
	for _, resultErr := range result.Errors {
		if strings.Contains(resultErr.Error(), "queen promote") {
			foundWriteFailure = true
			break
		}
	}
	if !foundWriteFailure {
		t.Fatalf("Queen write failure missing from result.Errors: %v", result.Errors)
	}
}

// TestReadQUEENMdIngestsInstinctsSection proves readQUEENMd ingests the
// "## Instincts" section, and that PromoteInstinct's exact entry format
// (- [instinct] **<domain>** (<conf>): When <trigger>, then <action>)
// yields the action text as the ingested value -- matching the existing
// "key: value" parsing branch rather than requiring parser changes.
func TestReadQUEENMdIngestsInstinctsSection(t *testing.T) {
	tmpDir := t.TempDir()
	queenPath := filepath.Join(tmpDir, "QUEEN.md")
	content := "## Instincts\n\n- [instinct] **testing** (0.90): When trigger-condition-x, then action-outcome-y\n"
	if err := os.WriteFile(queenPath, []byte(content), 0644); err != nil {
		t.Fatalf("write fixture QUEEN.md: %v", err)
	}

	result := readQUEENMd(queenPath)
	if len(result) == 0 {
		t.Fatalf("expected readQUEENMd to return a non-empty map for a QUEEN.md containing only ## Instincts")
	}

	found := false
	for _, v := range result {
		if strings.Contains(v, "action-outcome-y") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected readQUEENMd's ingested value to contain the action text 'action-outcome-y', got: %+v", result)
	}
}

// TestReadQUEENMdInstinctsDoesNotPullUnrelatedSections confirms widening the
// allowlist to include Instincts does not also pull User Preferences or
// Colony Charter lines into the wisdom map.
func TestReadQUEENMdInstinctsDoesNotPullUnrelatedSections(t *testing.T) {
	tmpDir := t.TempDir()
	queenPath := filepath.Join(tmpDir, "QUEEN.md")
	content := "## Instincts\n\n- [instinct] **testing** (0.90): When trigger-condition-x, then action-outcome-y\n\n" +
		"## User Preferences\n\n- prefer-plain-english-marker\n\n" +
		"## Colony Charter\n\n- **Name:** charter-name-marker\n"
	if err := os.WriteFile(queenPath, []byte(content), 0644); err != nil {
		t.Fatalf("write fixture QUEEN.md: %v", err)
	}

	result := readQUEENMd(queenPath)
	for k, v := range result {
		if strings.Contains(k, "prefer-plain-english-marker") || strings.Contains(v, "prefer-plain-english-marker") {
			t.Fatalf("readQUEENMd must not pull User Preferences lines into the wisdom map, got key=%q val=%q", k, v)
		}
		if strings.Contains(k, "charter-name-marker") || strings.Contains(v, "charter-name-marker") {
			t.Fatalf("readQUEENMd must not pull Colony Charter lines into the wisdom map, got key=%q val=%q", k, v)
		}
	}
}
