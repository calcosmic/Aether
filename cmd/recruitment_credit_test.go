package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// newCreditTrophallaxisFixture drives the real 203-10 public path (pack,
// acknowledge, record decision) to produce a genuine trophallaxisPacket with
// a recorded colony.LifecycleDecision, so credit tests attach to real
// recorded facts rather than a hand-built struct.
func newCreditTrophallaxisFixture(t *testing.T, suffix string) (packetID string, decisionID string) {
	t.Helper()
	seedTrophallaxisReceiver(t, "Mason-"+suffix)
	result := newTrophallaxisResultFixture("credit-" + suffix)
	packet, err := packTrophallaxisPacket(trophallaxisPackInput{
		Result:         result,
		Handoff:        newTrophallaxisHandoffFixture(),
		ParentReceiver: "Mason-" + suffix,
		Scope:          []string{trophallaxisSectionSummary},
	})
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	if _, err := acknowledgeTrophallaxisPacket(packet.PacketID, colony.SignalAcknowledgement{ActorID: "Mason-" + suffix, EvidenceID: "evidence-" + suffix}, ""); err != nil {
		t.Fatalf("acknowledge: %v", err)
	}
	decided, err := recordTrophallaxisDecision(packet.PacketID, "Mason-"+suffix, "did the thing for "+suffix, nil)
	if err != nil {
		t.Fatalf("record decision: %v", err)
	}
	if decided.Decision == nil {
		t.Fatal("fixture is broken: no decision recorded")
	}
	return packet.PacketID, decided.Decision.ID
}

// --- Task 1: a credit record that can say helped, did nothing, or made it worse ---

func TestRecruitmentCreditOutcomeVocabulary(t *testing.T) {
	names := recruitmentCreditOutcomeNames()
	if len(names) != 4 {
		t.Fatalf("recruitmentCreditOutcomeNames() = %v, want exactly 4 members", names)
	}
	for _, want := range []string{"helpful", "neutral", "harmful", "pending"} {
		found := false
		for _, got := range names {
			if got == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("recruitmentCreditOutcomeNames() = %v, missing declared member %q", names, want)
		}
	}

	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	_, decisionID := newCreditTrophallaxisFixture(t, "vocab")

	_, credited, err := recordRecruitmentCredit("contrib-vocab", recruitmentContributionNote, decisionID, "evidence-vocab", recruitmentCreditOutcome("bogus"), "")
	if err == nil {
		t.Fatal("expected an undeclared outcome to be refused")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Fatalf("refusal error = %q, want it to name the offending value %q", err.Error(), "bogus")
	}
	if credited {
		t.Fatal("an undeclared outcome must never report credited=true")
	}
}

func TestRecruitmentCreditRecordsAHarmfulOutcomeDirectlyOnTheFile(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	_, decisionID := newCreditTrophallaxisFixture(t, "harmful")

	record, credited, err := recordRecruitmentCredit("contrib-harmful", recruitmentContributionRecruitmentResult, decisionID, "evidence-harmful", recruitmentCreditOutcomeHarmful, "")
	if err != nil {
		t.Fatalf("record harmful credit: %v", err)
	}
	if !credited {
		t.Fatal("a fully-facted harmful contribution must be credited (recorded), not omitted")
	}
	if record.Outcome != recruitmentCreditOutcomeHarmful {
		t.Fatalf("stored outcome = %q, want harmful", record.Outcome)
	}

	raw, err := os.ReadFile(filepath.Join(store.BasePath(), recruitmentCreditPath))
	if err != nil {
		t.Fatalf("read credit store file: %v", err)
	}
	if !strings.Contains(string(raw), "harmful") {
		t.Fatalf("credit/records.json does not contain the harmful outcome on disk:\n%s", raw)
	}
}

func TestRecruitmentCreditReplayIsByteIdentical(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	_, decisionID := newCreditTrophallaxisFixture(t, "replay")

	if _, _, err := recordRecruitmentCredit("contrib-replay", recruitmentContributionNote, decisionID, "evidence-replay", recruitmentCreditOutcomeHelpful, "2026-09-13T00:00:00Z"); err != nil {
		t.Fatalf("first write: %v", err)
	}
	creditFilePath := filepath.Join(store.BasePath(), recruitmentCreditPath)
	before, err := os.ReadFile(creditFilePath)
	if err != nil {
		t.Fatalf("read before replay: %v", err)
	}

	record, credited, err := recordRecruitmentCredit("contrib-replay", recruitmentContributionNote, decisionID, "evidence-replay", recruitmentCreditOutcomeHelpful, "2026-09-13T00:00:00Z")
	if err != nil {
		t.Fatalf("replay write: %v", err)
	}
	if !credited {
		t.Fatal("a verified replay must still report credited=true (it returns the existing record)")
	}
	if record.Outcome != recruitmentCreditOutcomeHelpful {
		t.Fatalf("replayed record outcome = %q, want helpful", record.Outcome)
	}

	after, err := os.ReadFile(creditFilePath)
	if err != nil {
		t.Fatalf("read after replay: %v", err)
	}
	if string(before) != string(after) {
		t.Fatalf("replay mutated credit/records.json:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestRecruitmentCreditTwoContributionsOneDecision(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	_, decisionID := newCreditTrophallaxisFixture(t, "shared")

	if _, _, err := recordRecruitmentCredit("contrib-a", recruitmentContributionNote, decisionID, "evidence-a", recruitmentCreditOutcomeHelpful, ""); err != nil {
		t.Fatalf("record contrib-a: %v", err)
	}
	if _, _, err := recordRecruitmentCredit("contrib-b", recruitmentContributionMemoryItem, decisionID, "evidence-b", recruitmentCreditOutcomeNeutral, ""); err != nil {
		t.Fatalf("record contrib-b: %v", err)
	}

	records, err := recruitmentCreditForDecision(decisionID)
	if err != nil {
		t.Fatalf("query by decision: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("credit list for decision %q has %d entries, want 2: %+v", decisionID, len(records), records)
	}
	seen := map[string]bool{}
	for _, r := range records {
		seen[r.ContributionID] = true
	}
	if !seen["contrib-a"] || !seen["contrib-b"] {
		t.Fatalf("expected both contrib-a and contrib-b in the decision's credit list, got %+v", records)
	}
}

func TestRecruitmentCreditSortOrderIsStableAcrossRepeatedReads(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	_, decisionID := newCreditTrophallaxisFixture(t, "tie")

	sharedTimestamp := "2026-09-13T12:00:00Z"
	for _, c := range []string{"tie-c", "tie-a", "tie-b"} {
		if _, _, err := recordRecruitmentCredit(c, recruitmentContributionSpecialist, decisionID, "evidence-"+c, recruitmentCreditOutcomeNeutral, sharedTimestamp); err != nil {
			t.Fatalf("seed %s: %v", c, err)
		}
	}

	var previous []byte
	for i := 0; i < 10; i++ {
		records, err := recruitmentCreditForDecision(decisionID)
		if err != nil {
			t.Fatalf("read %d: %v", i, err)
		}
		encoded, err := json.Marshal(records)
		if err != nil {
			t.Fatalf("marshal read %d: %v", i, err)
		}
		if previous != nil && string(previous) != string(encoded) {
			t.Fatalf("read %d differs from prior read:\nprior: %s\nnow:   %s", i, previous, encoded)
		}
		previous = encoded
	}

	records, _ := recruitmentCreditForDecision(decisionID)
	var ids []string
	for _, r := range records {
		ids = append(ids, r.ContributionID)
	}
	want := []string{"tie-a", "tie-b", "tie-c"}
	if !reflect.DeepEqual(ids, want) {
		t.Fatalf("tie-break order = %v, want record-identifier ascending %v", ids, want)
	}
}

func TestRecruitmentCreditUncreditedRendersExistingNoMeasuredEffectWording(t *testing.T) {
	rendered := renderRecruitmentCreditOutcome(nil)
	if rendered != AgencyMeasuredEffectPending {
		t.Fatalf("uncredited rendering = %q, want the existing constant %q", rendered, AgencyMeasuredEffectPending)
	}

	pendingRecord := &recruitmentCreditRecord{Outcome: recruitmentCreditOutcomePending}
	pendingRendered := renderRecruitmentCreditOutcome(pendingRecord)
	if pendingRendered == AgencyMeasuredEffectPending {
		t.Fatalf("a decision-recorded-but-unverified contribution must render distinctly from the no-record-at-all wording, got %q for both", pendingRendered)
	}
	if pendingRendered != recruitmentCreditPendingWording {
		t.Fatalf("pending rendering = %q, want %q", pendingRendered, recruitmentCreditPendingWording)
	}

	helpfulRecord := &recruitmentCreditRecord{Outcome: recruitmentCreditOutcomeHelpful}
	if got := renderRecruitmentCreditOutcome(helpfulRecord); got != "helpful" {
		t.Fatalf("helpful rendering = %q, want %q", got, "helpful")
	}
}

func TestRecruitmentCreditNeitherFactRecordsNothing(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	record, credited, err := recordRecruitmentCredit("contrib-nothing", recruitmentContributionNote, "", "", "", "")
	if err != nil {
		t.Fatalf("expected no error for the neither-fact case, got %v", err)
	}
	if credited {
		t.Fatalf("expected credited=false for the neither-fact case, got record %+v", record)
	}

	all, err := recruitmentCreditAll()
	if err != nil {
		t.Fatalf("recruitmentCreditAll: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected zero records written for the neither-fact case, got %+v", all)
	}
}

func TestRecruitmentCreditEffectEvidenceWithoutDecisionIsRefused(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	_, credited, err := recordRecruitmentCredit("contrib-orphan-evidence", recruitmentContributionNote, "", "evidence-orphan", recruitmentCreditOutcomeHelpful, "")
	if err == nil {
		t.Fatal("expected effect evidence with no changed decision to be refused")
	}
	if credited {
		t.Fatal("a refused write must never report credited=true")
	}
}

func TestRecruitmentCreditDecisionOnlyIsPendingNotHelpful(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	_, decisionID := newCreditTrophallaxisFixture(t, "pending")

	record, credited, err := recordRecruitmentCredit("contrib-pending", recruitmentContributionNote, decisionID, "", "", "")
	if err != nil {
		t.Fatalf("record decision-only credit: %v", err)
	}
	if !credited {
		t.Fatal("a decision-only record is stored (as pending), not omitted")
	}
	if record.Outcome != recruitmentCreditOutcomePending {
		t.Fatalf("decision-only outcome = %q, want pending", record.Outcome)
	}
}

// --- Task 3: prove credit cannot be inferred ---

// TestDeliveredPlusPassingIsNotCredit drives a real note into a real worker
// brief and a real (trivially passing) verification step, then asserts zero
// credit records exist -- the precise failure mode CEC-07 names.
func TestDeliveredPlusPassingIsNotCredit(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	signal, _, err := writePheromoneSignal("FOCUS", "pay attention to the credit boundary", "normal", "test", "", "", 0, nil)
	if err != nil {
		t.Fatalf("write a real note: %v", err)
	}

	brief := resolvePheromoneSection()
	if !strings.Contains(brief, "pay attention to the credit boundary") {
		t.Fatalf("note was not actually delivered into the resolved worker brief:\n%s", brief)
	}

	step := runVerificationStep(context.Background(), tmpDir, "credit-boundary-check", true, "true", 5*time.Second)
	if !step.Passed {
		t.Fatalf("expected the phase's own check to pass, got %+v", step)
	}

	all, err := recruitmentCreditAll()
	if err != nil {
		t.Fatalf("recruitmentCreditAll: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("a delivered note plus a passing phase must never itself produce credit, got %+v (signal %s)", all, signal.ID)
	}
}

// TestCreditRequiresBothFacts is an AST-based scan of the cmd package
// mirroring cmd/recruitment_result_test.go's TestEveryResultWriteGoesThroughTheBinding:
// it derives every function whose body writes credit/records.json via the
// store, and asserts recordRecruitmentCredit is the only one -- and that its
// own signature carries both a changed-decision identifier and an
// effect-evidence identifier parameter, so no future write into the credit
// store can be guarded by only one of the two facts CEC-07 requires.
func TestCreditRequiresBothFacts(t *testing.T) {
	violations := scanForCreditWritesOutsideRecordRecruitmentCredit(t, ".")
	if len(violations) != 0 {
		t.Fatalf("found a credit-store write guarded by fewer than both required facts:\n%s", strings.Join(violations, "\n"))
	}

	t.Run("a synthetic single-identifier write is caught", func(t *testing.T) {
		fixtureSrc := `package cmd

func writeCreditWithOnlyOneFact(changedDecisionID string) error {
	var file recruitmentCreditFile
	return store.UpdateJSONAtomically(recruitmentCreditPath, &file, func() error {
		file.Entries = append(file.Entries, recruitmentCreditRecord{ChangedDecisionID: changedDecisionID})
		return nil
	})
}
`
		violations := scanSourceForCreditWritesOutsideRecordRecruitmentCredit(t, "fixture_credit_single_fact.go", fixtureSrc)
		if len(violations) == 0 {
			t.Fatal("scanner failed to detect a synthetic single-identifier credit write")
		}
	})
}

func scanForCreditWritesOutsideRecordRecruitmentCredit(t *testing.T, dir string) []string {
	t.Helper()
	names, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatalf("glob cmd package files: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("fixture is broken: no .go files found in the cmd package directory")
	}
	fset := token.NewFileSet()
	var violations []string
	found := false
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		fileFound, fileViolations := creditWriteViolationsInFile(fset, file)
		found = found || fileFound
		violations = append(violations, fileViolations...)
	}
	if !found {
		t.Fatal("fixture is broken: no write call referencing recruitmentCreditPath was found anywhere in the cmd package")
	}
	return violations
}

func scanSourceForCreditWritesOutsideRecordRecruitmentCredit(t *testing.T, filename, src string) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, 0)
	if err != nil {
		t.Fatalf("parse fixture source: %v", err)
	}
	_, violations := creditWriteViolationsInFile(fset, file)
	return violations
}

// recruitmentCreditPathWriters is the closed, by-symbol-name allowlist of
// every function permitted to write recruitmentCreditPath through the
// store. recordRecruitmentCredit is the original CEC-07 writer (both-facts
// contract, checked separately below). recordGuidanceApplicationState and
// recordGuidanceClaimUnverified (cmd/application_evidence.go,
// 204-06-PLAN.md, LEARN-03) were added when the sibling GuidanceApplications
// / GuidanceClaims arrays moved into this SAME file
// (recruitmentCreditFile) rather than a second data file -- extended by
// symbol name here, never by exempting a file path, so a THIRD writer
// dropped in anywhere in cmd/ is still caught by name.
var recruitmentCreditPathWriters = map[string]bool{
	"recordRecruitmentCredit":        true,
	"recordGuidanceApplicationState": true,
	"recordGuidanceClaimUnverified":  true,
}

// creditWriteViolationsInFile walks file for every call writing
// recruitmentCreditPath through the store, and requires the enclosing
// function to be one of recruitmentCreditPathWriters. recordRecruitmentCredit
// additionally must declare parameters naming both a changed-decision
// identifier and an effect-evidence identifier -- so a second write
// function reusing that ONE name's shape is still caught if it drops either
// parameter. The other allowed writers carry a different contract (a
// guidance state or an unverified claim, not a credit outcome) and are not
// held to that specific two-parameter shape.
func creditWriteViolationsInFile(fset *token.FileSet, file *ast.File) (found bool, violations []string) {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "UpdateJSONAtomically" {
				return true
			}
			if len(call.Args) == 0 {
				return true
			}
			ident, ok := call.Args[0].(*ast.Ident)
			if !ok || ident.Name != "recruitmentCreditPath" {
				return true
			}
			found = true
			fnName := "<unknown>"
			if fn.Name != nil {
				fnName = fn.Name.Name
			}
			if !recruitmentCreditPathWriters[fnName] {
				violations = append(violations, fmt.Sprintf(
					"%s: %s writes recruitmentCreditPath via store.%s -- only %v may write it",
					fset.Position(call.Pos()).String(), fnName, sel.Sel.Name, recruitmentCreditPathWriterNames(),
				))
				return true
			}
			if fnName != "recordRecruitmentCredit" {
				return true
			}
			hasChangedDecisionParam := false
			hasEffectEvidenceParam := false
			for _, field := range fn.Type.Params.List {
				for _, paramName := range field.Names {
					lower := strings.ToLower(paramName.Name)
					if strings.Contains(lower, "changeddecision") {
						hasChangedDecisionParam = true
					}
					if strings.Contains(lower, "effectevidence") {
						hasEffectEvidenceParam = true
					}
				}
			}
			if !hasChangedDecisionParam || !hasEffectEvidenceParam {
				violations = append(violations, fmt.Sprintf(
					"%s: %s writes recruitmentCreditPath without declaring both a changed-decision and an effect-evidence parameter",
					fset.Position(call.Pos()).String(), fnName,
				))
			}
			return true
		})
	}
	return found, violations
}

// recruitmentCreditPathWriterNames returns every allowed writer name, sorted,
// for a deterministic refusal message.
func recruitmentCreditPathWriterNames() []string {
	names := make([]string, 0, len(recruitmentCreditPathWriters))
	for name := range recruitmentCreditPathWriters {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// TestHarmfulOutcomeIsReachable drives the real public path -- a packed,
// acknowledged trophallaxis packet with a recorded decision, then a credit
// record -- to prove a harmful outcome is reachable end to end, not only
// from a unit fixture constructing a struct literal directly.
func TestHarmfulOutcomeIsReachable(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	packetID, decisionID := newCreditTrophallaxisFixture(t, "public-harmful")

	if _, _, err := recordRecruitmentCredit("contrib-public-harmful", recruitmentContributionRecruitmentResult, decisionID, packetID, recruitmentCreditOutcomeHarmful, ""); err != nil {
		t.Fatalf("record harmful credit via the public path: %v", err)
	}

	stored, ok, err := recruitmentCreditForContribution("contrib-public-harmful", decisionID)
	if err != nil {
		t.Fatalf("query stored record: %v", err)
	}
	if !ok {
		t.Fatal("expected a stored credit record for the public-path contribution")
	}
	if stored.Outcome != recruitmentCreditOutcomeHarmful {
		t.Fatalf("stored outcome = %q, want harmful", stored.Outcome)
	}
}
