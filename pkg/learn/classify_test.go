package learn

import "testing"

// TestClassifyEntry tests 4-way classification per D-10, D-11.
// Table-driven with cases covering all Classification values.
func TestClassifyEntry(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		scanResult PrivacyScanResult
		want       Classification
	}{
		// Blocked: secrets detected by privacy scan
		{
			name:    "API key blocked",
			content: "Use sk-abcdef1234567890abcdef12 for the client",
			scanResult: PrivacyScanResult{
				Blocked:  true,
				Findings: []string{"secret pattern matched: api_key"},
			},
			want: ClassBlocked,
		},
		{
			name:    "bearer token blocked",
			content: "Authorization: bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			scanResult: PrivacyScanResult{
				Blocked:  true,
				Findings: []string{"secret pattern matched: bearer_token"},
			},
			want: ClassBlocked,
		},
		{
			name:    "RSA private key blocked",
			content: "-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQ\n-----END RSA PRIVATE KEY-----",
			scanResult: PrivacyScanResult{
				Blocked:  true,
				Findings: []string{"secret pattern matched: private_key_rsa"},
			},
			want: ClassBlocked,
		},

		// RepoLocal: paths redacted by privacy scan (clean != original content)
		{
			name:    "local file path redacted",
			content: "See /Users/dev/project/file.go for the implementation",
			scanResult: PrivacyScanResult{
				Blocked: false,
				Clean:   "See [REDACTED_PATH] for the implementation",
			},
			want: ClassRepoLocal,
		},
		{
			name:    "home directory path redacted",
			content: "Check ~/code/test/main.go for the test",
			scanResult: PrivacyScanResult{
				Blocked: false,
				Clean:   "Check [REDACTED_PATH] for the test",
			},
			want: ClassRepoLocal,
		},

		// HiveShareable: clean content passes through unchanged + is generic
		{
			name:    "generic best practice",
			content: "Use table-driven tests for better readability",
			scanResult: PrivacyScanResult{
				Blocked: false,
				Clean:   "Use table-driven tests for better readability",
			},
			want: ClassHiveShareable,
		},
		{
			name:    "generic coding pattern",
			content: "Always handle errors before proceeding to the next step",
			scanResult: PrivacyScanResult{
				Blocked: false,
				Clean:   "Always handle errors before proceeding to the next step",
			},
			want: ClassHiveShareable,
		},

		// NeedsApproval: clean content passes through but is NOT generic
		{
			name:    "ambiguous project reference",
			content: "The Builder agent works well in cmd/",
			scanResult: PrivacyScanResult{
				Blocked: false,
				Clean:   "The Builder agent works well in cmd/",
			},
			want: ClassNeedsApproval,
		},
		{
			name:    "content with file extension",
			content: "Implement the handler in handler.go",
			scanResult: PrivacyScanResult{
				Blocked: false,
				Clean:   "Implement the handler in handler.go",
			},
			want: ClassNeedsApproval,
		},

		// Edge cases
		{
			name:    "empty content blocked",
			content: "",
			scanResult: PrivacyScanResult{
				Blocked:  true,
				Findings: []string{"empty content"},
			},
			want: ClassBlocked,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyEntry(tt.content, tt.scanResult)
			if got != tt.want {
				t.Errorf("ClassifyEntry(%q, %+v) = %q, want %q",
					tt.content, tt.scanResult, got, tt.want)
			}
		})
	}
}

// TestCollectEvidence_FullStruct verifies CollectEvidence assembles a complete
// Evidence struct from worker results and gate results.
func TestCollectEvidence_FullStruct(t *testing.T) {
	workers := []WorkerResult{
		{Name: "Builder-1", Caste: "builder", Status: "done", FilesTouched: []string{"cmd/main.go"}},
		{Name: "Watcher-1", Caste: "watcher", Status: "done", FilesTouched: []string{"pkg/learn/learn.go", "pkg/learn/evidence.go"}},
	}
	gates := GateResult{Passed: 3, Total: 3}

	got := CollectEvidence("run-42", 5, workers, gates, "repo-local")

	if got.RunID != "run-42" {
		t.Errorf("RunID = %q, want %q", got.RunID, "run-42")
	}
	if got.Phase != 5 {
		t.Errorf("Phase = %d, want %d", got.Phase, 5)
	}
	if len(got.Workers) != 2 {
		t.Fatalf("Workers = %d entries, want 2", len(got.Workers))
	}
	if got.Workers[0].Name != "Builder-1" {
		t.Errorf("Workers[0].Name = %q, want %q", got.Workers[0].Name, "Builder-1")
	}
	if got.Workers[0].Caste != "builder" {
		t.Errorf("Workers[0].Caste = %q, want %q", got.Workers[0].Caste, "builder")
	}
	if got.Workers[0].Status != "done" {
		t.Errorf("Workers[0].Status = %q, want %q", got.Workers[0].Status, "done")
	}
	// FilesTouched should aggregate all worker files
	expectedFiles := []string{"cmd/main.go", "pkg/learn/learn.go", "pkg/learn/evidence.go"}
	if len(got.FilesTouched) != len(expectedFiles) {
		t.Errorf("FilesTouched = %v (len %d), want %v (len %d)",
			got.FilesTouched, len(got.FilesTouched), expectedFiles, len(expectedFiles))
	}
	if got.GatesPassed != 3 {
		t.Errorf("GatesPassed = %d, want %d", got.GatesPassed, 3)
	}
	if got.GatesTotal != 3 {
		t.Errorf("GatesTotal = %d, want %d", got.GatesTotal, 3)
	}
	if got.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
	if got.Scope != "repo-local" {
		t.Errorf("Scope = %q, want %q", got.Scope, "repo-local")
	}
}

// TestCollectEvidence_ConfidenceComputed verifies confidence is computed via
// memory.Calculate trust scoring.
func TestCollectEvidence_ConfidenceComputed(t *testing.T) {
	workers := []WorkerResult{
		{Name: "Builder-1", Caste: "builder", Status: "done", FilesTouched: []string{"main.go"}},
	}
	gates := GateResult{Passed: 2, Total: 2}

	got := CollectEvidence("run-1", 1, workers, gates, "")

	if got.Confidence <= 0 {
		t.Errorf("Confidence = %f, want > 0 (computed via trust scoring)", got.Confidence)
	}
	// Fresh (DaysSince=0) with build_success/test_verified should give high score
	if got.Confidence < 0.8 {
		t.Errorf("Confidence = %f, want >= 0.8 for fresh build_success with test_verified", got.Confidence)
	}
}

// TestCollectEvidence_ScopeRepoLocal verifies default scope is "repo-local".
func TestCollectEvidence_ScopeRepoLocal(t *testing.T) {
	workers := []WorkerResult{
		{Name: "Builder-1", Caste: "builder", Status: "done"},
	}
	gates := GateResult{Passed: 1, Total: 1}

	// Empty scope should default to "repo-local"
	got := CollectEvidence("run-1", 1, workers, gates, "")
	if got.Scope != "repo-local" {
		t.Errorf("Scope = %q, want %q (default)", got.Scope, "repo-local")
	}

	// Explicit scope should be preserved
	got2 := CollectEvidence("run-2", 1, workers, gates, "hive-shareable")
	if got2.Scope != "hive-shareable" {
		t.Errorf("Scope = %q, want %q", got2.Scope, "hive-shareable")
	}
}

// TestIsGeneric tests the generic heuristic for hive-shareable detection.
func TestIsGeneric(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"plain text", "Use table-driven tests for better readability", true},
		{"generic advice", "Always handle errors before proceeding", true},
		{"empty string", "", true},
		{"single word", "Refactor", true},
		{"content with slash", "See cmd/aether/main.go for details", false},
		{"content with forward slash", "path/to/file", false},
		{"content with .go extension", "Update handler.go", false},
		{"content with .ts extension", "Fix component.ts", false},
		{"content with .json extension", "Read config.json", false},
		{"content with .md extension", "Edit README.md", false},
		{"content with short extension .c", "File.c", false},
		{"content with long extension .typescript", "File.typescript", true},
		{"content with slash and extension", "cmd/main.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsGeneric(tt.content)
			if got != tt.want {
				t.Errorf("IsGeneric(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}
