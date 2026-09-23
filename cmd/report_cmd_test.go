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

// TestReportBundleRedactsASecretValueInFailureText is CR-03's own
// regression guard (208-REVIEW.md): the sanitizers a midden entry's
// Message already passes through (colony.SanitizeSignalContent /
// colony.NeutralizeForRecord) catch prompt-injection and shell-injection
// shapes and secrets-file PATHS, never a literal secret VALUE a failing
// build or test genuinely printed to stdout/stderr. This seeds exactly
// that shape -- a fake API key embedded in captured failure output -- and
// asserts it never reaches the bundle `aether report` writes for a third
// party to read.
func TestReportBundleRedactsASecretValueInFailureText(t *testing.T) {
	saveGlobals(t)
	repo := bindCommandTestRepository(t)

	goal := "Prove secret values never reach the report bundle"
	createTestColonyState(t, repo.DataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	fakeKey := "sk-livefakekey1234567890abcdef"
	middenJSON := `{"entries":[{"id":"secret-1","timestamp":"2026-09-22T12:00:00Z","category":"test","source":"test","message":"test failed: auth rejected using ` + fakeKey + `","reviewed":false}]}`
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
	body := string(data)
	if strings.Contains(body, fakeKey) {
		t.Fatalf("expected the report bundle to never carry the secret value from a failure record, found it in:\n%s", body)
	}
	if !strings.Contains(body, "auth rejected using") {
		t.Errorf("expected the rest of the failure message to survive redaction, got:\n%s", body)
	}
	if !strings.Contains(body, "secret") {
		t.Errorf("expected the bundle's own \"What this is\" section to warn that raw output may still carry a secret, got:\n%s", body)
	}
}

// TestApplyReportOutputOverrideMovesTheBundle proves the happy path: a
// valid --output directory receives the bundle, and the returned path
// points at it with no warning.
func TestApplyReportOutputOverrideMovesTheBundle(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "report-source.md")
	if err := os.WriteFile(src, []byte("bundle body"), 0644); err != nil {
		t.Fatalf("seed source bundle: %v", err)
	}
	outputDir := filepath.Join(root, "wherever")

	finalPath, warning := applyReportOutputOverride(root, outputDir, src)
	if warning != "" {
		t.Fatalf("expected no warning for a valid --output directory, got: %q", warning)
	}
	wantPath := filepath.Join(outputDir, "report-source.md")
	if finalPath != wantPath {
		t.Fatalf("finalPath = %q, want %q", finalPath, wantPath)
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("expected the bundle to exist at %q: %v", wantPath, err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("expected the source bundle to be moved (gone), stat err = %v", err)
	}
}

// TestApplyReportOutputOverrideWarnsWhenMkdirFails is WR-02's own
// regression guard (208-REVIEW.md): before this fix, a failed MkdirAll or
// Rename fell through silently -- the command reported success and printed
// the DEFAULT path as if --output had been honored, with nothing telling
// the owner their explicit flag was ignored. This forces MkdirAll to fail
// (--output names a path THROUGH an existing regular file, which cannot
// have a directory created inside it) and asserts a warning is returned
// naming the real, unmoved path -- and that the source bundle is left
// exactly where it was, never silently dropped.
func TestApplyReportOutputOverrideWarnsWhenMkdirFails(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "report-source.md")
	if err := os.WriteFile(src, []byte("bundle body"), 0644); err != nil {
		t.Fatalf("seed source bundle: %v", err)
	}

	blocker := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatalf("seed blocking file: %v", err)
	}
	outputDir := filepath.Join(blocker, "sub")

	finalPath, warning := applyReportOutputOverride(root, outputDir, src)
	if warning == "" {
		t.Fatal("expected a warning naming the failed --output move, got none")
	}
	if !strings.Contains(warning, src) {
		t.Errorf("expected the warning to name the real, unmoved path %q, got: %q", src, warning)
	}
	if finalPath != src {
		t.Fatalf("finalPath = %q, want the unmoved source path %q", finalPath, src)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("expected the source bundle to remain at %q, stat err = %v", src, err)
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
