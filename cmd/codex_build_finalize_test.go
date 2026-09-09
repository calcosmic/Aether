package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func TestNormalizeExternalBuildStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"completed", "completed"},
		{"complete", "completed"},
		{"done", "completed"},
		{"success", "completed"},
		{"succeeded", "completed"},
		{"passed", "completed"},
		{"code_written", "completed"},
		{"CODE_WRITTEN", "completed"},
		{"Code_Written", "completed"},
		{"failed", "failed"},
		{"fail", "failed"},
		{"error", "failed"},
		{"timed_out", "timeout"},
		{"cancelled", "timeout"},
		{"manual", "manually-reconciled"},
		{"manually_reconciled", "manually-reconciled"},
		{"blocked", "blocked"},
		{"unknown_status", "unknown_status"},
	}

	for _, tc := range tests {
		got := normalizeExternalBuildStatus(tc.input)
		if got != tc.expected {
			t.Errorf("normalizeExternalBuildStatus(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestIsTerminalExternalBuildStatus(t *testing.T) {
	terminal := []string{"completed", "failed", "blocked", "timeout", "manually-reconciled"}
	for _, s := range terminal {
		if !isTerminalExternalBuildStatus(s) {
			t.Errorf("expected %q to be terminal", s)
		}
	}

	nonTerminal := []string{"pending", "running", ""}
	for _, s := range nonTerminal {
		if isTerminalExternalBuildStatus(s) {
			t.Errorf("expected %q to NOT be terminal", s)
		}
	}
}

func TestEffectiveNameUsesAntNameFallback(t *testing.T) {
	tests := []struct {
		name     string
		antName  string
		expected string
	}{
		{"Mason-67", "", "Mason-67"},
		{"", "Mason-67", "Mason-67"},
		{"Mason-67", "Other-99", "Mason-67"},
		{"  Mason-67  ", "", "Mason-67"},
		{"", "  Mason-67  ", "Mason-67"},
		{"", "", ""},
	}

	for _, tc := range tests {
		r := codexExternalBuildWorkerResult{Name: tc.name, AntName: tc.antName}
		got := r.effectiveName()
		if got != tc.expected {
			t.Errorf("effectiveName(Name=%q, AntName=%q) = %q, want %q", tc.name, tc.antName, got, tc.expected)
		}
	}
}

func TestEffectiveNameWithJSONAntName(t *testing.T) {
	jsonInput := `{"ant_name": "Hammer-23", "status": "code_written", "files_created": ["a.go"]}`

	var r codexExternalBuildWorkerResult
	if err := json.Unmarshal([]byte(jsonInput), &r); err != nil {
		t.Fatalf("parse: %v", err)
	}

	if r.effectiveName() != "Hammer-23" {
		t.Errorf("effectiveName() = %q, want %q", r.effectiveName(), "Hammer-23")
	}
}

func TestHasCompletedBuilders(t *testing.T) {
	tests := []struct {
		name       string
		dispatches []codexBuildDispatch
		expected   bool
	}{
		{
			"completed builder",
			[]codexBuildDispatch{{Caste: "builder", Status: "completed"}},
			true,
		},
		{
			"failed builder",
			[]codexBuildDispatch{{Caste: "builder", Status: "failed"}},
			false,
		},
		{
			"completed watcher",
			[]codexBuildDispatch{{Caste: "watcher", Status: "completed"}},
			false,
		},
		{
			"empty dispatches",
			[]codexBuildDispatch{},
			false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hasCompletedBuilders(tc.dispatches)
			if got != tc.expected {
				t.Errorf("hasCompletedBuilders() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestValidateBuildProvenanceRequiresCompletedBuilderFileEvidence(t *testing.T) {
	if err := validateBuildProvenance([]codexExternalBuildWorkerResult{
		{
			Name:          "Mason-67",
			Caste:         "builder",
			Status:        "code_written",
			FilesModified: []string{"cmd/codex_build_finalize.go"},
		},
	}); err != nil {
		t.Fatalf("expected completed builder with file evidence to pass provenance: %v", err)
	}

	if err := validateBuildProvenance([]codexExternalBuildWorkerResult{
		{
			Name:   "Mason-67",
			Caste:  "builder",
			Status: "completed",
		},
	}); err == nil {
		t.Fatal("expected completed builder without file evidence to fail provenance")
	}

	if err := validateBuildProvenance([]codexExternalBuildWorkerResult{
		{
			Name:          "Keen-13",
			Caste:         "watcher",
			Status:        "completed",
			FilesModified: []string{"cmd/codex_build_finalize_test.go"},
		},
	}); err == nil {
		t.Fatal("expected watcher-only file evidence to fail build provenance")
	}
}

func TestValidateBuildProvenanceForManifestAllowsVerificationOnlyOutputs(t *testing.T) {
	manifest := &codexBuildManifest{
		Tasks: []codexBuildTaskPlan{
			{ID: "3.1", Goal: "Run go test ./... from the repo root."},
			{ID: "3.2", Goal: "Verify TS host dependencies are available and Node satisfies >=20."},
		},
	}
	results := []codexExternalBuildWorkerResult{
		{
			Name:    "Brick-52",
			Caste:   "builder",
			Status:  "completed",
			TaskID:  "3.1",
			Outputs: []string{"go.mod"},
		},
	}

	if err := validateBuildProvenanceForManifest(manifest, results); err != nil {
		t.Fatalf("expected verification-only output evidence to pass provenance: %v", err)
	}
}

func TestValidateBuildProvenanceForManifestRejectsOutputOnlyMutationPhase(t *testing.T) {
	manifest := &codexBuildManifest{
		Tasks: []codexBuildTaskPlan{
			{ID: "2.1", Goal: "Update wrapper ceremony contract tests."},
		},
	}
	results := []codexExternalBuildWorkerResult{
		{
			Name:    "Weld-96",
			Caste:   "builder",
			Status:  "completed",
			TaskID:  "2.1",
			Outputs: []string{"cmd/codex_build_finalize.go"},
		},
	}

	if err := validateBuildProvenanceForManifest(manifest, results); err == nil {
		t.Fatal("expected output-only mutation phase to fail provenance")
	}
}

func TestValidateBuildProvenanceForManifestRejectsWatcherOnlyVerificationEvidence(t *testing.T) {
	manifest := &codexBuildManifest{
		Tasks: []codexBuildTaskPlan{
			{ID: "3.1", Goal: "Run go test ./... from the repo root."},
		},
	}
	results := []codexExternalBuildWorkerResult{
		{
			Name:    "Hawk-45",
			Caste:   "watcher",
			Status:  "completed",
			Outputs: []string{"go.mod"},
		},
	}

	if err := validateBuildProvenanceForManifest(manifest, results); err == nil {
		t.Fatal("expected watcher-only verification evidence to fail provenance")
	}
}

func TestMergeExternalBuildResultsWithCodeWritten(t *testing.T) {
	manifest := codexBuildManifest{
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Mason-67", Caste: "builder", Stage: "wave", TaskID: "1.1"},
		},
	}

	results := []codexExternalBuildWorkerResult{
		{
			Name:          "Mason-67",
			Status:        "code_written",
			Summary:       "Implemented task 1.1",
			FilesCreated:  []string{"src/main.go"},
			FilesModified: []string{"go.mod"},
		},
	}

	dispatches, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("mergeExternalBuildResults with code_written: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %+v", violations)
	}
	if dispatches[0].Status != "completed" {
		t.Errorf("status = %q, want completed", dispatches[0].Status)
	}
	if len(dispatches[0].Outputs) == 0 {
		t.Error("expected outputs to be populated")
	}
}

func TestMergeExternalBuildResultsWithAntName(t *testing.T) {
	manifest := codexBuildManifest{
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Mason-67", Caste: "builder", Stage: "wave", TaskID: "1.1"},
		},
	}

	results := []codexExternalBuildWorkerResult{
		{
			AntName:      "Mason-67",
			Status:       "completed",
			Summary:      "Implemented task 1.1",
			FilesCreated: []string{"src/new.go"},
		},
	}

	dispatches, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("mergeExternalBuildResults with ant_name: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %+v", violations)
	}
	if dispatches[0].Status != "completed" {
		t.Errorf("status = %q, want completed", dispatches[0].Status)
	}
}

func TestMergeExternalBuildResultsMatchesRetrySuffixDrift(t *testing.T) {
	manifest := codexBuildManifest{
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Hunt-33-r2", Caste: "builder", Stage: "wave", TaskID: "1.1"},
		},
	}

	results := []codexExternalBuildWorkerResult{
		{
			Name:         "Hunt-33",
			Caste:        "builder",
			Stage:        "wave",
			TaskID:       "1.1",
			Status:       "completed",
			Summary:      "Implemented task despite retry suffix drift",
			FilesCreated: []string{"cmd/reliability.go"},
		},
	}

	dispatches, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("mergeExternalBuildResults with retry suffix drift: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %+v", violations)
	}
	if dispatches[0].Status != "completed" {
		t.Errorf("status = %q, want completed", dispatches[0].Status)
	}
	if !contains(strings.Join(dispatches[0].Outputs, ","), "cmd/reliability.go") {
		t.Fatalf("expected outputs from suffix-matched result, got %+v", dispatches[0].Outputs)
	}
}

func TestMergeExternalBuildResultsRejectsAmbiguousRetrySuffixMatch(t *testing.T) {
	manifest := codexBuildManifest{
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Hunt-33-r4", Caste: "builder", Stage: "wave", TaskID: "1.1"},
		},
	}

	results := []codexExternalBuildWorkerResult{
		{Name: "Hunt-33-r2", Caste: "builder", Stage: "wave", TaskID: "1.1", Status: "completed", FilesCreated: []string{"a.go"}},
		{Name: "Hunt-33-r3", Caste: "builder", Stage: "wave", TaskID: "1.1", Status: "completed", FilesCreated: []string{"b.go"}},
	}

	_, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("expected no internal error for ambiguous retry suffix match, got: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 violation for ambiguous retry suffix match, got %d: %+v", len(violations), violations)
	}
	if violations[0].Rule != violationRuleDuplicateResult {
		t.Errorf("Rule = %q, want %q", violations[0].Rule, violationRuleDuplicateResult)
	}
	if !contains(violations[0].Message, "ambiguous external worker result") {
		t.Fatalf("expected ambiguous match violation message, got: %v", violations[0].Message)
	}
}

func TestClaimsOrAggregateWithAntName(t *testing.T) {
	root := t.TempDir()
	writeClaimFileForTest(t, root, "src/main.go")
	writeClaimFileForTest(t, root, "go.mod")
	writeClaimFileForTest(t, root, "src/main_test.go")
	completion := codexExternalBuildCompletion{
		DispatchManifest: &codexBuildManifest{PlanOnly: true},
		Dispatches: []codexExternalBuildWorkerResult{
			{
				AntName:       "Mason-67",
				Status:        "completed",
				FilesCreated:  []string{"src/main.go"},
				FilesModified: []string{"go.mod"},
				TestsWritten:  []string{"src/main_test.go"},
			},
		},
	}

	dispatches := []codexBuildDispatch{
		{Name: "Mason-67", Caste: "builder", Status: "completed", TaskID: "1.1"},
	}

	claims, err := completion.claimsOrAggregate(root, 1, time.Now().UTC(), dispatches)
	if err != nil {
		t.Fatalf("claimsOrAggregate: %v", err)
	}

	if len(claims.FilesCreated) == 0 {
		t.Error("expected FilesCreated to be populated from ant_name worker")
	}
	if claims.FilesCreated[0] != "src/main.go" {
		t.Errorf("FilesCreated[0] = %q, want src/main.go", claims.FilesCreated[0])
	}
	if len(claims.FilesModified) == 0 {
		t.Error("expected FilesModified to be populated")
	}
}

func TestClaimsOrAggregateMatchesRetrySuffixDrift(t *testing.T) {
	root := t.TempDir()
	writeClaimFileForTest(t, root, "cmd/reliability.go")
	completion := codexExternalBuildCompletion{
		Dispatches: []codexExternalBuildWorkerResult{{
			Name:          "Hunt-33",
			Caste:         "builder",
			Stage:         "wave",
			TaskID:        "1.1",
			Status:        "completed",
			FilesModified: []string{"cmd/reliability.go"},
		}},
	}
	dispatches := []codexBuildDispatch{{
		Name:   "Hunt-33-r4",
		Caste:  "builder",
		Stage:  "wave",
		Status: "completed",
		TaskID: "1.1",
	}}

	claims, err := completion.claimsOrAggregate(root, 1, time.Now().UTC(), dispatches)
	if err != nil {
		t.Fatalf("claimsOrAggregate with retry suffix drift: %v", err)
	}
	if len(claims.FilesModified) != 1 || claims.FilesModified[0] != "cmd/reliability.go" {
		t.Fatalf("FilesModified = %+v, want retry-suffix matched worker file", claims.FilesModified)
	}
	if len(claims.TaskClaims) != 1 || claims.TaskClaims[0].TaskID != "1.1" {
		t.Fatalf("TaskClaims = %+v, want task 1.1 claim", claims.TaskClaims)
	}
}

func TestClaimsOrAggregateRejectsUnsafeClaimPaths(t *testing.T) {
	root := t.TempDir()
	completion := codexExternalBuildCompletion{
		Dispatches: []codexExternalBuildWorkerResult{{
			Name:          "Mason-67",
			Status:        "completed",
			FilesModified: []string{"../outside.go"},
		}},
	}
	dispatches := []codexBuildDispatch{
		{Name: "Mason-67", Caste: "builder", Status: "completed", TaskID: "1.1"},
	}

	_, err := completion.claimsOrAggregate(root, 1, time.Now().UTC(), dispatches)
	if err == nil {
		t.Fatal("expected unsafe claim path to be rejected")
	}
	if !strings.Contains(err.Error(), "escapes repository") {
		t.Fatalf("expected repository escape error, got: %v", err)
	}
}

func TestClaimsOrAggregateRejectsAbsoluteClaimPaths(t *testing.T) {
	root := t.TempDir()
	writeClaimFileForTest(t, root, "src/generated.go")
	completion := codexExternalBuildCompletion{
		Dispatches: []codexExternalBuildWorkerResult{{
			Name:         "Mason-67",
			Status:       "completed",
			FilesCreated: []string{filepath.Join(root, "src", "generated.go")},
		}},
	}
	dispatches := []codexBuildDispatch{
		{Name: "Mason-67", Caste: "builder", Status: "completed", TaskID: "1.1"},
	}

	_, err := completion.claimsOrAggregate(root, 1, time.Now().UTC(), dispatches)
	if err == nil {
		t.Fatal("expected absolute claim path to be rejected")
	}
	if !strings.Contains(err.Error(), "repo-relative") {
		t.Fatalf("expected repo-relative error, got: %v", err)
	}
}

func TestClaimsOrAggregateRejectsAetherDataClaimPaths(t *testing.T) {
	root := t.TempDir()
	writeClaimFileForTest(t, root, ".aether/data/build/phase-1/outcome.md")
	completion := codexExternalBuildCompletion{
		Dispatches: []codexExternalBuildWorkerResult{{
			Name:         "Mason-67",
			Status:       "completed",
			FilesCreated: []string{".aether/data/build/phase-1/outcome.md"},
		}},
	}
	dispatches := []codexBuildDispatch{
		{Name: "Mason-67", Caste: "builder", Status: "completed", TaskID: "1.1"},
	}

	_, err := completion.claimsOrAggregate(root, 1, time.Now().UTC(), dispatches)
	if err == nil {
		t.Fatal("expected .aether/data claim path to be rejected")
	}
	if !strings.Contains(err.Error(), ".aether/data") {
		t.Fatalf("expected .aether/data error, got: %v", err)
	}
}

func TestClaimsOrAggregateToleratesSanctionedDataClaimPaths(t *testing.T) {
	root := t.TempDir()
	writeClaimFileForTest(t, root, "src/real.go")
	writeClaimFileForTest(t, root, ".aether/data/reviews/history/ledger.json")
	writeClaimFileForTest(t, root, ".aether/data/phase-research/phase-1-research.md")
	completion := codexExternalBuildCompletion{
		Dispatches: []codexExternalBuildWorkerResult{{
			Name:   "Digger-12",
			Status: "completed",
			FilesModified: []string{
				"src/real.go",
				".aether/data/reviews/history/ledger.json",
				".aether/data/phase-research/phase-1-research.md",
			},
		}},
	}
	dispatches := []codexBuildDispatch{
		{Name: "Digger-12", Caste: "archaeologist", Status: "completed", TaskID: ""},
	}

	claims, err := completion.claimsOrAggregate(root, 1, time.Now().UTC(), dispatches)
	if err != nil {
		t.Fatalf("sanctioned .aether/data claim must not fail the packet: %v", err)
	}
	modified := claims.FilesModified
	if len(modified) != 1 || modified[0] != "src/real.go" {
		t.Fatalf("sanctioned claims must be dropped from the claim set, got: %v", modified)
	}
}

func TestClaimsOrAggregateRejectsMissingClaimPaths(t *testing.T) {
	root := t.TempDir()
	completion := codexExternalBuildCompletion{
		Dispatches: []codexExternalBuildWorkerResult{{
			Name:          "Mason-67",
			Status:        "completed",
			FilesModified: []string{"src/missing.go"},
		}},
	}
	dispatches := []codexBuildDispatch{
		{Name: "Mason-67", Caste: "builder", Status: "completed", TaskID: "1.1"},
	}

	_, err := completion.claimsOrAggregate(root, 1, time.Now().UTC(), dispatches)
	if err == nil {
		t.Fatal("expected missing claim path to be rejected")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("expected missing path error, got: %v", err)
	}
}

func TestClaimsOrAggregateRejectsAmbiguousClaimPaths(t *testing.T) {
	root := t.TempDir()
	writeClaimFileForTest(t, root, "src/target.go")
	writeClaimFileForTest(t, root, "pkg/target.go")
	gitInitForTest(t, root)

	completion := codexExternalBuildCompletion{
		Dispatches: []codexExternalBuildWorkerResult{{
			Name:          "Mason-67",
			Status:        "completed",
			FilesModified: []string{"target.go"},
		}},
	}
	dispatches := []codexBuildDispatch{
		{Name: "Mason-67", Caste: "builder", Status: "completed", TaskID: "1.1"},
	}

	_, err := completion.claimsOrAggregate(root, 1, time.Now().UTC(), dispatches)
	if err == nil {
		t.Fatal("expected ambiguous claim path to be rejected")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguous path error, got: %v", err)
	}
}

func TestClaimsOrAggregateRejectsSymlinkClaimPaths(t *testing.T) {
	root := t.TempDir()
	writeClaimFileForTest(t, root, "real/target.go")
	linkPath := filepath.Join(root, "src", "linked.go")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatalf("mkdir symlink parent: %v", err)
	}
	if err := os.Symlink(filepath.Join(root, "real", "target.go"), linkPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	completion := codexExternalBuildCompletion{
		Dispatches: []codexExternalBuildWorkerResult{{
			Name:          "Mason-67",
			Status:        "completed",
			FilesModified: []string{"src/linked.go"},
		}},
	}
	dispatches := []codexBuildDispatch{
		{Name: "Mason-67", Caste: "builder", Status: "completed", TaskID: "1.1"},
	}

	_, err := completion.claimsOrAggregate(root, 1, time.Now().UTC(), dispatches)
	if err == nil {
		t.Fatal("expected symlink claim path to be rejected")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink path error, got: %v", err)
	}
}

func TestClaimsOrAggregateNormalizesValidRepoRelativeClaimPaths(t *testing.T) {
	root := t.TempDir()
	writeClaimFileForTest(t, root, "src/generated.go")
	writeClaimFileForTest(t, root, "src/main.go")
	writeClaimFileForTest(t, root, "src/main_test.go")
	completion := codexExternalBuildCompletion{
		Claims: &codexBuildClaims{
			FilesCreated:  []string{"src/generated.go"},
			FilesModified: []string{"src/main.go"},
			TestsWritten:  []string{"src/main_test.go"},
			TaskClaims: []codexBuildTaskClaim{{
				TaskID:        "1.1",
				FilesCreated:  []string{"src/generated.go"},
				FilesModified: []string{"src/main.go"},
				TestsWritten:  []string{"src/main_test.go"},
			}},
		},
	}

	claims, err := completion.claimsOrAggregate(root, 1, time.Now().UTC(), nil)
	if err != nil {
		t.Fatalf("claimsOrAggregate: %v", err)
	}
	if got, want := strings.Join(claims.FilesCreated, ","), "src/generated.go"; got != want {
		t.Fatalf("FilesCreated = %q, want %q", got, want)
	}
	if got, want := strings.Join(claims.FilesModified, ","), "src/main.go"; got != want {
		t.Fatalf("FilesModified = %q, want %q", got, want)
	}
	if got, want := strings.Join(claims.TestsWritten, ","), "src/main_test.go"; got != want {
		t.Fatalf("TestsWritten = %q, want %q", got, want)
	}
	if len(claims.TaskClaims) != 1 || strings.Join(claims.TaskClaims[0].FilesCreated, ",") != "src/generated.go" {
		t.Fatalf("TaskClaims not normalized: %#v", claims.TaskClaims)
	}
}

func TestManifestUsesExternalTask(t *testing.T) {
	tests := []struct {
		mode     string
		expected bool
	}{
		{"external-task", true},
		{"External-Task", true},
		{"EXTERNAL-TASK", true},
		{"", false},
		{"in-repo", false},
		{"simulated", false},
	}

	for _, tc := range tests {
		manifest := codexContinueManifest{
			Present: true,
			Data: codexBuildManifest{
				DispatchMode: tc.mode,
			},
		}
		got := manifestUsesExternalTask(manifest)
		if got != tc.expected {
			t.Errorf("manifestUsesExternalTask(%q) = %v, want %v", tc.mode, got, tc.expected)
		}
	}

	if manifestUsesExternalTask(codexContinueManifest{Present: false}) {
		t.Error("expected false for missing manifest")
	}
}

func TestParseGitNameOutput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty", "", nil},
		{"single", "file.go\n", []string{"file.go"}},
		{"multiple", "a.go\nb.go\nc.go\n", []string{"a.go", "b.go", "c.go"}},
		{"trailing newline", "file.go\n\n", []string{"file.go"}},
		{"whitespace", "  a.go  \n  b.go  \n", []string{"a.go", "b.go"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseGitNameOutput([]byte(tc.input))
			if len(got) != len(tc.expected) {
				t.Fatalf("got %d items, want %d: %v", len(got), len(tc.expected), got)
			}
			for i, v := range got {
				if v != tc.expected[i] {
					t.Errorf("item %d: got %q, want %q", i, v, tc.expected[i])
				}
			}
		})
	}
}

func TestNormalizeClaimPathsToRoot_SubdirectoryRelative(t *testing.T) {
	tmp := t.TempDir()
	nestedDir := filepath.Join(tmp, "app", "public", "wp-content", "themes", "mytheme", "resources", "js")
	if err := os.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatal(err)
	}
	existingFile := filepath.Join(nestedDir, "animations.js")
	if err := os.WriteFile(existingFile, []byte("// test"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Claim uses subdirectory-relative path.
	// Since this temp dir is not a git repo, findRepoRelativePath returns empty
	// (no filesystem walk fallback), so the original claimed path is kept as-is.
	claimed := "resources/js/animations.js"
	result := normalizeClaimPathsToRoot(tmp, []string{claimed})
	if len(result) != 1 {
		t.Fatalf("got %d results, want 1", len(result))
	}
	if result[0] != claimed {
		t.Errorf("got %q, want %q (original kept when not resolvable)", result[0], claimed)
	}
}

func TestNormalizeClaimPathsToRoot_AlreadyValid(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "src", "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Path already resolves from root
	result := normalizeClaimPathsToRoot(tmp, []string{"src/main.go"})
	if result[0] != "src/main.go" {
		t.Errorf("got %q, want %q", result[0], "src/main.go")
	}
}

func TestNormalizeClaimPathsToRoot_EmptyRoot(t *testing.T) {
	paths := []string{"foo/bar.go"}
	result := normalizeClaimPathsToRoot("", paths)
	if len(result) != 1 || result[0] != "foo/bar.go" {
		t.Errorf("empty root should return paths unchanged, got %v", result)
	}
}

func TestBestMatchForClaimedPath(t *testing.T) {
	tests := []struct {
		name       string
		claimed    string
		candidates []string
		want       string
	}{
		{
			name:    "single candidate",
			claimed: "resources/js/Foo.js",
			candidates: []string{
				"app/public/wp-content/themes/theme/resources/js/Foo.js",
			},
			want: "app/public/wp-content/themes/theme/resources/js/Foo.js",
		},
		{
			name:    "multiple candidates — best trailing match",
			claimed: "resources/js/animations.js",
			candidates: []string{
				"src/animations.js",
				"app/public/wp-content/themes/mytheme/resources/js/animations.js",
			},
			want: "app/public/wp-content/themes/mytheme/resources/js/animations.js",
		},
		{
			name:    "tiebreak by shortest path",
			claimed: "utils/helper.go",
			candidates: []string{
				"a/b/c/utils/helper.go",
				"pkg/utils/helper.go",
			},
			want: "pkg/utils/helper.go",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := bestMatchForClaimedPath(tc.claimed, tc.candidates)
			if got != tc.want {
				t.Errorf("bestMatchForClaimedPath(%q, %v) = %q, want %q", tc.claimed, tc.candidates, got, tc.want)
			}
		})
	}
}

func TestFindRepoRelativePath(t *testing.T) {
	tmp := t.TempDir()
	// Initialize git repo so git ls-files works
	if err := os.MkdirAll(filepath.Join(tmp, "deep", "nested", "dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(tmp, "deep", "nested", "dir", "target.go")
	if err := os.WriteFile(testFile, []byte("package dir"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run tests in a subprocess that initializes git
	// Since findRepoRelativePath uses git, we need a git repo
	t.Run("with git repo", func(t *testing.T) {
		// git init + add so ls-files tracks it
		gitInitForTest(t, tmp)
		gitAddForTest(t, tmp)

		claimed := "nested/dir/target.go"
		got := findRepoRelativePath(tmp, claimed)
		if got != "deep/nested/dir/target.go" {
			t.Errorf("findRepoRelativePath(%q, %q) = %q, want %q", tmp, claimed, got, "deep/nested/dir/target.go")
		}
	})
}

func TestFindRepoRelativePathIncludesUntrackedFiles(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "src", "components"), 0o755); err != nil {
		t.Fatal(err)
	}
	testFile := filepath.Join(tmp, "src", "components", "AddCardButton.test.tsx")
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	gitInitForTest(t, tmp)

	got := findRepoRelativePath(tmp, "AddCardButton.test.tsx")
	if got != "src/components/AddCardButton.test.tsx" {
		t.Errorf("findRepoRelativePath() = %q, want %q", got, "src/components/AddCardButton.test.tsx")
	}
}

// --- Finalizer validation rejection tests (Task 4.1) ---
// These tests document expected rejection behavior for malformed, mismatched,
// or invalid external worker results. They are intentionally written against
// current behavior and will be validated independently.

func TestMergeExternalBuildResults_RejectsMalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	malformedPath := filepath.Join(tmpDir, "completion.json")
	if err := os.WriteFile(malformedPath, []byte("{not valid json}"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := loadExternalBuildCompletion(malformedPath)
	if err == nil {
		t.Fatal("expected error for malformed JSON completion file")
	}
	// The error should mention parsing
	if !strings.Contains(err.Error(), "parse") {
		t.Fatalf("expected parse error, got: %v", err)
	}
}

func TestMergeExternalBuildResults_RejectsMissingManifest(t *testing.T) {
	tmpDir := t.TempDir()
	noManifestPath := filepath.Join(tmpDir, "completion.json")
	validJSON := `{"dispatches": [{"name": "Mason-67", "status": "completed"}]}`
	if err := os.WriteFile(noManifestPath, []byte(validJSON), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	_, err := loadExternalBuildCompletion(noManifestPath)
	if err == nil {
		t.Fatal("expected error for completion file missing dispatch_manifest")
	}
	if !strings.Contains(err.Error(), "dispatch_manifest") {
		t.Fatalf("expected dispatch_manifest error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Task 1 (163.1-08): loadExternalBuildCompletion keeps the wrapper's
// submitted bytes and no longer aborts decode on the first type mismatch.
// ---------------------------------------------------------------------------

// validCompletionManifestJSON is a minimal dispatch_manifest object literal
// satisfying every field completion-packet.schema.json requires, for tests
// that need a structurally valid manifest without going through
// runCodexBuildPlanOnly.
const validCompletionManifestJSON = `{"phase": 1, "phase_name": "x", "root": "/tmp", "colony_depth": "standard", "generated_at": "2026-01-01T00:00:00Z", "state": "ready", "checkpoint": "c", "claims_path": "p", "worker_briefs": [], "dispatches": [], "tasks": [], "success_criteria": []}`

func TestLoadExternalBuildCompletion_TypeErrorTolerated(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "completion.json")
	packet := `{"dispatch_manifest": ` + validCompletionManifestJSON + `, "dispatches": [{"name": "Hunt-1", "status": 5}]}`
	if err := os.WriteFile(path, []byte(packet), 0o644); err != nil {
		t.Fatalf("write completion: %v", err)
	}
	completion, err := loadExternalBuildCompletion(path)
	if err != nil {
		t.Fatalf("expected a single wrong-typed field to be tolerated, got error: %v", err)
	}
	if len(completion.Dispatches) != 1 || completion.Dispatches[0].Name != "Hunt-1" {
		t.Fatalf("expected partial decode to preserve Name, got %+v", completion.Dispatches)
	}
}

func TestLoadExternalBuildCompletion_CapturesSubmittedRawDirectForm(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "completion.json")
	packet := `{"dispatch_manifest": ` + validCompletionManifestJSON + `}`
	if err := os.WriteFile(path, []byte(packet), 0o644); err != nil {
		t.Fatalf("write completion: %v", err)
	}
	completion, err := loadExternalBuildCompletion(path)
	if err != nil {
		t.Fatalf("load completion: %v", err)
	}
	rawMap, ok := completion.submittedRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected submittedRaw to be a map[string]any, got %T", completion.submittedRaw)
	}
	if _, ok := rawMap["dispatch_manifest"]; !ok {
		t.Fatalf("expected submittedRaw to contain dispatch_manifest, got %+v", rawMap)
	}
}

func TestLoadExternalBuildCompletion_CapturesSubmittedRawEnvelopeForm(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "completion.json")
	packet := `{"result": {"dispatch_manifest": ` + validCompletionManifestJSON + `}}`
	if err := os.WriteFile(path, []byte(packet), 0o644); err != nil {
		t.Fatalf("write completion: %v", err)
	}
	completion, err := loadExternalBuildCompletion(path)
	if err != nil {
		t.Fatalf("load completion: %v", err)
	}
	rawMap, ok := completion.submittedRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected submittedRaw to be a map[string]any, got %T", completion.submittedRaw)
	}
	if _, ok := rawMap["dispatch_manifest"]; !ok {
		t.Fatalf("expected unwrapped submittedRaw to contain dispatch_manifest, got %+v", rawMap)
	}
	if _, ok := rawMap["result"]; ok {
		t.Fatalf("expected unwrapped submittedRaw to NOT contain the envelope's result key, got %+v", rawMap)
	}
}

func TestLoadExternalBuildCompletion_NullTopLevelManifestKeysAcceptEnvelope(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "completion.json")
	// JS/TS serializers commonly emit absent fields as explicit null. The
	// struct decode treats a null pointer field as absent, so the raw
	// envelope unwrap must too (WR-163.1-01): this packet was accepted via
	// the envelope path before the submitted-bytes rework and must stay
	// accepted, with submittedRaw unwrapped to the result value.
	packet := `{"dispatch_manifest": null, "manifest": null, "result": {"dispatch_manifest": ` + validCompletionManifestJSON + `}}`
	if err := os.WriteFile(path, []byte(packet), 0o644); err != nil {
		t.Fatalf("write completion: %v", err)
	}
	completion, err := loadExternalBuildCompletion(path)
	if err != nil {
		t.Fatalf("expected null top-level manifest keys to be treated as absent, got error: %v", err)
	}
	if completion.activeManifest() == nil {
		t.Fatal("expected the envelope's dispatch_manifest to be active")
	}
	rawMap, ok := completion.submittedRaw.(map[string]any)
	if !ok {
		t.Fatalf("expected submittedRaw to be a map[string]any, got %T", completion.submittedRaw)
	}
	if _, ok := rawMap["result"]; ok {
		t.Fatalf("expected unwrapped submittedRaw to NOT contain the envelope's result key, got %+v", rawMap)
	}
	if _, ok := rawMap["dispatch_manifest"].(map[string]any); !ok {
		t.Fatalf("expected unwrapped submittedRaw to carry the result's dispatch_manifest object, got %+v", rawMap)
	}
}

func TestLoadExternalBuildCompletion_StringDispatchManifestReturnsStructuralViolation(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "completion.json")
	packet := `{"dispatch_manifest": "not-an-object"}`
	if err := os.WriteFile(path, []byte(packet), 0o644); err != nil {
		t.Fatalf("write completion: %v", err)
	}
	_, err := loadExternalBuildCompletion(path)
	if err == nil {
		t.Fatal("expected a string-valued dispatch_manifest to be rejected")
	}
	var contractErr *completionContractError
	if !errorsAsCompletionContractError(err, &contractErr) {
		t.Fatalf("expected *completionContractError, got %v (%T)", err, err)
	}
	found := false
	for _, v := range contractErr.Violations {
		if v.Field == "/dispatch_manifest" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a violation with Field /dispatch_manifest, got %+v", contractErr.Violations)
	}
}

func TestLoadExternalBuildCompletion_WrongTypedDispatchManifestNotShadowedByValidManifest(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "completion.json")
	// WR-163.1-02: the tolerated type error on dispatch_manifest allocates a
	// zero-valued struct that activeManifest() prefers over the valid
	// manifest key. An either-key raw cross-check accepted this packet with
	// a corrupt active manifest and the structural violation was never
	// reported. The key-specific cross-check must reject it with a
	// violation naming /dispatch_manifest.
	packet := `{"dispatch_manifest": "bad", "manifest": ` + validCompletionManifestJSON + `}`
	if err := os.WriteFile(path, []byte(packet), 0o644); err != nil {
		t.Fatalf("write completion: %v", err)
	}
	_, err := loadExternalBuildCompletion(path)
	if err == nil {
		t.Fatal("expected a wrong-typed dispatch_manifest to be rejected even alongside a valid manifest")
	}
	var contractErr *completionContractError
	if !errorsAsCompletionContractError(err, &contractErr) {
		t.Fatalf("expected *completionContractError, got %v (%T)", err, err)
	}
	found := false
	for _, v := range contractErr.Violations {
		if v.Field == "/dispatch_manifest" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a violation with Field /dispatch_manifest, got %+v", contractErr.Violations)
	}
}

func TestLoadExternalBuildCompletion_EnvelopeWrongTypedDispatchManifestNotShadowed(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "completion.json")
	// Same shadowing bug on the envelope path: the wrong-typed
	// result.dispatch_manifest must not hide behind the valid
	// result.manifest object.
	packet := `{"result": {"dispatch_manifest": "bad", "manifest": ` + validCompletionManifestJSON + `}}`
	if err := os.WriteFile(path, []byte(packet), 0o644); err != nil {
		t.Fatalf("write completion: %v", err)
	}
	_, err := loadExternalBuildCompletion(path)
	if err == nil {
		t.Fatal("expected a wrong-typed dispatch_manifest inside the envelope to be rejected")
	}
	var contractErr *completionContractError
	if !errorsAsCompletionContractError(err, &contractErr) {
		t.Fatalf("expected *completionContractError, got %v (%T)", err, err)
	}
	found := false
	for _, v := range contractErr.Violations {
		if v.Field == "/dispatch_manifest" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a violation with Field /dispatch_manifest, got %+v", contractErr.Violations)
	}
}

func TestLoadExternalBuildCompletion_SubmittedRawDoesNotAffectDigest(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "completion.json")
	packet := `{"dispatch_manifest": ` + validCompletionManifestJSON + `}`
	if err := os.WriteFile(path, []byte(packet), 0o644); err != nil {
		t.Fatalf("write completion: %v", err)
	}
	completion, err := loadExternalBuildCompletion(path)
	if err != nil {
		t.Fatalf("load completion: %v", err)
	}
	if completion.submittedRaw == nil {
		t.Fatal("expected submittedRaw to be captured for a file-loaded completion")
	}
	withRaw, err := jsonSHA256(completion)
	if err != nil {
		t.Fatalf("hash with submittedRaw: %v", err)
	}
	cleared := completion
	cleared.submittedRaw = nil
	withoutRaw, err := jsonSHA256(cleared)
	if err != nil {
		t.Fatalf("hash without submittedRaw: %v", err)
	}
	if withRaw != withoutRaw {
		t.Fatalf("capturing submitted bytes changed the packet digest: %q vs %q", withRaw, withoutRaw)
	}
}

func TestMergeExternalBuildResults_RejectsWrongRoot(t *testing.T) {
	manifest := codexBuildManifest{
		PlanOnly: true,
		Root:     "/some/other/path",
		Dispatches: []codexBuildDispatch{
			{Name: "Mason-67", Caste: "builder", Stage: "wave", TaskID: "1.1"},
		},
	}
	results := []codexExternalBuildWorkerResult{
		{Name: "Mason-67", Status: "completed"},
	}
	_, violations, err := mergeExternalBuildResults(manifest, results)
	// mergeExternalBuildResults itself does not validate root — that happens
	// in runCodexBuildFinalize via validateFinalizerManifestRoot.
	// Verify that the merge itself succeeds (root is checked upstream).
	if err != nil {
		t.Fatalf("mergeExternalBuildResults should not check root (upstream concern): %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations (root is checked upstream), got %+v", violations)
	}
}

func TestMergeExternalBuildResults_RejectsWrongCaste(t *testing.T) {
	manifest := codexBuildManifest{
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Mason-67", Caste: "builder", Stage: "wave", TaskID: "1.1"},
		},
	}
	results := []codexExternalBuildWorkerResult{
		{Name: "Mason-67", Caste: "watcher", Status: "completed"},
	}
	_, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("expected no internal error for wrong caste, got: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 violation for wrong caste, got %d: %+v", len(violations), violations)
	}
	if violations[0].Rule != violationRuleIdentityMismatch {
		t.Errorf("Rule = %q, want %q", violations[0].Rule, violationRuleIdentityMismatch)
	}
	if !strings.Contains(violations[0].Message, "caste") {
		t.Fatalf("expected caste mismatch message, got: %v", violations[0].Message)
	}
}

func TestMergeExternalBuildResults_RejectsWrongStage(t *testing.T) {
	manifest := codexBuildManifest{
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Mason-67", Caste: "builder", Stage: "wave", TaskID: "1.1"},
		},
	}
	results := []codexExternalBuildWorkerResult{
		{Name: "Mason-67", Stage: "verification", Status: "completed"},
	}
	_, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("expected no internal error for wrong stage, got: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 violation for wrong stage, got %d: %+v", len(violations), violations)
	}
	if violations[0].Rule != violationRuleIdentityMismatch {
		t.Errorf("Rule = %q, want %q", violations[0].Rule, violationRuleIdentityMismatch)
	}
	if !strings.Contains(violations[0].Message, "stage") {
		t.Fatalf("expected stage mismatch message, got: %v", violations[0].Message)
	}
}

func TestMergeExternalBuildResults_RejectsWrongTaskID(t *testing.T) {
	manifest := codexBuildManifest{
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Mason-67", Caste: "builder", Stage: "wave", TaskID: "1.1"},
		},
	}
	results := []codexExternalBuildWorkerResult{
		{Name: "Mason-67", TaskID: "2.1", Status: "completed"},
	}
	_, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("expected no internal error for wrong task_id, got: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 violation for wrong task_id, got %d: %+v", len(violations), violations)
	}
	if violations[0].Rule != violationRuleIdentityMismatch {
		t.Errorf("Rule = %q, want %q", violations[0].Rule, violationRuleIdentityMismatch)
	}
	if !strings.Contains(violations[0].Message, "task_id") {
		t.Fatalf("expected task_id mismatch message, got: %v", violations[0].Message)
	}
}

func TestMergeExternalBuildResults_RejectsWrongWave(t *testing.T) {
	manifest := codexBuildManifest{
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Mason-67", Caste: "builder", Stage: "wave", Wave: 1, TaskID: "1.1"},
		},
	}
	results := []codexExternalBuildWorkerResult{
		{Name: "Mason-67", Wave: 2, Status: "completed"},
	}
	_, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("expected no internal error for wrong wave, got: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 violation for wrong wave, got %d: %+v", len(violations), violations)
	}
	if violations[0].Rule != violationRuleIdentityMismatch {
		t.Errorf("Rule = %q, want %q", violations[0].Rule, violationRuleIdentityMismatch)
	}
	if !strings.Contains(violations[0].Message, "wave") {
		t.Fatalf("expected wave mismatch message, got: %v", violations[0].Message)
	}
}

func TestMergeExternalBuildResults_RejectsWrongExecutionWave(t *testing.T) {
	manifest := codexBuildManifest{
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Mason-67", Caste: "builder", Stage: "wave", Wave: 1, ExecutionWave: 1, TaskID: "1.1"},
		},
	}
	results := []codexExternalBuildWorkerResult{
		{Name: "Mason-67", ExecutionWave: 3, Status: "completed"},
	}
	_, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("expected no internal error for wrong execution_wave, got: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 violation for wrong execution_wave, got %d: %+v", len(violations), violations)
	}
	if violations[0].Rule != violationRuleIdentityMismatch {
		t.Errorf("Rule = %q, want %q", violations[0].Rule, violationRuleIdentityMismatch)
	}
	if !strings.Contains(violations[0].Message, "execution_wave") {
		t.Fatalf("expected execution_wave mismatch message, got: %v", violations[0].Message)
	}
}

func TestMergeExternalBuildResults_RejectsDuplicateWorkerResult(t *testing.T) {
	manifest := codexBuildManifest{
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Mason-67", Caste: "builder", Stage: "wave", TaskID: "1.1"},
		},
	}
	results := []codexExternalBuildWorkerResult{
		{Name: "Mason-67", Status: "completed"},
		{Name: "Mason-67", Status: "completed"},
	}
	_, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("expected no internal error for duplicate worker result, got: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 violation for duplicate worker result, got %d: %+v", len(violations), violations)
	}
	if violations[0].Rule != violationRuleDuplicateResult {
		t.Errorf("Rule = %q, want %q", violations[0].Rule, violationRuleDuplicateResult)
	}
	if !strings.Contains(violations[0].Message, "duplicate") {
		t.Fatalf("expected duplicate message, got: %v", violations[0].Message)
	}
}

func TestMergeExternalBuildResults_RejectsNonTerminalStatus(t *testing.T) {
	manifest := codexBuildManifest{
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Mason-67", Caste: "builder", Stage: "wave", TaskID: "1.1"},
		},
	}
	results := []codexExternalBuildWorkerResult{
		{Name: "Mason-67", Status: "running"},
	}
	_, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("expected no internal error for non-terminal status, got: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 violation for non-terminal status, got %d: %+v", len(violations), violations)
	}
	if violations[0].Rule != violationRuleStatusTerminal {
		t.Errorf("Rule = %q, want %q", violations[0].Rule, violationRuleStatusTerminal)
	}
	if !strings.Contains(violations[0].Message, "non-terminal") {
		t.Fatalf("expected non-terminal status message, got: %v", violations[0].Message)
	}
}

func TestMergeExternalBuildResults_RejectsMissingResult(t *testing.T) {
	manifest := codexBuildManifest{
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Mason-67", Caste: "builder", Stage: "wave", TaskID: "1.1"},
			{Name: "Keen-33", Caste: "watcher", Stage: "verification", TaskID: ""},
		},
	}
	results := []codexExternalBuildWorkerResult{
		{Name: "Mason-67", Status: "completed"},
		// Keen-33 result is missing — build should reject
	}
	_, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("expected no internal error for missing worker result, got: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 violation for missing worker result, got %d: %+v", len(violations), violations)
	}
	if violations[0].Rule != violationRuleResultMissing {
		t.Errorf("Rule = %q, want %q", violations[0].Rule, violationRuleResultMissing)
	}
	if !strings.Contains(violations[0].Message, "missing") {
		t.Fatalf("expected missing result message, got: %v", violations[0].Message)
	}
}

func TestMergeExternalBuildResults_RejectsNamelessResult(t *testing.T) {
	manifest := codexBuildManifest{
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Mason-67", Caste: "builder", Stage: "wave", TaskID: "1.1"},
		},
	}
	results := []codexExternalBuildWorkerResult{
		{Name: "", Status: "completed"},
	}
	_, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("expected no internal error for nameless result, got: %v", err)
	}
	// The nameless result itself is one violation (worker.name_required); the
	// dispatch that never received a matching result is a second, independent
	// violation (worker.result_missing) -- proving accumulation keeps
	// checking later dispatches instead of stopping at the first problem.
	if len(violations) != 2 {
		t.Fatalf("expected exactly 2 violations for nameless result, got %d: %+v", len(violations), violations)
	}
	var sawNameRequired, sawResultMissing bool
	for _, v := range violations {
		switch v.Rule {
		case violationRuleNameRequired:
			sawNameRequired = true
			if !strings.Contains(v.Message, "missing name") {
				t.Errorf("name_required violation message = %q, want to contain %q", v.Message, "missing name")
			}
		case violationRuleResultMissing:
			sawResultMissing = true
		}
	}
	if !sawNameRequired {
		t.Errorf("expected a %s violation, got %+v", violationRuleNameRequired, violations)
	}
	if !sawResultMissing {
		t.Errorf("expected a %s violation, got %+v", violationRuleResultMissing, violations)
	}
}

// --- Claim-path violation accumulation (163.1-02 Task 1) ---

func TestValidateExternalWorkerResultClaimPathsAccumulatesAcrossWorkersAndFields(t *testing.T) {
	root := t.TempDir()
	writeClaimFileForTest(t, root, "src/ok.go")
	results := []codexExternalBuildWorkerResult{
		{
			Name:         "Mason-67",
			FilesCreated: []string{"/etc/passwd"},
		},
		{
			Name:         "Weaver-12",
			TestsWritten: []string{"../outside_test.go"},
		},
	}

	violations := validateExternalWorkerResultClaimPaths(root, results)
	if len(violations) != 2 {
		t.Fatalf("expected exactly 2 violations, got %d: %+v", len(violations), violations)
	}

	byWorker := map[string]contractViolation{}
	for _, v := range violations {
		byWorker[v.Worker] = v
	}
	masonV, ok := byWorker["Mason-67"]
	if !ok {
		t.Fatalf("expected a violation attributed to Mason-67, got %+v", violations)
	}
	if masonV.Field != "files_created" {
		t.Errorf("Mason-67 violation field = %q, want files_created", masonV.Field)
	}
	weaverV, ok := byWorker["Weaver-12"]
	if !ok {
		t.Fatalf("expected a violation attributed to Weaver-12, got %+v", violations)
	}
	if weaverV.Field != "tests_written" {
		t.Errorf("Weaver-12 violation field = %q, want tests_written", weaverV.Field)
	}
	if masonV.Worker == weaverV.Worker {
		t.Fatalf("expected distinct Worker values, got %q for both", masonV.Worker)
	}
}

func TestCollectClaimPathViolationsToleratesSanctionedPrefixesRejectsOthers(t *testing.T) {
	sanctioned := []string{
		".aether/data/planning/notes.md",
		".aether/data/phase-research/phase-1.md",
		".aether/data/survey/survey.json",
		".aether/data/worker-debug/debug.log",
		".aether/data/reviews/history/ledger.json",
	}
	if violations := collectClaimPathViolations(t.TempDir(), "Digger-12", "files_modified", sanctioned); len(violations) != 0 {
		t.Fatalf("expected zero violations for sanctioned .aether/data prefixes, got %+v", violations)
	}

	violations := collectClaimPathViolations(t.TempDir(), "Digger-12", "files_modified", []string{".aether/data/COLONY_STATE.json"})
	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 violation for unsanctioned .aether/data claim, got %d: %+v", len(violations), violations)
	}
	if violations[0].Rule != claimPathRuleAetherData {
		t.Errorf("Rule = %q, want %q", violations[0].Rule, claimPathRuleAetherData)
	}
}

func TestCollectClaimPathViolationsRuleIsAlwaysOneOfFourDocumentedStrings(t *testing.T) {
	root := t.TempDir()
	writeClaimFileForTest(t, root, "src/ok.go")
	valid := map[string]string{
		claimPathRuleNullByte:     "src/\x00null.go",
		claimPathRuleRepoRelative: "/absolute/path.go",
		claimPathRuleAetherData:   ".aether/data/COLONY_STATE.json",
		claimPathRuleEscapesRoot:  "../escape.go",
	}
	for wantRule, path := range valid {
		violations := collectClaimPathViolations(root, "Mason-67", "files_modified", []string{path})
		if len(violations) != 1 {
			t.Fatalf("path %q: expected exactly 1 violation, got %d: %+v", path, len(violations), violations)
		}
		if violations[0].Rule != wantRule {
			t.Errorf("path %q: Rule = %q, want %q", path, violations[0].Rule, wantRule)
		}
		if violations[0].Value != path {
			t.Errorf("path %q: Value = %q, want %q", path, violations[0].Value, path)
		}
	}
}

// --- One rejection, every violation (163.1-02 Task 2) ---

func TestMergeExternalBuildResultsReturnsAllViolations(t *testing.T) {
	manifest := codexBuildManifest{
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Mason-67", Caste: "builder", Stage: "wave", TaskID: "1.1"},
			{Name: "Keen-13", Caste: "watcher", Stage: "verification", TaskID: ""},
		},
	}
	results := []codexExternalBuildWorkerResult{
		{Name: "Mason-67", Caste: "watcher", Status: "completed"},                     // identity_mismatch (caste)
		{Name: "Keen-13", Caste: "watcher", Stage: "verification", Status: "running"}, // status_terminal
	}

	dispatches, violations, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("expected no internal error, got: %v", err)
	}
	if len(violations) != 2 {
		t.Fatalf("expected exactly 2 violations, got %d: %+v", len(violations), violations)
	}
	if len(dispatches) != 2 {
		t.Fatalf("expected 2 dispatch slots (unresolved dispatches keep their manifest entry), got %d", len(dispatches))
	}
	byWorker := map[string]contractViolation{}
	for _, v := range violations {
		byWorker[v.Worker] = v
	}
	if byWorker["Mason-67"].Rule != violationRuleIdentityMismatch {
		t.Errorf("Mason-67 Rule = %q, want %q", byWorker["Mason-67"].Rule, violationRuleIdentityMismatch)
	}
	if byWorker["Keen-13"].Rule != violationRuleStatusTerminal {
		t.Errorf("Keen-13 Rule = %q, want %q", byWorker["Keen-13"].Rule, violationRuleStatusTerminal)
	}
}

func TestValidateCompletionPacketSemanticsReturnsAllViolations(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	manifest, completion := prepareExternalBuildCompletionWithSecondWorker(t, root)
	_ = manifest

	// Plan 194-02 (D-07) shrank the build floor to the builder alone, so
	// this fixture's phase no longer produces a second dispatch on its own
	// -- prepareExternalBuildCompletionWithSecondWorker adds "Keen-6" (the
	// second worker this test needs) explicitly instead.
	for i := range completion.Dispatches {
		switch completion.Dispatches[i].effectiveName() {
		case "Forge-86":
			// Two independent violations from two different validation
			// layers on the SAME worker: a bad claim path (task 1) and an
			// invalid handoff (task 2's mergeExternalBuildResults).
			completion.Dispatches[i].FilesModified = []string{"/etc/passwd"}
			completion.Dispatches[i].Handoff.VerificationStatus = "not-a-real-status"
		case "Keen-6":
			// Two more independent violations, again spanning both layers,
			// on a second worker: an escaping claim path and a non-terminal
			// status.
			completion.Dispatches[i].TestsWritten = []string{"../outside.go"}
			completion.Dispatches[i].Status = "running"
		}
	}

	violations := validateCompletionPacketSemantics(root, completion)
	if len(violations) != 4 {
		t.Fatalf("expected exactly 4 violations, got %d: %+v", len(violations), violations)
	}

	rules := map[string]bool{}
	for _, v := range violations {
		rules[v.Rule] = true
		if v.Worker != "Forge-86" && v.Worker != "Keen-6" {
			t.Errorf("violation attributed to unexpected worker %q: %+v", v.Worker, v)
		}
	}
	if len(rules) != 4 {
		t.Fatalf("expected 4 distinct Rule values, got %d: %v", len(rules), rules)
	}
	wantRules := map[string]bool{
		claimPathRuleRepoRelative:   true,
		violationRuleHandoffValid:   true,
		claimPathRuleEscapesRoot:    true,
		violationRuleStatusTerminal: true,
	}
	for rule := range wantRules {
		if !rules[rule] {
			t.Errorf("expected rule %q among violations, got %v", rule, rules)
		}
	}
}

func TestValidateCompletionPacketSemanticsCleanPacketReturnsNoViolations(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	_, completion := prepareExternalBuildCompletion(t, root)

	if violations := validateCompletionPacketSemantics(root, completion); len(violations) != 0 {
		t.Fatalf("expected a clean packet to produce no violations, got %+v", violations)
	}
}

func TestCompletionContractErrorMessageHasOneLinePerViolation(t *testing.T) {
	err := &completionContractError{Violations: []contractViolation{
		{Worker: "Mason-67", Field: "files_modified", Rule: claimPathRuleRepoRelative, Message: "bad path"},
		{Worker: "Keen-13", Field: "status", Rule: violationRuleStatusTerminal, Message: "non-terminal"},
		{Field: "", Rule: "schema.marshal", Message: "no worker attribution"},
	}}
	lines := strings.Split(err.Error(), "\n")
	if len(lines) != len(err.Violations)+1 {
		t.Fatalf("expected %d lines (1 header + 1 per violation), got %d:\n%s", len(err.Violations)+1, len(lines), err.Error())
	}
	if !strings.HasPrefix(lines[0], "3 completion packet violation(s)") {
		t.Errorf("header line = %q, want to start with violation count", lines[0])
	}
	if !strings.Contains(lines[1], "Mason-67: ") {
		t.Errorf("expected worker prefix on attributed violation, got %q", lines[1])
	}
	if strings.Contains(lines[3], ": ") && strings.HasPrefix(strings.TrimSpace(lines[3]), ":") {
		t.Errorf("expected no dangling worker prefix for an unattributed violation, got %q", lines[3])
	}
}

// TestCompletionPacketRejectionIsAtomic locks in D-06: a rejected completion
// packet must leave the build attempt exactly as it found it -- no status
// transition, no completion digest bound. Everything before the semantic
// validation call in runCodexBuildFinalize (checkpoint save, beginBuildAttempt,
// bindBuildAttemptCompletion, transitionBuildAttempt) must never run for a
// packet that fails validation.
func TestCompletionPacketRejectionIsAtomic(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	_, completion := prepareExternalBuildCompletion(t, root)

	// runCodexBuildPlanOnly (inside prepareExternalBuildCompletion) already
	// minted a "prepared" build attempt with no completion digest bound --
	// capture that baseline so we can prove the rejected finalize call left
	// it byte-for-byte unchanged.
	attemptRel, before, ok := loadLatestBuildAttempt(1)
	if !ok {
		t.Fatal("expected a build attempt to exist from plan-only manifest generation")
	}
	if before.CompletionSHA256 != "" {
		t.Fatalf("baseline attempt must start with no bound completion digest, got %q", before.CompletionSHA256)
	}

	// Break exactly one worker's identity so the packet is rejected.
	for i := range completion.Dispatches {
		if completion.Dispatches[i].effectiveName() == "Forge-86" {
			completion.Dispatches[i].Caste = "watcher"
		}
	}

	if _, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false); err == nil {
		t.Fatal("expected a completion packet with a broken worker identity to be rejected")
	} else {
		var contractErr *completionContractError
		if !errorsAsCompletionContractError(err, &contractErr) {
			t.Fatalf("expected a *completionContractError, got %T: %v", err, err)
		}
		if len(contractErr.Violations) == 0 {
			t.Fatal("expected at least one violation on the rejected packet")
		}
	}

	var after buildAttemptRecord
	if err := store.LoadJSON(attemptRel, &after); err != nil {
		t.Fatalf("reload build attempt after rejected finalize: %v", err)
	}
	if after.CompletionSHA256 != "" {
		t.Fatalf("rejected finalize must not bind a completion digest, got %q", after.CompletionSHA256)
	}
	if after.Status != before.Status {
		t.Fatalf("rejected finalize must not transition attempt status: before=%q after=%q", before.Status, after.Status)
	}

	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload colony state: %v", err)
	}
	if state.State == colony.StateBUILT {
		t.Fatal("colony state must not advance to BUILT when the completion packet was rejected")
	}
}

// TestBuildFinalizeCLIRejectsMultiViolationPacketWithStructuredDetails proves
// the build-finalize CLI surface, not just the Go function: a multi-violation
// packet produces one JSON envelope with ok:false and a details array
// carrying every violation.
func TestBuildFinalizeCLIRejectsMultiViolationPacketWithStructuredDetails(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)
	forceBuildJSONOutput(t)
	_, completion := prepareExternalBuildCompletion(t, root)

	violationCount := 0
	for i := range completion.Dispatches {
		switch completion.Dispatches[i].effectiveName() {
		case "Forge-86":
			completion.Dispatches[i].FilesModified = []string{"/etc/passwd"}
			completion.Dispatches[i].Handoff.VerificationStatus = "not-a-real-status"
			violationCount += 2
		case "Check-80":
			completion.Dispatches[i].TestsWritten = []string{"../outside.go"}
			completion.Dispatches[i].Status = "running"
			violationCount += 2
		}
	}

	completionData, err := json.MarshalIndent(completion, "", "  ")
	if err != nil {
		t.Fatalf("marshal completion: %v", err)
	}
	completionPath := filepath.Join(root, "multi-violation-completion.json")
	if err := os.WriteFile(completionPath, completionData, 0644); err != nil {
		t.Fatalf("write completion: %v", err)
	}

	rootCmd.SetArgs([]string{"build-finalize", "1", "--completion-file", completionPath})
	if err := rootCmd.Execute(); err == nil {
		t.Fatal("expected build-finalize to fail on a multi-violation packet")
	}

	var envelope struct {
		OK      bool                `json:"ok"`
		Error   string              `json:"error"`
		Code    int                 `json:"code"`
		Details []contractViolation `json:"details"`
	}
	if err := json.Unmarshal(stderr.(*bytes.Buffer).Bytes(), &envelope); err != nil {
		t.Fatalf("failed to parse build-finalize error output: %v\n%s", err, stderr.(*bytes.Buffer).String())
	}
	if envelope.OK {
		t.Fatal("expected ok:false")
	}
	if len(envelope.Details) != violationCount {
		t.Fatalf("expected details array of length %d, got %d: %+v", violationCount, len(envelope.Details), envelope.Details)
	}
}

// errorsAsCompletionContractError is a tiny errors.As wrapper kept local to
// this test file to avoid importing "errors" solely for one assertion.
func errorsAsCompletionContractError(err error, target **completionContractError) bool {
	for err != nil {
		if ce, ok := err.(*completionContractError); ok {
			*target = ce
			return true
		}
		unwrapper, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = unwrapper.Unwrap()
	}
	return false
}

func gitInitForTest(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Skipf("git init failed: %v", err)
	}
}

func gitAddForTest(t *testing.T, dir string) {
	t.Helper()
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Skipf("git add failed: %v", err)
	}
}

func writeClaimFileForTest(t *testing.T, root, rel string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

// ---------------------------------------------------------------------------
// Task 2 (163-05): suggest-analyze wired into build finalize as its first
// live caller. Deliberate-regression proof (removing the collectPendingSuggestions
// call in runCodexBuildFinalize and re-running this suite) is recorded in
// 163-05-SUMMARY.md, not duplicated here.
// ---------------------------------------------------------------------------

func TestBuildFinalizeCollectsSuggestAnalyzeResults(t *testing.T) {
	t.Run("populates PendingSuggestions in COLONY_STATE.json", func(t *testing.T) {
		root := setupExternalBuildAttemptTest(t)
		if err := os.WriteFile(filepath.Join(root, ".env"), []byte("SECRET=abc123\n"), 0o644); err != nil {
			t.Fatalf("write .env fixture: %v", err)
		}
		_, completion := prepareExternalBuildCompletion(t, root)

		if _, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false); err != nil {
			t.Fatalf("finalize: %v", err)
		}

		var reloaded colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &reloaded); err != nil {
			t.Fatalf("reload colony state: %v", err)
		}
		if reloaded.PendingSuggestions == nil || len(*reloaded.PendingSuggestions) == 0 {
			t.Error("expected pending_suggestions to be populated after a completed build with analysable patterns")
		}
	})

	t.Run("result carries a pending suggestion count and the approve next action", func(t *testing.T) {
		root := setupExternalBuildAttemptTest(t)
		if err := os.WriteFile(filepath.Join(root, ".env"), []byte("SECRET=abc123\n"), 0o644); err != nil {
			t.Fatalf("write .env fixture: %v", err)
		}
		_, completion := prepareExternalBuildCompletion(t, root)

		result, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
		if err != nil {
			t.Fatalf("finalize: %v", err)
		}
		if result["suggest_analyze_ran"] != true {
			t.Errorf("expected suggest_analyze_ran=true, got %v", result["suggest_analyze_ran"])
		}
		count, ok := result["pending_suggestion_count"].(int)
		if !ok || count == 0 {
			t.Fatalf("expected pending_suggestion_count > 0, got %v (%T)", result["pending_suggestion_count"], result["pending_suggestion_count"])
		}
		if result["pending_suggestions_next"] != "aether suggest-approve" {
			t.Errorf("expected pending_suggestions_next to name the approve command, got %v", result["pending_suggestions_next"])
		}
	})

	t.Run("a suggest-analyze failure leaves finalize succeeding with a zero count", func(t *testing.T) {
		// collectPendingSuggestions is the exact seam runCodexBuildFinalize
		// calls; testing it directly with no store initialized proves the
		// non-blocking contract deterministically -- store can never be nil
		// mid-finalize (finalize itself requires it), so this is the only
		// way to force runSuggestAnalyze's one hard-error path without
		// fabricating an inconsistent finalize fixture.
		origStore := store
		store = nil
		defer func() { store = origStore }()

		ran, count := collectPendingSuggestions(".")
		if ran {
			t.Error("expected ran=false when suggest-analyze cannot run (no store)")
		}
		if count != 0 {
			t.Errorf("expected count=0 when suggest-analyze cannot run, got %d", count)
		}

		// Confirm the same non-blocking contract holds inside a real,
		// successful finalize: a colony with no analysable pattern changes
		// still finalizes successfully and reports zero pending suggestions
		// rather than an error.
		store = origStore
		root := setupExternalBuildAttemptTest(t)
		_, completion := prepareExternalBuildCompletion(t, root)
		result, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
		if err != nil {
			t.Fatalf("finalize with no analysable patterns should still succeed: %v", err)
		}
		if result["suggest_analyze_ran"] != true {
			t.Errorf("expected suggest_analyze_ran=true (ran, found nothing), got %v", result["suggest_analyze_ran"])
		}
	})

	t.Run("suggest-analyze is invoked exactly once per finalize, never per dispatch", func(t *testing.T) {
		// Plan 194-02 (D-07) shrank the build floor to the builder alone;
		// plan 194-05 (D-11) then removed the no-proposal keyword-scoring
		// fallback that used to score a watcher via relevance on this exact
		// fixture's wording, so a plain no-proposal build now produces
		// exactly one dispatch. This test's whole point is proving
		// per-finalize (not per-dispatch) invocation, so it needs a real
		// second dispatch in the ACTUAL build manifest --
		// prepareExternalBuildCompletionWithProposal reaches the judgement
		// path with an explicit proposal, unlike appending a synthetic
		// dispatch after the fact (which breaks the durable build-attempt
		// hash the finalizer checks against).
		root := setupExternalBuildAttemptTestWithVerifiableWork(t)
		if err := os.WriteFile(filepath.Join(root, ".env"), []byte("SECRET=abc123\n"), 0o644); err != nil {
			t.Fatalf("write .env fixture: %v", err)
		}
		_, completion := prepareExternalBuildCompletionWithProposal(t, root,
			[]string{"builder", "watcher"}, []string{"watcher=an independent check before this lands"})
		if len(completion.Dispatches) < 2 {
			t.Fatalf("fixture must have multiple dispatches to prove per-finalize (not per-dispatch) invocation, got %d", len(completion.Dispatches))
		}

		before := suggestAnalyzeInvocationCount
		if _, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false); err != nil {
			t.Fatalf("finalize: %v", err)
		}
		after := suggestAnalyzeInvocationCount

		if got := after - before; got != 1 {
			t.Errorf("expected runSuggestAnalyze to be invoked exactly once for %d dispatches, got %d invocations", len(completion.Dispatches), got)
		}
	})
}

// ---------------------------------------------------------------------------
// T-188-09 (D-10, D-11): validateBuildAttemptManifestBinding already detects
// a completion whose dispatch_manifest carries neither attempt_id nor
// attempt_path (binding.Legacy == true) -- the 2026-08-17 Audit Addendum
// keeps this branch on purpose, but before this fix nothing ever read
// binding.Legacy, so a legacy manifest was silently indistinguishable from a
// normal fresh completion. This must still be accepted (no new refusal) but
// must now warn loudly on stderr and leave a durable, phase-numbered Events
// entry -- on every occurrence, and never on the ordinary bound path.
// ---------------------------------------------------------------------------

func TestBuildFinalizeWarnsOnLegacyUnboundManifest(t *testing.T) {
	t.Run("a manifest with no attempt binding is accepted and warns loudly", func(t *testing.T) {
		root := setupExternalBuildAttemptTest(t)
		_, completion := prepareExternalBuildCompletion(t, root)

		var errBuf bytes.Buffer
		stderr = &errBuf

		// Simulate a genuinely old completion packet and repository: none of
		// the modern plan/attempt claims existed, while the historical
		// plan-only source owner/mode did. The plan-only helper necessarily
		// created modern attempt evidence, so remove only that temporary
		// fixture directory before exercising the compatibility adapter.
		makeLegacyUnboundCompletion200(t, root, &completion)

		result, updatedState, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
		if err != nil {
			t.Fatalf("a legacy unbound manifest must still be accepted (no new refusal), got error: %v", err)
		}
		if updatedState.State != colony.StateBUILT {
			t.Fatalf("expected the legacy manifest to finalize the build normally, state=%s", updatedState.State)
		}
		if result["idempotent"] != false {
			t.Errorf("expected a fresh (non-idempotent) finalize, got result=%+v", result)
		}

		warning := errBuf.String()
		if warning == "" {
			t.Fatal("expected an unconditional stderr warning for a legacy unbound manifest, got none")
		}
		if !strings.Contains(warning, "older, less strictly checked method") {
			t.Errorf("expected a plain-language stderr warning describing the older method, got: %q", warning)
		}
		if strings.Contains(strings.ToLower(warning), "legacy") || strings.Contains(strings.ToLower(warning), "binding") {
			t.Errorf("warning must explain itself in plain language, not the bare jargon words 'legacy'/'binding': %q", warning)
		}

		foundEvent := false
		for _, evt := range updatedState.Events {
			if strings.Contains(evt, "manifest_legacy_accepted") && strings.Contains(evt, "Phase 1") {
				foundEvent = true
				break
			}
		}
		if !foundEvent {
			t.Fatalf("expected state.Events to contain a manifest_legacy_accepted entry naming phase 1, got: %v", updatedState.Events)
		}
	})

	t.Run("an ordinary bound manifest stays silent on both channels", func(t *testing.T) {
		root := setupExternalBuildAttemptTest(t)
		_, completion := prepareExternalBuildCompletion(t, root)

		var errBuf bytes.Buffer
		stderr = &errBuf

		// Attempt-binding fields are left exactly as prepareExternalBuildCompletion
		// set them: this is the ordinary, fully-bound path.
		if completion.DispatchManifest.AttemptID == "" || completion.DispatchManifest.AttemptPath == "" {
			t.Fatal("fixture precondition: the ordinary path must start out fully bound")
		}

		_, updatedState, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
		if err != nil {
			t.Fatalf("bound manifest should finalize cleanly, got error: %v", err)
		}

		if warning := errBuf.String(); strings.Contains(warning, "older, less strictly checked method") {
			t.Errorf("the ordinary bound path must not emit the legacy warning, got stderr: %q", warning)
		}
		for _, evt := range updatedState.Events {
			if strings.Contains(evt, "manifest_legacy_accepted") {
				t.Errorf("the ordinary bound path must not append a manifest_legacy_accepted event, got: %v", updatedState.Events)
			}
		}
	})
}

// TestBuildStartLegacyBinding200 closes the compatibility lane around one
// explicit old shape. Missing modern fields are rebound by the canonical
// receipt-last transaction; any mixture of old and modern claims is an
// ambiguity, not an invitation to guess. Every refusal and exact replay is
// proved read-only with a full durable-data fingerprint.
func TestBuildStartLegacyBinding200(t *testing.T) {
	t.Run("genuine legacy input receives trusted canonical execution identity and replays read-only", func(t *testing.T) {
		root := setupExternalBuildAttemptTest(t)
		_, completion := prepareExternalBuildCompletion(t, root)
		makeLegacyUnboundCompletion200(t, root, &completion)

		first, state, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
		if err != nil {
			t.Fatalf("finalize genuine legacy completion: %v", err)
		}
		attemptRel, attempt, ok := loadLatestBuildAttempt(1)
		if !ok {
			t.Fatal("canonical legacy adapter did not persist an attempt")
		}
		wantOwner := buildExecutionOwner("external-task", false)
		if attempt.ExecutionOwner != wantOwner || attempt.DispatchMode != "external-task" {
			t.Fatalf("legacy execution identity trusted inbound values: owner/mode=%q/%q, want %q/external-task", attempt.ExecutionOwner, attempt.DispatchMode, wantOwner)
		}
		if attempt.PlanManifest == nil || attempt.PlanManifest.ExecutionBinding == nil {
			t.Fatalf("canonical attempt has no derived manifest binding: %+v", attempt.PlanManifest)
		}
		bound := attempt.PlanManifest
		if bound.AttemptID != attempt.ID || bound.ExecutionBinding.AttemptID != attempt.ID ||
			bound.ExecutionOwner != wantOwner || bound.DispatchMode != "external-task" ||
			!bound.PlanAuthority.Eligible {
			t.Fatalf("derived legacy binding is incomplete or untrusted: manifest=%+v attempt=%+v", *bound, attempt)
		}
		wantDisplay, pathErr := displayBuildAttemptDataPath(attemptRel)
		if pathErr != nil {
			t.Fatalf("display canonical attempt: %v", pathErr)
		}
		if bound.AttemptPath != wantDisplay || first["attempt"] != wantDisplay {
			t.Fatalf("legacy attempt paths = manifest %q/result %v, want %q", bound.AttemptPath, first["attempt"], wantDisplay)
		}
		receiptPath := filepath.Join(store.BasePath(), filepath.FromSlash(buildStartReceiptPath(1, attempt.ID)))
		if _, statErr := os.Stat(receiptPath); statErr != nil {
			t.Fatalf("canonical start receipt missing: %v", statErr)
		}
		if state.State != colony.StateBUILT {
			t.Fatalf("legacy finalize state=%s, want %s", state.State, colony.StateBUILT)
		}

		beforeReplay := snapshotProjectDataTree(t, store.BasePath())
		replayed, replayState, _, _, replayErr := runCodexBuildFinalize(root, 1, completion, false)
		if replayErr != nil {
			t.Fatalf("exact legacy replay: %v", replayErr)
		}
		afterReplay := snapshotProjectDataTree(t, store.BasePath())
		if !reflect.DeepEqual(beforeReplay, afterReplay) {
			t.Fatalf("exact legacy replay mutated durable data\nbefore: %#v\nafter:  %#v", beforeReplay, afterReplay)
		}
		if replayed["idempotent"] != true || replayed["attempt"] != wantDisplay || replayState.State != colony.StateBUILT {
			t.Fatalf("exact legacy replay result=%+v state=%s", replayed, replayState.State)
		}
	})

	refusals := []struct {
		name   string
		mutate func(*codexBuildManifest)
	}{
		{
			name: "attempt id without attempt path is partial",
			mutate: func(manifest *codexBuildManifest) {
				manifest.AttemptPath = ""
			},
		},
		{
			name: "execution binding without attempt identity is contradictory",
			mutate: func(manifest *codexBuildManifest) {
				manifest.AttemptID = ""
				manifest.AttemptPath = ""
			},
		},
		{
			name: "modern authority cannot claim the legacy lane",
			mutate: func(manifest *codexBuildManifest) {
				manifest.AttemptID = ""
				manifest.AttemptPath = ""
				manifest.ExecutionBinding = nil
			},
		},
		{
			name: "forged modern binding is refused",
			mutate: func(manifest *codexBuildManifest) {
				manifest.ExecutionBinding.ExecutionOwner = "forged-owner"
			},
		},
		{
			name: "legacy source owner and mode cannot contradict history",
			mutate: func(manifest *codexBuildManifest) {
				manifest.AttemptID = ""
				manifest.AttemptPath = ""
				manifest.ExecutionBinding = nil
				manifest.PlanAuthority = planAuthorityDecision{}
				manifest.PlanRevisionID = ""
				manifest.PlanStateHash = ""
				manifest.ExecutionOwner = "forged-owner"
				manifest.DispatchMode = "external-task"
			},
		},
	}
	for _, test := range refusals {
		t.Run(test.name, func(t *testing.T) {
			root := setupExternalBuildAttemptTest(t)
			_, completion := prepareExternalBuildCompletion(t, root)
			test.mutate(completion.DispatchManifest)
			before := snapshotProjectDataTree(t, store.BasePath())
			if _, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false); err == nil {
				t.Fatal("ambiguous or forged completion was accepted")
			}
			after := snapshotProjectDataTree(t, store.BasePath())
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("refusal mutated durable data\nbefore: %#v\nafter:  %#v", before, after)
			}
		})
	}
}

func makeLegacyUnboundCompletion200(t *testing.T, root string, completion *codexExternalBuildCompletion) {
	t.Helper()
	if completion == nil || completion.DispatchManifest == nil {
		t.Fatal("legacy fixture requires a dispatch manifest")
	}
	manifest := completion.DispatchManifest
	manifest.AttemptID = ""
	manifest.AttemptPath = ""
	manifest.ExecutionBinding = nil
	manifest.PlanAuthority = planAuthorityDecision{}
	manifest.PlanRevisionID = ""
	manifest.PlanStateHash = ""
	buildDir := filepath.Join(root, ".aether", "data", "build", fmt.Sprintf("phase-%d", manifest.Phase))
	if err := os.RemoveAll(buildDir); err != nil {
		t.Fatalf("remove modern-only attempt evidence from legacy fixture: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 195-06 Task 2: external/wrapper build-finalize reuses the SAME task-receipt
// trust boundary (admitCoherentJobTaskReceipts / finalizeCoherentJobTaskReceiptEvidence,
// via resolveCoherentJobDispatchReceipts, cmd/coherent_job_receipts.go) the
// native lane wired in 195-04, so a real `aether build-finalize` call --  not
// just the merge/resolve functions in isolation -- projects exact receipt
// evidence into exact task state and colony state (195-CONTEXT.md D-08/D-09).
// ---------------------------------------------------------------------------

// setupCoherentJobExternalFinalizeTest builds a real, plan-only manifest for
// six dependent tasks the runtime groups into ONE coherent job dispatch
// (planCoherentJobs), the exact shape D-08/D-09's partial-credit
// rulings are about. Returns root and the merged chain dispatch.
func setupCoherentJobExternalFinalizeTest(t *testing.T, goal string) (string, codexBuildManifest, codexBuildDispatch, []string) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	tasks, ids := sixChainedTasks()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0", Goal: &goal, State: colony.StateREADY, ColonyDepth: "standard", CurrentPhase: 0,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID: 1, Name: "Merged chain", Description: "One worker, six dependent steps",
			Status: colony.PhaseReady, Tasks: tasks,
		}}},
	})

	result, _, _, _, err := runCodexBuildPlanOnly(root, 1, nil)
	if err != nil {
		t.Fatalf("plan-only build: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)

	var chain codexBuildDispatch
	for _, dispatch := range manifest.Dispatches {
		if len(dispatch.CoveredTaskIDs) > 1 {
			chain = dispatch
			break
		}
	}
	if chain.Name == "" || len(chain.CoveredTaskIDs) != len(ids) {
		t.Fatalf("fixture did not produce one merged dispatch covering all six tasks; dispatches: %+v", manifest.Dispatches)
	}
	return root, manifest, chain, ids
}

// TestExternalGroupedFullSuccessCreditsAll is the regression guard for the
// legacy path: a real build-finalize call for a whole-success grouped
// dispatch (no receipts needed) still reaches BUILT with every covered task
// completed, exactly as before 195-06 wired receipt resolution into this
// entrypoint.
func TestExternalGroupedFullSuccessCreditsAll(t *testing.T) {
	root, manifest, chain, _ := setupCoherentJobExternalFinalizeTest(t, "Whole success still credits every covered task")

	if err := os.WriteFile(filepath.Join(root, "external-evidence.txt"), []byte("durable external work\n"), 0o644); err != nil {
		t.Fatalf("write external evidence: %v", err)
	}
	results := []codexExternalBuildWorkerResult{{
		Stage: chain.Stage, Wave: chain.Wave, ExecutionWave: normalizedDispatchWave(chain),
		Caste: chain.Caste, Name: chain.Name, TaskID: chain.TaskID,
		Status:        "completed",
		Summary:       chain.Name + " finished its chain",
		FilesModified: []string{"external-evidence.txt"},
		Handoff: codex.WorkerHandoff{
			VerificationStatus:     "pass",
			CommandsRun:            []string{"go test ./..."},
			NextWorkerInstructions: []string{"chain complete"},
		},
	}}
	completion := codexExternalBuildCompletion{DispatchManifest: &manifest, Dispatches: results}

	_, updatedState, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("finalize: %v", err)
	}
	if updatedState.State != colony.StateBUILT {
		t.Fatalf("whole-success grouped job state = %s, want %s", updatedState.State, colony.StateBUILT)
	}
	for _, task := range updatedState.Plan.Phases[0].Tasks {
		if task.Status != colony.TaskCompleted {
			t.Fatalf("task %s is %q after whole-success chain completion, want %q", *task.ID, task.Status, colony.TaskCompleted)
		}
	}
}

// TestExternalGroupedPartialPersistsExactTaskState is the external-lane
// counterpart of TestFailedGroupedDispatchCreditsExactlyReceiptedTasks
// (cmd/merged_dispatch_task_credit_test.go, native lane), run through the
// REAL `aether build-finalize` entrypoint rather than the bare merge/resolve
// functions: a single external worker covering six chained tasks that failed
// after finishing four, with valid per-task receipts for those four, must
// credit exactly those four, leave the other two pending, and never report
// the colony as BUILT.
//
// "An admitted candidate whose root artifact/evidence is absent is removed
// during finalization and remains pending" (this plan's own acceptance
// criteria) is already locked, lane-neutrally, by
// TestCoherentJobReceiptFinalization's "a candidate whose file is missing
// from root is not credited" subtest (cmd/coherent_job_receipts_test.go,
// 195-04) -- finalizeCoherentJobTaskReceiptEvidence is the exact same
// function this external lane now calls, unchanged, so that coverage already
// applies here and is not duplicated in this file. A packet-level attempt to
// reproduce a "vanished between admission and finalization" file at THIS
// entrypoint cannot reach that code path at all: validateExternalWorkerResultClaimPaths
// (part of validateCompletionPacketSemantics, which runs before merge/receipt
// resolution) already refuses a completion packet whose own files_modified
// names a path that does not exist in root at submission time, on any lane.
func TestExternalGroupedPartialPersistsExactTaskState(t *testing.T) {
	root, manifest, chain, ids := setupCoherentJobExternalFinalizeTest(t, "Partial credit persists exact task state")

	proven := ids[:4]
	pending := ids[4:]

	receipts := make([]codex.TaskReceipt, 0, len(proven))
	touchedFiles := make([]string, 0, len(proven))
	for _, id := range proven {
		receipts = append(receipts, receiptForTask(t, root, id))
		touchedFiles = append(touchedFiles, taskFileName(id))
	}

	results := []codexExternalBuildWorkerResult{{
		Stage: chain.Stage, Wave: chain.Wave, ExecutionWave: normalizedDispatchWave(chain),
		Caste: chain.Caste, Name: chain.Name, TaskID: chain.TaskID,
		Status:        "failed",
		Summary:       "crashed after finishing four of six steps",
		FilesModified: touchedFiles,
		Handoff: codex.WorkerHandoff{
			VerificationStatus: "fail",
			CommandsRun:        []string{"go test ./..."},
		},
		TaskReceipts: receipts,
	}}
	completion := codexExternalBuildCompletion{DispatchManifest: &manifest, Dispatches: results}

	_, updatedState, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
	if err != nil {
		t.Fatalf("build-finalize should accept a failed dispatch with genuine partial receipts, got error: %v", err)
	}
	if updatedState.State == colony.StateBUILT {
		t.Fatalf("colony state = %s after only 4 of 6 tasks were credited, want anything but BUILT (honest partial state)", updatedState.State)
	}

	statusByID := map[string]string{}
	for _, task := range updatedState.Plan.Phases[0].Tasks {
		statusByID[*task.ID] = string(task.Status)
	}
	for _, id := range proven {
		if statusByID[id] != string(colony.TaskCompleted) {
			t.Fatalf("task %s is %q, want %q", id, statusByID[id], colony.TaskCompleted)
		}
	}
	for _, id := range pending {
		if statusByID[id] == string(colony.TaskCompleted) {
			t.Fatalf("task %s (no receipt at all) is %q, want it to remain pending", id, statusByID[id])
		}
	}
}

// TestExternalGroupedFailureWithoutReceiptsCreditsNone is D-08's negative
// case at the real build-finalize entrypoint: an entirely failed grouped job
// with zero task receipts is rejected by the phantom-build provenance guard
// (SAFE-01) -- never silently credited from touched files or a summary
// naming every task -- so no grouped task state changes at all.
func TestExternalGroupedFailureWithoutReceiptsCreditsNone(t *testing.T) {
	root, manifest, chain, ids := setupCoherentJobExternalFinalizeTest(t, "No receipts, no credit")

	touchedFiles := make([]string, 0, len(ids))
	for _, id := range ids {
		file := taskFileName(id)
		if err := os.WriteFile(filepath.Join(root, file), []byte("package fixture\n"), 0o644); err != nil {
			t.Fatalf("write fixture file for task %s: %v", id, err)
		}
		touchedFiles = append(touchedFiles, file)
	}

	results := []codexExternalBuildWorkerResult{{
		Stage: chain.Stage, Wave: chain.Wave, ExecutionWave: normalizedDispatchWave(chain),
		Caste: chain.Caste, Name: chain.Name, TaskID: chain.TaskID,
		Status:        "failed",
		Summary:       "touched every file for tasks " + fmt.Sprint(ids) + " but has no per-task receipt",
		FilesModified: touchedFiles,
		Handoff: codex.WorkerHandoff{
			VerificationStatus: "fail",
			CommandsRun:        []string{"go test ./..."},
		},
	}}
	completion := codexExternalBuildCompletion{DispatchManifest: &manifest, Dispatches: results}

	if _, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false); err == nil {
		t.Fatal("expected build-finalize to reject an entirely failed job with no task receipts at all (phantom-build guard, SAFE-01)")
	}

	var reloaded colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &reloaded); err != nil {
		t.Fatalf("reload colony state: %v", err)
	}
	for _, task := range reloaded.Plan.Phases[0].Tasks {
		if task.Status == colony.TaskCompleted {
			t.Fatalf("task %s is %q after a rejected completion packet, want no state change at all", *task.ID, task.Status)
		}
	}
}

// TestExternalPartialClaimsMatchTaskState proves two things (195-06 Task 2):
// (1) in-repo mode, the dispatches runCodexBuildFinalize returns carry
// CompletedTaskIDs/TaskClaims that agree exactly with the persisted task
// state it just committed -- result diagnostics and task state can never
// silently disagree; and (2) a worktree-backed completion's dispatches, as
// they exist right after mergeExternalBuildResults and before any
// resolveCoherentJobDispatchReceipts call, have NO CompletedTaskIDs at all --
// runCodexBuildFinalize's own worktree gate skips that call entirely, since
// root does not yet have the synced files, and plan 195-08's sync-first
// adapter is what consumes the still-intact TaskReceipts afterward.
func TestExternalPartialClaimsMatchTaskState(t *testing.T) {
	t.Run("in-repo: returned dispatches agree with committed task state", func(t *testing.T) {
		root, manifest, chain, ids := setupCoherentJobExternalFinalizeTest(t, "Claims match task state")

		receiptedIDs := ids[:4]
		receipts := make([]codex.TaskReceipt, 0, len(receiptedIDs))
		touchedFiles := make([]string, 0, len(receiptedIDs))
		for _, id := range receiptedIDs {
			receipts = append(receipts, receiptForTask(t, root, id))
			touchedFiles = append(touchedFiles, taskFileName(id))
		}
		results := []codexExternalBuildWorkerResult{{
			Stage: chain.Stage, Wave: chain.Wave, ExecutionWave: normalizedDispatchWave(chain),
			Caste: chain.Caste, Name: chain.Name, TaskID: chain.TaskID,
			Status:        "failed",
			Summary:       "crashed after finishing four of six steps",
			FilesModified: touchedFiles,
			Handoff: codex.WorkerHandoff{
				VerificationStatus: "fail",
				CommandsRun:        []string{"go test ./..."},
			},
			TaskReceipts: receipts,
		}}
		completion := codexExternalBuildCompletion{DispatchManifest: &manifest, Dispatches: results}

		_, updatedState, _, dispatches, err := runCodexBuildFinalize(root, 1, completion, false)
		if err != nil {
			t.Fatalf("finalize: %v", err)
		}

		statusByID := map[string]string{}
		for _, task := range updatedState.Plan.Phases[0].Tasks {
			statusByID[*task.ID] = string(task.Status)
		}

		var merged codexBuildDispatch
		for _, d := range dispatches {
			if d.Name == chain.Name {
				merged = d
				break
			}
		}
		gotCompleted := append([]string{}, merged.CompletedTaskIDs...)
		sort.Strings(gotCompleted)
		if fmt.Sprint(gotCompleted) != fmt.Sprint(receiptedIDs) {
			t.Fatalf("returned dispatch CompletedTaskIDs = %v, want %v", gotCompleted, receiptedIDs)
		}
		for _, id := range gotCompleted {
			if statusByID[id] != string(colony.TaskCompleted) {
				t.Fatalf("dispatch claims task %s complete but committed task state says %q", id, statusByID[id])
			}
		}
		gotClaimIDs := make([]string, 0, len(merged.TaskClaims))
		for _, c := range merged.TaskClaims {
			gotClaimIDs = append(gotClaimIDs, c.TaskID)
		}
		sort.Strings(gotClaimIDs)
		if fmt.Sprint(gotClaimIDs) != fmt.Sprint(receiptedIDs) {
			t.Fatalf("returned dispatch TaskClaims cover %v, want exactly %v", gotClaimIDs, receiptedIDs)
		}
	})

	t.Run("worktree-backed: no pre-sync CompletedTaskIDs", func(t *testing.T) {
		root := t.TempDir()
		ids := []string{"1.1", "1.2", "1.3", "1.4", "1.5", "1.6"}
		chain := codexBuildDispatch{Name: "Weaver-1", Caste: "builder", TaskID: ids[0], CoveredTaskIDs: ids, Status: "failed"}
		manifest := codexBuildManifest{Phase: 1, Dispatches: []codexBuildDispatch{chain}}

		receiptedIDs := ids[:4]
		receipts := make([]codex.TaskReceipt, 0, len(receiptedIDs))
		touchedFiles := make([]string, 0, len(receiptedIDs))
		for _, id := range receiptedIDs {
			receipts = append(receipts, receiptForTask(t, root, id))
			touchedFiles = append(touchedFiles, taskFileName(id))
		}
		results := []codexExternalBuildWorkerResult{{
			Name: chain.Name, TaskID: chain.TaskID, Status: "failed",
			Summary:       "crashed after finishing four of six steps",
			FilesModified: touchedFiles,
			Handoff: codex.WorkerHandoff{
				VerificationStatus: "fail",
				CommandsRun:        []string{"go test ./..."},
			},
			TaskReceipts: receipts,
		}}

		dispatches, violations, err := mergeExternalBuildResults(manifest, results)
		if err != nil {
			t.Fatalf("mergeExternalBuildResults: %v", err)
		}
		if len(violations) > 0 {
			t.Fatalf("unexpected violations: %+v", violations)
		}

		// This IS the exact state runCodexBuildFinalize's own worktree gate
		// leaves dispatches in -- it never calls
		// resolveCoherentJobDispatchReceipts for a worktree-backed completion
		// (root does not yet have the synced files); plan 195-08's
		// sync-first adapter is what runs admission/finalization afterward.
		for _, d := range dispatches {
			if len(d.CompletedTaskIDs) != 0 || len(d.TaskClaims) != 0 {
				t.Fatalf("a worktree-backed dispatch must have no pre-sync CompletedTaskIDs/TaskClaims, got %v / %+v", d.CompletedTaskIDs, d.TaskClaims)
			}
			if len(d.TaskReceipts) != len(receiptedIDs) {
				t.Fatalf("worktree lane must still carry TaskReceipts through unchanged for 195-08's later admission, got %d, want %d", len(d.TaskReceipts), len(receiptedIDs))
			}
		}
	})
}
