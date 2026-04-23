package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// TestExtractEmojisFromMarkdown
// ---------------------------------------------------------------------------

func TestExtractEmojisFromMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "no emojis",
			content:  "plain text without any emojis",
			expected: nil,
		},
		{
			name:     "single emoji",
			content:  "build command 🔨",
			expected: []string{"🔨"},
		},
		{
			name:     "multiple emojis deduplicated",
			content:  "🔨 build and 🔨 again with 👁️",
			expected: []string{"🔨", "👁️"},
		},
		{
			name:     "emoji in markdown description",
			content:  "---\ndescription: \"🔨 Build a phase\"\n---\n\nContent here.",
			expected: []string{"🔨"},
		},
		{
			name:     "complex emoji content",
			content:  "🥚 init 🗺️ colonize 📋 plan 🔨 build 👁️ continue",
			expected: []string{"🥚", "🗺️", "📋", "🔨", "👁️"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := extractEmojisFromMarkdown(tc.content)
			if len(result) != len(tc.expected) {
				t.Fatalf("expected %d emojis, got %d: %v", len(tc.expected), len(result), result)
			}
			for i, e := range tc.expected {
				if i >= len(result) || result[i] != e {
					t.Errorf("emoji[%d]: expected %q, got %q", i, e, result[i])
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestEmojiConsistencyMatching
// ---------------------------------------------------------------------------

func TestEmojiConsistencyMatching(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude", "commands", "ant")
	opencodeDir := filepath.Join(dir, ".opencode", "commands", "ant")
	dataDir := filepath.Join(dir, ".aether", "data")

	for _, d := range []string{claudeDir, opencodeDir, dataDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	// build wrapper with correct emoji 🔨
	writeFile(t, claudeDir, "build.md", []byte("---\ndescription: \"🔨 Build a phase\"\n---"))
	writeFile(t, opencodeDir, "build.md", []byte("---\ndescription: \"🔨 Build a phase\"\n---"))

	fc := newFileChecker(dataDir)
	issues := checkEmojiConsistency(fc)

	for _, issue := range issues {
		if issue.Severity == "warning" {
			t.Errorf("matching wrapper produced warning: %s", issue.Message)
		}
	}
}

// ---------------------------------------------------------------------------
// TestEmojiConsistencyMismatch
// ---------------------------------------------------------------------------

func TestEmojiConsistencyMismatch(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude", "commands", "ant")
	dataDir := filepath.Join(dir, ".aether", "data")

	for _, d := range []string{claudeDir, dataDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	// build wrapper with WRONG emoji 🗺️ instead of 🔨
	writeFile(t, claudeDir, "build.md", []byte("---\ndescription: \"🗺️ Build a phase\"\n---"))

	fc := newFileChecker(dataDir)
	issues := checkEmojiConsistency(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && contains(issue.Message, "build") && contains(issue.Message, "runtime expects") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected mismatch warning for build command; got issues: %+v", issues)
	}
}

// ---------------------------------------------------------------------------
// TestEmojiConsistencyMissing
// ---------------------------------------------------------------------------

func TestEmojiConsistencyMissing(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude", "commands", "ant")
	dataDir := filepath.Join(dir, ".aether", "data")

	for _, d := range []string{claudeDir, dataDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	// build wrapper with no emoji at all
	writeFile(t, claudeDir, "build.md", []byte("---\ndescription: \"Build a phase\"\n---"))

	fc := newFileChecker(dataDir)
	issues := checkEmojiConsistency(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "info" && contains(issue.Message, "no emoji") && contains(issue.Message, "build") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected info for missing emoji; got issues: %+v", issues)
	}
}

// ---------------------------------------------------------------------------
// TestCheckStageMarkersPresent
// ---------------------------------------------------------------------------

func TestCheckStageMarkersPresent(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude", "commands", "ant")
	yamlDir := filepath.Join(dir, ".aether", "commands")
	dataDir := filepath.Join(dir, ".aether", "data")

	for _, d := range []string{claudeDir, yamlDir, dataDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	// Create wrappers with stage markers for all state-changing commands
	for _, cmd := range stateChangingCommands {
		writeFile(t, claudeDir, cmd+".md", []byte("── Context ──\n── Tasks ──\ncontent"))
		writeFile(t, yamlDir, cmd+".yaml", []byte("name: "+cmd))
	}

	fc := newFileChecker(dataDir)
	issues := checkStageMarkers(fc)

	for _, issue := range issues {
		t.Errorf("state-changing commands with markers produced issue: [%s] %s", issue.Severity, issue.Message)
	}
}

// ---------------------------------------------------------------------------
// TestCheckStageMarkersMissing
// ---------------------------------------------------------------------------

func TestCheckStageMarkersMissing(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude", "commands", "ant")
	yamlDir := filepath.Join(dir, ".aether", "commands")
	dataDir := filepath.Join(dir, ".aether", "data")

	for _, d := range []string{claudeDir, yamlDir, dataDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	// Create wrapper without stage markers
	writeFile(t, claudeDir, "build.md", []byte("plain content with no markers"))
	writeFile(t, yamlDir, "build.yaml", []byte("name: build"))

	fc := newFileChecker(dataDir)
	issues := checkStageMarkers(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && contains(issue.Message, "no stage markers") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for missing stage markers; got issues: %+v", issues)
	}
}

// ---------------------------------------------------------------------------
// TestCheckStageMarkersMissingYAML
// ---------------------------------------------------------------------------

func TestCheckStageMarkersMissingYAML(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude", "commands", "ant")
	dataDir := filepath.Join(dir, ".aether", "data")

	for _, d := range []string{claudeDir, dataDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	// Wrapper with markers but no YAML source
	writeFile(t, claudeDir, "build.md", []byte("── Context ──\ncontent"))

	fc := newFileChecker(dataDir)
	issues := checkStageMarkers(fc)

	found := false
	for _, issue := range issues {
		if issue.Severity == "warning" && contains(issue.Message, "YAML source") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for missing YAML source; got issues: %+v", issues)
	}
}

// ---------------------------------------------------------------------------
// TestCheckContextClearGuidance
// ---------------------------------------------------------------------------

func TestCheckContextClearGuidance(t *testing.T) {
	t.Run("no hardcoded patterns", func(t *testing.T) {
		dir := t.TempDir()
		continueDir := filepath.Join(dir, ".claude", "commands", "ant")
		dataDir := filepath.Join(dir, ".aether", "data")

		for _, d := range []string{continueDir, dataDir} {
			if err := os.MkdirAll(d, 0755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
		}

		// continue.md that references runtime-owned context-clear
		writeFile(t, continueDir, "continue.md", []byte(
			"The runtime emits context-clear guidance automatically -- do not duplicate it.\n"))

		fc := newFileChecker(dataDir)
		issues := checkContextClearGuidance(fc)

		for _, issue := range issues {
			if issue.Severity == "warning" {
				t.Errorf("runtime-owned context-clear produced warning: %s", issue.Message)
			}
		}
	})

	t.Run("hardcoded pattern detected", func(t *testing.T) {
		dir := t.TempDir()
		continueDir := filepath.Join(dir, ".claude", "commands", "ant")
		dataDir := filepath.Join(dir, ".aether", "data")

		for _, d := range []string{continueDir, dataDir} {
			if err := os.MkdirAll(d, 0755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
		}

		// continue.md with hard-coded context-clear
		writeFile(t, continueDir, "continue.md", []byte(
			"It's safe to clear your context now. Run /ant-resume.\n"))

		fc := newFileChecker(dataDir)
		issues := checkContextClearGuidance(fc)

		found := false
		for _, issue := range issues {
			if issue.Severity == "warning" && contains(issue.Message, "hardcoded value") {
				found = true
			}
		}
		if !found {
			t.Errorf("expected warning for hard-coded context-clear; got issues: %+v", issues)
		}
	})

	t.Run("missing continue.md", func(t *testing.T) {
		dir := t.TempDir()
		dataDir := filepath.Join(dir, ".aether", "data")
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		fc := newFileChecker(dataDir)
		issues := checkContextClearGuidance(fc)

		found := false
		for _, issue := range issues {
			if issue.Severity == "warning" && contains(issue.Message, "not found") {
				found = true
			}
		}
		if !found {
			t.Errorf("expected warning for missing continue.md; got issues: %+v", issues)
		}
	})
}

// ---------------------------------------------------------------------------
// TestScanCeremonyIntegrityIntegration
// ---------------------------------------------------------------------------

func TestScanCeremonyIntegrityIntegration(t *testing.T) {
	dir := t.TempDir()
	claudeDir := filepath.Join(dir, ".claude", "commands", "ant")
	opencodeDir := filepath.Join(dir, ".opencode", "commands", "ant")
	yamlDir := filepath.Join(dir, ".aether", "commands")
	dataDir := filepath.Join(dir, ".aether", "data")

	for _, d := range []string{claudeDir, opencodeDir, yamlDir, dataDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	// Healthy setup: correct emojis, stage markers, no hardcoded context-clear
	for _, cmd := range stateChangingCommands {
		expectedEmoji := getCommandEmoji(cmd)
		writeFile(t, claudeDir, cmd+".md", []byte(
			"── Context ──\n"+expectedEmoji+" "+cmd+" command\n"))
		writeFile(t, yamlDir, cmd+".yaml", []byte("name: "+cmd))
	}

	writeFile(t, claudeDir, "build.md", []byte("── Context ──\n🔨 build command\n"))
	writeFile(t, opencodeDir, "build.md", []byte("── Context ──\n🔨 build command\n"))
	writeFile(t, claudeDir, "continue.md", []byte(
		"The runtime emits context-clear guidance automatically.\n"))

	fc := newFileChecker(dataDir)
	issues := scanCeremonyIntegrity(fc)

	for _, issue := range issues {
		if issue.Severity == "critical" {
			t.Errorf("healthy ceremony setup produced critical: %s", issue.Message)
		}
	}
}
