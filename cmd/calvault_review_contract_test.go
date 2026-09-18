package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Exercise the real producer schema, parser and continue consumer together.
// Separate producer/consumer tests missed the impossible Auditor contract.
func TestCalVaultAuditorProducerConsumerContract(t *testing.T) {
	const output = `{
		"ant_name":"Exam-83", "caste":"auditor", "task_id":"audit", "status":"completed",
		"summary":"Audit completed", "files_created":[], "files_modified":[], "tests_written":[],
		"task_receipts":[], "tool_count":3, "blockers":[], "spawns":[],
		"artifacts":{"research_file":null,"survey_file":null,"plan_file":null,
			"review":{"overall_score":87,"findings":[{"domain":"quality","severity":"LOW",
			"file":"main.go","line":12,"category":"maintainability","title":"Duplicated branch",
			"description":"Two branches repeat a check","suggestion":"Share the check","blocking":false}]}},
		"handoff":{"changed_files":[],"commands_run":["go test ./..."],"verification_status":"pass",
			"known_failures":[],"open_decisions":[],"assumptions":[],"next_worker_instructions":["Read review"],
			"do_not_repeat":[],"freshness":"current"}
	}`
	for _, caste := range []string{"auditor", "gatekeeper", "builder"} {
		t.Run(caste, func(t *testing.T) {
			schemaBytes, err := codex.WorkerOutputSchema(codex.WorkerConfig{Caste: caste})
			if err != nil {
				t.Fatal(err)
			}
			var schemaDoc any
			if err := json.Unmarshal(schemaBytes, &schemaDoc); err != nil {
				t.Fatal(err)
			}
			compiler := jsonschema.NewCompiler()
			compiler.UseLoader(nativeGapNoSchemaLoader{})
			if err := compiler.AddResource("urn:calvault:actual-worker-schema", schemaDoc); err != nil {
				t.Fatal(err)
			}
			schema, err := compiler.Compile("urn:calvault:actual-worker-schema")
			if err != nil {
				t.Fatal(err)
			}
			for _, mutation := range []string{"valid", "missing", "null", "score", "severity", "extra"} {
				t.Run(mutation, func(t *testing.T) {
					var doc map[string]any
					if err := json.Unmarshal([]byte(output), &doc); err != nil {
						t.Fatal(err)
					}
					doc["caste"] = caste
					artifacts := doc["artifacts"].(map[string]any)
					review := artifacts["review"].(map[string]any)
					if caste == "gatekeeper" {
						delete(review, "overall_score")
					}
					switch mutation {
					case "missing":
						delete(artifacts, "review")
					case "null":
						artifacts["review"] = nil
					case "score":
						review["overall_score"] = float64(101)
					case "severity":
						review["findings"].([]any)[0].(map[string]any)["severity"] = "NOT_A_SEVERITY"
					case "extra":
						artifacts["invented"] = "not allowed"
					}
					if caste == "builder" && mutation == "valid" {
						delete(artifacts, "review")
					}
					validationErr := schema.Validate(doc)
					if mutation == "valid" && validationErr != nil {
						t.Fatalf("actual schema rejects valid %s: %v", caste, validationErr)
					}
					if mutation == "extra" && validationErr == nil {
						t.Fatal("strict artifacts accepted an unknown property")
					}
					raw, err := json.Marshal(doc)
					if err != nil {
						t.Fatal(err)
					}
					claims, err := codex.ParseWorkerOutput(string(raw))
					if err != nil {
						t.Fatal(err)
					}
					step := normalizeContinueReviewEvidence(codexContinueWorkerFlowStep{
						Stage: "review", Name: claims.AntName, Caste: claims.Caste, Status: claims.Status,
					}, claims.Artifacts)
					if mutation == "valid" {
						if len(step.EvidenceErrors) != 0 {
							t.Fatalf("producer result rejected by consumer: %v", step.EvidenceErrors)
						}
						if caste == "gatekeeper" && (step.OverallScore != nil || len(step.Findings) != 1) {
							t.Fatalf("lost Gatekeeper findings or invented score: %+v", step)
						}
						if caste == "auditor" && (step.OverallScore == nil || *step.OverallScore != 87 || len(step.Findings) != 1) {
							t.Fatalf("lost review: %+v", step)
						}
					} else if caste == "auditor" && mutation != "extra" && len(step.EvidenceErrors) == 0 {
						t.Fatalf("invalid completed Auditor evidence accepted: %s", mutation)
					}
				})
			}
		})
	}
}

func TestCalVaultTimeoutMeasurementsRemainUnknown(t *testing.T) {
	saveGlobals(t)
	setVisualOutputMode(t, "visual")
	for _, tc := range []struct {
		name     string
		known    bool
		observed int
		want     string
	}{
		{"silent timeout", false, 0, "tool calls " + spendMarkNotReported},
		{"partial activity", false, 2, "tool calls " + spendMarkNotReported},
		{"reported zero", true, 0, "0 tool calls"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			worker := codex.WorkerResult{WorkerName: "Hammer-71", Status: "timeout", Duration: 10 * time.Minute,
				ToolCountReported: tc.known, ObservedToolCalls: tc.observed, DiagnosticPath: ".aether/data/worker-debug/timeout.json"}
			var out strings.Builder
			old := stdout
			stdout = &out
			t.Cleanup(func() { stdout = old })
			emitCodexDispatchWorkerFinished(codex.WorkerDispatch{WorkerName: worker.WorkerName, Caste: "builder"},
				codex.DispatchResult{WorkerName: worker.WorkerName, Status: worker.Status, WorkerResult: &worker})
			if !strings.Contains(out.String(), tc.want) {
				t.Fatalf("misleading timeout: %s", out.String())
			}
			mapped := mapInternalWorkerResult(worker, nil)
			if mapped.ToolCountReported != tc.known || mapped.ObservedToolCalls != tc.observed || mapped.DiagnosticPath != worker.DiagnosticPath {
				t.Fatalf("adapter lost timeout evidence: %+v", mapped)
			}
		})
	}
}
