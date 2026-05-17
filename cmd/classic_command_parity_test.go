package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type classicCommandParityMatrix struct {
	SchemaVersion    string                       `json:"schema_version"`
	RuntimeAuthority string                       `json:"runtime_authority"`
	TSWritePolicy    string                       `json:"ts_write_policy"`
	Commands         []classicCommandParityRecord `json:"commands"`
}

type classicCommandParityRecord struct {
	Name               string                        `json:"name"`
	Category           string                        `json:"category"`
	RuntimeCommand     string                        `json:"runtime_command"`
	TSHostSurface      string                        `json:"ts_host_surface"`
	StateMutationOwner string                        `json:"state_mutation_owner"`
	MutatesAetherData  bool                          `json:"mutates_aether_data"`
	TSWritePolicy      string                        `json:"ts_write_policy"`
	FinalizerRequired  bool                          `json:"finalizer_required"`
	FinalizerCommand   string                        `json:"finalizer_command"`
	CeremonySteps      []string                      `json:"ceremony_steps"`
	WrapperCoverage    classicCommandWrapperCoverage `json:"wrapper_coverage"`
	RawBypass          bool                          `json:"raw_bypass"`
	ClassicBehavior    string                        `json:"classic_behavior"`
	RestoreTarget      string                        `json:"restore_target"`
}

type classicCommandWrapperCoverage struct {
	YAML     bool `json:"yaml"`
	Claude   bool `json:"claude"`
	OpenCode bool `json:"opencode"`
}

var requiredClassicParityCommands = []string{
	"init",
	"discuss",
	"colonize",
	"plan",
	"build",
	"continue",
	"seal",
	"oracle",
	"swarm",
	"watch",
	"status",
	"history",
	"phase",
	"resume",
	"focus",
	"redirect",
	"feedback",
	"pheromones",
}

func TestClassicCommandParityMatrixCoversCoreCommands(t *testing.T) {
	matrix := loadClassicCommandParityMatrix(t)
	records := classicCommandParityRecordsByName(t, matrix)

	for _, command := range requiredClassicParityCommands {
		if _, ok := records[command]; !ok {
			t.Errorf("classic command parity matrix missing %q", command)
		}
	}
}

func TestClassicCommandParityMatrixMatchesCommandGuideAndWrappers(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	matrix := loadClassicCommandParityMatrix(t)
	records := classicCommandParityRecordsByName(t, matrix)

	for _, command := range requiredClassicParityCommands {
		record := records[command]
		guide, err := buildCommandGuide(command, "codex")
		if err != nil {
			t.Fatalf("buildCommandGuide(%q): %v", command, err)
		}
		if record.Category != guide.Category {
			t.Errorf("%s category = %q, want command-guide category %q", command, record.Category, guide.Category)
		}
		if record.RawBypass != (guide.RawBypass != "") {
			t.Errorf("%s raw_bypass = %v, want %v from command-guide", command, record.RawBypass, guide.RawBypass != "")
		}
		assertClassicCommandWrapperExists(t, repoRoot, command, "yaml", record.WrapperCoverage.YAML)
		assertClassicCommandWrapperExists(t, repoRoot, command, "claude", record.WrapperCoverage.Claude)
		assertClassicCommandWrapperExists(t, repoRoot, command, "opencode", record.WrapperCoverage.OpenCode)
	}
}

func TestClassicCommandParityMatrixDocumentsStateBoundary(t *testing.T) {
	matrix := loadClassicCommandParityMatrix(t)
	if !strings.Contains(matrix.RuntimeAuthority, "not runtime authority") {
		t.Fatalf("matrix must explicitly say it is not runtime authority, got %q", matrix.RuntimeAuthority)
	}
	if matrix.TSWritePolicy != "The TypeScript host may orchestrate, render, and pass arguments, but must never write .aether/data directly." {
		t.Fatalf("unexpected top-level TS write policy: %q", matrix.TSWritePolicy)
	}

	records := classicCommandParityRecordsByName(t, matrix)
	for _, command := range requiredClassicParityCommands {
		record := records[command]
		if record.RuntimeCommand == "" {
			t.Errorf("%s missing runtime_command", command)
		}
		if record.TSHostSurface == "" {
			t.Errorf("%s missing ts_host_surface", command)
		}
		if record.TSWritePolicy != "never-write-aether-data" {
			t.Errorf("%s ts_write_policy = %q, want never-write-aether-data", command, record.TSWritePolicy)
		}
		if record.MutatesAetherData && !strings.HasPrefix(record.StateMutationOwner, "go-") {
			t.Errorf("%s mutates .aether/data but state_mutation_owner = %q", command, record.StateMutationOwner)
		}
		if record.FinalizerRequired && !strings.Contains(record.FinalizerCommand, "--completion-file") {
			t.Errorf("%s requires finalizer but finalizer_command = %q", command, record.FinalizerCommand)
		}
		if len(record.CeremonySteps) == 0 {
			t.Errorf("%s missing ceremony_steps", command)
		}
		if record.ClassicBehavior == "" || record.RestoreTarget == "" {
			t.Errorf("%s must document classic_behavior and restore_target", command)
		}
	}
}

func TestClassicCommandParityMatrixDocumentsFinalizerContracts(t *testing.T) {
	records := classicCommandParityRecordsByName(t, loadClassicCommandParityMatrix(t))
	expected := map[string]struct {
		required bool
		command  string
	}{
		"colonize": {required: true, command: "aether colonize-finalize --completion-file <file>"},
		"plan":     {required: true, command: "aether plan-finalize --completion-file <file>"},
		"build":    {required: true, command: "aether build-finalize <phase> --completion-file <file>"},
		"continue": {required: false, command: "aether continue-finalize --completion-file <file>"},
		"seal":     {required: true, command: "aether seal-finalize --completion-file <file>"},
		"oracle":   {required: false, command: "aether oracle-iterate-finalize --completion-file <file>"},
		"swarm":    {required: true, command: "aether swarm-finalize --completion-file <file>"},
	}

	for command, want := range expected {
		record := records[command]
		if record.FinalizerRequired != want.required {
			t.Errorf("%s finalizer_required = %v, want %v", command, record.FinalizerRequired, want.required)
		}
		if record.FinalizerCommand != want.command {
			t.Errorf("%s finalizer_command = %q, want %q", command, record.FinalizerCommand, want.command)
		}
		if want.required && record.StateMutationOwner != "go-finalizer" {
			t.Errorf("%s required finalizer should use go-finalizer state owner, got %q", command, record.StateMutationOwner)
		}
	}

	for command, record := range records {
		if _, ok := expected[command]; ok {
			continue
		}
		if record.FinalizerRequired {
			t.Errorf("%s unexpectedly requires a finalizer", command)
		}
		if record.FinalizerCommand != "" {
			t.Errorf("%s unexpectedly documents finalizer command %q", command, record.FinalizerCommand)
		}
	}
}

func loadClassicCommandParityMatrix(t *testing.T) classicCommandParityMatrix {
	t.Helper()
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}
	path := filepath.Join(repoRoot, ".aether", "commands", "classic-command-parity.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var matrix classicCommandParityMatrix
	if err := json.Unmarshal(data, &matrix); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	if matrix.SchemaVersion == "" {
		t.Fatal("classic command parity matrix missing schema_version")
	}
	if len(matrix.Commands) == 0 {
		t.Fatal("classic command parity matrix has no commands")
	}
	return matrix
}

func classicCommandParityRecordsByName(t *testing.T, matrix classicCommandParityMatrix) map[string]classicCommandParityRecord {
	t.Helper()
	records := make(map[string]classicCommandParityRecord, len(matrix.Commands))
	for _, record := range matrix.Commands {
		if record.Name == "" {
			t.Fatal("classic command parity matrix contains command with empty name")
		}
		if _, exists := records[record.Name]; exists {
			t.Fatalf("classic command parity matrix contains duplicate command %q", record.Name)
		}
		records[record.Name] = record
	}

	var names []string
	for name := range records {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		if !slices.Contains(requiredClassicParityCommands, name) {
			t.Fatalf("classic command parity matrix contains unexpected command %q", name)
		}
	}
	return records
}

func assertClassicCommandWrapperExists(t *testing.T, repoRoot, command, surface string, expected bool) {
	t.Helper()
	var path string
	switch surface {
	case "yaml":
		path = filepath.Join(repoRoot, ".aether", "commands", command+".yaml")
	case "claude":
		path = filepath.Join(repoRoot, ".claude", "commands", "ant", command+".md")
	case "opencode":
		path = filepath.Join(repoRoot, ".opencode", "commands", "ant", command+".md")
	default:
		t.Fatalf("unknown command wrapper surface %q", surface)
	}

	_, err := os.Stat(path)
	exists := err == nil
	if expected && !exists {
		t.Errorf("%s wrapper for %s missing at %s", surface, command, path)
	}
	if !expected && exists {
		t.Errorf("%s wrapper for %s exists but matrix says it is uncovered at %s", surface, command, path)
	}
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}

func TestClassicCommandParityMatrixKeepsTSHostAsConductorNotAuthority(t *testing.T) {
	matrix := loadClassicCommandParityMatrix(t)
	records := classicCommandParityRecordsByName(t, matrix)

	hostCommands := []string{"plan", "build", "continue", "oracle", "swarm", "watch"}
	for _, command := range hostCommands {
		record := records[command]
		if record.TSHostSurface == "none" {
			t.Errorf("%s should document its TypeScript host surface", command)
		}
		if record.StateMutationOwner == "typescript-host" {
			t.Errorf("%s must not make TypeScript the state mutation owner", command)
		}
	}

	for _, command := range []string{"status", "focus", "redirect", "feedback", "pheromones"} {
		record := records[command]
		if record.Category != commandGuideCategoryLiteral {
			t.Errorf("%s category = %q, want literal", command, record.Category)
		}
		if !strings.Contains(record.RestoreTarget, "runtime") && !strings.Contains(record.RestoreTarget, "Go") {
			t.Errorf("%s restore_target should preserve runtime ownership, got %q", command, record.RestoreTarget)
		}
	}
}

func TestClassicCommandParityMatrixNamesKnownFutureHostGaps(t *testing.T) {
	records := classicCommandParityRecordsByName(t, loadClassicCommandParityMatrix(t))
	for _, command := range []string{"colonize", "seal"} {
		record := records[command]
		if record.TSHostSurface != "missing-orchestration-target" {
			t.Errorf("%s ts_host_surface = %q, want missing-orchestration-target", command, record.TSHostSurface)
		}
		if !strings.Contains(record.RestoreTarget, fmt.Sprintf("aether host %s", command)) {
			t.Errorf("%s restore_target should name future host command, got %q", command, record.RestoreTarget)
		}
	}
}
