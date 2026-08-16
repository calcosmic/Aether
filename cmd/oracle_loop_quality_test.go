package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func planWithUntouchedQuestions(n int) oraclePlanFile {
	questions := make([]oracleQuestion, 0, n)
	for i := 1; i <= n; i++ {
		questions = append(questions, oracleQuestion{
			ID:                fmt.Sprintf("q%d", i),
			Text:              fmt.Sprintf("Question %d?", i),
			Status:            "open",
			KeyFindings:       []oracleFinding{},
			IterationsTouched: []int{},
		})
	}
	return oraclePlanFile{Version: "1.1", Sources: map[string]oracleSource{}, Questions: questions}
}

// TestOracleDeepRunLeavesSurveyBeforeAQuarterOfBudget asserts a proportion, not
// a fixed round. Survey used to hold until every question had been touched
// once; with the nine questions a real run generated, that was the first nine
// rounds of a thirty-round run spent breadth-first.
func TestOracleDeepRunLeavesSurveyBeforeAQuarterOfBudget(t *testing.T) {
	plan := planWithUntouchedQuestions(9)
	const maxIterations = 30
	cap := oracleSurveyIterationCap(maxIterations)

	if cap > maxIterations/4+1 {
		t.Fatalf("survey cap %d exceeds a quarter of a %d-round run", cap, maxIterations)
	}

	state := oracleStateFile{MaxIterations: maxIterations, Iteration: cap + 1, OverallConfidence: 20, TargetConfidence: 95}
	if phase := nextOraclePhase(plan, state); phase == "survey" {
		t.Fatalf("still in survey at round %d of %d with %d questions untouched; a deep run must move on to investigate them properly",
			state.Iteration, maxIterations, len(plan.Questions))
	}

	// Early rounds still survey: breadth-first is the right start.
	early := oracleStateFile{MaxIterations: maxIterations, Iteration: 1, TargetConfidence: 95}
	if phase := nextOraclePhase(plan, early); phase != "survey" {
		t.Errorf("round 1 of a deep run is %q, want survey — the loop should still open breadth-first", phase)
	}
}

// A short run must still get a real survey; the cap has a floor.
func TestOracleQuickRunStillSurveys(t *testing.T) {
	plan := planWithUntouchedQuestions(5)
	state := oracleStateFile{MaxIterations: 5, Iteration: 2, TargetConfidence: 60}
	if phase := nextOraclePhase(plan, state); phase != "survey" {
		t.Errorf("a 5-round run left survey at round 2 (%q); the cap floor should keep at least three survey rounds", phase)
	}
}

// TestOracleDeepSurveyUsesMediumReasoning: asking for depth must not buy the
// shallowest setting for the opening third of the run.
func TestOracleDeepSurveyUsesMediumReasoning(t *testing.T) {
	deep := defaultOracleAttemptPolicy("survey", 1, 30)
	if deep.ReasoningEffort != "medium" {
		t.Errorf("deep-run survey reasoning = %q, want medium", deep.ReasoningEffort)
	}
	if deep.Timeout < 5*time.Minute {
		t.Errorf("deep-run survey watchdog = %s, want at least 5m", deep.Timeout)
	}

	quick := defaultOracleAttemptPolicy("survey", 1, 5)
	if quick.ReasoningEffort != "low" {
		t.Errorf("quick-run survey reasoning = %q, want low — a quick pass should stay cheap", quick.ReasoningEffort)
	}
	if quick.Timeout != 3*time.Minute {
		t.Errorf("quick-run survey watchdog = %s, want 3m", quick.Timeout)
	}

	// The other phases are unchanged by depth.
	if verify := defaultOracleAttemptPolicy("verify", 1, 5); verify.ReasoningEffort != "high" {
		t.Errorf("verify reasoning = %q, want high", verify.ReasoningEffort)
	}
}

func writeOracleColonyState(t *testing.T, root, goal string, state colony.State) {
	t.Helper()
	dir := filepath.Join(root, ".aether", "data")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	payload := map[string]interface{}{"goal": goal, "state": string(state)}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "COLONY_STATE.json"), data, 0644); err != nil {
		t.Fatalf("write state: %v", err)
	}
}

// TestOracleQuestionsOmitColonyGoalWithoutActiveColony: standalone research --
// the flow where someone researches first and starts a colony afterwards --
// must not inherit the goal of whatever project was there last.
func TestOracleQuestionsOmitColonyGoalWithoutActiveColony(t *testing.T) {
	t.Run("no colony at all", func(t *testing.T) {
		root := t.TempDir()
		if goal := loadColonyGoal(root); goal != "" {
			t.Errorf("a repo with no colony reported goal %q", goal)
		}
	})

	t.Run("idle colony", func(t *testing.T) {
		root := t.TempDir()
		writeOracleColonyState(t, root, "Fix TS host typecheck, resolve double-dispatch", colony.StateIDLE)
		if goal := loadColonyGoal(root); goal != "" {
			t.Errorf("an idle colony leaked its goal into research: %q", goal)
		}
	})

	t.Run("active colony", func(t *testing.T) {
		root := t.TempDir()
		writeOracleColonyState(t, root, "Ship the ingest rewrite", colony.StateEXECUTING)
		if goal := loadColonyGoal(root); goal != "Ship the ingest rewrite" {
			t.Errorf("an active colony's goal was dropped: %q", goal)
		}
	})
}

// TestOracleColonyGoalQuestionStaysReadable pins the invariant the live q6
// violated: a question spliced from a long topic and a long goal that ran past
// 300 characters and could not be answered as asked.
func TestOracleColonyGoalQuestionStaysReadable(t *testing.T) {
	longTopic := strings.Repeat("Comprehensive Aether system review and recovery planning across every surface. ", 12)
	longGoal := strings.Repeat("Fix TS host typecheck, resolve double-dispatch, restore ceremony surfaces. ", 4)
	brief := "## Colony Goal\n" + longGoal + "\n\n## Codebase Structure\n- go.mod\n"

	profile, err := resolveOracleScope(longTopic, "repo")
	if err != nil {
		t.Fatalf("resolve scope: %v", err)
	}
	questions := buildBriefInformedQuestions(longTopic, brief, "go", profile)

	found := false
	for _, question := range questions {
		if !strings.Contains(question.Text, "active colony goal") {
			continue
		}
		found = true
		if length := len([]rune(question.Text)); length > 240 {
			t.Errorf("the colony-goal question is %d characters:\n%s", length, question.Text)
		}
	}
	if !found {
		t.Fatal("no colony-goal question was generated for an active colony")
	}

	// Every generated question should be answerable as written.
	for _, question := range questions {
		if length := len([]rune(question.Text)); length > 400 {
			t.Errorf("question %s is %d characters, too long to answer as asked:\n%s", question.ID, length, question.Text)
		}
	}
}
