package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// allowedLiveWrapperSuppressionCount is a ceiling on known-benign stderr
// suppression in the two LIVE wrapper directories (.claude/commands/ant/ and
// .opencode/commands/ant/ — the literal prompt text injected when a user runs
// a slash command). Measured directly against the current tree: 8 total (4 per
// platform mirror) — a `git diff --stat` prose call in dream.md and three
// `grep ... || true` / `grep ... | head` fallbacks in archaeology.md, mirrored
// identically across both platforms. None of the counted sites is an `aether`
// invocation. This constant is a ceiling on known-benign suppression, not a
// target to defend — a legitimate new benign site requires bumping this
// constant deliberately in the same commit that adds it, so the change is
// reviewed rather than silent.
const allowedLiveWrapperSuppressionCount = 8

// TestLiveWrapperStderrSuppressionCount is the count-invariant half of
// LOUD-06. It walks every .md file under the two live wrapper directories,
// counts total occurrences of the stderr-suppression token, and asserts the
// total equals allowedLiveWrapperSuppressionCount — so any *new* suppression
// site (in a new or existing file) trips this test regardless of what it's
// called.
//
// A second, sharper assertion is independent of the count: no line containing
// the suppression token may also contain the token "aether". That is the
// assertion that actually encodes LOUD-06's requirement — suppression on a
// non-aether shell fallback (grep, git) is benign; suppression on an aether
// invocation silently swallows a colony failure from the operator.
func TestLiveWrapperStderrSuppressionCount(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	wrapperDirs := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant"),
	}

	// Build the suppression token from parts at runtime so this test file's
	// own source never contributes an occurrence if the scan is ever widened.
	suppressionToken := "2" + ">" + "/dev/null"

	type siteHit struct {
		file string
		line int
		text string
	}

	var allHits []siteHit
	perFileCounts := map[string]int{}

	for _, dir := range wrapperDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			// filepath.Base(dir) alone would collide ("ant") across both
			// platform mirrors, so report the platform-qualified relative path.
			relName := filepath.Join(filepath.Base(filepath.Dir(filepath.Dir(dir))), filepath.Base(filepath.Dir(dir)), filepath.Base(dir), entry.Name())

			for i, line := range strings.Split(string(data), "\n") {
				if !strings.Contains(line, suppressionToken) {
					continue
				}
				allHits = append(allHits, siteHit{file: relName, line: i + 1, text: strings.TrimSpace(line)})
				perFileCounts[relName]++

				if strings.Contains(line, "aether") {
					t.Errorf(
						"%s:%d suppresses stderr on a line containing an `aether` invocation — this is forbidden regardless of the allowed count, because it can silently swallow a colony failure: %q",
						relName, i+1, strings.TrimSpace(line),
					)
				}
			}
		}
	}

	if len(allHits) != allowedLiveWrapperSuppressionCount {
		var detail strings.Builder
		for _, hit := range allHits {
			fmt.Fprintf(&detail, "  %s:%d: %s\n", hit.file, hit.line, hit.text)
		}
		var perFile strings.Builder
		for file, count := range perFileCounts {
			fmt.Fprintf(&perFile, "  %s: %d\n", file, count)
		}
		t.Errorf(
			"live wrapper stderr suppression count changed: expected %d, measured %d.\nPer-file counts:\n%sAll sites:\n%s",
			allowedLiveWrapperSuppressionCount, len(allHits), perFile.String(), detail.String(),
		)
	}
}
