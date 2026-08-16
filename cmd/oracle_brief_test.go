package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// hashOracleWorkspace fingerprints every file under .aether/oracle so a test
// can prove a command wrote nothing at all, rather than only checking for the
// one file it expected not to appear.
func hashOracleWorkspace(t *testing.T, root string) string {
	t.Helper()
	dir := filepath.Join(root, ".aether", "oracle")
	hasher := sha256.New()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		fmt.Fprintf(hasher, "%s:%x\n", rel, sha256.Sum256(data))
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("hash oracle workspace: %v", err)
	}
	return hex.EncodeToString(hasher.Sum(nil))
}

func TestOracleProposeSuggestsTemplateDepthAndQuestionCount(t *testing.T) {
	root := t.TempDir()

	bug, err := runOraclePropose(root, "the sync job crashes with a nil pointer after a retry")
	if err != nil {
		t.Fatalf("propose bug topic: %v", err)
	}
	bugSuggested := bug["suggested"].(map[string]interface{})
	if got := bugSuggested["template"]; got != "bug-investigation" {
		t.Errorf("bug-shaped topic proposed template %v, want bug-investigation", got)
	}

	choice, err := runOraclePropose(root, "should we use SQLite or Postgres for the local cache?")
	if err != nil {
		t.Fatalf("propose choice topic: %v", err)
	}
	if got := choice["suggested"].(map[string]interface{})["template"]; got != "tech-eval" {
		t.Errorf("a choice between two technologies proposed template %v, want tech-eval", got)
	}

	// A vague topic must ask for more scoping than a precise one. Asserting the
	// relationship rather than exact counts keeps the test meaningful if the
	// thresholds are retuned.
	vague, err := runOraclePropose(root, "look into improving the app")
	if err != nil {
		t.Fatalf("propose vague topic: %v", err)
	}
	precise, err := runOraclePropose(root, "Does the retry policy in the sync worker double-count failed batches when the queue is drained mid-flight?")
	if err != nil {
		t.Fatalf("propose precise topic: %v", err)
	}
	vagueCount := vague["suggested"].(map[string]interface{})["clarifying_questions"].(int)
	preciseCount := precise["suggested"].(map[string]interface{})["clarifying_questions"].(int)
	if vagueCount <= preciseCount {
		t.Errorf("vague topic suggested %d clarifying questions, precise topic %d; vague must ask for more", vagueCount, preciseCount)
	}

	// Every depth the user can pick must be offered with its round count, so
	// the wrapper never has to invent the numbers.
	options, ok := precise["depth_options"].([]oracleDepthOption)
	if !ok || len(options) != 4 {
		t.Fatalf("depth_options = %#v, want 4 options", precise["depth_options"])
	}
	for _, option := range options {
		if option.MaxIterations <= 0 {
			t.Errorf("depth option %q offered without a round count", option.Value)
		}
	}
	if len(precise["confidence_options"].([]oracleConfidenceOption)) != 4 {
		t.Errorf("confidence_options must offer all four accuracy targets")
	}
	if len(precise["scoping_options"].([]oracleScopingOption)) != 4 {
		t.Errorf("scoping_options must offer skip/quick/standard/thorough")
	}
}

func TestOracleProposeMutatesNothing(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".aether", "oracle"), 0755); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
	before := hashOracleWorkspace(t, root)

	if _, err := runOraclePropose(root, "should we adopt a queue for the ingest path?"); err != nil {
		t.Fatalf("propose: %v", err)
	}

	if after := hashOracleWorkspace(t, root); after != before {
		t.Fatalf("oracle propose modified the workspace; it must be read-only")
	}
}

func TestOracleFromBriefFailsWithoutApprovedBrief(t *testing.T) {
	root := t.TempDir()

	if _, err := resolveOracleBriefRun(root); err == nil {
		t.Fatal("--from-brief succeeded with no approved brief; the setup ritual is not enforced")
	}

	// A brief that exists but carries no question is just as unusable.
	if _, err := runOracleBriefApprove(root, oracleBriefOptions{
		Topic:        "caching",
		CoreQuestion: "Should the cache be write-through?",
		Depth:        "quick",
	}, false); err != nil {
		t.Fatalf("approve brief: %v", err)
	}
	briefPath := oraclePendingBriefPath(root)
	if err := os.WriteFile(briefPath, []byte(`{"version":"1.0","topic":"caching","core_question":"   "}`), 0644); err != nil {
		t.Fatalf("write hollow brief: %v", err)
	}
	if _, err := resolveOracleBriefRun(root); err == nil {
		t.Fatal("--from-brief accepted a brief with an empty core question")
	}
}

func TestOracleBriefRejectsMalformedCoreQuestion(t *testing.T) {
	cases := []struct {
		name     string
		question string
	}{
		{"empty", "   "},
		{"not a question", "Look at the caching layer and tell me what you think"},
		{"too long", strings.Repeat("Is this the question we should be asking here? ", 12)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if _, err := runOracleBriefApprove(root, oracleBriefOptions{
				Topic:        "caching",
				CoreQuestion: tc.question,
				Depth:        "quick",
			}, false); err == nil {
				t.Fatalf("brief accepted a malformed core question (%s)", tc.name)
			}
			if _, err := os.Stat(oraclePendingBriefPath(root)); !os.IsNotExist(err) {
				t.Fatalf("a rejected brief was still written to disk")
			}
		})
	}
}

func TestOracleBriefDryRunWritesNothing(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".aether", "oracle"), 0755); err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
	before := hashOracleWorkspace(t, root)

	result, err := runOracleBriefApprove(root, oracleBriefOptions{
		Topic:           "caching",
		CoreQuestion:    "Should the local cache use SQLite or Postgres?",
		SuccessCriteria: []string{"A recommendation with reasons"},
		Depth:           "deep",
	}, true)
	if err != nil {
		t.Fatalf("brief dry run: %v", err)
	}
	if panel, _ := result["panel"].(string); !strings.Contains(panel, "Should the local cache use SQLite or Postgres?") {
		t.Errorf("dry run did not render the brief for approval:\n%s", panel)
	}
	if after := hashOracleWorkspace(t, root); after != before {
		t.Fatalf("oracle brief --dry-run wrote to the workspace")
	}
}

func TestOracleBriefRecordsApprovedSettings(t *testing.T) {
	root := t.TempDir()
	if _, err := runOracleBriefApprove(root, oracleBriefOptions{
		Topic:            "cache storage",
		CoreQuestion:     "Should the local cache use SQLite or Postgres?",
		Context:          "Choosing storage before the sync feature is built.",
		SuccessCriteria:  []string{"A recommendation with reasons", "Migration cost"},
		Depth:            "deep",
		TargetConfidence: 90,
	}, false); err != nil {
		t.Fatalf("approve brief: %v", err)
	}

	brief, err := loadOraclePendingBrief(root)
	if err != nil {
		t.Fatalf("load approved brief: %v", err)
	}
	if brief.Depth != "deep" || brief.TargetConfidence != 90 {
		t.Errorf("brief recorded depth=%q target=%d, want deep/90", brief.Depth, brief.TargetConfidence)
	}
	if len(brief.SuccessCriteria) != 2 {
		t.Errorf("brief recorded %d success criteria, want 2", len(brief.SuccessCriteria))
	}
	if brief.Template != "tech-eval" {
		t.Errorf("brief resolved template %q, want tech-eval", brief.Template)
	}
}

// TestApprovedCoreQuestionBecomesFirstResearchQuestion is the reason the brief
// gate exists. Before it, the loop opened on a generated question that spliced
// the raw topic into a template -- one live run spent its first iteration on a
// 300-character question the operator never asked.
func TestApprovedCoreQuestionBecomesFirstResearchQuestion(t *testing.T) {
	topic := strings.Repeat("Comprehensive review of the ingest and sync subsystems with recovery planning. ", 6)
	coreQuestion := "Should the local cache use SQLite or Postgres?"
	profile, err := resolveOracleScope(topic, "both")
	if err != nil {
		t.Fatalf("resolve scope: %v", err)
	}

	questions := buildOracleQuestionPlan(topic, "# Oracle Research Brief\n", "go", coreQuestion, profile)
	if len(questions) == 0 {
		t.Fatal("no questions generated")
	}
	if questions[0].Text != coreQuestion {
		t.Fatalf("first research question = %q, want the approved core question %q", questions[0].Text, coreQuestion)
	}
	if questions[0].ID != "q1" {
		t.Errorf("approved question got ID %q, want q1", questions[0].ID)
	}
	for i, question := range questions {
		want := fmt.Sprintf("q%d", i+1)
		if question.ID != want {
			t.Fatalf("question %d has ID %q, want %q -- IDs must stay contiguous after the core question is prepended", i, question.ID, want)
		}
	}

	// Without an approved brief the generated questions are unchanged, so the
	// express path keeps working.
	fallback := buildOracleQuestionPlan(topic, "# Oracle Research Brief\n", "go", "", profile)
	generated := buildBriefInformedQuestions(topic, "# Oracle Research Brief\n", "go", profile)
	if len(fallback) != len(generated) {
		t.Fatalf("no-brief path changed the generated question set: %d vs %d", len(fallback), len(generated))
	}
}

func TestApprovedBriefOnlyAppliesToItsOwnTopic(t *testing.T) {
	root := t.TempDir()
	if _, err := runOracleBriefApprove(root, oracleBriefOptions{
		Topic:        "cache storage",
		CoreQuestion: "Should the local cache use SQLite or Postgres?",
		Depth:        "quick",
	}, false); err != nil {
		t.Fatalf("approve brief: %v", err)
	}

	if brief := approvedBriefForTopic(root, "cache storage"); brief == nil {
		t.Fatal("the brief did not apply to the topic it was approved for")
	}
	if brief := approvedBriefForTopic(root, "how should we structure the CI pipeline"); brief != nil {
		t.Fatalf("a brief approved for %q attached itself to an unrelated topic", brief.Topic)
	}
}

func TestOracleStartConsumesPendingBrief(t *testing.T) {
	root := t.TempDir()
	if _, err := runOracleBriefApprove(root, oracleBriefOptions{
		Topic:        "cache storage",
		CoreQuestion: "Should the local cache use SQLite or Postgres?",
		Depth:        "quick",
	}, false); err != nil {
		t.Fatalf("approve brief: %v", err)
	}

	// Starting a run archives the workspace, which is what retires the brief.
	paths := oracleWorkspacePaths(root)
	if err := archiveOracleWorkspace(paths); err != nil {
		t.Fatalf("archive workspace: %v", err)
	}

	if _, err := os.Stat(oraclePendingBriefPath(root)); !os.IsNotExist(err) {
		t.Fatal("the approved brief survived the run that used it; a later run could silently inherit it")
	}
	archived, _ := filepath.Glob(filepath.Join(paths.Dir, "archive", "*", oraclePendingBriefFileName))
	if len(archived) == 0 {
		t.Error("the approved brief was discarded rather than archived with its run")
	}
}

func TestOracleCoreQuestionStaysReadable(t *testing.T) {
	// The live defect this guards: a question built by splicing a 1400-char
	// topic into a template, which no worker can answer as asked.
	long := "Should we " + strings.Repeat("rework the ingest path and ", 30) + "ship it?"
	if err := validateOracleCoreQuestion(long); err == nil {
		t.Fatalf("a %d-character core question was accepted", len([]rune(long)))
	}
	if err := validateOracleCoreQuestion("Should the local cache use SQLite or Postgres?"); err != nil {
		t.Fatalf("a well-formed core question was rejected: %v", err)
	}
}
