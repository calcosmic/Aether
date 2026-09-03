package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

func TestFrontDoorHelpGroups(t *testing.T) {
	root := t.TempDir()
	got := frontDoorHelpOutput199(t, root, 100)

	groups := []string{"Normal journey", "Steer and inspect", "Expert maintenance"}
	last := -1
	for _, group := range groups {
		index := strings.Index(got, group)
		if index < 0 {
			t.Fatalf("help is missing %q:\n%s", group, got)
		}
		if index <= last {
			t.Fatalf("help group %q is out of order:\n%s", group, got)
		}
		last = index
	}

	compact := strings.Join(strings.Fields(got), " ")
	wants := []string{
		`/ant-init "goal" Start a guided colony for one goal.`,
		`/ant-run Autopilot the remaining accepted phases within the displayed safety contract.`,
		`/ant-status Show the complete authoritative colony snapshot.`,
		`/ant-pause Stop at a safe boundary and save one resumable handoff.`,
		`/ant-resume Validate and restore the safest honest recovery point.`,
		`/ant-seal Close a verified colony, or explicitly record an owner-forced incomplete closure.`,
		`/ant-entomb Archive and clear the sealed colony.`,
		`/ant-maintenance Inspect or repair Aether internals with preview and rollback.`,
	}
	for _, want := range wants {
		if !strings.Contains(compact, want) {
			t.Errorf("help is missing exact command copy %q", want)
		}
	}
	for _, forbidden := range []string{"pause-colony", "resume-colony", "/ant-recover", "/ant-abandon", "lay-eggs", "finalize", "protocol"} {
		if strings.Contains(strings.ToLower(got), strings.ToLower(forbidden)) {
			t.Errorf("ordinary help exposes hidden plumbing %q", forbidden)
		}
	}
}

func TestFrontDoorHelpEmptyStanding(t *testing.T) {
	got := frontDoorHelpOutput199(t, t.TempDir(), 100)
	lines := strings.Split(strings.TrimSpace(got), "\n")
	want := []string{
		"No colony is active",
		`Start a guided colony for one goal with /ant-init "goal".`,
	}
	if len(lines) < len(want) || lines[0] != want[0] || lines[1] != want[1] {
		t.Fatalf("empty help opening = %q, want %q", lines[:min(len(lines), len(want))], want)
	}
}

func TestFrontDoorHelpActiveStanding(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, ".aether", "data")
	writeFrontDoorFile199(t, filepath.Join(dataDir, "COLONY_STATE.json"), `{
  "version":"3.0",
  "goal":"Ship the bridge",
  "colony_name":"Atlas",
  "state":"READY",
  "current_phase":2,
  "session_id":"bridge_7",
  "accepted_charter":{"schema_version":"accepted-charter/v1","episode_id":"bridge_7","goal":"Ship the bridge","provenance":"owner-provided","accepted_at":"2026-09-03T12:00:00Z"},
  "plan":{"phases":[{"id":1,"name":"Foundations","status":"completed"},{"id":2,"name":"Front door","status":"ready"},{"id":3,"name":"Finish","status":"pending"}]},
  "memory":{},"errors":{}
}`)
	writeFrontDoorFile199(t, filepath.Join(dataDir, "spawn-tree.txt"), strings.Join([]string{
		"2026-09-03T12:00:00Z|queen|builder|Mason-7|Build the front door|1|active",
		"2026-09-03T12:00:01Z|queen|watcher|Keen-2|Verify the front door|1|spawned",
	}, "\n")+"\n")

	got := frontDoorHelpOutput199(t, root, 180)
	first := strings.SplitN(strings.TrimSpace(got), "\n", 2)[0]
	want := "Colony: Atlas | Goal: Ship the bridge | Episode: bridge_7 | Phase: 2/3 | Standing: READY | Ants: Builder, Watcher | Blockers: 0 | Next Up: /ant-build 2 or /ant-run"
	if first != want {
		t.Fatalf("active standing line:\n got: %q\nwant: %q", first, want)
	}
}

func TestFrontDoorHelpResponsive(t *testing.T) {
	for _, width := range []int{63, 64, 100} {
		t.Run(fmt.Sprintf("width-%d", width), func(t *testing.T) {
			got := frontDoorHelpOutput199(t, t.TempDir(), width)
			for lineNumber, line := range strings.Split(got, "\n") {
				if len([]rune(line)) > width {
					t.Errorf("line %d is %d columns at width %d: %q", lineNumber+1, len([]rune(line)), width, line)
				}
			}
			for _, command := range []string{"/ant-init", "/ant-plan", "/ant-build", "/ant-run", "/ant-maintenance"} {
				if !strings.Contains(got, command) {
					t.Errorf("width %d truncated command meaning %q", width, command)
				}
			}
		})
	}
}

func TestFrontDoorInitBootstraps(t *testing.T) {
	root, _ := prepareFrontDoorInit199(t)
	if err := runFrontDoorInit199(t, "Build a calm front door"); err != nil {
		t.Fatalf("init: %v", err)
	}
	for _, rel := range []string{
		".aether/WHAT-IS-THIS.md",
		".aether/.gitignore",
		".aether/QUEEN.md",
		".aether/dreams",
		".aether/oracle",
		".aether/checkpoints",
		".aether/locks",
		".aether/data/COLONY_STATE.json",
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			t.Errorf("automatic bootstrap did not create %s: %v", rel, err)
		}
	}
}

func TestFrontDoorInitFiveStages(t *testing.T) {
	_, output := prepareFrontDoorInit199(t)
	if err := runFrontDoorInit199(t, "Build a calm front door"); err != nil {
		t.Fatalf("init: %v", err)
	}
	got := output.String()
	stages := []string{
		"── 1. Queen opening ──",
		"── 2. Setup ──",
		"── 3. Accepted intent ──",
		"── 4. Territory ──",
		"── 5. Closeout ──",
	}
	last := -1
	for _, stage := range stages {
		index := strings.Index(got, stage)
		if index < 0 {
			t.Fatalf("init is missing stage %q:\n%s", stage, got)
		}
		if index <= last {
			t.Fatalf("init stage %q is out of order:\n%s", stage, got)
		}
		last = index
	}
	if !strings.Contains(got, "Bootstrapped") || !strings.HasSuffix(strings.TrimSpace(got), "Next Up: /ant-plan") {
		t.Fatalf("init setup/closeout contract is incomplete:\n%s", got)
	}
}

func TestFrontDoorInitTerritoryStates(t *testing.T) {
	cases := []struct {
		name   string
		result SurveyFreshnessResult
		want   string
	}{
		{"fresh", SurveyFreshnessResult{Freshness: colony.SurveyFreshnessFresh}, "Fresh"},
		{"refreshed", SurveyFreshnessResult{Freshness: colony.SurveyFreshnessFresh, Refreshed: true}, "Refreshed"},
		{"stale", SurveyFreshnessResult{Freshness: colony.SurveyFreshnessStale}, "Stale—refresh required"},
		{"missing", SurveyFreshnessResult{Freshness: colony.SurveyFreshnessMissing}, "Stale—refresh required"},
		{"unavailable", SurveyFreshnessResult{Freshness: colony.SurveyFreshnessUnavailable}, "Unavailable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.result.OutcomeLabel(); got != tc.want {
				t.Fatalf("typed territory label = %q, want %q", got, tc.want)
			}
		})
	}

	_, output := prepareFrontDoorInit199(t)
	if err := runFrontDoorInit199(t, "Survey this repository"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if !strings.Contains(output.String(), "Territory: Stale—refresh required") {
		t.Fatalf("init did not render its typed territory result:\n%s", output.String())
	}
}

func TestFrontDoorInitCharterPersists(t *testing.T) {
	root, output := prepareFrontDoorInit199(t)
	if err := runFrontDoorInit199(t, "Build a calm front door", "--charter-json", `{"constraints":"No destructive setup","goals":"One normal journey"}`); err != nil {
		t.Fatalf("init: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(root, ".aether", "data", "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("read state: %v", err)
	}
	var state map[string]any
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	accepted, ok := state["accepted_charter"].(map[string]any)
	if !ok {
		t.Fatalf("state has no reloadable accepted_charter: %s", raw)
	}
	for field, want := range map[string]string{
		"schema_version": "accepted-charter/v1",
		"goal":           "Build a calm front door",
		"provenance":     "owner-provided",
	} {
		if got := fmt.Sprint(accepted[field]); got != want {
			t.Errorf("accepted_charter.%s = %q, want %q", field, got, want)
		}
	}
	if strings.TrimSpace(fmt.Sprint(accepted["episode_id"])) == "" || strings.TrimSpace(fmt.Sprint(accepted["accepted_at"])) == "" {
		t.Errorf("accepted charter is missing episode/time provenance: %#v", accepted)
	}
	for _, visible := range []string{"Build a calm front door", "No destructive setup", fmt.Sprint(accepted["episode_id"])} {
		if !strings.Contains(output.String(), visible) {
			t.Errorf("persisted charter value %q was not visible at acceptance", visible)
		}
	}
}

func TestFrontDoorInitRefusalZeroWrite(t *testing.T) {
	root, output := prepareFrontDoorInit199(t)
	goal := "Keep the active work"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan:         colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Active", Status: colony.PhaseInProgress}}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("seed active state: %v", err)
	}
	gitSurveyFreshness199(t, root, "init")
	gitSurveyFreshness199(t, root, "config", "user.email", "front-door@aether.invalid")
	gitSurveyFreshness199(t, root, "config", "user.name", "Front Door Test")
	gitSurveyFreshness199(t, root, "add", ".")
	gitSurveyFreshness199(t, root, "commit", "-m", "active fixture")

	home := os.Getenv("HOME")
	before := fingerprintLifecycleFactSurfaces(t, root, []string{root, home})
	if err := runFrontDoorInit199(t, "Replace the active work"); err != nil {
		t.Fatalf("refused init returned a Cobra error: %v", err)
	}
	after := fingerprintLifecycleFactSurfaces(t, root, []string{root, home})
	if fmt.Sprintf("%#v", after) != fmt.Sprintf("%#v", before) {
		t.Fatalf("active-colony refusal mutated repository, hub, registry, or Git:\nbefore=%#v\nafter=%#v", before, after)
	}
	for _, want := range []string{"active colony", "Keep the active work", "/ant-status"} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("refusal is missing %q:\n%s", want, output.String())
		}
	}
}

func frontDoorHelpOutput199(t *testing.T, root string, width int) string {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("AETHER_ROOT", root)
	t.Setenv("COLONY_DATA_DIR", filepath.Join(root, ".aether", "data"))
	t.Setenv("AETHER_PLATFORM", "claude")
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("NO_COLOR", "1")
	t.Setenv("COLUMNS", fmt.Sprint(width))
	store = nil
	var output bytes.Buffer
	stdout = &output
	stderr = &output
	rootCmd.SetOut(&output)
	rootCmd.SetErr(&output)
	t.Cleanup(func() { rootCmd.SetErr(os.Stderr) })
	rootCmd.SetArgs([]string{"--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("root help: %v", err)
	}
	return output.String()
}

func prepareFrontDoorInit199(t *testing.T) (string, *bytes.Buffer) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	root := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AETHER_ROOT", root)
	t.Setenv("COLONY_DATA_DIR", filepath.Join(root, ".aether", "data"))
	t.Setenv("AETHER_PLATFORM", "claude")
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("AETHER_HIVE_POLICY", "off")
	t.Setenv("NO_COLOR", "1")
	withWorkingDir(t, root)
	s, err := storage.NewStore(filepath.Join(root, ".aether", "data"))
	if err != nil {
		t.Fatalf("create test store: %v", err)
	}
	store = s
	output := &bytes.Buffer{}
	stdout = output
	stderr = output
	rootCmd.SetOut(output)
	rootCmd.SetErr(output)
	t.Cleanup(func() { rootCmd.SetErr(os.Stderr) })
	return root, output
}

func runFrontDoorInit199(t *testing.T, goal string, extra ...string) error {
	t.Helper()
	resetFlags(rootCmd)
	args := []string{"init", goal}
	args = append(args, extra...)
	rootCmd.SetArgs(args)
	return rootCmd.Execute()
}

func writeFrontDoorFile199(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
