package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
)

// Plan 173-08 (SPAWN-07).
//
// spawn-tree-active's JSON payload has been a flat array since it was
// written; this file proves the visual rendering Task 1 added on top of it
// four ways: that it is indented by depth while the run is still going, not
// after it ends (D-14/ROADMAP criterion 6); that it reads as English rather
// than JSON with the field names translated (D-14); that it mutates nothing,
// the same class of defect CLAUDE.md's Definition of Done names in
// `consolidation-phase-end --dry-run` (T-173-40); and that the pre-existing
// JSON contract survives the rendering change untouched.

// findLineContaining returns the first line in text containing substr, or
// fails the test -- a test that could not find the line it means to assert
// against would pass vacuously.
func findLineContaining(t *testing.T, text, substr string) string {
	t.Helper()
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, substr) {
			return line
		}
	}
	t.Fatalf("no line in output contains %q:\n%s", substr, text)
	return ""
}

// leadingSpaces counts the run of leading ASCII spaces on a line.
func leadingSpaces(line string) int {
	n := 0
	for _, r := range line {
		if r != ' ' {
			break
		}
		n++
	}
	return n
}

// containsStandaloneWord reports whether word appears in text as its own
// word, not merely as a substring of a longer word (so "spawn" inside
// "spawned" does not falsely trip a check for the raw token "spawned", and
// vice versa).
func containsStandaloneWord(text, word string) bool {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(word) + `\b`)
	return re.MatchString(text)
}

// hashAllFilesForTest walks dir and returns a path-to-SHA-256 map of every
// regular file under it, using the same hashFileForTest helper the
// consolidation dry-run purity tests use (consolidation_dryrun_test.go) --
// the same class of test, proving the same invariant, for a different
// command.
func hashAllFilesForTest(t *testing.T, dir string) map[string]string {
	t.Helper()
	hashes := map[string]string{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			rel = path
		}
		hashes[rel] = hashFileForTest(t, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return hashes
}

// diffHashes returns a human-readable description of every path whose hash
// changed or appeared between before and after, or "" if identical.
func diffHashes(before, after map[string]string) string {
	var diffs []string
	for k, v := range before {
		if after[k] != v {
			diffs = append(diffs, fmt.Sprintf("%s: %s -> %s", k, v, after[k]))
		}
	}
	for k, v := range after {
		if _, ok := before[k]; !ok {
			diffs = append(diffs, fmt.Sprintf("%s: (absent) -> %s", k, v))
		}
	}
	sort.Strings(diffs)
	return strings.Join(diffs, "; ")
}

// suppressFirstRunBanner pre-writes the ".welcomed" marker checkAndEmitFirstRun
// (cmd/ux_firstrun.go) looks for, so a fresh test store in visual mode does not
// print the first-run banner ahead of spawn-tree-active's own output -- and,
// more importantly for TestSpawnTreeActiveMutatesNothing, does not have
// checkAndEmitFirstRun itself write that marker file as a side effect of
// running a command, which would be an unrelated mutation this plan's own
// purity test must not be tripped by.
func suppressFirstRunBanner(t *testing.T, dataDir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dataDir, ".welcomed"), []byte(""), 0600); err != nil {
		t.Fatalf("pre-write .welcomed marker: %v", err)
	}
}

// TestSpawnTreeActiveRendersIndentedByDepthMidRun is the direct D-14/ROADMAP
// criterion-6 proof: a Queen -> W1 -> H1 chain, recorded with a run begun and
// every entry left in a live status (no spawn-complete called on any of
// them -- this is the requirement, not a convenience: a test that completed
// the entries first would prove the tree renders correctly once finished,
// which is the opposite of "while the run is in progress").
func TestSpawnTreeActiveRendersIndentedByDepthMidRun(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	suppressFirstRunBanner(t, s.BasePath())

	st := agent.NewSpawnTree(s, "spawn-tree.txt")
	if _, err := st.BeginRun("test", time.Now().UTC()); err != nil {
		t.Fatalf("begin run: %v", err)
	}

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgsWithCasteTask("Queen", "W1", "builder", "fix the login form"))
	runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgsWithCasteTask("W1", "H1", "watcher", "check the fix"))

	buf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"spawn-tree-active"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("spawn-tree-active: %v", err)
	}
	out := buf.String()

	w1Line := findLineContaining(t, out, "W1 is still working")
	h1Line := findLineContaining(t, out, "H1 is still working")

	w1Indent := leadingSpaces(w1Line)
	h1Indent := leadingSpaces(h1Line)

	if h1Indent != w1Indent+2 {
		t.Fatalf("H1's indentation (%d spaces) is not exactly two more than W1's (%d spaces):\n%s", h1Indent, w1Indent, out)
	}

	// If a depth-0 root line is rendered at all, W1's indentation must be
	// exactly two more than it. Nothing here is ever rendered at depth 0 in
	// practice -- the coordinator sentinel is never itself a recorded
	// entry -- but the assertion is written to hold regardless.
	if rootLine := firstLineWithIndent(out, 0); rootLine != "" && strings.TrimSpace(rootLine) != "" {
		if w1Indent != leadingSpaces(rootLine)+2 {
			t.Fatalf("W1's indentation (%d spaces) is not exactly two more than the depth-0 root line's (%d spaces):\n%s", w1Indent, leadingSpaces(rootLine), out)
		}
	}

	if !strings.Contains(h1Line, "W1") {
		t.Fatalf("H1's line does not attribute the spawn to W1:\n%s", h1Line)
	}
}

// firstLineWithIndent returns the first non-blank line in text whose
// leading-space count equals want, or "" if none.
func firstLineWithIndent(text string, want int) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if leadingSpaces(line) == want {
			return line
		}
	}
	return ""
}

// TestSpawnTreeActiveShowsNoRawIdentifiersOrJSON is the D-14 invariant test:
// it asserts the ABSENCE of JSON punctuation and raw status tokens and the
// PRESENCE of the English caste label, rather than that a named section
// exists -- a test asserting only that a section exists could not catch its
// replacement by a JSON dump.
func TestSpawnTreeActiveShowsNoRawIdentifiersOrJSON(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	suppressFirstRunBanner(t, s.BasePath())

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgsWithCasteTask("Queen", "W1", "builder", "fix the login form"))

	buf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"spawn-tree-active"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("spawn-tree-active: %v", err)
	}
	out := buf.String()

	for _, forbidden := range []string{"{", "}", "\":"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("visual output contains raw JSON punctuation %q:\n%s", forbidden, out)
		}
	}

	for _, token := range []string{"spawned", "active", "running", "completed"} {
		if containsStandaloneWord(out, token) {
			t.Fatalf("visual output contains the raw status token %q as a standalone word:\n%s", token, out)
		}
	}

	label := casteLabel("builder")
	if !strings.Contains(out, label) {
		t.Fatalf("visual output does not contain the English caste label %q for the recorded caste:\n%s", label, out)
	}
}

// TestSpawnTreeActiveMutatesNothing is the CLAUDE.md Definition of Done
// corollary test: an inspection command must not mutate state. Hashes every
// file in the test store, runs the visual command twice, and asserts every
// hash is unchanged and both runs' outputs are byte-identical. This is the
// same class of defect this repo shipped in
// `consolidation-phase-end --dry-run` for months.
func TestSpawnTreeActiveMutatesNothing(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	suppressFirstRunBanner(t, s.BasePath())

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgsWithCasteTask("Queen", "W1", "builder", "fix the login form"))
	runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgsWithCasteTask("W1", "H1", "watcher", "check the fix"))

	dataDir := s.BasePath()
	before := hashAllFilesForTest(t, dataDir)

	buf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"spawn-tree-active"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("first run: %v", err)
	}
	firstOutput := buf.String()

	afterFirst := hashAllFilesForTest(t, dataDir)
	if diff := diffHashes(before, afterFirst); diff != "" {
		t.Fatalf("first run mutated the store: %s", diff)
	}

	buf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"spawn-tree-active"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("second run: %v", err)
	}
	secondOutput := buf.String()

	afterSecond := hashAllFilesForTest(t, dataDir)
	if diff := diffHashes(before, afterSecond); diff != "" {
		t.Fatalf("second run mutated the store: %s", diff)
	}

	if firstOutput != secondOutput {
		t.Fatalf("two runs over the same unchanged tree produced different output:\nfirst=%q\nsecond=%q", firstOutput, secondOutput)
	}
}

// TestSpawnTreeActiveKeepsItsJSONContract proves the rendering change did
// not silently break the machine reader: every pre-existing key, at both
// the envelope and the per-entry level, is still present under
// AETHER_OUTPUT_MODE=json.
func TestSpawnTreeActiveKeepsItsJSONContract(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf

	runSpawnLogExpectingSuccess(t, &buf, &errBuf, spawnLogArgsWithCasteTask("Queen", "W1", "builder", "fix the login form"))

	buf.Reset()
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"spawn-tree-active"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("spawn-tree-active: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok=true, got: %s", buf.String())
	}
	result, _ := env["result"].(map[string]interface{})
	if result == nil {
		t.Fatalf("no result in payload: %s", buf.String())
	}
	if _, ok := result["active"]; !ok {
		t.Fatalf("result missing 'active' key: %s", buf.String())
	}
	if _, ok := result["count"]; !ok {
		t.Fatalf("result missing 'count' key: %s", buf.String())
	}

	active, _ := result["active"].([]interface{})
	if len(active) == 0 {
		t.Fatalf("expected at least one active entry: %s", buf.String())
	}
	entry, _ := active[0].(map[string]interface{})
	if entry == nil {
		t.Fatalf("active entry is not an object: %s", buf.String())
	}
	for _, key := range []string{"name", "parent", "caste", "task", "depth", "status", "spawned_at"} {
		if _, ok := entry[key]; !ok {
			t.Fatalf("active entry missing pre-existing key %q: %s", key, buf.String())
		}
	}
}
