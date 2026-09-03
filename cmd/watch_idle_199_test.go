package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func seedWatchIdle199Fixture(t *testing.T) (string, string) {
	t.Helper()
	dataDir, root := seedRunFixture(t, 1)
	stale := time.Date(2026, 9, 1, 8, 30, 0, 0, time.UTC).Format(time.RFC3339)
	mutateRunFixtureState(t, func(state *colony.ColonyState) {
		state.Events = []string{stale + "|build_completed|build|Phase 1 build was recorded"}
	})
	line := stale + "|Queen|builder|Mason-11|Implement Phase 1|1|working\n"
	if err := os.WriteFile(filepath.Join(dataDir, "spawn-tree.txt"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	return dataDir, root
}

func runWatchIdle199(t *testing.T, mode string) (map[string]interface{}, string) {
	t.Helper()
	t.Setenv("AETHER_OUTPUT_MODE", mode)
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"watch", "--once"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("watch returned error: %v", err)
	}
	rendered := stdout.(*bytes.Buffer).String()
	if mode == "json" {
		envelope := parseEnvelope(t, rendered)
		result, ok := envelope["result"].(map[string]interface{})
		if !ok {
			t.Fatalf("watch result = %#v", envelope["result"])
		}
		return result, rendered
	}
	return nil, rendered
}

func TestWatchIdle199StatusFallback(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	_, _ = seedWatchIdle199Fixture(t)

	result, _ := runWatchIdle199(t, "json")
	if result["idle_message"] != "No ants are active right now" {
		t.Fatalf("idle message = %v", result["idle_message"])
	}
	if result["authoritative_snapshot"] != "status" {
		t.Fatalf("authoritative snapshot = %v, want status", result["authoritative_snapshot"])
	}
	if result["projection_revision"] != LifecycleProjectionRevision || strings.TrimSpace(stringValue(result["captured_at"])) == "" {
		t.Fatalf("watch lacks projection revision/timestamp: %+v", result)
	}
	status, ok := result["status"].(map[string]interface{})
	if !ok {
		t.Fatalf("compact status projection = %#v", result["status"])
	}
	if stringValue(status["goal"]) == "" || intValue(status["current_phase"]) != 1 || intValue(status["total_phases"]) != 1 {
		t.Fatalf("compact status facts = %+v", status)
	}
	activity, ok := result["recent_activity"].([]interface{})
	if !ok || len(activity) == 0 {
		t.Fatalf("watch omitted recorded recent activity: %#v", result["recent_activity"])
	}
	row, _ := activity[0].(map[string]interface{})
	if strings.TrimSpace(stringValue(row["timestamp"])) == "" {
		t.Fatalf("recorded activity lacks its evidence timestamp: %+v", row)
	}
}

func TestWatchIdle199NoFakeLiveness(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	_, _ = seedWatchIdle199Fixture(t)

	result, _ := runWatchIdle199(t, "json")
	if intValue(result["active_count"]) != 0 || result["live_capability"] != "unsupported" {
		t.Fatalf("idle watch fabricated live capability: %+v", result)
	}
	if _, exists := result["current_run_id"]; exists {
		t.Fatalf("idle fallback inferred a current process/run: %+v", result)
	}

	_, visual := runWatchIdle199(t, "visual")
	for _, want := range []string{"No ants are active right now", "Status is the authoritative snapshot", "Recent recorded activity", "Live event source: unsupported"} {
		if !strings.Contains(visual, want) {
			t.Errorf("idle watch visual lacks %q:\n%s", want, visual)
		}
	}
	for _, forbidden := range []string{"Live colony activity view", "Refreshing automatically", "Press Ctrl+C", "Workers: 1 active", "spinner"} {
		if strings.Contains(visual, forbidden) {
			t.Errorf("idle watch visual contains fake-liveness phrase %q:\n%s", forbidden, visual)
		}
	}
}

func TestWatchIdle199ReadOnly(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	_, root := seedWatchIdle199Fixture(t)
	before := hashDirContents(t, root)
	_, _ = runWatchIdle199(t, "json")
	after := hashDirContents(t, root)
	if before != after {
		t.Fatalf("idle watch mutated the workspace: %s -> %s", before, after)
	}
	for _, forbidden := range []string{"watch-status.txt", "watch-progress.txt"} {
		if _, err := os.Stat(filepath.Join(root, ".aether", "data", forbidden)); !os.IsNotExist(err) {
			t.Fatalf("idle watch wrote deprecated snapshot artifact %s", forbidden)
		}
	}
}
