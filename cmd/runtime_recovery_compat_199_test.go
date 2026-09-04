package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestRuntimeRecoveryCompatibility199 locks the input-only legacy command
// migration at the single compatibility boundary. Everything after that
// boundary must operate on canonical pause/resume command IDs.
func TestRuntimeRecoveryCompatibility199(t *testing.T) {
	const legacyResume = "resume-colony"

	t.Run("legacy load normalizes to canonical resume", func(t *testing.T) {
		if got := normalizeLegacySessionCommand(legacyResume, "1.28.9"); got != "resume" {
			t.Fatalf("normalized legacy command = %q, want resume", got)
		}
	})

	t.Run("canonical writes do not retain legacy token", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		state := colony.ColonyState{State: colony.StateEXECUTING}
		ensureSessionSummary(state, legacyResume, "aether continue", "compatibility write")

		var session colony.SessionFile
		if err := s.LoadJSON("session.json", &session); err != nil {
			t.Fatalf("load session: %v", err)
		}
		if session.LastCommand != "resume" {
			t.Fatalf("LastCommand = %q, want canonical resume", session.LastCommand)
		}
	})

	t.Run("hook grace accepts loaded legacy state", func(t *testing.T) {
		saveGlobals(t)
		s, _ := newTestStore(t)
		store = s
		if err := s.SaveJSON("session.json", colony.SessionFile{
			LastCommand: legacyResume, LastCommandAt: time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			t.Fatalf("save session: %v", err)
		}
		if !allowStopAfterRecentResume() {
			t.Fatal("legacy persisted resume did not receive post-resume grace")
		}
	})

	t.Run("exact token only", func(t *testing.T) {
		for _, input := range []string{"resume-colonies", "resume colony", "Resume-colony"} {
			if got := normalizeLegacySessionCommand(input, "1.28.9"); got != input {
				t.Errorf("normalizeLegacySessionCommand(%q) = %q, want unchanged", input, got)
			}
		}
	})

	t.Run("expiry retains original token", func(t *testing.T) {
		if got := normalizeLegacySessionCommand(legacyResume, "1.29.0"); got != legacyResume {
			t.Fatalf("expired legacy command = %q, want unchanged", got)
		}
	})

	t.Run("recovery candidates use canonical command", func(t *testing.T) {
		if got := normalizeBaseCommand(legacyResume); got != "resume" {
			t.Fatalf("recovery command = %q, want canonical resume", got)
		}
	})

	t.Run("recovery output keeps diagnosis read only and restoration canonical", func(t *testing.T) {
		state := colony.ColonyState{State: colony.StateEXECUTING}
		for _, issue := range []HealthIssue{
			{Category: "stale_spawned", Severity: "critical", Fixable: true},
			{Category: "dirty_worktree", Severity: "critical", Fixable: true},
			{Category: "missing_agents", Severity: "warning", Fixable: true},
		} {
			output := renderRecoverDiagnosis([]HealthIssue{issue}, state, nil)
			if !strings.Contains(output, "aether maintenance recovery-inspect") || !strings.Contains(output, "aether resume") {
				t.Fatalf("recovery output must provide inspection and resume routes:\n%s", output)
			}
			assertNoRetiredRuntimeRecoverySuggestion199(t, output)
		}
	})

	t.Run("owned runtime literals and parity guide have no retired suggestions", func(t *testing.T) {
		for _, name := range []string{"recover_visuals.go", "recover_repair.go", "worktree.go"} {
			assertNoRetiredRuntimeRecoveryLiterals199(t, name)
		}
		parityPath := filepath.Join("..", ".aether", "docs", "PARITY_CLASSIC_VS_GO.md")
		body, err := os.ReadFile(parityPath)
		if err != nil {
			t.Fatalf("read parity guide: %v", err)
		}
		for _, retired := range []string{"aether recover", "recover --apply", "/ant-recover"} {
			if strings.Contains(string(body), retired) {
				t.Errorf("parity guide retains retired route %q", retired)
			}
		}
		if !strings.Contains(string(body), "aether maintenance recovery-inspect") || !strings.Contains(string(body), "aether resume") {
			t.Fatal("parity guide must prove maintenance inspection and resume")
		}
	})
}
