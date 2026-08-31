package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLifecycleWrappersHaveVisualCloseoutAfterJSONFinalizer(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	workflows := []string{"build", "plan", "colonize", "continue", "seal", "swarm"}
	for _, platformDir := range []string{".claude/commands/ant", ".opencode/commands/ant"} {
		for _, workflow := range workflows {
			wrapperPath := filepath.Join(repoRoot, platformDir, workflow+".md")
			content, err := os.ReadFile(wrapperPath)
			if err != nil {
				t.Fatalf("read %s: %v", wrapperPath, err)
			}
			text := string(content)
			finalizer := "AETHER_OUTPUT_MODE=json aether " + workflow + "-finalize"
			if workflow == "build" {
				finalizer = "AETHER_OUTPUT_MODE=json aether build-finalize"
			}
			closeout := "AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow " + workflow
			assertSubstringsInOrder(t, wrapperPath, text, []string{finalizer, closeout})
		}
	}
}

func TestLifecycleWrappersRenderRuntimeCeremonySurfaces(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	// Workflows that render full worker ceremony in wrappers
	workerTheatreWorkflows := []string{"colonize", "seal", "swarm"}
	// Workflows that delegate manifest fetch to TS host
	hostWorkflows := []string{"colonize", "build", "plan", "continue", "seal"}

	for _, platformDir := range []string{".claude/commands/ant", ".opencode/commands/ant"} {
		for _, workflow := range workerTheatreWorkflows {
			wrapperPath := filepath.Join(repoRoot, platformDir, workflow+".md")
			content, err := os.ReadFile(wrapperPath)
			if err != nil {
				t.Fatalf("read %s: %v", wrapperPath, err)
			}
			text := string(content)
			for _, want := range []string{
				"AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow " + workflow,
				"AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow " + workflow,
				"AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow " + workflow,
			} {
				if !strings.Contains(text, want) {
					t.Errorf("%s missing runtime ceremony call %q", wrapperPath, want)
				}
			}
		}
		for _, workflow := range hostWorkflows {
			wrapperPath := filepath.Join(repoRoot, platformDir, workflow+".md")
			content, err := os.ReadFile(wrapperPath)
			if err != nil {
				t.Fatalf("read %s: %v", wrapperPath, err)
			}
			text := string(content)
			// TS host workflows delegate manifest orchestration to aether host and retain visual closeout.
			for _, want := range []string{
				"aether host " + workflow,
				"AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow " + workflow,
			} {
				if !strings.Contains(text, want) {
					t.Errorf("%s missing TS host visual contract %q", wrapperPath, want)
				}
			}
		}
	}
}

func TestWrapperOrchestratedCommandsPreserveLiveWorkerCeremony(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	// Commands with full inline worker ceremony
	workerTheatreCommands := []string{"colonize", "seal", "swarm"}
	// TS host commands with host-backed manifest generation
	hostCommands := []string{"colonize", "build", "plan", "continue", "seal"}

	for _, platformDir := range []string{".claude/commands/ant", ".opencode/commands/ant"} {
		for _, command := range workerTheatreCommands {
			wrapperPath := filepath.Join(repoRoot, platformDir, command+".md")
			content, err := os.ReadFile(wrapperPath)
			if err != nil {
				t.Fatalf("read %s: %v", wrapperPath, err)
			}
			text := string(content)
			for _, want := range []string{
				"visible live Task/subagent",
				"caste-labelled",
				"Do not set `run_in_background`",
				"background agents",
				"markdown worker table",
			} {
				if !strings.Contains(text, want) {
					t.Errorf("%s missing live worker ceremony contract %q", wrapperPath, want)
				}
			}
		}
		for _, command := range hostCommands {
			wrapperPath := filepath.Join(repoRoot, platformDir, command+".md")
			content, err := os.ReadFile(wrapperPath)
			if err != nil {
				t.Fatalf("read %s: %v", wrapperPath, err)
			}
			text := string(content)
			// TS host wrappers delegate orchestration; only require core safety markers
			for _, want := range []string{
				"Do not set `run_in_background`",
				"background agents",
			} {
				if !strings.Contains(text, want) {
					t.Errorf("%s missing live worker ceremony contract %q", wrapperPath, want)
				}
			}
		}
	}
}

func TestRuntimeOwnedWrappersDelegateVisually(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	commands := map[string]string{
		"bump-version":   "AETHER_OUTPUT_MODE=visual aether bump-version",
		"data-clean":     "AETHER_OUTPUT_MODE=visual aether data-clean",
		"export-signals": "AETHER_OUTPUT_MODE=visual aether export-signals",
		"feedback":       "AETHER_OUTPUT_MODE=visual aether feedback",
		"flag":           "AETHER_OUTPUT_MODE=visual aether flag",
		"flags":          "AETHER_OUTPUT_MODE=visual aether flags",
		"focus":          "AETHER_OUTPUT_MODE=visual aether focus",
		"history":        "AETHER_OUTPUT_MODE=visual aether history",
		"import-signals": "AETHER_OUTPUT_MODE=visual aether import-signals",
		"insert-phase":   "AETHER_OUTPUT_MODE=visual aether insert-phase",
		"maturity":       "AETHER_OUTPUT_MODE=visual aether maturity",
		"memory-details": "AETHER_OUTPUT_MODE=visual aether memory-details",
		"migrate-state":  "AETHER_OUTPUT_MODE=visual aether migrate-state",
		"quick":          "AETHER_OUTPUT_MODE=visual aether quick",
		"redirect":       "AETHER_OUTPUT_MODE=visual aether redirect",
		"tunnels":        "AETHER_OUTPUT_MODE=visual aether tunnels",
		"verify-castes":  "AETHER_OUTPUT_MODE=visual aether verify-castes",
	}

	for _, platformDir := range []string{".claude/commands/ant", ".opencode/commands/ant"} {
		for name, want := range commands {
			wrapperPath := filepath.Join(repoRoot, platformDir, name+".md")
			content, err := os.ReadFile(wrapperPath)
			if err != nil {
				t.Fatalf("read %s: %v", wrapperPath, err)
			}
			if !strings.Contains(string(content), want) {
				t.Errorf("%s missing visual runtime delegation %q", wrapperPath, want)
			}
		}
	}
}

func TestInsertPhaseWrapperParityAndGoOwnership(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	paths := []string{
		filepath.Join(repoRoot, ".aether", "commands", "insert-phase.yaml"),
		filepath.Join(repoRoot, ".claude", "commands", "ant", "insert-phase.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "insert-phase.md"),
	}
	contents := make([]string, 0, len(paths))
	for _, path := range paths {
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("read %s: %v", path, readErr)
		}
		text := string(content)
		contents = append(contents, text)
		for _, want := range []string{
			"AETHER_OUTPUT_MODE=visual aether insert-phase $ARGUMENTS",
			`aether insert-phase "problem to stabilise"`,
			`aether insert-phase --after 2 --name "Stabilize login retries" --description "login retries lose state" --constraints "do not change the provider"`,
		} {
			if !strings.Contains(text, want) {
				t.Errorf("%s missing insert-phase contract %q", path, want)
			}
		}
	}

	claudeBody := normalizeCommandWrapper(contents[1])
	opencodeBody := normalizeCommandWrapper(contents[2])
	if claudeBody != opencodeBody {
		t.Fatal("insert-phase wrapper body drift between Claude and OpenCode")
	}
	for _, forbidden := range []string{"derivePhaseInsertName", "resolvePhaseInsertRequest", "COLONY_STATE.json"} {
		if strings.Contains(claudeBody, forbidden) {
			t.Errorf("managed wrappers contain Go-owned implementation detail %q", forbidden)
		}
	}
}

func TestBumpVersionWrapperPreservesReleaseFollowUp(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	for _, platformDir := range []string{".claude/commands/ant", ".opencode/commands/ant"} {
		wrapperPath := filepath.Join(repoRoot, platformDir, "bump-version.md")
		content, err := os.ReadFile(wrapperPath)
		if err != nil {
			t.Fatalf("read %s: %v", wrapperPath, err)
		}
		text := string(content)
		for _, want := range []string{
			"AETHER_OUTPUT_MODE=visual aether bump-version",
			"go test ./...",
			"aether publish --channel stable",
			"git commit -m",
			"git tag v<new_version>",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("%s missing release follow-up %q", wrapperPath, want)
			}
		}
	}
}

func TestTunnelsWrapperDocumentsRuntimeCompareAndImport(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	for _, platformDir := range []string{".claude/commands/ant", ".opencode/commands/ant"} {
		wrapperPath := filepath.Join(repoRoot, platformDir, "tunnels.md")
		content, err := os.ReadFile(wrapperPath)
		if err != nil {
			t.Fatalf("read %s: %v", wrapperPath, err)
		}
		text := string(content)
		for _, want := range []string{
			"AETHER_OUTPUT_MODE=visual aether tunnels",
			"two chambers: side-by-side comparison",
			"--import-signals",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("%s missing restored tunnels behavior %q", wrapperPath, want)
			}
		}
	}
}

func TestWrappersDoNotHandRollNextUpWithJQ(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	for _, platformDir := range []string{".claude/commands/ant", ".opencode/commands/ant"} {
		wrapperDir := filepath.Join(repoRoot, platformDir)
		entries, err := os.ReadDir(wrapperDir)
		if err != nil {
			t.Fatalf("read %s: %v", wrapperDir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
				continue
			}
			wrapperPath := filepath.Join(wrapperDir, entry.Name())
			content, err := os.ReadFile(wrapperPath)
			if err != nil {
				t.Fatalf("read %s: %v", wrapperPath, err)
			}
			text := string(content)
			for _, forbidden := range []string{"state=$(jq", "current_phase=$(jq", "total_phases=$(jq"} {
				if strings.Contains(text, forbidden) {
					t.Errorf("%s still hand-rolls Next Up with %q", wrapperPath, forbidden)
				}
			}
			if strings.Contains(text, "aether print-next-up") && !strings.Contains(text, "AETHER_OUTPUT_MODE=visual aether print-next-up") {
				t.Errorf("%s invokes print-next-up without visual mode", wrapperPath)
			}
		}
	}
}

func TestRestoredVisualCommandsHaveEmojiEntries(t *testing.T) {
	for _, command := range []string{
		"closeout",
		"print-next-up",
		"export-signals",
		"import-signals",
		"reference-index",
		"reference-list",
		"reference-match",
		"shelf-list",
		"shelf-add",
		"shelf-promote",
		"shelf-dismiss",
		"queen-init",
		"queen-read",
		"queen-promote",
		"queen-thresholds",
		"queen-compose",
		"porter",
	} {
		if emoji := strings.TrimSpace(commandEmojiMap[command]); emoji == "" {
			t.Errorf("commandEmojiMap[%q] missing", command)
		}
	}
}
