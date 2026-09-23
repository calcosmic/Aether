package cmd

// 208-11-PLAN.md (UED-10, D-03 "the program does it itself when nobody is
// there"): proves attemptRefusalSelfRecovery (cmd/refusal_self_recovery.go)
// end to end through the real `colonize --plan-only` command (Task 1), and
// the guards that keep the recovery honest (Task 2): attended behaviour
// untouched, one decision shared by both colonize sites, and only a
// refusal that has declared itself recoverable is ever recovered from.

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
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

// TestAttendedColonizeStillStopsAndAsks (208-11-PLAN.md Task 2, D-03's
// attended constraint): with the is-anyone-here fact unset -- the ordinary,
// attended case -- both colonize sites still stop and still name their own
// next command, byte-identical to the refusal 208-09 already proved. One
// subtest per lane.
func TestAttendedColonizeStillStopsAndAsks(t *testing.T) {
	row, ok := refusalForID("colonize-existing-survey-found")
	if !ok {
		t.Fatal("refusalRegistry needs a colonize-existing-survey-found row for this test")
	}

	t.Run("plan-only lane", func(t *testing.T) {
		setupColonizeExistingSurveyFixture(t)
		// Deliberately not setting unattendedEnvVar -- the ordinary case.
		t.Setenv("AETHER_OUTPUT_MODE", "json")

		rootCmd.SetArgs([]string{"colonize", "--plan-only"})
		if execErr := rootCmd.Execute(); execErr != nil {
			t.Fatalf("colonize --plan-only returned error: %v", execErr)
		}
		rendered := captureStdoutBuffer(t).String() + stderr.(*bytes.Buffer).String()
		if !strings.Contains(rendered, row.What) || !strings.Contains(rendered, row.NextCommand) {
			t.Fatalf("attended plan-only colonize did not stop and name its next command:\n%s", rendered)
		}
		for _, entry := range refusalLogEntries(50) {
			if entry.ID == row.ID && entry.Recovered {
				t.Fatalf("an attended run must never be marked recovered: %+v", entry)
			}
		}
	})

	t.Run("direct lane", func(t *testing.T) {
		setupColonizeExistingSurveyFixture(t)
		t.Setenv("AETHER_OUTPUT_MODE", "json")

		rootCmd.SetArgs([]string{"colonize"})
		if execErr := rootCmd.Execute(); execErr != nil {
			t.Fatalf("colonize returned error: %v", execErr)
		}
		rendered := captureStdoutBuffer(t).String() + stderr.(*bytes.Buffer).String()
		if !strings.Contains(rendered, row.What) || !strings.Contains(rendered, row.NextCommand) {
			t.Fatalf("attended direct colonize did not stop and name its next command:\n%s", rendered)
		}
		for _, entry := range refusalLogEntries(50) {
			if entry.ID == row.ID && entry.Recovered {
				t.Fatalf("an attended run must never be marked recovered: %+v", entry)
			}
		}
	})
}

// TestUnattendedDirectColonizeRefreshesTheMapItself (208-11-PLAN.md Task 2):
// the direct-lane recovery proof. An unattended run of the direct
// (non-plan-only) lane against an existing survey carries on and rewrites
// the survey rather than returning the refusal.
func TestUnattendedDirectColonizeRefreshesTheMapItself(t *testing.T) {
	row, ok := refusalForID("colonize-existing-survey-found")
	if !ok {
		t.Fatal("refusalRegistry needs a colonize-existing-survey-found row for this test")
	}

	dataDir := filepath.Join(setupColonizeExistingSurveyFixture(t), ".aether", "data")
	t.Setenv(unattendedEnvVar, "1")
	t.Setenv("AETHER_OUTPUT_MODE", "json")

	before, err := os.Stat(filepath.Join(dataDir, "survey", "PROVISIONS.md"))
	if err != nil {
		t.Fatalf("stat pre-existing survey artifact: %v", err)
	}

	rootCmd.SetArgs([]string{"colonize"})
	if execErr := rootCmd.Execute(); execErr != nil {
		t.Fatalf("colonize returned error: %v", execErr)
	}

	rawOutput := captureStdoutBuffer(t).String()
	var envelope struct {
		OK     bool                   `json:"ok"`
		Result map[string]interface{} `json:"result"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(rawOutput)), &envelope); err != nil {
		t.Fatalf("machine-readable output did not parse as JSON: %v\noutput: %s", err, rawOutput)
	}
	if !envelope.OK {
		t.Fatalf("expected the unattended direct lane to carry on rather than refuse, output: %s", rawOutput)
	}
	forced, isBool := envelope.Result["force_resurvey"].(bool)
	if !isBool || !forced {
		t.Fatalf("expected result.force_resurvey=true, got %#v\noutput: %s", envelope.Result["force_resurvey"], rawOutput)
	}

	after, err := os.Stat(filepath.Join(dataDir, "survey", "PROVISIONS.md"))
	if err != nil {
		t.Fatalf("stat post-run survey artifact: %v", err)
	}
	if !after.ModTime().After(before.ModTime()) && after.Size() == before.Size() {
		t.Fatalf("expected the direct lane to rewrite the survey artifact on disk; before=%v after=%v", before, after)
	}

	recovered := oneRecoveredLogEntry(t, row.ID)
	if !recovered.Recovered {
		t.Fatalf("expected the log record for %q to be marked recovered, got: %+v", row.ID, recovered)
	}
}

// TestSelfRecoveryHasOneDecision (208-11-PLAN.md Task 2): the one-decision
// guard. It walks every non-test Go file in the module and fails by name if
// any file other than cmd/refusal_self_recovery.go names the opt-in map, or
// if any file other than that one both reads the is-anyone-here fact
// (through sessionHasNoOneToAsk, the identifier this repository's existing
// TestTheIsAnyoneHereFactHasOneReader already confines to
// cmd/unattended_session.go) and acts on a refusal (calls refuse(...)) --
// the second-copy failure mode this repository has hit before. It cannot
// pass vacuously: it fails if the walk finds too few files, or if it never
// actually visits the one file it expects to allow.
//
// What this cannot catch, stated honestly rather than claimed away: a
// second decision expressed without ever naming refusalSelfRecoveryTable or
// calling both sessionHasNoOneToAsk and refuse in the same file -- for
// example, a helper that always returns a hard-coded true. Nothing in this
// repository writes a self-recovery decision that way.
func TestSelfRecoveryHasOneDecision(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	const soleDecisionFile = "cmd/refusal_self_recovery.go"

	var goFiles []string
	walkErr := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", "vendor", "node_modules", "testdata", ".planning", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		goFiles = append(goFiles, path)
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk %s: %v", repoRoot, walkErr)
	}
	if len(goFiles) < 50 {
		t.Fatalf("only %d non-test .go files found under %s -- the walk is not reaching the module, so this guard would pass vacuously", len(goFiles), repoRoot)
	}

	var sawSoleDecisionFile bool
	var offenders []string
	fset := token.NewFileSet()
	for _, path := range goFiles {
		rel, relErr := filepath.Rel(repoRoot, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		if rel == soleDecisionFile {
			sawSoleDecisionFile = true
			continue
		}
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", rel, parseErr)
		}

		var namesTheMap bool
		var callsSessionHasNoOneToAsk bool
		var callsRefuse bool
		ast.Inspect(file, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.Ident:
				if node.Name == "refusalSelfRecoveryTable" {
					namesTheMap = true
				}
				if node.Name == "sessionHasNoOneToAsk" {
					callsSessionHasNoOneToAsk = true
				}
			case *ast.CallExpr:
				if fn, ok := node.Fun.(*ast.Ident); ok && fn.Name == "refuse" {
					callsRefuse = true
				}
			}
			return true
		})

		var reasons []string
		if namesTheMap {
			reasons = append(reasons, "names refusalSelfRecoveryTable")
		}
		if callsSessionHasNoOneToAsk && callsRefuse {
			reasons = append(reasons, "both calls sessionHasNoOneToAsk and refuse(...) in the same file")
		}
		if len(reasons) > 0 {
			offenders = append(offenders, rel+" ("+strings.Join(reasons, "; ")+")")
		}
	}

	if !sawSoleDecisionFile {
		t.Fatalf("%s was never visited by the walk -- this guard cannot be passing for the right reason", soleDecisionFile)
	}
	if len(offenders) > 0 {
		t.Fatalf("only %s may hold a self-recovery decision; found a second one also in: %v", soleDecisionFile, offenders)
	}
}

// TestOnlyASafeRefusalCanRecoverItself (208-11-PLAN.md Task 2): the opt-in
// contract guard. For every id in refusalSelfRecoveryTable: it exists in
// the one refusal table, its disposition is still a stop, it still
// protects work, and its next command is still present and carries no
// unsubstituted placeholder. The map must be non-empty, so this cannot pass
// vacuously. It then proves it can genuinely fail, against a locally
// constructed table -- never the real registry -- in which one row has
// been downgraded.
func TestOnlyASafeRefusalCanRecoverItself(t *testing.T) {
	if len(refusalSelfRecoveryTable) == 0 {
		t.Fatal("refusalSelfRecoveryTable must not be empty -- this guard would pass vacuously")
	}
	if problems := refusalSelfRecoveryContractProblems(refusalSelfRecoveryTable); len(problems) > 0 {
		t.Fatalf("the real refusalSelfRecoveryTable violates its own contract: %v", problems)
	}

	// Prove the guard bites: a locally built table naming a row that does
	// not protect work should be reported by name, without ever touching
	// the real registry.
	downgraded := map[string]string{
		"colonize-finalize-timestamp-in-future": "a row that does not protect work must never be offered self-recovery",
	}
	problems := refusalSelfRecoveryContractProblems(downgraded)
	if len(problems) == 0 {
		t.Fatal("expected the contract check to report a problem for a non-work-protecting row, got none")
	}
	found := false
	for _, p := range problems {
		if strings.Contains(p, "colonize-finalize-timestamp-in-future") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the offending row to be named by id, got: %v", problems)
	}
}

// refusalSelfRecoveryContractProblems checks every id in table against the
// one refusal registry: it must exist, stay a "stop" disposition, protect
// work, and carry a next command with no unsubstituted `<...>` placeholder.
func refusalSelfRecoveryContractProblems(table map[string]string) []string {
	var problems []string
	for id := range table {
		row, ok := refusalForID(id)
		if !ok {
			problems = append(problems, id+": not a registered refusal row")
			continue
		}
		if row.Disposition != "stop" {
			problems = append(problems, id+": disposition is "+row.Disposition+", not \"stop\"")
		}
		if !row.ProtectsWork {
			problems = append(problems, id+": does not protect work")
		}
		next := strings.TrimSpace(row.NextCommand)
		if next == "" {
			problems = append(problems, id+": has no next command")
		} else if strings.ContainsAny(next, "<>") {
			problems = append(problems, id+": next command still carries an unsubstituted placeholder: "+next)
		}
	}
	return problems
}
