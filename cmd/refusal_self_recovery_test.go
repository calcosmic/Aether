package cmd

// 208-11-PLAN.md Task 1 (UED-10, D-03 "the program does it itself when
// nobody is there"): proves attemptRefusalSelfRecovery
// (cmd/refusal_self_recovery.go) end to end through the real
// `colonize --plan-only` command -- an unattended session meeting an
// existing survey carries on with a forced re-survey, says so only in
// visual mode, and records the recovery in the refusal log, rather than
// stopping.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// setupColonizeExistingSurveyFixture binds a fresh command-test repository
// with an existing survey document on disk -- the same fixture shape
// refusalDrivers["colonize-existing-survey-found"] (cmd/refusal_behaviour_test.go)
// uses to drive this row's real call site, reused rather than re-invented.
func setupColonizeExistingSurveyFixture(t *testing.T) string {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/aether-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "survey"), 0755); err != nil {
		t.Fatalf("create survey dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "survey", "PROVISIONS.md"), []byte("# old survey\n"), 0644); err != nil {
		t.Fatalf("write old survey: %v", err)
	}
	goal := "Survey the repo"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY,
		Plan: colony.Plan{Phases: []colony.Phase{}},
	})
	return root
}

// captureStdoutBuffer returns the *bytes.Buffer the current test binding's
// stdout points at -- setupBuildFlowTest/bindCommandTestRepository install
// one -- failing loudly if that contract ever changes rather than silently
// asserting against the real os.Stdout.
func captureStdoutBuffer(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf, ok := stdout.(*bytes.Buffer)
	if !ok {
		t.Fatalf("stdout is not a *bytes.Buffer in this test binding: %T", stdout)
	}
	return buf
}

// oneRecoveredLogEntry asserts the refusal log carries exactly one record
// for id and returns it.
func oneRecoveredLogEntry(t *testing.T, id string) refusalLogEntry {
	t.Helper()
	var found []refusalLogEntry
	for _, entry := range refusalLogEntries(50) {
		if entry.ID == id {
			found = append(found, entry)
		}
	}
	if len(found) != 1 {
		t.Fatalf("expected exactly 1 refusal-log record for %q, got %d: %+v", id, len(found), found)
	}
	return found[0]
}

// TestNoOneHereMeansAetherRefreshesTheMapItself is the end-to-end proof
// (208-11-PLAN.md Task 1): an unattended `colonize --plan-only` meeting an
// existing survey carries on with a forced re-survey instead of stopping,
// its machine-readable output stays valid JSON with no notice leaked into
// it, a visual-mode run names the exact command Aether ran, and the refusal
// log carries exactly one record for this id marked recovered.
func TestNoOneHereMeansAetherRefreshesTheMapItself(t *testing.T) {
	row, ok := refusalForID("colonize-existing-survey-found")
	if !ok {
		t.Fatal("refusalRegistry needs a colonize-existing-survey-found row for this test")
	}

	t.Run("machine-readable output stays valid JSON and reports the forced re-survey", func(t *testing.T) {
		setupColonizeExistingSurveyFixture(t)
		t.Setenv(unattendedEnvVar, "1")
		t.Setenv("AETHER_OUTPUT_MODE", "json")

		rootCmd.SetArgs([]string{"colonize", "--plan-only"})
		if execErr := rootCmd.Execute(); execErr != nil {
			t.Fatalf("colonize --plan-only returned error: %v", execErr)
		}

		rawOutput := captureStdoutBuffer(t).String()

		// json.Unmarshal on the exact trimmed byte slice -- not merely
		// finding a JSON object somewhere inside it -- fails if anything
		// else (a banner, a notice) was also written to stdout, since
		// trailing non-JSON bytes make the whole parse fail.
		var envelope struct {
			OK     bool                   `json:"ok"`
			Result map[string]interface{} `json:"result"`
		}
		trimmed := strings.TrimSpace(rawOutput)
		if err := json.Unmarshal([]byte(trimmed), &envelope); err != nil {
			t.Fatalf("machine-readable output did not parse as JSON (nothing else may be written to stdout in this mode): %v\noutput: %s", err, rawOutput)
		}
		if !envelope.OK {
			t.Fatalf("expected ok:true, output: %s", rawOutput)
		}
		forced, isBool := envelope.Result["force_resurvey"].(bool)
		if !isBool || !forced {
			t.Fatalf("expected result.force_resurvey=true, got %#v\noutput: %s", envelope.Result["force_resurvey"], rawOutput)
		}

		for _, leak := range []string{"Carrying On Without You", unattendedGuidanceSentence} {
			if strings.Contains(rawOutput, leak) {
				t.Fatalf("the recovery notice leaked into machine-readable output (found %q):\n%s", leak, rawOutput)
			}
		}

		recovered := oneRecoveredLogEntry(t, row.ID)
		if !recovered.Recovered {
			t.Fatalf("expected the log record for %q to be marked recovered, got: %+v", row.ID, recovered)
		}
	})

	t.Run("visual output names the exact command Aether carried out", func(t *testing.T) {
		setupColonizeExistingSurveyFixture(t)
		t.Setenv(unattendedEnvVar, "1")
		t.Setenv("AETHER_OUTPUT_MODE", "visual")
		// Pinned so this test's expectation is derived deterministically
		// rather than depending on ambient host signals detectPlatform()
		// reads (codex.DetectActivePlatform et al). writeVisualOutput
		// rewrites an `aether <verb>` mention for the detected platform
		// (translateHintCommandsForPlatform) -- the same rewrite is applied
		// below to derive what the notice must contain, still read from the
		// refusal's own row.NextCommand, never a hand-typed literal.
		t.Setenv("AETHER_PLATFORM", "claude")

		rootCmd.SetArgs([]string{"colonize", "--plan-only"})
		if execErr := rootCmd.Execute(); execErr != nil {
			t.Fatalf("colonize --plan-only returned error: %v", execErr)
		}

		rendered := captureStdoutBuffer(t).String()
		want := translateHintCommandsForPlatform(row.NextCommand, "claude")
		if !strings.Contains(rendered, want) {
			t.Fatalf("visual notice does not name the refusal's own next command (rendered as %q for this platform):\n%s", want, rendered)
		}
	})
}

// TestOldShapedRefusalLogRecordStillReadsAsNotRecovered (208-11-PLAN.md
// Task 1 acceptance criterion): a refusal-log line written before the
// Recovered field existed -- no "recovered" key at all -- still parses and
// reads as not recovered, proving the additive field (cmd/refusal_log.go)
// is safe against every record already on disk. No schema-version bump was
// needed for this, and this test is what proves that decision was correct.
func TestOldShapedRefusalLogRecordStillReadsAsNotRecovered(t *testing.T) {
	saveGlobals(t)
	repo := bindCommandTestRepository(t)

	oldShaped := `{"schema_version":1,"recorded_at":"2026-01-01T00:00:00Z","id":"colonize-existing-survey-found",` +
		`"what":"A territory survey already exists for this project.",` +
		`"next_command":"aether colonize --force-resurvey","protects_work":true,"command":"colonize"}` + "\n"
	if err := os.WriteFile(filepath.Join(repo.DataDir, refusalLogPath), []byte(oldShaped), 0644); err != nil {
		t.Fatalf("write old-shaped refusal log: %v", err)
	}

	entries := refusalLogEntries(10)
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 entry from the old-shaped line, got %d: %+v", len(entries), entries)
	}
	if entries[0].Recovered {
		t.Fatalf("an old-shaped record with no recovered key must read as not recovered, got: %+v", entries[0])
	}
	if entries[0].ID != "colonize-existing-survey-found" || entries[0].NextCommand == "" {
		t.Fatalf("old-shaped record did not parse its other fields correctly: %+v", entries[0])
	}
}
