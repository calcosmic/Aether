package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
)

func TestNarratorLauncherOffSuppressesLaunch(t *testing.T) {
	saveGlobals(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("AETHER_NARRATOR", "off")
	stdout = &bytes.Buffer{}

	started := false
	withNarratorLookPath(t, func(file string) (string, error) {
		started = true
		return "", nil
	})

	if launcher := maybeLaunchNarrator(context.Background(), t.TempDir()); launcher != nil {
		t.Fatal("expected narrator launcher to be suppressed")
	}
	if started {
		t.Fatal("launcher looked up node despite AETHER_NARRATOR=off")
	}
}

func TestNarratorLauncherAutoSkipsJSONMode(t *testing.T) {
	saveGlobals(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	t.Setenv("AETHER_NARRATOR", "auto")
	stdout = &bytes.Buffer{}

	if shouldLaunchNarrator() {
		t.Fatal("auto narrator must not launch in JSON mode")
	}
}

func TestNarratorLauncherOnSkipsJSONMode(t *testing.T) {
	saveGlobals(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	t.Setenv("AETHER_NARRATOR", "on")
	stdout = &bytes.Buffer{}

	if shouldLaunchNarrator() {
		t.Fatal("forced narrator must not pollute JSON mode")
	}
}

func TestNarratorLauncherAutoSkipsWhenNodeMissing(t *testing.T) {
	saveGlobals(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("AETHER_NARRATOR", "auto")
	stdout = &bytes.Buffer{}

	withNarratorLookPath(t, func(file string) (string, error) {
		return "", exec.ErrNotFound
	})

	if launcher := maybeLaunchNarrator(context.Background(), t.TempDir()); launcher != nil {
		t.Fatal("expected missing node to disable narrator without failing")
	}
}

func TestNarratorLauncherMissingRuntimeDoesNotFail(t *testing.T) {
	saveGlobals(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("AETHER_NARRATOR", "on")
	stdout = &bytes.Buffer{}

	withNarratorLookPath(t, func(file string) (string, error) {
		return "/usr/bin/node", nil
	})
	withNarratorRuntimePath(t, func(root string) (string, bool) {
		return "", false
	})

	if launcher := maybeLaunchNarrator(context.Background(), t.TempDir()); launcher != nil {
		t.Fatal("expected missing narrator runtime to disable narrator without failing")
	}
}

func TestNarratorLauncherUsesDistRuntimeDirectly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake node shell script test is POSIX-only")
	}
	saveGlobals(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("AETHER_NARRATOR", "on")
	var out bytes.Buffer
	stdout = &out

	root := t.TempDir()
	runtimePath := filepath.Join(root, ".aether", "ts", "dist", "narrator.js")
	if err := os.MkdirAll(filepath.Dir(runtimePath), 0755); err != nil {
		t.Fatalf("mkdir runtime dir: %v", err)
	}
	if err := os.WriteFile(runtimePath, []byte("runtime fixture\n"), 0644); err != nil {
		t.Fatalf("write runtime fixture: %v", err)
	}
	logPath := filepath.Join(root, "node-args.log")
	fakeNode := writeFakeNode(t, root, `#!/bin/sh
printf '%s\n' "$@" > "$AETHER_FAKE_NODE_ARGS"
while IFS= read -r line; do
  printf '[FAKE-NARRATOR] %s\n' "$line"
done
`)
	t.Setenv("AETHER_FAKE_NODE_ARGS", logPath)
	withNarratorLookPath(t, func(file string) (string, error) {
		return fakeNode, nil
	})

	launcher := maybeLaunchNarrator(context.Background(), root)
	if launcher == nil {
		t.Fatal("expected narrator launcher")
	}
	launcher.EmitEvent(testCeremonyEvent(t, events.CeremonyTopicBuildSpawn, events.CeremonyPayload{
		Phase:  2,
		Caste:  "builder",
		Name:   "Mason-67",
		Status: "starting",
	}))
	launcher.Close()

	args := readFileString(t, logPath)
	if !strings.Contains(filepath.ToSlash(args), ".aether/ts/dist/narrator.js") {
		t.Fatalf("launcher did not pass dist runtime path, args: %s", args)
	}
	for _, forbidden := range []string{"npm", "npx", "tsx", "narrator.ts"} {
		if strings.Contains(args, forbidden) {
			t.Fatalf("launcher args contained forbidden runtime tool %q: %s", forbidden, args)
		}
	}
	if !strings.Contains(out.String(), "[FAKE-NARRATOR]") {
		t.Fatalf("expected child stdout to route through Go visual output, got: %s", out.String())
	}
}

func TestNarratorLauncherOnStreamsCeremonyEventsToBundledRuntime(t *testing.T) {
	nodePath, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not found; skipping bundled narrator launcher smoke")
	}
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}
	runtimePath := filepath.Join(repoRoot, ".aether", "ts", "dist", "narrator.js")
	if _, err := os.Stat(runtimePath); err != nil {
		t.Fatalf("narrator runtime missing: %v", err)
	}

	saveGlobals(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("AETHER_NARRATOR", "on")
	var out bytes.Buffer
	stdout = &out
	withNarratorLookPath(t, func(file string) (string, error) {
		return nodePath, nil
	})

	launcher := maybeLaunchNarrator(context.Background(), repoRoot)
	if launcher == nil {
		t.Fatal("expected narrator launcher")
	}
	launcher.EmitEvent(testCeremonyEvent(t, events.CeremonyTopicBuildSpawn, events.CeremonyPayload{
		Phase:  2,
		Wave:   1,
		Caste:  "builder",
		Name:   "Mason-67",
		Status: "streamed",
	}))
	launcher.Close()

	if !strings.Contains(out.String(), "[CEREMONY] ceremony.build.spawn phase=2 wave=1") {
		t.Fatalf("narrator output mismatch:\n%s", out.String())
	}
}

func TestNarratorLauncherCloseCancelsStreamAndWaitsForRuntime(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake node shell script test is POSIX-only")
	}
	saveGlobals(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("AETHER_NARRATOR", "on")
	stdout = &bytes.Buffer{}

	root := t.TempDir()
	runtimePath := filepath.Join(root, ".aether", "ts", "dist", "narrator.js")
	if err := os.MkdirAll(filepath.Dir(runtimePath), 0755); err != nil {
		t.Fatalf("mkdir runtime dir: %v", err)
	}
	if err := os.WriteFile(runtimePath, []byte("runtime fixture\n"), 0644); err != nil {
		t.Fatalf("write runtime fixture: %v", err)
	}
	closedPath := filepath.Join(root, "closed")
	fakeNode := writeFakeNode(t, root, `#!/bin/sh
while IFS= read -r line; do
  :
done
printf closed > "$AETHER_FAKE_NODE_CLOSED"
`)
	t.Setenv("AETHER_FAKE_NODE_CLOSED", closedPath)
	withNarratorLookPath(t, func(file string) (string, error) {
		return fakeNode, nil
	})

	launcher := maybeLaunchNarrator(context.Background(), root)
	if launcher == nil {
		t.Fatal("expected narrator launcher")
	}
	start := time.Now()
	launcher.Close()
	if elapsed := time.Since(start); elapsed > narratorCloseTimeout+time.Second {
		t.Fatalf("launcher close took too long: %v", elapsed)
	}
	if got := readFileString(t, closedPath); got != "closed" {
		t.Fatalf("fake runtime did not observe stdin close, got %q", got)
	}
	launcher.Close()
}

func TestNarratorLauncherHandlesEarlyRuntimeExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake node shell script test is POSIX-only")
	}
	saveGlobals(t)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("AETHER_NARRATOR", "on")
	stdout = &bytes.Buffer{}

	root := t.TempDir()
	runtimePath := filepath.Join(root, ".aether", "ts", "dist", "narrator.js")
	if err := os.MkdirAll(filepath.Dir(runtimePath), 0755); err != nil {
		t.Fatalf("mkdir runtime dir: %v", err)
	}
	if err := os.WriteFile(runtimePath, []byte("runtime fixture\n"), 0644); err != nil {
		t.Fatalf("write runtime fixture: %v", err)
	}
	fakeNode := writeFakeNode(t, root, "#!/bin/sh\nexit 0\n")
	withNarratorLookPath(t, func(file string) (string, error) {
		return fakeNode, nil
	})

	launcher := maybeLaunchNarrator(context.Background(), root)
	if launcher == nil {
		t.Fatal("expected narrator launcher")
	}
	launcher.EmitEvent(testCeremonyEvent(t, events.CeremonyTopicBuildSpawn, events.CeremonyPayload{Name: "Mason-67"}))
	launcher.Close()
}

func TestBuildSyntheticNarratorDoesNotPolluteJSONOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	t.Setenv("AETHER_NARRATOR", "auto")

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("chdir test root: %v", err)
	}
	defer os.Chdir(oldDir)

	goal := "JSON output remains machine readable"
	taskID := "1.1"
	createTestColonyState(t, dataDir, testBuildState(goal, taskID))

	rootCmd.SetArgs([]string{"build", "1", "--synthetic"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}
	raw := stdout.(*bytes.Buffer).String()
	if strings.Contains(raw, "[CEREMONY]") {
		t.Fatalf("JSON output was polluted by narrator text:\n%s", raw)
	}
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &envelope); err != nil {
		t.Fatalf("build output is not valid JSON: %v\n%s", err, raw)
	}
	if envelope["ok"] != true {
		t.Fatalf("expected ok:true envelope, got %#v", envelope)
	}

	lines, err := store.ReadJSONL("event-bus.jsonl")
	if err != nil {
		t.Fatalf("expected ceremony events persisted in JSON mode: %v", err)
	}
	if len(lines) == 0 {
		t.Fatal("expected at least one persisted ceremony event")
	}
}

func withNarratorLookPath(t *testing.T, fn func(string) (string, error)) {
	t.Helper()
	original := narratorLookPath
	narratorLookPath = fn
	t.Cleanup(func() { narratorLookPath = original })
}

func withNarratorRuntimePath(t *testing.T, fn func(string) (string, bool)) {
	t.Helper()
	original := narratorRuntimePath
	narratorRuntimePath = fn
	t.Cleanup(func() { narratorRuntimePath = original })
}

func writeFakeNode(t *testing.T, dir, script string) string {
	t.Helper()
	path := filepath.Join(dir, "node")
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("write fake node: %v", err)
	}
	return path
}

func testCeremonyEvent(t *testing.T, topic string, payload events.CeremonyPayload) events.Event {
	t.Helper()
	raw, err := payload.RawMessage()
	if err != nil {
		t.Fatalf("payload marshal: %v", err)
	}
	now := time.Now().UTC()
	return events.Event{
		ID:        "evt_test",
		Topic:     topic,
		Payload:   raw,
		Source:    "unit-test",
		Timestamp: events.FormatTimestamp(now),
		TTLDays:   events.DefaultTTL,
		ExpiresAt: events.FormatTimestamp(events.ComputeExpiry(now, events.DefaultTTL)),
	}
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ""
		}
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
