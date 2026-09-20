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
	// D-02 (Phase 196) pinned the routine roles to the cheaper model, so
	// they follow the sonnet slot's override like every other sonnet role.
	// This assertion previously expected "session" — the leftover-model
	// behaviour D-02 abolished; see TestRoutineBuilderIsSonnetNeverInherit.
	if got := resolveCasteModel("chronicler"); got != "glm-5-turbo" {
		t.Fatalf("resolveCasteModel(chronicler) = %q, want the env override glm-5-turbo — it is pinned to the sonnet slot", got)
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

// agentModelLines reads the `model:` frontmatter line out of every
// `.claude/agents/ant/aether-*.md` file and returns it keyed by caste. This
// is the source that ACTUALLY routes models (the platform reads the agent
// file; the Go table is display only), so the D-02 tests below assert
// against it as well as against the resolved value — a pin that exists only
// in the display table would be a pin that does not route.
func agentModelLines(t *testing.T) map[string]string {
	t.Helper()
	agentDir := filepath.Join("..", ".claude", "agents", "ant")
	entries, err := os.ReadDir(agentDir)
	if err != nil {
		t.Fatalf("read agent dir: %v", err)
	}
	models := map[string]string{}
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
		for _, line := range strings.Split(string(raw), "\n") {
			if strings.HasPrefix(line, "model:") {
				models[strings.ReplaceAll(role, "-", "_")] = strings.TrimSpace(strings.TrimPrefix(line, "model:"))
				break
			}
		}
	}
	if len(models) < 20 {
		t.Fatalf("only %d agent file(s) declared a model — an enumeration that finds nothing passes forever", len(models))
	}
	return models
}

// TestRoutineBuilderIsSonnetNeverInherit is D-02's first half (196-CONTEXT.md).
// The documentation writer, the knowledge-keeper and the accessibility
// checker are pinned to the cheaper model, and NO role anywhere is left on
// the sentinel that means "whatever model happened to run last" — an
// unpinned role's cost is both unpredictable and unattributable.
//
// It asserts the RESOLVED model, not merely the text of a config line: a pin
// that the resolver does not honour is not a pin.
func TestRoutineBuilderIsSonnetNeverInherit(t *testing.T) {
	t.Setenv("ANTHROPIC_DEFAULT_SONNET_MODEL", "")
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "")

	agentModels := agentModelLines(t)

	// Named by what they do, the way the owner reads them — the role word is
	// only the lookup key.
	routine := []struct{ caste, plain string }{
		{"chronicler", "the documentation writer"},
		{"keeper", "the knowledge-keeper"},
		{"includer", "the accessibility checker"},
	}
	for _, role := range routine {
		if got := resolveCasteModel(role.caste); got != "sonnet" {
			t.Errorf("%s (%s) resolves to %q, want sonnet — D-02 pins the routine roles to the cheaper model", role.caste, role.plain, got)
		}
		if got := agentModels[role.caste]; got != "sonnet" {
			t.Errorf("%s (%s) declares model %q in its own agent file, want sonnet — a pin that lives only in the display table does not route", role.caste, role.plain, got)
		}
	}

	// The sentinel is gone everywhere, in both the routing source and the
	// display table, and nothing resolves to the session's model.
	for caste, model := range agentModels {
		if model == "inherit" {
			t.Errorf("%s still declares model: inherit — no role may run on whatever model happened to run last (D-02)", caste)
		}
	}
	for caste, slot := range casteModelSlot {
		if slot == "inherit" {
			t.Errorf("casteModelSlot[%q] is still \"inherit\" — no role may run on whatever model happened to run last (D-02)", caste)
		}
		if got := resolveCasteModel(caste); got == "session" {
			t.Errorf("%s resolves to %q — that is the session's leftover model, which D-02 forbids", caste, got)
		}
	}
}

// casteModelReasonForbiddenWords is the vocabulary a reason may not contain.
// Two sources, both deliberate:
//
//  1. Words this repository invented (CLAUDE.md's own translation table).
//     They mean nothing to the person reading the card.
//  2. Every role's own lookup word. A reason that says "the auditor checks
//     quality" explains nothing to someone who does not already know what
//     that word was chosen to mean — it names the role instead of saying
//     what it does, which is the exact failure this list exists to catch.
//
// Matched as whole words after lowercasing, never as substrings: "ant" is a
// forbidden word but "important" is an ordinary one.
func casteModelReasonForbiddenWords() map[string]bool {
	forbidden := map[string]bool{}
	for _, word := range []string{
		"colony", "colonies", "colonize", "caste", "castes", "pheromone", "pheromones",
		"instinct", "instincts", "hive", "midden", "entomb", "entombed", "seal", "sealed",
		"ratchet", "allowlist", "wrapper", "wrappers", "playbook", "playbooks", "orphan",
		"orphans", "nest", "brood", "anthill", "ant", "ants", "swarm", "forage",
	} {
		forbidden[word] = true
	}
	for caste := range casteModelSlot {
		for _, part := range strings.Split(caste, "_") {
			forbidden[part] = true
		}
	}
	return forbidden
}

func casteModelReasonWords(reason string) []string {
	return strings.FieldsFunc(strings.ToLower(reason), func(r rune) bool {
		return r < 'a' || r > 'z'
	})
}

// TestOpusRequiresRecordedReason is D-02's second half: any role kept on the
// expensive model carries a WRITTEN reason. The table is checked in both
// directions, so it can go stale in neither: an expensive role with no
// reason fails by name, and a reason recorded for a role that is not on the
// expensive model fails by name too.
//
// The reason is owner-facing text, so it is held to CLAUDE.md's rule for
// owner-facing text: plain English, no vocabulary this repository invented,
// no file paths, and it must say what a cheaper model would get wrong —
// otherwise it justifies nothing.
func TestOpusRequiresRecordedReason(t *testing.T) {
	expensive := map[string]bool{}
	for caste, slot := range casteModelSlot {
		if slot == "opus" {
			expensive[caste] = true
		}
	}
	if len(expensive) == 0 {
		t.Fatal("no role is on the expensive model — this test would then assert nothing at all")
	}

	for caste := range expensive {
		reason := casteModelReason(caste)
		if strings.TrimSpace(reason) == "" {
			t.Errorf("%s runs on the expensive model but records no reason why — D-02 requires a written one", caste)
			continue
		}
		if len(reason) < 60 {
			t.Errorf("%s: the recorded reason is too short to justify anything: %q", caste, reason)
		}
		lower := strings.ToLower(reason)
		for _, placeholder := range []string{"todo", "tbd", "fixme", "placeholder", "n/a", "xxx"} {
			if strings.Contains(lower, placeholder) {
				t.Errorf("%s: the recorded reason is a placeholder, not a reason: %q", caste, reason)
			}
		}
		if !strings.Contains(lower, "cheaper") {
			t.Errorf("%s: the reason never says what a cheaper model would get wrong, so it does not justify the expense: %q", caste, reason)
		}
		for _, path := range []string{"/", ".go", ".md", ".json"} {
			if strings.Contains(reason, path) {
				t.Errorf("%s: the reason contains a file path (%q), which the owner cannot read as an explanation: %q", caste, path, reason)
			}
		}
		forbidden := casteModelReasonForbiddenWords()
		for _, word := range casteModelReasonWords(reason) {
			if forbidden[word] {
				t.Errorf("%s: the reason uses %q, a word this repository invented or a role's own name — it explains nothing to someone who has never opened a file here: %q", caste, word, reason)
			}
		}
	}

	// The other direction: no reason may survive its role moving off the
	// expensive model, and no reason may exist for a role that never was.
	for caste := range casteModelReasons {
		if !expensive[caste] {
			t.Errorf("a reason is recorded for %q, which does not run on the expensive model — a stale justification is worse than none", caste)
		}
	}
}
