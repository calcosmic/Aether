package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestReportBundleNamesWhatTheOwnerNeedsToSend runs the real
// runReportBundle against a seeded store and asserts the written file
// contains all six section headings, the version string, at least one
// recorded refusal row carrying its next command, and the next command
// from resolveNextAction.
func TestReportBundleNamesWhatTheOwnerNeedsToSend(t *testing.T) {
	saveGlobals(t)
	repo := bindCommandTestRepository(t)

	goal := "Ship the report bundle"
	createTestColonyState(t, repo.DataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})
	currentStreamingCommand = "report"
	appendRefusalToLog(refuse("colonize-finalize-missing-timestamp"))

	path, err := runReportBundle(repo.Root)
	if err != nil {
		t.Fatalf("runReportBundle: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report bundle: %v", err)
	}
	body := string(data)

	for _, heading := range []string{
		"## What this is",
		"## Version",
		"## Where the project is",
		"## What Aether refused",
		"## What went wrong",
		"## What is still open",
	} {
		if !strings.Contains(body, heading) {
			t.Errorf("expected bundle to contain heading %q, got:\n%s", heading, body)
		}
	}
	if !strings.Contains(body, resolveVersion()) {
		t.Errorf("expected bundle to contain the version string %q", resolveVersion())
	}
	if !strings.Contains(body, "aether colonize --plan-only --force-resurvey") {
		t.Errorf("expected bundle to name the recorded refusal's next command, got:\n%s", body)
	}
	answer := resolveNextAction(loadNextActionInputForCommand("report"))
	if !strings.Contains(body, answer.Command) {
		t.Errorf("expected bundle to carry the shared next-action command %q, got:\n%s", answer.Command, body)
	}
}

// TestReportBundleFilenamesNeverCollide calls runReportBundle twice with
// the clock pinned to one instant and asserts two distinct paths exist,
// both non-empty.
func TestReportBundleFilenamesNeverCollide(t *testing.T) {
	saveGlobals(t)
	repo := bindCommandTestRepository(t)

	originalNow := reportBundleNow
	pinned := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	reportBundleNow = func() time.Time { return pinned }
	t.Cleanup(func() { reportBundleNow = originalNow })

	first, err := runReportBundle(repo.Root)
	if err != nil {
		t.Fatalf("first runReportBundle: %v", err)
	}
	second, err := runReportBundle(repo.Root)
	if err != nil {
		t.Fatalf("second runReportBundle: %v", err)
	}
	if first == second {
		t.Fatalf("expected two distinct paths, got the same path twice: %s", first)
	}
	for _, path := range []string{first, second} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
		if info.Size() == 0 {
			t.Errorf("expected %s to be non-empty", path)
		}
	}
}

// TestReportBundleWithoutAProject runs it in an empty temp dir and asserts
// exit is success, the file exists, and its body names `aether init`.
func TestReportBundleWithoutAProject(t *testing.T) {
	saveGlobals(t)
	store = nil
	root := t.TempDir()

	path, err := runReportBundle(root)
	if err != nil {
		t.Fatalf("expected success with no project, got error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report bundle: %v", err)
	}
	if !strings.Contains(string(data), "aether init") {
		t.Errorf("expected bundle body to name `aether init`, got:\n%s", string(data))
	}
}

// TestReportBundleIsValidUTF8ForEveryRecord seeds a refusal-log line and a
// failure record each carrying an invalid byte sequence and asserts
// utf8.Valid on the whole written file.
func TestReportBundleIsValidUTF8ForEveryRecord(t *testing.T) {
	saveGlobals(t)
	repo := bindCommandTestRepository(t)

	goal := "Prove the bundle stays valid UTF-8"
	createTestColonyState(t, repo.DataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	badRefusalLine := "{\"schema_version\":1,\"recorded_at\":\"2026-09-22T12:00:00Z\",\"id\":\"bad-bytes\",\"what\":\"bad \xff byte\",\"next_command\":\"aether patrol\",\"protects_work\":false,\"command\":\"report\"}\n"
	if err := appendRawLine(filepath.Join(repo.DataDir, refusalLogPath), badRefusalLine); err != nil {
		t.Fatalf("seed refusal log: %v", err)
	}

	middenJSON := "{\"entries\":[{\"id\":\"bad-1\",\"timestamp\":\"2026-09-22T12:00:00Z\",\"category\":\"bad\",\"source\":\"test\",\"message\":\"bad \xff byte\",\"reviewed\":false}]}"
	if err := os.WriteFile(filepath.Join(repo.DataDir, "midden.json"), []byte(middenJSON), 0644); err != nil {
		t.Fatalf("seed midden file: %v", err)
	}

	path, err := runReportBundle(repo.Root)
	if err != nil {
		t.Fatalf("runReportBundle: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report bundle: %v", err)
	}
	if !utf8.Valid(data) {
		t.Fatalf("expected the written report bundle to be valid UTF-8")
	}
}

// TestReportBundleOrdersRecordsNewestFirst seeds three refusals, two
// sharing a timestamp, and asserts the newest is first and the two
// equal-timestamp rows appear in the order they were appended.
func TestReportBundleOrdersRecordsNewestFirst(t *testing.T) {
	saveGlobals(t)
	repo := bindCommandTestRepository(t)

	goal := "Prove newest-first ordering"
	createTestColonyState(t, repo.DataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	path := filepath.Join(repo.DataDir, refusalLogPath)
	lines := []string{
		`{"schema_version":1,"recorded_at":"2026-09-22T10:00:00Z","id":"oldest","what":"oldest refusal","next_command":"aether patrol","protects_work":false,"command":"report"}`,
		`{"schema_version":1,"recorded_at":"2026-09-22T11:00:00Z","id":"tie-first","what":"tie refusal written first","next_command":"aether patrol","protects_work":false,"command":"report"}`,
		`{"schema_version":1,"recorded_at":"2026-09-22T11:00:00Z","id":"tie-second","what":"tie refusal written second","next_command":"aether patrol","protects_work":false,"command":"report"}`,
	}
	if err := appendRawLine(path, strings.Join(lines, "\n")+"\n"); err != nil {
		t.Fatalf("seed refusal log: %v", err)
	}

	bundlePath, err := runReportBundle(repo.Root)
	if err != nil {
		t.Fatalf("runReportBundle: %v", err)
	}
	data, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read report bundle: %v", err)
	}
	body := string(data)

	idxTieFirst := strings.Index(body, "tie refusal written first")
	idxTieSecond := strings.Index(body, "tie refusal written second")
	idxOldest := strings.Index(body, "oldest refusal")
	if idxTieFirst == -1 || idxTieSecond == -1 || idxOldest == -1 {
		t.Fatalf("expected all three seeded refusals to appear in the bundle, got:\n%s", body)
	}
	if !(idxTieFirst < idxOldest && idxTieSecond < idxOldest) {
		t.Errorf("expected the newest (tied) entries before the oldest entry:\n%s", body)
	}
	if idxTieFirst > idxTieSecond {
		t.Errorf("expected equal-timestamp rows to keep append order (first written, first shown), got second before first:\n%s", body)
	}
}

// TestReportBundleCarriesNoEnvironmentSecrets sets a uniquely-valued
// environment variable before the call and asserts that value does not
// appear in the written file.
func TestReportBundleCarriesNoEnvironmentSecrets(t *testing.T) {
	saveGlobals(t)
	repo := bindCommandTestRepository(t)

	secret := "sk-report-bundle-secret-9f3ae7d1"
	t.Setenv("AETHER_REPORT_TEST_SECRET", secret)

	goal := "Prove no environment leak"
	createTestColonyState(t, repo.DataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	path, err := runReportBundle(repo.Root)
	if err != nil {
		t.Fatalf("runReportBundle: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read report bundle: %v", err)
	}
	if strings.Contains(string(data), secret) {
		t.Fatalf("expected the report bundle to never carry an environment variable value, found the secret in:\n%s", string(data))
	}
}

// appendRawLine writes raw content, verbatim, appended to path -- used to
// seed a JSONL line (or a whole raw JSON file body) carrying bytes
// encoding/json's own Marshal would otherwise sanitize away, so a test can
// prove the report bundle stays valid UTF-8 even when the underlying data
// did not.
func appendRawLine(path, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(content)
	return err
}
