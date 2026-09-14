package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

// improvementReportTestSentinels is the fixture sentinel list every test in
// this file uses, mirroring the shape of the real
// cmd/testdata/eval-gates/sentinels.json entries this report cross-
// references for its hard-failure consequence text.
func improvementReportTestSentinels() []evalGateSentinel {
	return []evalGateSentinel{
		{Test: "TestOneAdmissionAuthority", Package: "cmd", Consequence: "a helper's backup request could bypass the single admission gate."},
	}
}

// openAndCloseEpisode is a small test helper that writes a real open record
// and then a real closed/terminal record for episodeID through
// recordEpisodeOutcome -- every fixture in this file goes through the real
// writer, never a hand-built struct literal, so a test proves something
// about records the runtime can actually produce.
func openAndCloseEpisode(t *testing.T, episodeID, startedAt, endedAt string, closed episodeLedgerRecord) {
	t.Helper()
	if _, _, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind: episodeLedgerRecordKindOpened,
		EpisodeID:  episodeID,
		StartedAt:  startedAt,
	}); err != nil {
		t.Fatalf("open %s: %v", episodeID, err)
	}
	closed.RecordKind = episodeLedgerRecordKindClosed
	closed.EpisodeID = episodeID
	if closed.EndedAt == "" {
		closed.EndedAt = endedAt
	}
	if _, _, err := recordEpisodeOutcome(closed); err != nil {
		t.Fatalf("close %s: %v", episodeID, err)
	}
}

// recordIntervention writes a real intervention_recorded record for
// episodeID through recordEpisodeOutcome.
func recordIntervention(t *testing.T, episodeID, at, category string) {
	t.Helper()
	if _, _, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind:    episodeLedgerRecordKindIntervention,
		EpisodeID:     episodeID,
		StartedAt:     at,
		Interventions: []string{category},
	}); err != nil {
		t.Fatalf("record intervention on %s: %v", episodeID, err)
	}
}

// verifiedSuccessClosedRecord returns a closed-record template that
// isVerifiedUsefulSuccess accepts: a completed terminal result, at least one
// evidence id, and every recorded hard gate passing.
func verifiedSuccessClosedRecord() episodeLedgerRecord {
	return episodeLedgerRecord{
		TerminalResult:  improvementReportSuccessTerminalStatus,
		EvidenceIDs:     []string{"evidence-1"},
		HardGateResults: map[string]bool{"TestOneAdmissionAuthority": true},
	}
}

// TestTwoFiguresAreNeverCombined asserts, structurally, that improvementReport
// declares no combined score field and that renderImprovementReport never
// emits a single line carrying both figures.
func TestTwoFiguresAreNeverCombined(t *testing.T) {
	allowedFields := map[string]bool{
		"Window":                   true,
		"VerifiedUsefulSuccess":    true,
		"PreventableInterventions": true,
		"TotalEpisodes":            true,
		"UnclassifiedEpisodes":     true,
		"HardFailures":             true,
		"Rows":                     true,
	}
	typ := reflect.TypeOf(improvementReport{})
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		if !allowedFields[name] {
			t.Fatalf("improvementReport declares an unexpected field %q -- if this is a combined score, it must not exist (see this file's top-of-file comment)", name)
		}
	}

	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s
	openAndCloseEpisode(t, "ep-combo-check", "2026-09-01T00:00:00Z", "2026-09-01T00:01:00Z", verifiedSuccessClosedRecord())
	recordIntervention(t, "ep-combo-check", "2026-09-01T00:00:30Z", "declined a forced reviewer")

	records, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	report := buildImprovementReport(records, improvementReportTestSentinels(), reportWindow{})
	rendered := renderImprovementReport(report)

	// The two figures' own distinguishing narrative phrases (never the
	// numeric ratio string, which can coincide by chance when the two
	// counts happen to be equal, as they are in this very fixture).
	const successPhrase = "Finished something genuinely useful"
	const interventionPhrase = "had to step in"
	for _, line := range strings.Split(rendered, "\n") {
		if strings.Contains(line, successPhrase) && strings.Contains(line, interventionPhrase) {
			t.Fatalf("rendered line combines both figures: %q", line)
		}
	}
}

// TestThresholdBoundaries drives a ratio at exactly, one below, and one
// above each declared threshold and asserts the declared classification.
func TestThresholdBoundaries(t *testing.T) {
	t.Run("success threshold", func(t *testing.T) {
		cases := []struct {
			name  string
			ratio reportRatio
			want  string
		}{
			{"exactly at threshold", ratioOf(8, 10), "meeting the bar"},
			{"one below threshold", ratioOf(7, 10), "below the bar"},
			{"one above threshold", ratioOf(9, 10), "meeting the bar"},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				if got := improvementReportSuccessClassification(c.ratio); got != c.want {
					t.Fatalf("classification for %+v = %q, want %q", c.ratio, got, c.want)
				}
			})
		}
	})

	t.Run("intervention threshold", func(t *testing.T) {
		cases := []struct {
			name  string
			ratio reportRatio
			want  string
		}{
			{"exactly at threshold", ratioOf(2, 10), "elevated"},
			{"one below threshold", ratioOf(1, 10), "within bounds"},
			{"one above threshold", ratioOf(3, 10), "elevated"},
		}
		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				if got := improvementReportInterventionClassification(c.ratio); got != c.want {
					t.Fatalf("classification for %+v = %q, want %q", c.ratio, got, c.want)
				}
			})
		}
	})
}

// TestEmptyWindowReportsZeroNotPerfect asserts a window with zero episodes
// reports zero and zero, with a stated total of zero, and no classification
// claiming success.
func TestEmptyWindowReportsZeroNotPerfect(t *testing.T) {
	report := buildImprovementReport(nil, improvementReportTestSentinels(), reportWindow{})
	if report.VerifiedUsefulSuccess != (reportRatio{0, 0}) {
		t.Fatalf("VerifiedUsefulSuccess = %+v, want 0 out of 0", report.VerifiedUsefulSuccess)
	}
	if report.PreventableInterventions != (reportRatio{0, 0}) {
		t.Fatalf("PreventableInterventions = %+v, want 0 out of 0", report.PreventableInterventions)
	}
	if report.TotalEpisodes != 0 {
		t.Fatalf("TotalEpisodes = %d, want 0", report.TotalEpisodes)
	}
	if got := improvementReportSuccessClassification(report.VerifiedUsefulSuccess); got == "meeting the bar" {
		t.Fatalf("empty window classified as meeting the bar: %q", got)
	}
	rendered := renderImprovementReport(report)
	if !strings.Contains(rendered, "0 out of 0") {
		t.Fatalf("rendered report does not show 0 out of 0:\n%s", rendered)
	}
}

// TestReportOrderingIsTotalAndStable asserts rows are ordered by episode
// identifier ascending, identically across repeated builds from the same
// records.
func TestReportOrderingIsTotalAndStable(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	for _, id := range []string{"ep-order-c", "ep-order-a", "ep-order-b"} {
		openAndCloseEpisode(t, id, "2026-09-01T00:00:00Z", "2026-09-01T00:01:00Z", verifiedSuccessClosedRecord())
	}

	records, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	first := buildImprovementReport(records, improvementReportTestSentinels(), reportWindow{})
	second := buildImprovementReport(records, improvementReportTestSentinels(), reportWindow{})

	if len(first.Rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(first.Rows))
	}
	wantOrder := []string{"ep-order-a", "ep-order-b", "ep-order-c"}
	for i, id := range wantOrder {
		if first.Rows[i].EpisodeID != id {
			t.Fatalf("row %d = %q, want %q", i, first.Rows[i].EpisodeID, id)
		}
	}
	for i := range first.Rows {
		if first.Rows[i].EpisodeID != second.Rows[i].EpisodeID {
			t.Fatalf("ordering differs between repeated builds at index %d: %q vs %q", i, first.Rows[i].EpisodeID, second.Rows[i].EpisodeID)
		}
	}
}

// TestOneEpisodeCanCountInBothFigures asserts an episode that produced both
// a verified success and a preventable intervention increments both
// figures and is merged into neither.
func TestOneEpisodeCanCountInBothFigures(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	openAndCloseEpisode(t, "ep-both", "2026-09-01T00:00:00Z", "2026-09-01T00:02:00Z", verifiedSuccessClosedRecord())
	recordIntervention(t, "ep-both", "2026-09-01T00:01:00Z", "declined a forced reviewer")

	records, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	report := buildImprovementReport(records, improvementReportTestSentinels(), reportWindow{})

	if report.VerifiedUsefulSuccess.Count != 1 {
		t.Fatalf("VerifiedUsefulSuccess.Count = %d, want 1", report.VerifiedUsefulSuccess.Count)
	}
	if report.PreventableInterventions.Count != 1 {
		t.Fatalf("PreventableInterventions.Count = %d, want 1", report.PreventableInterventions.Count)
	}
	if len(report.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(report.Rows))
	}
	row := report.Rows[0]
	if !row.VerifiedSuccess || !row.PreventableIntervention {
		t.Fatalf("row = %+v, want both VerifiedSuccess and PreventableIntervention true", row)
	}
	if len(report.UnclassifiedEpisodes) != 0 {
		t.Fatalf("UnclassifiedEpisodes = %v, want none", report.UnclassifiedEpisodes)
	}
}

// TestHardFailureCannotBeExcludedOrOffset drives a window whose successes
// outnumber its hard failures and asserts every hard failure still appears.
func TestHardFailureCannotBeExcludedOrOffset(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	for i := 0; i < 9; i++ {
		id := "ep-success-" + string(rune('a'+i))
		openAndCloseEpisode(t, id, "2026-09-01T00:00:00Z", "2026-09-01T00:01:00Z", verifiedSuccessClosedRecord())
	}
	failing := episodeLedgerRecord{
		TerminalResult:  improvementReportSuccessTerminalStatus,
		EvidenceIDs:     []string{"evidence-1"},
		HardGateResults: map[string]bool{"TestOneAdmissionAuthority": false},
	}
	openAndCloseEpisode(t, "ep-hard-failure", "2026-09-01T00:00:00Z", "2026-09-01T00:01:00Z", failing)

	records, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	report := buildImprovementReport(records, improvementReportTestSentinels(), reportWindow{})

	if report.VerifiedUsefulSuccess.Count != 9 {
		t.Fatalf("VerifiedUsefulSuccess.Count = %d, want 9 (9 successes should not exclude the failure)", report.VerifiedUsefulSuccess.Count)
	}
	if len(report.HardFailures) != 1 {
		t.Fatalf("HardFailures = %+v, want exactly 1 entry", report.HardFailures)
	}
	if report.HardFailures[0].EpisodeID != "ep-hard-failure" || report.HardFailures[0].Gate != "TestOneAdmissionAuthority" {
		t.Fatalf("HardFailures[0] = %+v, want ep-hard-failure/TestOneAdmissionAuthority", report.HardFailures[0])
	}
	if report.HardFailures[0].Consequence == "" {
		t.Fatalf("HardFailures[0].Consequence is empty, want the matching sentinel's consequence text")
	}

	rendered := renderImprovementReport(report)
	if !strings.Contains(rendered, "ep-hard-failure") {
		t.Fatalf("rendered report does not name the failing episode:\n%s", rendered)
	}
}

// TestPercentagesAreNeverCompared is an abstract-syntax-tree check that no
// comparison operator in cmd/improvement_report.go is applied to a
// Percent() call result.
func TestPercentagesAreNeverCompared(t *testing.T) {
	violations := scanForPercentComparisons(t, "improvement_report.go")
	if len(violations) != 0 {
		t.Fatalf("found a comparison against Percent():\n%s", strings.Join(violations, "\n"))
	}

	t.Run("a synthetic violation is caught", func(t *testing.T) {
		fixtureSrc := `package cmd

func sneakyPercentCompare(r reportRatio) bool {
	return r.Percent() >= 50.0
}
`
		violations := scanPercentComparisonSource(t, "fixture_percent_compare.go", fixtureSrc)
		if len(violations) == 0 {
			t.Fatal("scanner failed to detect a synthetic comparison against Percent()")
		}
		if !strings.Contains(violations[0], "sneakyPercentCompare") {
			t.Fatalf("violation %q does not name the offending function", violations[0])
		}
	})
}

func scanForPercentComparisons(t *testing.T, filename string) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", filename, err)
	}
	return percentComparisonViolationsInFile(file)
}

func scanPercentComparisonSource(t *testing.T, filename, src string) []string {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, 0)
	if err != nil {
		t.Fatalf("parse fixture source: %v", err)
	}
	return percentComparisonViolationsInFile(file)
}

func percentComparisonViolationsInFile(file *ast.File) []string {
	var violations []string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		fnName := "<unknown>"
		if fn.Name != nil {
			fnName = fn.Name.Name
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			be, ok := n.(*ast.BinaryExpr)
			if !ok {
				return true
			}
			if !isComparisonToken(be.Op) {
				return true
			}
			if callsPercentMethod(be.X) || callsPercentMethod(be.Y) {
				violations = append(violations, fnName+" compares a Percent() result")
			}
			return true
		})
	}
	return violations
}

func isComparisonToken(op token.Token) bool {
	switch op {
	case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
		return true
	default:
		return false
	}
}

func callsPercentMethod(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	return sel.Sel.Name == "Percent"
}

// TestPercentRoundingIsHalfToEven uses values that round differently under
// half-to-even than under Go's default half-away-from-zero rounding.
func TestPercentRoundingIsHalfToEven(t *testing.T) {
	cases := []struct {
		name  string
		value float64
		want  float64
	}{
		{"0.25 rounds down to even 0.2", 0.25, 0.2},
		{"0.35 rounds up to even 0.4", 0.35, 0.4},
		{"exactly on an even digit stays", 0.20, 0.2},
		{"non-tie rounds normally down", 0.24, 0.2},
		{"non-tie rounds normally up", 0.26, 0.3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := roundHalfToEven(c.value, reportPercentDecimalPlaces)
			if got != c.want {
				t.Fatalf("roundHalfToEven(%v, 1) = %v, want %v", c.value, got, c.want)
			}
		})
	}

	// A ratio whose exact percentage already sits on one decimal place
	// (12.5%) needs no rounding at all -- confirms Percent() does not
	// perturb an already-exact value.
	if got := ratioOf(1, 8).Percent(); got != 12.5 {
		t.Fatalf("ratioOf(1,8).Percent() = %v, want 12.5 (exact, no rounding needed)", got)
	}
}
