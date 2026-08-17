package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// WS3 — worker model visibility. Display only: the platform routes agents
// natively from `.claude/agents/ant/*.md` frontmatter, and automatic model
// selection stays rejected (2026-07-28). These tests pin (1) the tag on the
// spawn moment, (2) env-var resolution, and (3) that the static display
// table can never drift from the agent frontmatter that actually routes.

func TestSpawnLineCarriesModelTag(t *testing.T) {
	t.Setenv("ANTHROPIC_DEFAULT_SONNET_MODEL", "")
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "")

	identity := casteIdentityWithModel("builder")
	if !strings.HasSuffix(identity, "[sonnet]") {
		t.Fatalf("builder spawn identity does not end with its model tag: %q", identity)
	}
	// The house style stays intact underneath the tag.
	if !strings.HasPrefix(identity, "🔨🐜 ") {
		t.Fatalf("model tag broke the caste identity house style: %q", identity)
	}

	// Wiring: the ceremony dispatch line (the wave-start spawn roster the
	// user actually watches) carries the tag.
	var b strings.Builder
	writeCeremonyDispatchLine(&b, ceremonyDispatch{Caste: "builder", Name: "Mason-67", Task: "lay bricks"}, "  ")
	if !strings.Contains(b.String(), "[sonnet]") {
		t.Fatalf("ceremony dispatch line lost the model tag: %q", b.String())
	}

	// A caste with no agent definition gets no tag — never a fabricated one.
	if got := casteIdentityWithModel("dreamer"); strings.Contains(got, "[") {
		t.Fatalf("caste without an agent file got a model tag: %q", got)
	}
}

func TestModelTagEnvOverride(t *testing.T) {
	t.Setenv("ANTHROPIC_DEFAULT_SONNET_MODEL", "glm-5-turbo")
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "")

	if got := resolveCasteModel("builder"); got != "glm-5-turbo" {
		t.Fatalf("resolveCasteModel(builder) = %q, want the env override glm-5-turbo", got)
	}
	// The opus slot is untouched by the sonnet override.
	if got := resolveCasteModel("queen"); got != "opus" {
		t.Fatalf("resolveCasteModel(queen) = %q, want opus", got)
	}
	// inherit agents run on the session's model.
	if got := resolveCasteModel("chronicler"); got != "session" {
		t.Fatalf("resolveCasteModel(chronicler) = %q, want session", got)
	}
	if got := resolveCasteModel("unknown-caste"); got != "" {
		t.Fatalf("resolveCasteModel(unknown) = %q, want empty", got)
	}
}

// TestCasteModelSlotMatchesAgentFrontmatter derives the caste→slot mapping
// from the agent files that actually route models and asserts the display
// table matches in BOTH directions. Changing one `model:` line without
// updating the table (or vice versa) fails here — the table cannot go stale
// silently.
func TestCasteModelSlotMatchesAgentFrontmatter(t *testing.T) {
	agentDir := filepath.Join("..", ".claude", "agents", "ant")
	entries, err := os.ReadDir(agentDir)
	if err != nil {
		t.Fatalf("read agent dir: %v", err)
	}

	derived := map[string]string{}
	surveyorSlots := map[string]string{}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "aether-") || !strings.HasSuffix(name, ".md") {
			continue
		}
		role := strings.TrimSuffix(strings.TrimPrefix(name, "aether-"), ".md")
		raw, err := os.ReadFile(filepath.Join(agentDir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		model := ""
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.HasPrefix(line, "model:") {
				model = strings.TrimSpace(strings.TrimPrefix(line, "model:"))
				break
			}
		}
		if model == "" {
			t.Fatalf("%s declares no model: frontmatter", name)
		}
		if strings.HasPrefix(role, "surveyor-") {
			surveyorSlots[role] = model
			continue
		}
		derived[strings.ReplaceAll(role, "-", "_")] = model
	}
	// The four surveyor agents collapse to the single "surveyor" caste key;
	// they must agree on a slot for that collapse to be honest.
	first := ""
	for role, slot := range surveyorSlots {
		if first == "" {
			first = slot
			continue
		}
		if slot != first {
			t.Fatalf("surveyor agents disagree on model (%s says %s, another says %s) — the shared surveyor tag would lie", role, slot, first)
		}
	}
	if first != "" {
		derived["surveyor"] = first
	}

	for caste, want := range derived {
		got, ok := casteModelSlot[caste]
		if !ok {
			t.Errorf("agent file declares caste %q (model %s) but casteModelSlot has no entry", caste, want)
			continue
		}
		if got != want {
			t.Errorf("casteModelSlot[%q] = %q but the agent frontmatter says %q — update the table", caste, got, want)
		}
	}
	for caste := range casteModelSlot {
		if _, ok := derived[caste]; !ok {
			t.Errorf("casteModelSlot has %q but no agent file declares it — remove the entry", caste)
		}
	}
}
