package llm

import (
	"testing"
)

func TestParseAgentSpec(t *testing.T) {
	content := []byte(`---
name: aether-builder
description: "Build things"
tools: [Read, Write, Edit, Bash, Grep, Glob]
color: yellow
model: sonnet
---
Some body content here.
`)

	cfg, err := ParseAgentSpec(content)
	if err != nil {
		t.Fatalf("ParseAgentSpec() returned unexpected error: %v", err)
	}

	if cfg.Name != "aether-builder" {
		t.Errorf("Name = %q, want %q", cfg.Name, "aether-builder")
	}
	if cfg.Description != "Build things" {
		t.Errorf("Description = %q, want %q", cfg.Description, "Build things")
	}
	if cfg.Color != "yellow" {
		t.Errorf("Color = %q, want %q", cfg.Color, "yellow")
	}
	if cfg.Model != "sonnet" {
		t.Errorf("Model = %q, want %q", cfg.Model, "sonnet")
	}
	if len(cfg.Tools) != 6 {
		t.Fatalf("len(Tools) = %d, want 6", len(cfg.Tools))
	}
	expectedTools := []string{"Read", "Write", "Edit", "Bash", "Grep", "Glob"}
	for i, tool := range expectedTools {
		if cfg.Tools[i] != tool {
			t.Errorf("Tools[%d] = %q, want %q", i, cfg.Tools[i], tool)
		}
	}
}

func TestParseAgentSpecMinimal(t *testing.T) {
	content := []byte(`---
name: minimal-agent
---
body
`)

	cfg, err := ParseAgentSpec(content)
	if err != nil {
		t.Fatalf("ParseAgentSpec() returned unexpected error: %v", err)
	}

	if cfg.Name != "minimal-agent" {
		t.Errorf("Name = %q, want %q", cfg.Name, "minimal-agent")
	}
	if cfg.Description != "" {
		t.Errorf("Description = %q, want empty", cfg.Description)
	}
	if len(cfg.Tools) != 0 {
		t.Errorf("len(Tools) = %d, want 0", len(cfg.Tools))
	}
	if cfg.Color != "" {
		t.Errorf("Color = %q, want empty", cfg.Color)
	}
	if cfg.Model != "" {
		t.Errorf("Model = %q, want empty", cfg.Model)
	}
}

func TestParseAgentSpecMissingOpen(t *testing.T) {
	content := []byte(`name: no-delimiter
description: "Missing opening delimiter"
---
body
`)

	_, err := ParseAgentSpec(content)
	if err == nil {
		t.Fatal("ParseAgentSpec() should return error for missing opening ---")
	}
}

func TestParseAgentSpecMissingClose(t *testing.T) {
	content := []byte(`---
name: unclosed
description: "Missing closing delimiter"
`)

	_, err := ParseAgentSpec(content)
	if err == nil {
		t.Fatal("ParseAgentSpec() should return error for missing closing ---")
	}
}

func TestParseAgentSpecEmptyName(t *testing.T) {
	content := []byte(`---
name: ""
description: "Empty name"
---
body
`)

	_, err := ParseAgentSpec(content)
	if err == nil {
		t.Fatal("ParseAgentSpec() should return error for empty name")
	}
}

func TestParseAgentSpecInvalidYAML(t *testing.T) {
	content := []byte(`---
name: broken
description: [invalid yaml
---
body
`)

	_, err := ParseAgentSpec(content)
	if err == nil {
		t.Fatal("ParseAgentSpec() should return error for invalid YAML")
	}
}

func TestParseAgentSpecBodyWithDashes(t *testing.T) {
	content := []byte(`---
name: body-dashes
description: "Body contains ---"
---
This body has ---
multiple --- dashes
---
`)

	cfg, err := ParseAgentSpec(content)
	if err != nil {
		t.Fatalf("ParseAgentSpec() returned unexpected error: %v", err)
	}
	if cfg.Name != "body-dashes" {
		t.Errorf("Name = %q, want %q", cfg.Name, "body-dashes")
	}
}

func TestParseAgentSpecToolsList(t *testing.T) {
	content := []byte(`---
name: tool-agent
tools:
  - Read
  - Write
  - Bash
---
body
`)

	cfg, err := ParseAgentSpec(content)
	if err != nil {
		t.Fatalf("ParseAgentSpec() returned unexpected error: %v", err)
	}

	if len(cfg.Tools) != 3 {
		t.Fatalf("len(Tools) = %d, want 3", len(cfg.Tools))
	}
	expectedTools := []string{"Read", "Write", "Bash"}
	for i, tool := range expectedTools {
		if cfg.Tools[i] != tool {
			t.Errorf("Tools[%d] = %q, want %q", i, cfg.Tools[i], tool)
		}
	}
}

func TestParseAgentSpecTriggers(t *testing.T) {
	content := []byte(`---
name: triggered-agent
triggers:
  - topic: "learning.*"
    filter:
      confidence: 0.8
  - topic: "memory.*"
---
body
`)

	cfg, err := ParseAgentSpec(content)
	if err != nil {
		t.Fatalf("ParseAgentSpec() returned unexpected error: %v", err)
	}

	if len(cfg.Triggers) != 2 {
		t.Fatalf("len(Triggers) = %d, want 2", len(cfg.Triggers))
	}

	if cfg.Triggers[0].Topic != "learning.*" {
		t.Errorf("Triggers[0].Topic = %q, want %q", cfg.Triggers[0].Topic, "learning.*")
	}
	if cfg.Triggers[0].Filter == nil {
		t.Fatal("Triggers[0].Filter should not be nil")
	}
	if conf, ok := cfg.Triggers[0].Filter["confidence"]; !ok {
		t.Error("Triggers[0].Filter missing 'confidence' key")
	} else {
		// YAML unmarshals numbers as float64 by default
		if v, ok := conf.(float64); !ok || v != 0.8 {
			t.Errorf("Triggers[0].Filter[\"confidence\"] = %v, want 0.8", conf)
		}
	}

	if cfg.Triggers[1].Topic != "memory.*" {
		t.Errorf("Triggers[1].Topic = %q, want %q", cfg.Triggers[1].Topic, "memory.*")
	}
}

func TestParseAgentSpecLeadingWhitespace(t *testing.T) {
	content := []byte(`
---
name: whitespace-agent
---
body
`)

	cfg, err := ParseAgentSpec(content)
	if err != nil {
		t.Fatalf("ParseAgentSpec() returned unexpected error: %v", err)
	}
	if cfg.Name != "whitespace-agent" {
		t.Errorf("Name = %q, want %q", cfg.Name, "whitespace-agent")
	}
}
