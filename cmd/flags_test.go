package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestFlagsListJSON(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	bindSeededCommandTestRepository(t)

	rootCmd.SetArgs([]string{"flag-list", "--json"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("flag-list --json returned error: %v", err)
	}

	output := buf.String()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
	}
	if envelope["ok"] != true {
		t.Errorf("expected ok=true, got: %v", envelope["ok"])
	}
	result, ok := envelope["result"].(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}
	flags, ok := result["flags"].([]interface{})
	if !ok {
		t.Fatalf("result.flags is not an array, got: %T", result["flags"])
	}
	if len(flags) != 3 {
		t.Errorf("expected 3 flags, got %d", len(flags))
	}
}

func TestFlagsListJSONEmpty(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	// Create store with no flags file.
	binding := bindCommandTestRepository(t)

	state := `{"version":"3.0","goal":"test","state":"READY","current_phase":1,"plan":{"phases":[]},"events":[],"memory":{"phase_learnings":[],"decisions":[],"instincts":[]},"errors":{"records":[]}}`
	if err := os.WriteFile(binding.DataDir+"/COLONY_STATE.json", []byte(state), 0644); err != nil {
		t.Fatalf("write empty colony state: %v", err)
	}

	rootCmd.SetArgs([]string{"flag-list", "--json"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("flag-list --json with no flags returned error: %v", err)
	}

	output := buf.String()
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("output is not valid JSON: %v, got: %s", err, output)
	}
	if envelope["ok"] != true {
		t.Errorf("expected ok=true, got: %v", envelope["ok"])
	}
	result := envelope["result"].(map[string]interface{})
	flags := result["flags"].([]interface{})
	if len(flags) != 0 {
		t.Errorf("expected 0 flags for empty case, got %d", len(flags))
	}
}

func TestFlagsList(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	bindSeededCommandTestRepository(t)

	rootCmd.SetArgs([]string{"flag-list"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("flag-list returned error: %v", err)
	}

	output := buf.String()
	// Testdata has 3 flags: blocker, issue, note
	if !strings.Contains(output, "flag_001") {
		t.Errorf("expected flag_001, got: %s", output)
	}
	if !strings.Contains(output, "flag_002") {
		t.Errorf("expected flag_002, got: %s", output)
	}
	if !strings.Contains(output, "Critical dependency missing") {
		t.Errorf("expected flag description, got: %s", output)
	}
}

func TestFlagsAlias(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	bindSeededCommandTestRepository(t)

	rootCmd.SetArgs([]string{"flags"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("flags alias returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "flag_001") {
		t.Errorf("expected flag_001 via alias, got: %s", output)
	}
}

func TestFlagsFilterByType(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	bindSeededCommandTestRepository(t)

	rootCmd.SetArgs([]string{"flag-list", "--type", "blocker"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("flag-list --type blocker returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "flag_001") {
		t.Errorf("expected blocker flag_001, got: %s", output)
	}
	if strings.Contains(output, "flag_002") {
		t.Errorf("did not expect issue flag_002 when filtering by blocker, got: %s", output)
	}
	if strings.Contains(output, "flag_003") {
		t.Errorf("did not expect note flag_003 when filtering by blocker, got: %s", output)
	}
}

func TestFlagsFilterByStatus(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	bindSeededCommandTestRepository(t)

	rootCmd.SetArgs([]string{"flag-list", "--status", "resolved"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("flag-list --status resolved returned error: %v", err)
	}

	output := buf.String()
	// Only flag_003 is resolved
	if !strings.Contains(output, "flag_003") {
		t.Errorf("expected resolved flag_003, got: %s", output)
	}
	if strings.Contains(output, "flag_001") {
		t.Errorf("did not expect active flag_001 when filtering by resolved, got: %s", output)
	}
}

func TestFlagAddCreatesFlag(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	binding := bindCommandTestRepository(t)
	s := binding.Store

	rootCmd.SetArgs([]string{"flag-add", "--title", "Test blocker", "--type", "blocker", "--severity", "high", "--description", "Something is broken"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("flag-add returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("invalid JSON output: %v, got: %s", err, output)
	}
	if envelope["ok"] != true {
		t.Fatalf("expected ok:true, got: %s", output)
	}
	result := envelope["result"].(map[string]interface{})
	if result["created"] != true {
		t.Fatalf("expected created:true, got: %v", result["created"])
	}

	// Verify filesystem
	var ff colony.FlagsFile
	if err := s.LoadJSON("pending-decisions.json", &ff); err != nil {
		t.Fatalf("failed to load flags: %v", err)
	}
	if len(ff.Decisions) != 1 {
		t.Fatalf("expected 1 flag, got %d", len(ff.Decisions))
	}
	if ff.Decisions[0].Type != "blocker" {
		t.Fatalf("expected type=blocker, got %q", ff.Decisions[0].Type)
	}
}

func TestFlagResolveUpdatesFlag(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	binding := bindCommandTestRepository(t)
	s := binding.Store

	// Add a flag first
	rootCmd.SetArgs([]string{"flag-add", "--title", "To resolve", "--type", "issue", "--severity", "low"})
	_ = rootCmd.Execute()

	// Extract the flag ID from output
	output := strings.TrimSpace(buf.String())
	var envelope map[string]interface{}
	json.Unmarshal([]byte(output), &envelope)
	result := envelope["result"].(map[string]interface{})
	flagObj := result["flag"].(map[string]interface{})
	id := flagObj["id"].(string)

	// Resolve it
	buf.Reset()
	rootCmd.SetArgs([]string{"flag-resolve", "--id", id, "--message", "Fixed in commit abc"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("flag-resolve returned error: %v", err)
	}

	output = strings.TrimSpace(buf.String())
	json.Unmarshal([]byte(output), &envelope)
	if envelope["ok"] != true {
		t.Fatalf("expected ok:true, got: %s", output)
	}
	resResult := envelope["result"].(map[string]interface{})
	if resResult["resolved"] != true {
		t.Fatalf("expected resolved:true, got: %v", resResult["resolved"])
	}

	// Verify filesystem
	var ff colony.FlagsFile
	if err := s.LoadJSON("pending-decisions.json", &ff); err != nil {
		t.Fatalf("failed to load flags: %v", err)
	}
	if !ff.Decisions[0].Resolved {
		t.Fatalf("expected flag to be resolved")
	}
	if ff.Decisions[0].Resolution != "Fixed in commit abc" {
		t.Fatalf("expected resolution message, got %q", ff.Decisions[0].Resolution)
	}
}

func TestFlagCheckBlockersCounts(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf

	bindCommandTestRepository(t)

	// Add flags of different types
	rootCmd.SetArgs([]string{"flag-add", "--title", "Blocker 1", "--type", "blocker", "--severity", "critical"})
	_ = rootCmd.Execute()
	buf.Reset()
	rootCmd.SetArgs([]string{"flag-add", "--title", "Issue 1", "--type", "issue", "--severity", "high"})
	_ = rootCmd.Execute()
	buf.Reset()
	rootCmd.SetArgs([]string{"flag-add", "--title", "Note 1", "--type", "note", "--severity", "low"})
	_ = rootCmd.Execute()

	buf.Reset()
	rootCmd.SetArgs([]string{"flag-check-blockers"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("flag-check-blockers returned error: %v", err)
	}

	output := strings.TrimSpace(buf.String())
	var envelope map[string]interface{}
	json.Unmarshal([]byte(output), &envelope)
	if envelope["ok"] != true {
		t.Fatalf("expected ok:true, got: %s", output)
	}
	result := envelope["result"].(map[string]interface{})
	if result["blockers"] != float64(1) {
		t.Fatalf("expected blockers=1, got %v", result["blockers"])
	}
	if result["issues"] != float64(1) {
		t.Fatalf("expected issues=1, got %v", result["issues"])
	}
	if result["notes"] != float64(1) {
		t.Fatalf("expected notes=1, got %v", result["notes"])
	}
	if result["has_blockers"] != true {
		t.Fatalf("expected has_blockers=true, got %v", result["has_blockers"])
	}
}
