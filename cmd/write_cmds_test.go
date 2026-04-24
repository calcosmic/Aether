package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/storage"
)

// newTestStore creates a fresh store in a temp dir for testing.
// It sets COLONY_DATA_DIR so rootCmd.PersistentPreRunE resolves correctly.
func newTestStore(t *testing.T) (*storage.Store, string) {
	t.Helper()
	origColonyDataDir := os.Getenv("COLONY_DATA_DIR")
	t.Cleanup(func() {
		os.Setenv("COLONY_DATA_DIR", origColonyDataDir)
	})
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/.aether/data"
	os.MkdirAll(dataDir, 0755)
	os.Setenv("COLONY_DATA_DIR", dataDir)
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	return s, tmpDir
}

// parseEnvelope parses a json_ok/json_err envelope.
func parseEnvelope(t *testing.T, output string) map[string]interface{} {
	t.Helper()
	output = strings.TrimSpace(output)
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(output), &m); err != nil {
		t.Fatalf("invalid JSON output: %s\nraw: %s", err, output)
	}
	return m
}

// --- Pheromone Write Tests ---

func TestPheromoneWrite(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"pheromone-write", "--type", "FOCUS", "--content", "test signal"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["created"] != true {
		t.Errorf("created = %v, want true", result["created"])
	}
	if result["total"] != float64(1) {
		t.Errorf("total = %v, want 1", result["total"])
	}
}

func TestPheromoneWriteRedirect(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"pheromone-write", "--type", "REDIRECT", "--content", "avoid this"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	signal := result["signal"].(map[string]interface{})
	if signal["priority"] != "high" {
		t.Errorf("REDIRECT priority = %v, want high", signal["priority"])
	}
	if signal["expires_at"] == nil {
		t.Error("REDIRECT should have expires_at set")
	}
}

func TestPheromoneWriteFeedback(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"pheromone-write", "--type", "FEEDBACK", "--content", "nice work"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	signal := result["signal"].(map[string]interface{})
	if signal["priority"] != "low" {
		t.Errorf("FEEDBACK priority = %v, want low", signal["priority"])
	}
}

func TestPheromoneWriteDedup(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// First write
	buf.Reset()
	rootCmd.SetArgs([]string{"pheromone-write", "--type", "FOCUS", "--content", "test dedup"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("first write error: %v", err)
	}
	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("first write not ok: %v", env)
	}

	// Second write with identical type and content
	buf.Reset()
	rootCmd.SetArgs([]string{"pheromone-write", "--type", "FOCUS", "--content", "test dedup"})
	err = rootCmd.Execute()
	if err != nil {
		t.Fatalf("second write error: %v", err)
	}
	env = parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("second write not ok: %v", env)
	}

	// Load pheromones.json and verify dedup behavior
	var pf colony.PheromoneFile
	s.LoadJSON("pheromones.json", &pf)

	if len(pf.Signals) != 1 {
		t.Errorf("signal count = %d, want 1 (duplicate should reinforce, not append)", len(pf.Signals))
	}

	if pf.Signals[0].ReinforcementCount == nil || *pf.Signals[0].ReinforcementCount < 1 {
		t.Errorf("reinforcement_count = %v, want >= 1 after one reinforcement", pf.Signals[0].ReinforcementCount)
	}
}

func TestPheromoneWriteInvalidType(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"pheromone-write", "--type", "INVALID", "--content", "test"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for invalid type, got: %v", env["ok"])
	}
}

func TestPheromoneWriteMissingFlags(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"pheromone-write"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for missing flags, got: %v", env["ok"])
	}
}

func TestPheromoneExpire(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// First create a signal
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:        "sig_test_1234",
				Type:      "FOCUS",
				Active:    true,
				CreatedAt: "2026-01-01T00:00:00Z",
				Content:   json.RawMessage(`{"text":"test"}`),
			},
		},
	}
	s.SaveJSON("pheromones.json", pf)

	rootCmd.SetArgs([]string{"pheromone-expire", "--id", "sig_test_1234"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["expired"] != true {
		t.Errorf("expired = %v, want true", result["expired"])
	}

	// Verify signal was deactivated
	var updated colony.PheromoneFile
	s.LoadJSON("pheromones.json", &updated)
	if updated.Signals[0].Active {
		t.Error("signal should be inactive after expire")
	}
}

func TestPheromoneExpireNotFound(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	s.SaveJSON("pheromones.json", colony.PheromoneFile{Signals: []colony.PheromoneSignal{}})

	rootCmd.SetArgs([]string{"pheromone-expire", "--id", "sig_nonexistent"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for non-existent signal, got: %v", env["ok"])
	}
}

// --- Flag Tests ---

func TestFlagAdd(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"flag-add", "--title", "test blocker", "--severity", "critical"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["created"] != true {
		t.Errorf("created = %v, want true", result["created"])
	}

	flag := result["flag"].(map[string]interface{})
	if flag["type"] != "issue" {
		t.Errorf("default type = %v, want issue", flag["type"])
	}
}

func TestFlagAddBlocker(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"flag-add", "--title", "build fails", "--severity", "critical", "--type", "blocker"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	flag := result["flag"].(map[string]interface{})
	if flag["type"] != "blocker" {
		t.Errorf("type = %v, want blocker", flag["type"])
	}
}

func TestFlagAddWithDescriptionAndPhase(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"flag-add", "--title", "build fails", "--severity", "critical", "--description", "detailed error info", "--source", "chaos-testing", "--phase", "3"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	flag := result["flag"].(map[string]interface{})

	if flag["description"] != "detailed error info" {
		t.Errorf("description = %v, want 'detailed error info'", flag["description"])
	}

	phaseVal := flag["phase"]
	phaseFloat, ok := phaseVal.(float64)
	if !ok || int(phaseFloat) != 3 {
		t.Errorf("phase = %v (%T), want 3", phaseVal, phaseVal)
	}
}

func TestFlagAddLegacyPositionalForm(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"flag-add", "blocker", "build fails", "detailed error info", "chaos-testing", "3"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	flag := result["flag"].(map[string]interface{})
	if got, want := flag["type"], "blocker"; got != want {
		t.Fatalf("type = %v, want %q", got, want)
	}
	if got, want := flag["description"], "detailed error info"; got != want {
		t.Fatalf("description = %v, want %q", got, want)
	}
	if got, want := flag["source"], "chaos-testing"; got != want {
		t.Fatalf("source = %v, want %q", got, want)
	}
	if got := int(flag["phase"].(float64)); got != 3 {
		t.Fatalf("phase = %d, want 3", got)
	}
}

func TestFlagAddPhaseZeroOmitsPhase(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"flag-add", "--title", "note flag", "--severity", "low", "--phase", "0"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	flag := result["flag"].(map[string]interface{})

	if flag["phase"] != nil {
		t.Errorf("phase = %v, want nil when phase is 0", flag["phase"])
	}
}

func TestFlagAddInvalidSeverity(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"flag-add", "--title", "test", "--severity", "invalid"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for invalid severity, got: %v", env["ok"])
	}
}

func TestFlagCheckBlockers(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	ff := colony.FlagsFile{
		Decisions: []colony.FlagEntry{
			{ID: "f1", Type: "blocker", Description: "critical bug", Resolved: false},
			{ID: "f2", Type: "issue", Description: "minor issue", Resolved: false},
		},
	}
	s.SaveJSON("pending-decisions.json", ff)

	rootCmd.SetArgs([]string{"flag-check-blockers"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["blockers"] != float64(1) {
		t.Errorf("blockers = %v, want 1", result["blockers"])
	}
	if result["issues"] != float64(1) {
		t.Errorf("issues = %v, want 1", result["issues"])
	}
	if result["has_blockers"] != true {
		t.Errorf("has_blockers = %v, want true", result["has_blockers"])
	}
}

func TestFlagResolve(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	ff := colony.FlagsFile{
		Decisions: []colony.FlagEntry{
			{ID: "flag_test_1234", Type: "blocker", Description: "test", Resolved: false},
		},
	}
	s.SaveJSON("pending-decisions.json", ff)

	rootCmd.SetArgs([]string{"flag-resolve", "--id", "flag_test_1234"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["resolved"] != true {
		t.Errorf("resolved = %v, want true", result["resolved"])
	}

	var updated colony.FlagsFile
	s.LoadJSON("pending-decisions.json", &updated)
	if !updated.Decisions[0].Resolved {
		t.Error("flag should be resolved after flag-resolve")
	}
}

func TestFlagAutoResolve(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create flags with old timestamps
	ff := colony.FlagsFile{
		Decisions: []colony.FlagEntry{
			{ID: "f1", Type: "issue", CreatedAt: "2020-01-01T00:00:00Z"},
			{ID: "f2", Type: "note", CreatedAt: "2026-01-01T00:00:00Z"},
		},
	}
	s.SaveJSON("pending-decisions.json", ff)

	rootCmd.SetArgs([]string{"flag-auto-resolve", "--max-days", "30"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["resolved"] != float64(2) {
		t.Errorf("resolved = %v, want 2", result["resolved"])
	}
}

func TestFlagResolveWithMessage(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	ff := colony.FlagsFile{
		Decisions: []colony.FlagEntry{
			{ID: "flag_msg_001", Type: "blocker", Description: "test blocker", Resolved: false},
		},
	}
	s.SaveJSON("pending-decisions.json", ff)

	rootCmd.SetArgs([]string{"flag-resolve", "--id", "flag_msg_001", "--message", "Fixed in commit abc123"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["resolved"] != true {
		t.Errorf("resolved = %v, want true", result["resolved"])
	}

	var updated colony.FlagsFile
	s.LoadJSON("pending-decisions.json", &updated)
	if !updated.Decisions[0].Resolved {
		t.Error("flag should be resolved after flag-resolve")
	}
	if updated.Decisions[0].Resolution != "Fixed in commit abc123" {
		t.Errorf("resolution = %q, want %q", updated.Decisions[0].Resolution, "Fixed in commit abc123")
	}
	if updated.Decisions[0].ResolvedAt == "" {
		t.Error("resolved_at should be set after flag-resolve")
	}
}

func TestFlagAcknowledgePersists(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	ff := colony.FlagsFile{
		Decisions: []colony.FlagEntry{
			{ID: "flag_ack_001", Type: "issue", Description: "test issue", Acknowledged: false},
		},
	}
	s.SaveJSON("pending-decisions.json", ff)

	rootCmd.SetArgs([]string{"flag-acknowledge", "--id", "flag_ack_001"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["acknowledged"] != true {
		t.Errorf("acknowledged = %v, want true", result["acknowledged"])
	}

	var updated colony.FlagsFile
	s.LoadJSON("pending-decisions.json", &updated)
	if !updated.Decisions[0].Acknowledged {
		t.Error("flag should be acknowledged in persisted file")
	}
}

// --- Spawn Tests ---

func TestSpawnLog(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"spawn-log", "--parent", "queen", "--caste", "builder", "--name", "worker-1", "--task", "build feature", "--depth", "1"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["recorded"] != true {
		t.Errorf("recorded = %v, want true", result["recorded"])
	}
	if result["parent"] != "queen" {
		t.Errorf("parent = %v, want queen", result["parent"])
	}
	if result["depth"] != float64(1) {
		t.Errorf("depth = %v, want 1", result["depth"])
	}
	if strings.TrimSpace(result["event_id"].(string)) == "" {
		t.Fatal("expected spawn-log to publish a ceremony event")
	}

	bus := events.NewBus(s, events.DefaultConfig())
	evts, err := bus.Query(context.Background(), events.CeremonyTopicBuildSpawn, time.Time{}, 10)
	if err != nil {
		t.Fatalf("query ceremony events: %v", err)
	}
	if len(evts) != 1 {
		t.Fatalf("ceremony events = %d, want 1", len(evts))
	}
	var payload events.CeremonyPayload
	if err := json.Unmarshal(evts[0].Payload, &payload); err != nil {
		t.Fatalf("decode ceremony payload: %v", err)
	}
	if payload.Caste != "builder" || payload.Name != "worker-1" || payload.Status != "spawned" {
		t.Fatalf("unexpected ceremony payload: %+v", payload)
	}
}

func TestSpawnLogLegacyFlagAliases(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"spawn-log", "--name", "Queen", "--caste", "builder", "--id", "Hammer-42", "--description", "implementing auth"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if got, want := result["parent"], "Queen"; got != want {
		t.Fatalf("parent = %v, want %q", got, want)
	}
	if got, want := result["name"], "Hammer-42"; got != want {
		t.Fatalf("name = %v, want %q", got, want)
	}
	if got, want := result["task"], "implementing auth"; got != want {
		t.Fatalf("task = %v, want %q", got, want)
	}
	if got := int(result["depth"].(float64)); got != 0 {
		t.Fatalf("depth = %d, want 0", got)
	}
}

func TestSpawnComplete(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create spawn tree with an active entry
	s.AtomicWrite("spawn-tree.txt", []byte("2026-01-01T00:00:00Z|queen|builder|worker-1|build|1|spawned\n"))

	rootCmd.SetArgs([]string{"spawn-complete", "--name", "worker-1"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["completed"] != true {
		t.Errorf("completed = %v, want true", result["completed"])
	}
	if result["status"] != "completed" {
		t.Errorf("status = %v, want completed", result["status"])
	}
	if strings.TrimSpace(result["event_id"].(string)) == "" {
		t.Fatal("expected spawn-complete to publish a ceremony event")
	}

	bus := events.NewBus(s, events.DefaultConfig())
	evts, err := bus.Query(context.Background(), events.CeremonyTopicBuildSpawn, time.Time{}, 10)
	if err != nil {
		t.Fatalf("query ceremony events: %v", err)
	}
	if len(evts) != 1 {
		t.Fatalf("ceremony events = %d, want 1", len(evts))
	}
	var payload events.CeremonyPayload
	if err := json.Unmarshal(evts[0].Payload, &payload); err != nil {
		t.Fatalf("decode ceremony payload: %v", err)
	}
	if payload.Caste != "builder" || payload.Name != "worker-1" || payload.Status != "completed" || payload.Task != "build" {
		t.Fatalf("unexpected ceremony payload: %+v", payload)
	}
}

func TestSpawnCompleteWithSummary(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create spawn tree with an active entry
	s.AtomicWrite("spawn-tree.txt", []byte("2026-01-01T00:00:00Z|queen|builder|worker-1|build|1|spawned\n"))

	rootCmd.SetArgs([]string{"spawn-complete", "--name", "worker-1", "--summary", "All tests passing"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["completed"] != true {
		t.Errorf("completed = %v, want true", result["completed"])
	}
	if result["status"] != "completed" {
		t.Errorf("status = %v, want completed", result["status"])
	}
	if result["summary"] != "All tests passing" {
		t.Errorf("summary = %v, want 'All tests passing'", result["summary"])
	}
}

func TestSpawnCompleteSummaryStoredInRecord(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create spawn tree with an active entry
	s.AtomicWrite("spawn-tree.txt", []byte("2026-01-01T00:00:00Z|queen|builder|worker-1|build|1|spawned\n"))

	rootCmd.SetArgs([]string{"spawn-complete", "--name", "worker-1", "--summary", "Fixed 3 bugs"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the summary is persisted in the spawn tree file
	st := agent.NewSpawnTree(s, "spawn-tree.txt")
	entries, err := st.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Summary != "Fixed 3 bugs" {
		t.Errorf("entry summary = %q, want %q", entries[0].Summary, "Fixed 3 bugs")
	}
}

func TestSpawnCompleteWithoutSummaryBackwardCompat(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create spawn tree with an active entry
	s.AtomicWrite("spawn-tree.txt", []byte("2026-01-01T00:00:00Z|queen|builder|worker-1|build|1|spawned\n"))

	rootCmd.SetArgs([]string{"spawn-complete", "--name", "worker-1"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["completed"] != true {
		t.Errorf("completed = %v, want true", result["completed"])
	}

	// Summary should not appear in output when not provided
	if _, exists := result["summary"]; exists {
		t.Errorf("summary should not be present in output when not provided, got %v", result["summary"])
	}

	// Verify the spawn tree file is valid and can be re-parsed
	st := agent.NewSpawnTree(s, "spawn-tree.txt")
	entries, err := st.Parse()
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Status != "completed" {
		t.Errorf("status = %q, want completed", entries[0].Status)
	}
}

func TestSpawnCanSpawn(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"spawn-can-spawn", "--depth", "3"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["can_spawn"] != true {
		t.Errorf("can_spawn = %v, want true", result["can_spawn"])
	}
}

func TestSpawnTreeDepth(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	s.AtomicWrite("spawn-tree.txt", []byte(
		"2026-01-01T00:00:00Z|queen|builder|w1|task|1|completed\n"+
			"2026-01-01T00:00:00Z|w1|builder|w2|task|2|completed\n"+
			"2026-01-01T00:00:00Z|w2|builder|w3|task|3|completed\n"))

	rootCmd.SetArgs([]string{"spawn-tree-depth"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["max_depth"] != float64(3) {
		t.Errorf("max_depth = %v, want 3", result["max_depth"])
	}
}

func TestSpawnEfficiency(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	s.AtomicWrite("spawn-tree.txt", []byte(
		"2026-01-01T00:00:00Z|queen|builder|w1|task|1|completed\n"+
			"2026-01-01T00:00:00Z|queen|builder|w2|task|1|spawned\n"))

	rootCmd.SetArgs([]string{"spawn-efficiency"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["total"] != float64(2) {
		t.Errorf("total = %v, want 2", result["total"])
	}
	if result["completed"] != float64(1) {
		t.Errorf("completed = %v, want 1", result["completed"])
	}
}

// --- State Mutation Tests ---

func TestStateMutate(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "build the thing"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"state-mutate", "--field", "goal", "--value", "new goal"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["updated"] != true {
		t.Errorf("updated = %v, want true", result["updated"])
	}
	if result["field"] != "goal" {
		t.Errorf("field = %v, want goal", result["field"])
	}

	// Verify the file was actually updated
	var updated colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &updated)
	if *updated.Goal != "new goal" {
		t.Errorf("goal = %q, want %q", *updated.Goal, "new goal")
	}
}

func TestStateMutateInvalidTransition(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	// READY -> BUILT is not allowed
	rootCmd.SetArgs([]string{"state-mutate", "--field", "state", "--value", "BUILT"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for invalid transition, got: %v", env["ok"])
	}
}

func TestLoadState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test goal"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"load-state"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["version"] != "3.0" {
		t.Errorf("version = %v, want 3.0", result["version"])
	}
}

func TestValidateState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"validate-state"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["valid"] != true {
		t.Errorf("valid = %v, want true", result["valid"])
	}
}

func TestValidateStateInvalid(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// State with missing required fields
	state := colony.ColonyState{Version: "3.0"}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"validate-state"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["valid"] != false {
		t.Errorf("valid = %v, want false for incomplete state", result["valid"])
	}
}

// --- Colony Tests ---

func TestColonyName(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	sessionID := "myproject_20260101T000000Z"
	goal := "Build the thing"
	state := colony.ColonyState{
		Version:   "3.0",
		SessionID: &sessionID,
		Goal:      &goal,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"colony-name"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["name"] != "myproject" {
		t.Errorf("name = %v, want myproject", result["name"])
	}
}

func TestColonyNameFallbackToGoal(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "Build the API"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"colony-name"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["name"] != "build" {
		t.Errorf("name = %v, want build (first word of goal, lowercased)", result["name"])
	}
}

func TestColonyDepthGet(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"colony-depth", "get"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["depth"] != "standard" {
		t.Errorf("depth = %v, want standard (default)", result["depth"])
	}
}

func TestColonyDepthSet(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test"
	state := colony.ColonyState{Version: "3.0", Goal: &goal}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"colony-depth", "set", "--depth", "deep"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["depth"] != "deep" {
		t.Errorf("depth = %v, want deep", result["depth"])
	}
}

func TestColonyDepthSetInvalid(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test"
	state := colony.ColonyState{Version: "3.0", Goal: &goal}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"colony-depth", "set", "--depth", "invalid"})

	rootCmd.Execute()

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok:false for invalid depth, got: %v", env["ok"])
	}
}

func TestDomainDetect(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create go.mod in project root (parent of .aether/data)
	projectRoot := filepath.Dir(filepath.Dir(s.BasePath()))
	os.WriteFile(filepath.Join(projectRoot, "go.mod"), []byte("module test\n"), 0644)

	rootCmd.SetArgs([]string{"domain-detect"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	domains := result["domains"].([]interface{})
	found := false
	for _, d := range domains {
		if d == "go" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("domains = %v, expected to contain go", domains)
	}
}

// --- Learning Tests ---

func TestLearningObserve(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"learning-observe", "--content", "test pattern", "--type", "pattern"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["captured"] != true {
		t.Errorf("captured = %v, want true", result["captured"])
	}
	if result["is_new"] != true {
		t.Errorf("is_new = %v, want true for first observation", result["is_new"])
	}
}

func TestLearningCheckPromotion(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create an observation with high trust score
	score := 0.8
	obs := colony.LearningFile{
		Observations: []colony.Observation{
			{
				ContentHash: "sha256:test123",
				Content:     "test pattern",
				WisdomType:  "pattern",
				TrustScore:  &score,
			},
		},
	}
	s.SaveJSON("learning-observations.json", obs)

	rootCmd.SetArgs([]string{"learning-check-promotion", "--observation-id", "sha256:test123"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["promotable"] != true {
		t.Errorf("promotable = %v, want true for high trust score", result["promotable"])
	}
}

func TestLearningPromoteAuto(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	score := 0.8
	obs := colony.LearningFile{
		Observations: []colony.Observation{
			{
				ContentHash: "sha256:eligible1",
				WisdomType:  "pattern",
				TrustScore:  &score,
			},
			{
				ContentHash:      "sha256:noteligible",
				WisdomType:       "philosophy",
				ObservationCount: 1,
			},
		},
	}
	s.SaveJSON("learning-observations.json", obs)

	rootCmd.SetArgs([]string{"learning-promote-auto"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["promoted"] != float64(1) {
		t.Errorf("promoted = %v, want 1", result["promoted"])
	}
}

// --- Activity Log Tests ---

func TestActivityLog(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"activity-log", "--command", "build", "--phase", "1"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["logged"] != true {
		t.Errorf("logged = %v, want true", result["logged"])
	}
}

func TestActivityLogRead(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Write an entry first
	s.AppendJSONL("activity-log.jsonl", map[string]string{
		"command": "test",
		"phase":   "1",
	})

	rootCmd.SetArgs([]string{"activity-log-read"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["count"] != float64(1) {
		t.Errorf("count = %v, want 1", result["count"])
	}
}

// --- Changelog Tests ---

func TestChangelogCollectPlanData(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Create a test plan file in the data dir (where store resolves)
	planContent := "---\nphase: 50\nplan: 3\ntype: execute\n---\nSome content"
	s.AtomicWrite("test-plan.md", []byte(planContent))

	rootCmd.SetArgs([]string{"changelog-collect-plan-data", "--plan-file", "test-plan.md"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env["ok"])
	}

	result := env["result"].(map[string]interface{})
	if result["phase"] != "50" {
		t.Errorf("phase = %v, want 50", result["phase"])
	}
	if result["plan"] != "3" {
		t.Errorf("plan = %v, want 3", result["plan"])
	}
}

// --- Pheromone Prime Tests ---

func TestPheromonePrime(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:      "sig_1",
				Type:    "FOCUS",
				Active:  true,
				Content: json.RawMessage(`{"text":"pay attention here"}`),
			},
			{
				ID:      "sig_2",
				Type:    "REDIRECT",
				Active:  true,
				Content: json.RawMessage(`{"text":"avoid this"}`),
			},
		},
	}
	s.SaveJSON("pheromones.json", pf)

	rootCmd.SetArgs([]string{"pheromone-prime"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["signal_count"] != float64(2) {
		t.Errorf("signal_count = %v, want 2", result["signal_count"])
	}
	if result["redirect_count"] != float64(1) {
		t.Errorf("redirect_count = %v, want 1", result["redirect_count"])
	}
	if result["focus_count"] != float64(1) {
		t.Errorf("focus_count = %v, want 1", result["focus_count"])
	}

	section := result["section"].(string)
	if !strings.Contains(section, "REDIRECT") {
		t.Error("section should contain REDIRECT signals")
	}
	if !strings.Contains(section, "FOCUS") {
		t.Error("section should contain FOCUS signals")
	}
}

// --- Validate Worker Response Test ---

func TestValidateWorkerResponse(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"validate-worker-response", "--response", `{"ok":true,"result":"done"}`, "--expect-json"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["valid"] != true {
		t.Errorf("valid = %v, want true for valid JSON", result["valid"])
	}
}

func TestValidateWorkerResponseInvalidJSON(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"validate-worker-response", "--response", "not json", "--expect-json"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["valid"] != false {
		t.Errorf("valid = %v, want false for invalid JSON", result["valid"])
	}
}
