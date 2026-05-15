package smoke

import (
	"testing"
)

func TestParseYAMLCommands(t *testing.T) {
	cmds, err := ParseYAMLCommands("../../.aether/commands/*.yaml")
	if err != nil {
		t.Fatalf("ParseYAMLCommands failed: %v", err)
	}

	if len(cmds) < 60 {
		t.Errorf("expected at least 60 commands, got %d", len(cmds))
	}

	// Check --help is extracted for at least 50 commands
	helpCount := 0
	for _, cmd := range cmds {
		for _, f := range cmd.Flags {
			if f == "--help" {
				helpCount++
				break
			}
		}
	}
	// --help is excluded from extraction, so we expect 0
	if helpCount != 0 {
		t.Errorf("expected --help to be excluded from all commands, found %d", helpCount)
	}

	// Check no duplicate flags within a single command
	for _, cmd := range cmds {
		seen := make(map[string]bool)
		for _, f := range cmd.Flags {
			if seen[f] {
				t.Errorf("duplicate flag %q in command %q", f, cmd.Name)
			}
			seen[f] = true
		}
	}

	// Check IsHost is true for exactly 6 commands (lifecycle.yaml does not exist)
	hostCount := 0
	for _, cmd := range cmds {
		if cmd.IsHost {
			hostCount++
		}
	}
	if hostCount != 6 {
		t.Errorf("expected exactly 6 host commands, got %d", hostCount)
	}

	// Verify specific host commands exist
	hostNames := map[string]bool{
		"ant-plan":     false,
		"ant-build":    false,
		"ant-continue": false,
		"ant-oracle":   false,
		"ant-watch":    false,
		"ant-swarm":    false,
	}
	for _, cmd := range cmds {
		if _, ok := hostNames[cmd.Name]; ok {
			hostNames[cmd.Name] = true
		}
	}
	for name, found := range hostNames {
		if !found {
			t.Errorf("expected host command %q to be found", name)
		}
	}
}

func TestExtractFlags(t *testing.T) {
	tests := []struct {
		name     string
		texts    []string
		expected []string
	}{
		{
			name:     "simple flag",
			texts:    []string{"run `aether build --depth balanced`"},
			expected: []string{"--depth"},
		},
		{
			name:     "flag in prose only (not standalone)",
			texts:    []string{"The --depth parameter controls granularity"},
			expected: []string{"--depth"},
		},
		{
			name:     "embedded flag (not standalone)",
			texts:    []string{"pre-depth-post"},
			expected: nil,
		},
		{
			name:     "multiple flags",
			texts:    []string{"`aether plan --depth balanced --planning-depth standard`"},
			expected: []string{"--depth", "--planning-depth"},
		},
		{
			name:     "help excluded",
			texts:    []string{"run `aether --help` or `aether build --help`"},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFlags(tt.texts)
			if len(got) != len(tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, got)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("expected %v, got %v", tt.expected, got)
					return
				}
			}
		})
	}
}

func TestIsStandaloneToken(t *testing.T) {
	tests := []struct {
		text string
		flag string
		want bool
	}{
		{"run --depth balanced", "--depth", true},
		{"pre-depth-post", "--depth", false},
		{"`--depth`", "--depth", true},
		{"(--depth)", "--depth", true},
		{"[--depth]", "--depth", true},
		{"--depth.", "--depth", true},
		{"--depth,", "--depth", true},
		{"\"--depth\"", "--depth", true},
		{"'--depth'", "--depth", true},
		{"<--depth>", "--depth", true},
		{"|--depth|", "--depth", true},
		{"--depth;", "--depth", true},
		{"--depth:", "--depth", true},
		{"--depth!", "--depth", true},
		{"--depth?", "--depth", true},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := isStandaloneToken(tt.text, tt.flag)
			if got != tt.want {
				t.Errorf("isStandaloneToken(%q, %q) = %v, want %v", tt.text, tt.flag, got, tt.want)
			}
		})
	}
}
