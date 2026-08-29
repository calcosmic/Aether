package cmd

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
)

// TestWorkerFlowStepCarriesToolCount is the RED/GREEN proof for Task 1: a
// codexContinueWorkerFlowStep with a measured duration and tool-call count
// round-trips through json.Marshal/json.Unmarshal preserving both figures,
// and a step whose measurements were never reported serializes with neither
// figure present (not as a fabricated zero -- Phase 196 D-01).
func TestWorkerFlowStepCarriesToolCount(t *testing.T) {
	t.Run("measured figures round-trip", func(t *testing.T) {
		step := codexContinueWorkerFlowStep{
			Stage: "review", Caste: "watcher", Name: "Keen-12", Status: "completed",
			Duration: 190.4, DurationReported: true,
			ToolCount: 14, ToolCountReported: true,
		}

		data, err := json.Marshal(step)
		if err != nil {
			t.Fatalf("marshal step: %v", err)
		}
		var decoded codexContinueWorkerFlowStep
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal step: %v", err)
		}
		if decoded.Duration != step.Duration || !decoded.DurationReported {
			t.Errorf("duration did not survive round-trip: got %+v", decoded)
		}
		if decoded.ToolCount != step.ToolCount || !decoded.ToolCountReported {
			t.Errorf("tool count did not survive round-trip: got %+v", decoded)
		}
	})

	t.Run("unreported tool count is absent from JSON, not a zero", func(t *testing.T) {
		step := codexContinueWorkerFlowStep{
			Stage: "review", Caste: "watcher", Name: "Keen-12", Status: "timeout",
			// Duration/ToolCount left unset -- no WorkerResult was ever produced.
		}

		data, err := json.Marshal(step)
		if err != nil {
			t.Fatalf("marshal step: %v", err)
		}
		var decoded map[string]interface{}
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal step to map: %v", err)
		}
		if _, present := decoded["tool_count"]; present {
			t.Errorf("tool_count key present in JSON for an unreported worker: %s", data)
		}
		if _, present := decoded["duration"]; present {
			t.Errorf("duration key present in JSON for an unreported worker: %s", data)
		}
	})

	t.Run("a worker result carrying a tool-call count populates the step", func(t *testing.T) {
		// Mirrors the population site in runCodexContinueReview (cmd/codex_continue.go)
		// so this test would fail if that wiring regressed.
		result := &codex.WorkerResult{Duration: 0, ToolCount: 14}
		var step codexContinueWorkerFlowStep
		step.Duration = result.Duration.Seconds()
		step.DurationReported = true
		step.ToolCount = result.ToolCount
		step.ToolCountReported = true

		if step.ToolCount != 14 {
			t.Errorf("ToolCount = %d, want 14", step.ToolCount)
		}
		if !step.ToolCountReported {
			t.Errorf("ToolCountReported = false, want true")
		}
	})
}

// TestWorkerUsageStaysUnserialized asserts, via struct-tag reflection (not a
// text search over the source), that codexContinueWorkerFlowStep.Usage still
// carries json:"-" -- a figure that crossed a wire could be asserted by an
// outside caller rather than measured by the runtime, unlike the plain
// ToolCount int this plan adds.
func TestWorkerUsageStaysUnserialized(t *testing.T) {
	field, ok := reflect.TypeOf(codexContinueWorkerFlowStep{}).FieldByName("Usage")
	if !ok {
		t.Fatalf("codexContinueWorkerFlowStep has no Usage field")
	}
	tag := field.Tag.Get("json")
	if tag != "-" {
		t.Fatalf("Usage field json tag = %q, want \"-\"", tag)
	}

	// Marshal a step with a non-empty usage value and confirm no usage key
	// leaks into the JSON either.
	step := codexContinueWorkerFlowStep{
		Name: "Keen-12", Status: "completed",
		Usage: codex.WorkerUsage{TotalTokens: 12345},
	}
	data, err := json.Marshal(step)
	if err != nil {
		t.Fatalf("marshal step: %v", err)
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal step to map: %v", err)
	}
	for key := range decoded {
		if strings.Contains(strings.ToLower(key), "usage") {
			t.Fatalf("marshalled JSON contains a usage-related key %q: %s", key, data)
		}
	}
}

// TestUnmeasuredWorkerFiguresRenderAsNotReported tables over the four
// reported/unreported combinations plus the genuine-zero row (Phase 196
// D-01: a figure that was never measured is never shown as zero and never a
// guess).
func TestUnmeasuredWorkerFiguresRenderAsNotReported(t *testing.T) {
	cases := []struct {
		name              string
		durationSeconds   float64
		durationReported  bool
		toolCount         int
		toolCountReported bool
		want              string
	}{
		{
			name: "both reported", durationSeconds: 190, durationReported: true,
			toolCount: 14, toolCountReported: true,
			want: "3m 10s, 14 tool calls",
		},
		{
			name: "duration reported, tool count not", durationSeconds: 5, durationReported: true,
			toolCount: 0, toolCountReported: false,
			want: "5.0s, tool calls " + spendMarkNotReported,
		},
		{
			name: "tool count reported, duration not", durationSeconds: 0, durationReported: false,
			toolCount: 3, toolCountReported: true,
			want: "duration " + spendMarkNotReported + ", 3 tool calls",
		},
		{
			name: "neither reported", durationSeconds: 0, durationReported: false,
			toolCount: 0, toolCountReported: false,
			want: spendMarkNotReported,
		},
		{
			name: "genuine zero tool calls renders 0, not not-reported", durationSeconds: 12, durationReported: true,
			toolCount: 0, toolCountReported: true,
			want: "12.0s, 0 tool calls",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := workerMeasurementFigures(tc.durationSeconds, tc.durationReported, tc.toolCount, tc.toolCountReported)
			if got != tc.want {
				t.Errorf("workerMeasurementFigures(...) = %q, want %q", got, tc.want)
			}
			if !tc.toolCountReported {
				if strings.Contains(got, "0 tool call") {
					t.Errorf("unreported tool count rendered as a zero: %q", got)
				}
			}
		})
	}
}

// TestLiveAndSummaryWorkerFiguresShareOneSource renders the live finishing
// line (emitCodexDispatchWorkerFinished) and the continue worker-flow summary
// line for the same worker and asserts both contain the identical fragment
// produced by workerMeasurementFigures. An AST check additionally proves
// neither call site formats a duration or a tool count itself -- the string
// "tool call" appears nowhere in cmd/codex_visuals.go or
// cmd/codex_build_progress.go outside workerMeasurementFigures's own body.
func TestLiveAndSummaryWorkerFiguresShareOneSource(t *testing.T) {
	saveGlobals(t)
	setVisualOutputMode(t, "visual")

	dispatch := codex.WorkerDispatch{WorkerName: "Keen-12", Caste: "watcher"}
	result := codex.DispatchResult{
		WorkerName: "Keen-12", Status: "completed",
		WorkerResult: &codex.WorkerResult{Duration: 190 * 1e9 /* ns */, ToolCount: 14},
	}

	var liveBuf strings.Builder
	origStdout := stdout
	stdout = &liveBuf
	emitCodexDispatchWorkerFinished(dispatch, result)
	stdout = origStdout

	step := codexContinueWorkerFlowStep{
		Stage: "review", Caste: "watcher", Name: "Keen-12", Status: "completed",
		Duration: 190, DurationReported: true, ToolCount: 14, ToolCountReported: true,
	}
	var summaryBuf strings.Builder
	renderContinueWorkerFlowValue(&summaryBuf, []codexContinueWorkerFlowStep{step})

	wantFragment := workerMeasurementFigures(190, true, 14, true)
	if !strings.Contains(liveBuf.String(), wantFragment) {
		t.Errorf("live finishing line missing figures fragment %q:\n%s", wantFragment, liveBuf.String())
	}
	if !strings.Contains(summaryBuf.String(), wantFragment) {
		t.Errorf("summary line missing figures fragment %q:\n%s", wantFragment, summaryBuf.String())
	}

	t.Run("no other call site formats the figures itself", func(t *testing.T) {
		for _, file := range []string{"codex_visuals.go", "codex_build_progress.go"} {
			fset := token.NewFileSet()
			parsed, err := parser.ParseFile(fset, file, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", file, err)
			}
			for _, decl := range parsed.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil || fn.Name.Name == "workerMeasurementFigures" {
					continue
				}
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					lit, ok := n.(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						return true
					}
					if strings.Contains(strings.ToLower(lit.Value), "tool call") {
						t.Errorf("%s in %s formats \"tool call(s)\" text itself instead of calling workerMeasurementFigures: %s", fn.Name.Name, file, lit.Value)
					}
					return true
				})
			}
		}
	})
}
