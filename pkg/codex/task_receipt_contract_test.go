package codex

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestTaskReceiptContractRoundTripPreservesTaskEvidence(t *testing.T) {
	want := TaskReceipt{
		TaskID:        "task-copy-models",
		Status:        TaskReceiptStatusCompleted,
		Summary:       "Copied the model templates and verified the generated output.",
		FilesCreated:  []string{"models/new.go"},
		FilesModified: []string{"models/existing.go"},
		TestsWritten:  []string{"models/existing_test.go"},
		Handoff: WorkerHandoff{
			ChangedFiles:       []string{"models/new.go", "models/existing.go", "models/existing_test.go"},
			CommandsRun:        []string{"go test ./models"},
			VerificationStatus: "pass",
			Assumptions:        []string{"The generated fixture is authoritative."},
			NextWorkerInstructions: []string{
				"Use the verified model output when wiring completion credit.",
			},
			DoNotRepeat: []string{"Do not regenerate the fixture by hand."},
			Freshness:   "2026-08-27T00:00:00Z",
		},
	}

	payload, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal TaskReceipt: %v", err)
	}

	var got TaskReceipt
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal TaskReceipt: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("TaskReceipt round trip\n got: %#v\nwant: %#v", got, want)
	}

	var wire map[string]json.RawMessage
	if err := json.Unmarshal(payload, &wire); err != nil {
		t.Fatalf("unmarshal TaskReceipt wire object: %v", err)
	}
	for _, forbidden := range []string{"requirements", "criteria", "hash"} {
		if _, exists := wire[forbidden]; exists {
			t.Fatalf("TaskReceipt must not carry authoritative %q data: %s", forbidden, payload)
		}
	}
}

func TestWorkerClaimsTaskReceiptsPreserveTaskSpecificEvidence(t *testing.T) {
	payload := `{
		"ant_name":"Mason-67",
		"caste":"builder",
		"task_id":"job-copy-templates",
		"status":"completed",
		"summary":"Completed both covered tasks.",
		"files_created":["aggregate/new.go"],
		"files_modified":["aggregate/existing.go"],
		"tests_written":["aggregate/existing_test.go"],
		"artifacts":{},
		"tool_count":3,
		"blockers":[],
		"spawns":[],
		"handoff":{"changed_files":[],"commands_run":[],"verification_status":"pass","known_failures":[],"open_decisions":[],"assumptions":[],"next_worker_instructions":[],"do_not_repeat":[],"freshness":"2026-08-27T00:00:00Z"},
		"task_receipts":[
			{
				"task_id":"task-create",
				"status":"completed",
				"summary":"Created the new model.",
				"files_created":["models/new.go"],
				"files_modified":[],
				"tests_written":["models/new_test.go"],
				"handoff":{"changed_files":["models/new.go","models/new_test.go"],"commands_run":["go test ./models -run TestNew"],"verification_status":"pass","known_failures":[],"open_decisions":[],"assumptions":[],"next_worker_instructions":["Wire task-create credit."],"do_not_repeat":[],"freshness":"2026-08-27T00:00:00Z"}
			},
			{
				"task_id":"task-update",
				"status":"completed_no_change",
				"summary":"Verified the existing model already matched the contract.",
				"files_created":[],
				"files_modified":["models/existing.go"],
				"tests_written":["models/existing_test.go"],
				"handoff":{"changed_files":["models/existing.go","models/existing_test.go"],"commands_run":["go test ./models -run TestExisting"],"verification_status":"pass","known_failures":[],"open_decisions":[],"assumptions":["Existing output is authoritative."],"next_worker_instructions":[],"do_not_repeat":[],"freshness":"2026-08-27T00:01:00Z"}
			}
		]
	}`

	claims, err := ParseWorkerOutput(payload)
	if err != nil {
		t.Fatalf("ParseWorkerOutput: %v", err)
	}
	if len(claims.TaskReceipts) != 2 {
		t.Fatalf("TaskReceipts = %#v, want two task-specific receipts", claims.TaskReceipts)
	}

	created := claims.TaskReceipts[0]
	if created.TaskID != "task-create" || created.Status != TaskReceiptStatusCompleted {
		t.Fatalf("first receipt identity/status = %#v", created)
	}
	if !reflect.DeepEqual(created.FilesCreated, []string{"models/new.go"}) ||
		!reflect.DeepEqual(created.TestsWritten, []string{"models/new_test.go"}) {
		t.Fatalf("first receipt lost task-specific paths: %#v", created)
	}
	if created.Handoff.VerificationStatus != "pass" ||
		!reflect.DeepEqual([]string(created.Handoff.CommandsRun), []string{"go test ./models -run TestNew"}) ||
		!reflect.DeepEqual([]string(created.Handoff.ChangedFiles), []string{"models/new.go", "models/new_test.go"}) {
		t.Fatalf("first receipt lost verification evidence: %#v", created.Handoff)
	}

	verified := claims.TaskReceipts[1]
	if verified.TaskID != "task-update" || verified.Status != TaskReceiptStatusCompletedNoChange {
		t.Fatalf("second receipt identity/status = %#v", verified)
	}
	if !reflect.DeepEqual(verified.FilesModified, []string{"models/existing.go"}) ||
		!reflect.DeepEqual(verified.TestsWritten, []string{"models/existing_test.go"}) {
		t.Fatalf("second receipt lost task-specific paths: %#v", verified)
	}
}

func TestWorkerClaimsWithoutTaskReceiptsIsBackwardCompatible(t *testing.T) {
	payload := `{"ant_name":"Mason-67","caste":"builder","task_id":"legacy-task","status":"completed","summary":"Legacy whole-job success.","files_created":[],"files_modified":[],"tests_written":[],"artifacts":{},"tool_count":0,"blockers":[],"spawns":[],"handoff":{"changed_files":[],"commands_run":["go test ./..."],"verification_status":"pass","known_failures":[],"open_decisions":[],"assumptions":[],"next_worker_instructions":[],"do_not_repeat":[],"freshness":"2026-08-27T00:00:00Z"}}`

	claims, err := ParseWorkerOutput(payload)
	if err != nil {
		t.Fatalf("legacy ParseWorkerOutput: %v", err)
	}
	if claims.Status != "completed" || claims.TaskID != "legacy-task" || claims.Summary != "Legacy whole-job success." {
		t.Fatalf("legacy claims changed: %#v", claims)
	}
	if len(claims.TaskReceipts) != 0 {
		t.Fatalf("legacy claims unexpectedly gained receipts: %#v", claims.TaskReceipts)
	}
}

func TestTaskReceiptContractWorkerSchemaIsStrictAndAdditive(t *testing.T) {
	schema := workerClaimsSchema()
	receipts, ok := schema.Properties["task_receipts"].(map[string]interface{})
	if !ok {
		t.Fatalf("worker schema task_receipts missing or wrong type: %#v", schema.Properties["task_receipts"])
	}
	if receipts["type"] != "array" {
		t.Fatalf("task_receipts type = %#v, want array", receipts["type"])
	}
	item, ok := receipts["items"].(map[string]interface{})
	if !ok {
		t.Fatalf("task_receipts items missing or wrong type: %#v", receipts["items"])
	}
	if additional, ok := item["additionalProperties"].(bool); !ok || additional {
		t.Fatalf("task receipt additionalProperties = %#v, want false", item["additionalProperties"])
	}

	properties, ok := item["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("task receipt properties missing or wrong type: %#v", item["properties"])
	}
	wantProperties := []string{
		"task_id", "status", "summary", "files_created", "files_modified", "tests_written", "handoff",
	}
	if len(properties) != len(wantProperties) {
		t.Fatalf("task receipt properties = %#v, want exactly %v", properties, wantProperties)
	}
	for _, name := range wantProperties {
		if _, exists := properties[name]; !exists {
			t.Fatalf("task receipt schema missing %q: %#v", name, properties)
		}
	}
	for _, forbidden := range []string{"requirements", "criteria", "hash"} {
		if _, exists := properties[forbidden]; exists {
			t.Fatalf("task receipt schema must not declare %q", forbidden)
		}
	}

	for _, existingRequired := range []string{
		"ant_name", "caste", "task_id", "status", "summary", "files_created", "files_modified",
		"tests_written", "artifacts", "tool_count", "blockers", "spawns", "handoff",
	} {
		if !stringListContains(schema.Required, existingRequired) {
			t.Fatalf("worker schema lost existing required field %q: %v", existingRequired, schema.Required)
		}
	}
}
