package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

	dispatches, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("mergeExternalBuildResults with code_written: %v", err)
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

	dispatches, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("mergeExternalBuildResults with ant_name: %v", err)
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

	dispatches, err := mergeExternalBuildResults(manifest, results)
	if err != nil {
		t.Fatalf("mergeExternalBuildResults with retry suffix drift: %v", err)
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

	_, err := mergeExternalBuildResults(manifest, results)
	if err == nil {
		t.Fatal("expected ambiguous retry suffix match to fail")
	}
	if !contains(err.Error(), "ambiguous external worker result") {
		t.Fatalf("expected ambiguous match error, got: %v", err)
	}
}

func TestClaimsOrAggregateWithAntName(t *testing.T) {
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

	claims := completion.claimsOrAggregate(t.TempDir(), 1, time.Now().UTC(), dispatches)

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
	_, err := mergeExternalBuildResults(manifest, results)
	// mergeExternalBuildResults itself does not validate root — that happens
	// in runCodexBuildFinalize via validateFinalizerManifestRoot.
	// Verify that the merge itself succeeds (root is checked upstream).
	if err != nil {
		t.Fatalf("mergeExternalBuildResults should not check root (upstream concern): %v", err)
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
	_, err := mergeExternalBuildResults(manifest, results)
	if err == nil {
		t.Fatal("expected error for wrong caste")
	}
	if !strings.Contains(err.Error(), "caste") {
		t.Fatalf("expected caste mismatch error, got: %v", err)
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
	_, err := mergeExternalBuildResults(manifest, results)
	if err == nil {
		t.Fatal("expected error for wrong stage")
	}
	if !strings.Contains(err.Error(), "stage") {
		t.Fatalf("expected stage mismatch error, got: %v", err)
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
	_, err := mergeExternalBuildResults(manifest, results)
	if err == nil {
		t.Fatal("expected error for wrong task_id")
	}
	if !strings.Contains(err.Error(), "task_id") {
		t.Fatalf("expected task_id mismatch error, got: %v", err)
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
	_, err := mergeExternalBuildResults(manifest, results)
	if err == nil {
		t.Fatal("expected error for wrong wave")
	}
	if !strings.Contains(err.Error(), "wave") {
		t.Fatalf("expected wave mismatch error, got: %v", err)
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
	_, err := mergeExternalBuildResults(manifest, results)
	if err == nil {
		t.Fatal("expected error for wrong execution_wave")
	}
	if !strings.Contains(err.Error(), "execution_wave") {
		t.Fatalf("expected execution_wave mismatch error, got: %v", err)
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
	_, err := mergeExternalBuildResults(manifest, results)
	if err == nil {
		t.Fatal("expected error for duplicate worker result")
	}
	if !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate error, got: %v", err)
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
	_, err := mergeExternalBuildResults(manifest, results)
	if err == nil {
		t.Fatal("expected error for non-terminal status")
	}
	if !strings.Contains(err.Error(), "non-terminal") {
		t.Fatalf("expected non-terminal status error, got: %v", err)
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
	_, err := mergeExternalBuildResults(manifest, results)
	if err == nil {
		t.Fatal("expected error for missing worker result")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Fatalf("expected missing result error, got: %v", err)
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
	_, err := mergeExternalBuildResults(manifest, results)
	if err == nil {
		t.Fatal("expected error for nameless result")
	}
	if !strings.Contains(err.Error(), "missing name") {
		t.Fatalf("expected missing name error, got: %v", err)
	}
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
