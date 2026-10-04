package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Claude Code reads a command's description only from front matter at the very
// top of the file. Every managed command file used to start with the
// Aether-managed comment, so the owner's command menu showed that comment
// instead of the real description (2026-10-04, WINDOWS 86). The marker now sits
// on the line after the front matter. The updater must still recognise files
// installed in the old layout, or it would stop replacing them.
const testManagedMarker = "<!-- Aether-managed: runtime spec at .aether/commands/build.yaml. Synced by aether update. -->"

func TestManagedCommandFilesStartWithFrontMatter(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	checked := 0
	for _, dir := range []string{".claude/commands/ant", ".claude/commands", ".opencode/commands/ant"} {
		entries, err := os.ReadDir(filepath.Join(repoRoot, filepath.FromSlash(dir)))
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(dir), entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if !isGeneratedAetherCommandWrapper(data) {
				continue // user-authored or unmanaged file
			}
			checked++
			if !strings.HasPrefix(string(data), "---\n") {
				t.Errorf("%s/%s is Aether-managed but does not start with front matter, so its description is hidden", dir, entry.Name())
			}
		}
	}
	if checked < 60 {
		t.Fatalf("only %d managed command files found; the check is not looking at the real commands", checked)
	}
}

func TestManagedMarkerIsRecognisedInBothLayouts(t *testing.T) {
	legacy := []byte(testManagedMarker + "\n---\nname: ant-build\ndescription: \"d\"\n---\n\nbody\n")
	current := []byte("---\nname: ant-build\ndescription: \"d\"\n---\n" + testManagedMarker + "\n\nbody\n")
	crlf := []byte(strings.ReplaceAll(string(current), "\n", "\r\n"))
	for name, data := range map[string][]byte{"legacy first line": legacy, "after front matter": current, "after front matter, CRLF": crlf} {
		if !isGeneratedAetherCommandWrapper(data) {
			t.Errorf("%s: not recognised as Aether-managed", name)
		}
	}
	for name, data := range map[string][]byte{
		"user file with front matter": []byte("---\nname: mine\ndescription: x\n---\nhello\n"),
		"marker buried in the body":   []byte("---\nname: mine\ndescription: x\n---\nhello\n" + testManagedMarker + "\n"),
		"marker inside front matter":  []byte("---\nname: mine\n" + testManagedMarker + "\n---\nhello\n"),
		"empty":                       nil,
	} {
		if isGeneratedAetherCommandWrapper(data) {
			t.Errorf("%s: wrongly treated as Aether-managed (would let update overwrite it)", name)
		}
	}
}

// splitManagedMarker returns a command file's Aether-managed marker line and
// the file text with that line removed, wherever the marker sits (line one in
// the older layout, the line after the front matter now). What remains is
// exactly the text the YAML spec's `wrapper` field holds, so tests can compare
// it to the spec without caring where the marker was.
func splitManagedMarker(text string) (marker, rest string) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	first, remainder, _ := strings.Cut(text, "\n")
	if first != "---" {
		return first, remainder
	}
	end := strings.Index(text, "\n---\n")
	if end < 0 {
		return "", text
	}
	cut := end + len("\n---\n")
	line, tail, _ := strings.Cut(text[cut:], "\n")
	return line, text[:cut] + tail
}

func TestSplitManagedMarkerRemovesTheMarkerInBothLayouts(t *testing.T) {
	legacy := testManagedMarker + "\n---\nname: x\n---\n\nbody\n"
	current := "---\nname: x\n---\n" + testManagedMarker + "\n\nbody\n"
	want := "---\nname: x\n---\n\nbody\n"
	for name, text := range map[string]string{"legacy": legacy, "current": current} {
		marker, rest := splitManagedMarker(text)
		if marker != testManagedMarker || rest != want {
			t.Errorf("%s: marker=%q rest=%q, want the marker and %q", name, marker, rest, want)
		}
	}
}
