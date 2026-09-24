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
	"regexp"
	"sort"
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

		// WR-07 (208-REVIEW-GAP2.md): anchor on the notice itself as well --
		// its own banner heading (read from the real, unmutated renderBanner
		// helper, not retyped as a copy of its formatted output) and its
		// no-one-is-here line -- so this assertion cannot pass on unrelated
		// colonize output that merely happens to mention the same command
		// string.
		bannerHeading := strings.TrimRight(renderBanner("🙅", "Carrying On Without You"), "\n")
		if !strings.Contains(rendered, bannerHeading) {
			t.Fatalf("the self-recovery notice's own banner heading never appeared:\n%s", rendered)
		}
		if !strings.Contains(rendered, "No one is here to answer") {
			t.Fatalf("the self-recovery notice's own no-one-is-here line never appeared:\n%s", rendered)
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
		// WR-04 (208-REVIEW-GAP2.md): pinned explicitly to empty -- a person
		// is present -- rather than left unset. The attended guarantee must
		// not depend on the ambient shell/harness never having exported the
		// unattended value (the journey gate's own unattended walks do
		// exactly that on their `claude` children); this is the only proof
		// of that guarantee, so it must not turn red for an environmental
		// reason indistinguishable from a real regression.
		t.Setenv(unattendedEnvVar, "")
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
		// WR-04 (208-REVIEW-GAP2.md): see the plan-only lane's comment above.
		t.Setenv(unattendedEnvVar, "")
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

// mayHoldASelfRecoveryDecision is the closed, checked-in allow-list of
// specific reviewed places -- not whole files -- permitted to read the
// is-anyone-here fact (sessionHasNoOneToAsk) or name the opt-in
// self-recovery table (refusalSelfRecoveryTable) today. Each key is
// "<repo-relative file>::<enclosing declaration name>": the file a person
// actually reviewed, and the exact function or top-level declaration inside
// it the read sits in. TestSelfRecoveryHasOneDecision fails by name --
// naming both the file and the enclosing declaration -- on any read of
// either identifier that does not land inside one of these five places.
//
// This replaces 208-13's file-keyed mayReadTheIsAnyoneHereFact
// (208-18-PLAN.md CR-02 / 208-REVIEW-GAP3.md): excusing a whole file moved
// the blind spot from an unreviewed file into a reviewed file at an
// unreviewed line -- a rival "carry on without asking" decision planted
// anywhere else inside cmd/refusal.go (a file the old, file-keyed guard
// exempted wholesale) sailed straight through unnoticed. Keying by place
// instead of file closes that: only the five specific, already-reviewed
// reads named below are excused, and a rival decision written anywhere else
// in one of these same three files is caught by name, exactly like a rival
// decision in a fourth file always was.
//
// The list may shrink; it must not grow without its own owner ruling -- a
// new entry here is exactly the "second decision" failure mode this guard
// exists to catch, so adding one is a decision for a person, not a reflex
// to make a test pass. An entry that no longer matches anything in the
// current source (the place it named was removed, renamed, or stopped
// reading the fact) fails the test by name rather than sitting there as a
// stale excuse -- see the "excuse list can only shrink honestly" check
// inside TestSelfRecoveryHasOneDecision below.
//
// What this still cannot catch, stated honestly rather than claimed away: a
// second decision expressed without ever naming refusalSelfRecoveryTable or
// calling sessionHasNoOneToAsk -- for example, a helper that always returns
// a hard-coded true. Nothing in this repository writes a self-recovery
// decision that way.
var mayHoldASelfRecoveryDecision = map[string]bool{
	// The definition of the fact itself.
	"cmd/unattended_session.go::sessionHasNoOneToAsk": true,
	// Appends the shared guidance sentence on the one-line refusal lane
	// (208-09-PLAN.md) -- reads the fact only to decide whether to append
	// that sentence, never to decide whether to recover.
	"cmd/refusal.go::Error": true,
	// Appends the same sentence on the drawn-block lane.
	"cmd/refusal.go::renderRefusal": true,
	// The opt-in table's own declaration, which sits outside any function.
	"cmd/refusal_self_recovery.go::refusalSelfRecoveryTable": true,
	// The one decision.
	"cmd/refusal_self_recovery.go::attemptRefusalSelfRecovery": true,
}

// TestSelfRecoveryHasOneDecision (208-11-PLAN.md Task 2, widened by
// 208-13-PLAN.md Task 1, narrowed from files to places by 208-18-PLAN.md
// Task 1): the one-decision guard. It walks every non-test Go file in the
// module and, for each top-level declaration, attributes every read of
// sessionHasNoOneToAsk or refusalSelfRecoveryTable to the declaration it
// sits inside -- a function's own name for a *ast.FuncDecl, or a spec's own
// first declared name for a *ast.GenDecl (a var, const, or type). It fails
// by name -- the file AND the enclosing declaration -- on any such read
// whose "<file>::<declaration>" key is not in mayHoldASelfRecoveryDecision.
// It cannot pass vacuously: it fails if the walk finds too few files, if it
// never actually visits the one decision file it expects to allow, or if
// any excused place in mayHoldASelfRecoveryDecision was never observed
// (a stale excuse).
//
// This walk also skips a "worktrees" directory at any depth (208-18-PLAN.md
// WR-01 / 208-REVIEW-GAP3.md), matching the sibling walk in
// cmd/colonize_snapshot_refresh_test.go and the pre-existing precedent in
// cmd/build_attempt_external_test.go: without it, a leftover linked
// worktree checked out inside this repository (as one genuinely is, right
// now, under .claude/worktrees/) is parsed as if it were part of the tree
// under review, which can produce a spurious failure naming a path nobody
// is actually reviewing, or mask a genuine offender behind noise from a
// stale, unrelated checkout.
//
// What this cannot catch, stated honestly rather than claimed away: a second
// decision expressed without ever naming refusalSelfRecoveryTable or calling
// sessionHasNoOneToAsk -- for example, a helper that always returns a
// hard-coded true. Nothing in this repository writes a self-recovery
// decision that way.
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
			case ".git", "vendor", "node_modules", "testdata", ".planning", "dist", "worktrees":
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
	observed := make(map[string]bool, len(mayHoldASelfRecoveryDecision))
	fset := token.NewFileSet()
	for _, path := range goFiles {
		rel, relErr := filepath.Rel(repoRoot, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		if rel == soleDecisionFile {
			sawSoleDecisionFile = true
		}
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", rel, parseErr)
		}

		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Name == nil {
					continue
				}
				recordSelfRecoveryReads(d, rel, d.Name.Name, observed, &offenders)
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					var enclosing string
					switch s := spec.(type) {
					case *ast.ValueSpec:
						if len(s.Names) == 0 {
							continue
						}
						enclosing = s.Names[0].Name
					case *ast.TypeSpec:
						if s.Name == nil {
							continue
						}
						enclosing = s.Name.Name
					default:
						continue
					}
					recordSelfRecoveryReads(spec, rel, enclosing, observed, &offenders)
				}
			}
		}
	}

	if !sawSoleDecisionFile {
		t.Fatalf("%s was never visited by the walk -- this guard cannot be passing for the right reason", soleDecisionFile)
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Fatalf("a self-recovery decision or the is-anyone-here fact may only be read at the five reviewed places named in mayHoldASelfRecoveryDecision; found a read outside that list: %v", offenders)
	}

	var stale []string
	for key := range mayHoldASelfRecoveryDecision {
		if !observed[key] {
			stale = append(stale, key)
		}
	}
	if len(stale) > 0 {
		sort.Strings(stale)
		t.Fatalf("mayHoldASelfRecoveryDecision names a place that matches nothing in the current source -- an excused place that matches nothing is a stale excuse, and the list may only shrink by the place genuinely going away: %v", stale)
	}
}

// recordSelfRecoveryReads walks node -- a *ast.FuncDecl or a single Spec
// belonging to a *ast.GenDecl -- for every *ast.Ident named
// sessionHasNoOneToAsk or refusalSelfRecoveryTable, and attributes each one
// found to the key "<rel>::<enclosing>". A key present in
// mayHoldASelfRecoveryDecision is marked observed; any other key is
// recorded as an offender naming the file, the enclosing declaration, and
// which identifier it read.
func recordSelfRecoveryReads(node ast.Node, rel string, enclosing string, observed map[string]bool, offenders *[]string) {
	var namesTheMap bool
	var callsSessionHasNoOneToAsk bool
	ast.Inspect(node, func(n ast.Node) bool {
		if ident, ok := n.(*ast.Ident); ok {
			if ident.Name == "refusalSelfRecoveryTable" {
				namesTheMap = true
			}
			if ident.Name == "sessionHasNoOneToAsk" {
				callsSessionHasNoOneToAsk = true
			}
		}
		return true
	})
	if !namesTheMap && !callsSessionHasNoOneToAsk {
		return
	}
	key := rel + "::" + enclosing
	if mayHoldASelfRecoveryDecision[key] {
		observed[key] = true
		return
	}
	var reasons []string
	if namesTheMap {
		reasons = append(reasons, "names refusalSelfRecoveryTable")
	}
	if callsSessionHasNoOneToAsk {
		reasons = append(reasons, "reads the is-anyone-here fact")
	}
	*offenders = append(*offenders, key+" ("+strings.Join(reasons, "; ")+")")
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
	downgraded := map[string]refusalSelfRecoveryRow{
		"colonize-finalize-timestamp-in-future": {
			Reason: "a row that does not protect work must never be offered self-recovery",
			Action: "doing something it should never be offered",
		},
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

	// WR-03 (208-REVIEW-GAP3.md): a row with a Reason but a blank Action
	// must be refused by name too -- a future row cannot be added without
	// supplying its own action wording. Never touches the real registry;
	// the id is taken from the real table's own key rather than typed, so
	// this keeps covering the real opted-in row even if its id ever
	// changes.
	var declaredID string
	for id := range refusalSelfRecoveryTable {
		declaredID = id
		break
	}
	if declaredID == "" {
		t.Fatal("refusalSelfRecoveryTable is empty -- there is no declared id to build this case from")
	}
	blankAction := map[string]refusalSelfRecoveryRow{
		declaredID: {Reason: "a reason is present", Action: "   "},
	}
	blankActionProblems := refusalSelfRecoveryContractProblems(blankAction)
	if len(blankActionProblems) == 0 {
		t.Fatal("expected the contract check to report a problem for a row with a blank action, got none")
	}
	foundBlankAction := false
	for _, p := range blankActionProblems {
		if strings.Contains(p, declaredID) {
			foundBlankAction = true
		}
	}
	if !foundBlankAction {
		t.Fatalf("expected the row with a blank action to be named by id (%s), got: %v", declaredID, blankActionProblems)
	}
}

// refusalSelfRecoveryContractProblems checks every id in table against the
// one refusal registry: it must exist, stay a "stop" disposition, protect
// work, and carry a next command with no unsubstituted `<...>` placeholder.
// It also reports, by id, a row whose own Reason or Action is blank after
// trimming (208-18-PLAN.md Task 2, WR-03 / 208-REVIEW-GAP3.md): a row with
// no wording of its own cannot supply the closing "next" line's clause, so
// it is refused here rather than silently rendering an empty sentence.
func refusalSelfRecoveryContractProblems(table map[string]refusalSelfRecoveryRow) []string {
	var problems []string
	for id, entry := range table {
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
		if strings.TrimSpace(entry.Reason) == "" {
			problems = append(problems, id+": has no reason")
		}
		if strings.TrimSpace(entry.Action) == "" {
			problems = append(problems, id+": has no action")
		}
	}
	return problems
}

// TestOnlyADeclaredRefusalIsEverRecoveredFrom (208-13-PLAN.md Task 1, CR-01 /
// 208-REVIEW-GAP2.md): drives attemptRefusalSelfRecovery directly -- never
// through a cobra command -- covering each of its remaining gates on its
// own, with the session pinned as having nobody present. Every id, command
// and flag is derived from the real refusalRegistry/refusalSelfRecoveryTable
// through refuse/refusalForID; none is typed as a literal, so this test
// cannot drift into a shape the runtime cannot produce.
//
// Before this test existed, deleting the opt-in table lookup, the
// ProtectsWork check and the NextCommand check from attemptRefusalSelfRecovery
// left every named guard in this file green, because every other test that
// exercises the function does so exclusively through the one real registered
// refusal that happens to satisfy all three conditions at once.
func TestOnlyADeclaredRefusalIsEverRecoveredFrom(t *testing.T) {
	setupColonizeExistingSurveyFixture(t)
	t.Setenv(unattendedEnvVar, "1")
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	// Case 1: a registered, work-protecting stop that names a real command
	// but is NOT a key of the opt-in table -- found by scanning the real
	// registry for the first row that qualifies, never by typing an id.
	var notDeclared refusal
	for _, row := range refusalRegistry {
		if _, listed := refusalSelfRecoveryTable[row.ID]; listed {
			continue
		}
		if row.Disposition == "stop" && row.ProtectsWork && strings.TrimSpace(row.NextCommand) != "" {
			notDeclared = refuse(row.ID)
			break
		}
	}
	if notDeclared.ID == "" {
		t.Fatal("no undeclared work-protecting stop row exists in refusalRegistry to drive this guard -- the test cannot proceed honestly")
	}
	assertRefusalNeverRecovered(t, notDeclared, "is not a key of the opt-in table")

	// The one declared (opt-in) id, taken from the table's own key rather
	// than typed -- so this test keeps covering the real opted-in row even
	// if its id ever changes.
	var declaredID string
	for id := range refusalSelfRecoveryTable {
		declaredID = id
		break
	}
	if declaredID == "" {
		t.Fatal("refusalSelfRecoveryTable is empty -- there is no declared row to drive this test with")
	}
	declared := refuse(declaredID)

	// Case 2: the declared row with ProtectsWork forced false on a copy.
	noWork := declared
	noWork.ProtectsWork = false
	assertRefusalNeverRecovered(t, noWork, "does not protect work")

	// Case 3: the declared row with its next command blanked to whitespace
	// on a copy.
	noCommand := declared
	noCommand.NextCommand = "   "
	assertRefusalNeverRecovered(t, noCommand, "names no command")

	// Positive control: the untouched declared row must still be recovered
	// from. Without this, the three negative cases above could be passing
	// only because the function returns false unconditionally.
	if !attemptRefusalSelfRecovery(declared) {
		t.Fatalf("%s is opted in, protects work, and names a real command -- it must be recovered from", declared.ID)
	}
}

// assertRefusalNeverRecovered drives attemptRefusalSelfRecovery with r and
// asserts it returned false, wrote nothing to stdout, and added no
// refusal-log record -- the shape every negative case in
// TestOnlyADeclaredRefusalIsEverRecoveredFrom must have.
func assertRefusalNeverRecovered(t *testing.T, r refusal, because string) {
	t.Helper()
	captureStdoutBuffer(t).Reset()
	before := len(refusalLogEntries(200))
	if attemptRefusalSelfRecovery(r) {
		t.Fatalf("%s %s and must never be recovered from", r.ID, because)
	}
	if got := captureStdoutBuffer(t).String(); got != "" {
		t.Fatalf("a refused self-recovery must print nothing, got:\n%s", got)
	}
	if after := len(refusalLogEntries(200)); after != before {
		t.Fatalf("a refused self-recovery must record nothing: %d -> %d", before, after)
	}
}

// planningDecisionIDPattern matches a planning-decision identifier like
// "D-03" -- internal bookkeeping CLAUDE.md's "READ THIS BEFORE YOU WRITE
// ANYTHING TO THE OWNER" section names as never belonging on the owner's
// screen.
var planningDecisionIDPattern = regexp.MustCompile(`\bD-\d+\b`)

// TestSelfRecoveryNoticeSaysOnlyWhatActuallyHappened (208-13-PLAN.md Task 2,
// WR-02 / 208-REVIEW-GAP2.md): the rendered notice never claims, in the past
// tense, that the named command has already run. That claim is true on the
// direct colonize lane but false on the plan-only lane -- at the moment this
// notice prints, runCodexColonizePlanOnly has only set ForceResurvey and
// gone on to build a manifest a host will later dispatch surveyors from;
// nothing has actually re-surveyed the project yet. Reads the notice through
// the real renderer; never compares against a full copy of the expected
// sentence typed into the test.
func TestSelfRecoveryNoticeSaysOnlyWhatActuallyHappened(t *testing.T) {
	row, ok := refusalForID("colonize-existing-survey-found")
	if !ok {
		t.Fatal("refusalRegistry needs a colonize-existing-survey-found row for this test")
	}
	entry, ok := refusalSelfRecoveryTable[row.ID]
	if !ok {
		t.Fatalf("refusalSelfRecoveryTable needs an entry for %q", row.ID)
	}
	rendered := renderRefusalSelfRecoveryNotice(refuse(row.ID), entry.Reason, entry.Action, row.NextCommand)

	if strings.Contains(rendered, "ran `") || strings.Contains(rendered, "on your behalf") || strings.Contains(rendered, "carried out") {
		t.Fatalf("the notice claims the command has already run, which is false on the plan-only lane at the moment it prints:\n%s", rendered)
	}
	if !strings.Contains(rendered, "going ahead") {
		t.Fatalf("the notice must say Aether is going ahead instead of stopping to ask -- true on both colonize lanes:\n%s", rendered)
	}
}

// TestSelfRecoveryNoticeSpeaksPlainEnglish (208-13-PLAN.md Task 2, WR-03 /
// 208-REVIEW-GAP2.md): the rendered notice carries no planning-decision
// identifier, no planning-directory filename, and no word this repository
// invented left unexplained -- reusing untranslatedRepoWords
// (next_action_card_test.go), the same predicate every other voiced screen
// is checked against, rather than a second, competing definition of plain
// English. Reads the notice through the real renderer; never compares
// against a full copy of the expected sentence typed into the test.
func TestSelfRecoveryNoticeSpeaksPlainEnglish(t *testing.T) {
	row, ok := refusalForID("colonize-existing-survey-found")
	if !ok {
		t.Fatal("refusalRegistry needs a colonize-existing-survey-found row for this test")
	}
	entry, ok := refusalSelfRecoveryTable[row.ID]
	if !ok {
		t.Fatalf("refusalSelfRecoveryTable needs an entry for %q", row.ID)
	}
	rendered := renderRefusalSelfRecoveryNotice(refuse(row.ID), entry.Reason, entry.Action, row.NextCommand)

	if m := planningDecisionIDPattern.FindString(rendered); m != "" {
		t.Fatalf("the notice carries a planning-decision identifier (%q):\n%s", m, rendered)
	}
	if strings.Contains(rendered, ".planning") || strings.Contains(rendered, ".md") {
		t.Fatalf("the notice carries a planning-directory filename:\n%s", rendered)
	}
	if violations := untranslatedRepoWords(rendered); len(violations) > 0 {
		t.Fatalf("the notice uses words this repository invented without explaining them:\n  %s\n\nfull notice:\n%s", strings.Join(violations, "\n  "), rendered)
	}
}

// TestEveryRecoveryRowSuppliesItsOwnActionWording (208-18-PLAN.md Task 2,
// WR-03 / 208-REVIEW-GAP3.md): the notice's closing sentence is built from
// the row that fired, never from a fixed phrase baked into the renderer.
// Two parts, both reading through the real renderer and never comparing
// against a full copy of a sentence typed into the test.
func TestEveryRecoveryRowSuppliesItsOwnActionWording(t *testing.T) {
	// Part 1: every row in the real table supplies its own non-blank
	// Reason and Action, the two answer different questions (so they must
	// not be the same string), and the notice rendered for that row
	// contains both.
	for id, entry := range refusalSelfRecoveryTable {
		reason := strings.TrimSpace(entry.Reason)
		action := strings.TrimSpace(entry.Action)
		if reason == "" {
			t.Fatalf("%s: Reason is blank", id)
		}
		if action == "" {
			t.Fatalf("%s: Action is blank", id)
		}
		if reason == action {
			t.Fatalf("%s: Reason and Action are the same string -- Reason answers why Aether is going ahead, Action answers what it is doing, and collapsing them into one hides that", id)
		}
		rendered := renderRefusalSelfRecoveryNotice(refuse(id), entry.Reason, entry.Action, "aether report")
		if !strings.Contains(rendered, reason) {
			t.Fatalf("%s: rendered notice does not contain its own Reason:\n%s", id, rendered)
		}
		if !strings.Contains(rendered, action) {
			t.Fatalf("%s: rendered notice does not contain its own Action:\n%s", id, rendered)
		}
	}

	// Part 2: the future-second-row property. A locally built row whose
	// Action describes something that is not about rebuilding a map of
	// code at all -- arranging a saved copy of work before carrying on --
	// must show up on screen verbatim when rendered with its own values,
	// and the real colonize row's own Action (read out of the real table
	// at assertion time, never typed as a literal) must NOT leak into a
	// notice rendered for this different row. If the closing clause were
	// still written into the screen as a fixed phrase rather than read
	// from the row that fired, this second assertion would fail.
	row, ok := refusalForID("colonize-existing-survey-found")
	if !ok {
		t.Fatal("refusalRegistry needs a colonize-existing-survey-found row for this test")
	}
	realEntry, ok := refusalSelfRecoveryTable[row.ID]
	if !ok {
		t.Fatalf("refusalSelfRecoveryTable needs an entry for %q", row.ID)
	}
	realAction := strings.TrimSpace(realEntry.Action)
	if realAction == "" {
		t.Fatalf("refusalSelfRecoveryTable's %q entry has a blank Action -- this test cannot prove leakage against an empty string", row.ID)
	}

	secondRowReason := "nobody is here to answer, and this recovery is safe for Aether to carry out on its own"
	secondRowAction := "arranging a saved copy of your work before carrying on"
	rendered := renderRefusalSelfRecoveryNotice(refuse(row.ID), secondRowReason, secondRowAction, row.NextCommand)

	if !strings.Contains(rendered, secondRowAction) {
		t.Fatalf("a notice rendered with a locally built row's own Action does not contain it:\n%s", rendered)
	}
	if strings.Contains(rendered, realAction) {
		t.Fatalf("a notice rendered for a different row's Action still contains the real colonize row's Action (%q) -- the closing clause is still a fixed phrase, not read from the row that fired:\n%s", realAction, rendered)
	}
}
