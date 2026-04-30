package cmd

import (
	"strings"
	"testing"
)

// TestRecoveryExcludesFailedCommand verifies that recovery options never
// suggest re-running the command that just failed (LOOP-05 guarantee).
func TestRecoveryExcludesFailedCommand(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		errorMsg  string
		forbidden []string
	}{
		{
			name:      "seal failure never suggests seal",
			command:   "seal",
			errorMsg:  "no colony initialized",
			forbidden: []string{"seal", "aether seal"},
		},
		{
			name:      "entomb failure never suggests entomb",
			command:   "entomb",
			errorMsg:  "no colony initialized",
			forbidden: []string{"entomb", "aether entomb"},
		},
		{
			name:      "status failure never suggests status",
			command:   "status",
			errorMsg:  "failed to load colony state",
			forbidden: []string{"status", "aether status"},
		},
		{
			name:      "resume failure never suggests resume or resume-colony",
			command:   "resume",
			errorMsg:  "failed to restore runnable colony state",
			forbidden: []string{"resume", "aether resume", "aether resume-colony"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := recoveryOptionsForCommand(tt.command, tt.errorMsg)

			if len(options) == 0 {
				t.Fatal("expected at least 1 recovery option, got 0")
			}

			for _, opt := range options {
				optLower := strings.ToLower(opt.Command)
				for _, forbidden := range tt.forbidden {
					if strings.Contains(optLower, strings.ToLower(forbidden)) {
						t.Errorf("recovery option %q contains forbidden command %q", opt.Command, forbidden)
					}
				}
			}
		})
	}
}

// TestRecoveryNormalizesCommand verifies command normalization strips prefixes and flags.
func TestRecoveryNormalizesCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"aether seal --force", "seal"},
		{"aether resume-colony", "resume"},
		{"seal", "seal"},
		{"entomb", "entomb"},
		{"aether status", "status"},
		{"aether resume", "resume"},
		{"resume-colony", "resume"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeBaseCommand(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeBaseCommand(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// TestClassifyError verifies error message classification.
func TestClassifyError(t *testing.T) {
	tests := []struct {
		errorMsg string
		expected string
	}{
		{"no colony initialized", "no_colony"},
		{"No Colony Initialized", "no_colony"},
		{"failed to load colony state", "state_corruption"},
		{"json: unexpected end", "state_corruption"},
		{"no project plan", "missing_prerequisite"},
		{"Colony has not been sealed", "missing_prerequisite"},
		{"CROWNED-ANTHILL.md not found", "missing_prerequisite"},
		{"permission denied", "permission_denied"},
		{"something unexpected happened", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.errorMsg, func(t *testing.T) {
			got := classifyError(tt.errorMsg)
			if got != tt.expected {
				t.Errorf("classifyError(%q) = %q, want %q", tt.errorMsg, got, tt.expected)
			}
		})
	}
}

// TestRenderRecoveryMenu verifies the recovery menu renders correctly.
func TestRenderRecoveryMenu(t *testing.T) {
	result := renderRecoveryMenu("seal", "no colony initialized", nil)

	if result == "" {
		t.Fatal("renderRecoveryMenu returned empty string")
	}

	// Must contain "Recovery" in the banner
	if !strings.Contains(result, "Recovery") {
		t.Error("renderRecoveryMenu output should contain 'Recovery' in the banner")
	}

	// Must contain a numbered list (at least "1.")
	if !strings.Contains(result, "1.") {
		t.Error("renderRecoveryMenu output should contain numbered options")
	}
}

// TestRecoveryIncludesExpectedOptions verifies recovery options include expected commands.
func TestRecoveryIncludesExpectedOptions(t *testing.T) {
	t.Run("seal no colony includes init", func(t *testing.T) {
		options := recoveryOptionsForCommand("seal", "no colony initialized")
		found := false
		for _, opt := range options {
			if strings.Contains(opt.Command, "aether init") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected 'aether init' in recovery options for seal with no colony")
		}
	})

	t.Run("seal state corruption includes patrol", func(t *testing.T) {
		options := recoveryOptionsForCommand("seal", "failed to load colony state")
		found := false
		for _, opt := range options {
			if strings.Contains(opt.Command, "aether patrol") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected 'aether patrol' in recovery options for seal with state corruption")
		}
	})
}

// TestRecoveryMinimumOptions verifies at least 2 options for every lifecycle command.
func TestRecoveryMinimumOptions(t *testing.T) {
	lifecycleCommands := []string{"seal", "entomb", "status", "resume"}
	errorMessages := []string{
		"no colony initialized",
		"failed to load colony state",
		"no project plan",
		"Colony has not been sealed",
		"permission denied",
		"something unknown went wrong",
	}

	for _, cmd := range lifecycleCommands {
		for _, errMsg := range errorMessages {
			name := cmd + " / " + errMsg
			t.Run(name, func(t *testing.T) {
				options := recoveryOptionsForCommand(cmd, errMsg)
				if len(options) < 2 {
					t.Errorf("expected at least 2 recovery options for %s with error %q, got %d", cmd, errMsg, len(options))
				}
			})
		}
	}
}

// TestRecoveryExcludesFlagVariants verifies that command flag variants are excluded.
func TestRecoveryExcludesFlagVariants(t *testing.T) {
	// "aether seal --force" should be treated as "seal" and excluded when seal fails
	options := recoveryOptionsForCommand("seal", "no colony initialized")
	for _, opt := range options {
		if strings.Contains(strings.ToLower(opt.Command), "seal") {
			t.Errorf("recovery option %q should not contain 'seal' when seal command failed", opt.Command)
		}
	}
}
