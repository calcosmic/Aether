package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const currentVocabulary199Path = "cmd/testdata/current-vocabulary-199.json"

type currentVocabulary199Document struct {
	Version string                       `json:"version"`
	Buckets []currentVocabulary199Bucket `json:"buckets"`
}

type currentVocabulary199Bucket struct {
	Name    string                      `json:"name"`
	Entries []currentVocabulary199Entry `json:"entries"`
}

type currentVocabulary199Entry struct {
	Path          string `json:"path"`
	Family        string `json:"family"`
	Count         int    `json:"count"`
	Locator       string `json:"locator"`
	Purpose       string `json:"purpose"`
	Owner         string `json:"owner"`
	Host          string `json:"host"`
	GeneratedFrom string `json:"generated_from"`
	AllowedRoute  string `json:"allowed_route"`
	Expiry        string `json:"expiry"`
}

func TestCurrentVocabulary199(t *testing.T) {
	root := findTestModuleRoot(t)
	document := loadCurrentVocabulary199(t, root)
	entries := flattenCurrentVocabulary199(t, document)

	t.Run("four-exclusive-buckets", func(t *testing.T) {
		want := map[string]bool{
			"active_current_documentation_template_generated":  true,
			"active_runtime_state_routing":                     true,
			"hidden_parser_compatibility":                      true,
			"historical_evidence_versioned_fixture_test_input": true,
		}
		if len(document.Buckets) != len(want) {
			t.Fatalf("bucket count = %d, want %d", len(document.Buckets), len(want))
		}
		for _, bucket := range document.Buckets {
			if !want[bucket.Name] {
				t.Fatalf("unexpected bucket %q", bucket.Name)
			}
			delete(want, bucket.Name)
		}
		if len(want) != 0 {
			t.Fatalf("missing buckets: %v", mapsKeysCurrentVocabulary199(want))
		}
		validateCurrentVocabulary199Entries(t, document, entries)
	})

	t.Run("tracked-occurrences-are-exhaustively-classified", func(t *testing.T) {
		actual := trackedVocabularyOccurrences199(t, root)
		inventory := map[string]int{}
		for _, entry := range entries {
			if entry.Family == "seal_status_entomb_init" {
				continue
			}
			key := entry.Path + "\x00" + entry.Family
			inventory[key] += entry.Count
		}
		if len(actual) != len(inventory) {
			t.Fatalf("tracked occurrence keys = %d, inventory keys = %d; inventory must classify every exact tracked path and family", len(actual), len(inventory))
		}
		for key, count := range actual {
			if inventory[key] != count {
				t.Errorf("%q count = %d, inventory = %d", key, count, inventory[key])
			}
		}
		for key, count := range inventory {
			if actual[key] != count {
				t.Errorf("inventory %q count = %d, tracked bytes = %d", key, count, actual[key])
			}
		}
	})

	t.Run("required-current-paths-are-independently-seeded", func(t *testing.T) {
		for _, path := range requiredCurrentVocabularyPaths199() {
			found := false
			for _, entry := range entries {
				if entry.Path == path && (entry.Family == "seal_status_entomb_init" || entry.Family == "current_surface") {
					found = true
				}
			}
			if !found {
				t.Errorf("required current path %s has no exact inventory row", path)
			}
		}
	})

	t.Run("hidden-compatibility-is-exact-and-expiring", func(t *testing.T) {
		for _, entry := range entries {
			if entry.Family != "legacy_pause" && entry.Family != "legacy_resume" {
				continue
			}
			if entry.Path != "cmd/normalize_args.go" || entry.Expiry != "1.29" || entry.AllowedRoute != "input-only-to-canonical-pause-or-resume" {
				t.Errorf("legacy entry must be exact expiring parser compatibility: %+v", entry)
			}
		}
		content, err := os.ReadFile(filepath.Join(root, "cmd", "normalize_args.go"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(content), `case "pause`+`-colony":`) || !strings.Contains(string(content), `case "resume`+`-colony":`) || !strings.Contains(string(content), `legacySessionRedirectExpiry = "1.29.0"`) {
			t.Fatal("normalizer no longer contains the bounded exact-token 1.29 compatibility boundary")
		}
	})

	t.Run("current-sources-never-teach-retired-command-shapes", func(t *testing.T) {
		for _, entry := range entries {
			if entry.Family != "current_surface" && entry.Family != "seal_status_entomb_init" {
				continue
			}
			content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(entry.Path)))
			if err != nil {
				t.Fatal(err)
			}
			for _, retired := range currentVocabularyRetiredTokens199() {
				if strings.Contains(string(content), retired) {
					t.Errorf("current surface %s teaches retired command shape %q", entry.Path, retired)
				}
			}
		}
	})

	t.Run("executed-runtime-routes-and-compatibility", func(t *testing.T) {
		for name, output := range collectRuntimeRecoverySuggestions199(t) {
			if !strings.Contains(output, "aether resume") || strings.Contains(output, "aether "+"recover") {
				t.Errorf("%s does not retain only the canonical restoration route: %s", name, output)
			}
		}
		command := exec.Command("go", "test", "./cmd", "-run", "^TestRuntimeRecoveryCompatibility199$", "-count=1")
		command.Dir = root
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("Plan 34 compatibility proof failed: %v\n%s", err, output)
		}
	})

	t.Run("d17-status-primary-generated-parity", func(t *testing.T) {
		guides := []d17Guide199{
			{path: "README.md", host: "README", seal: "/ant-seal", status: "/ant-status", entomb: "/ant-entomb", init: "/ant-init"},
			{path: "AGENTS.md", host: "root guide", seal: "aether seal", status: "aether status", entomb: "aether entomb", init: "aether init"},
			{path: ".aether/rules/aether-colony.md", host: "canonical colony rule", seal: "/ant-seal", status: "/ant-status", entomb: "/ant-entomb", init: "/ant-init"},
			{path: ".claude/rules/aether-colony.md", host: "generated colony rule", seal: "/ant-seal", status: "/ant-status", entomb: "/ant-entomb", init: "/ant-init"},
			{path: ".aether/templates/opencode-md-template.md", host: "canonical OpenCode template", seal: "/ant-seal", status: "/ant-status", entomb: "/ant-entomb", init: "/ant-init"},
			{path: "cmd/.opencode/OPENCODE.md", host: "generated OpenCode guide", seal: "/ant-seal", status: "/ant-status", entomb: "/ant-entomb", init: "/ant-init"},
			{path: ".aether/templates/agents-md-template.md", host: "canonical Codex template", seal: "aether seal", status: "aether status", entomb: "aether entomb", init: "aether init"},
			{path: "cmd/AGENTS.md", host: "generated Codex guide", seal: "aether seal", status: "aether status", entomb: "aether entomb", init: "aether init"},
		}
		for _, guide := range guides {
			section := readD17GuideSection199(t, root, guide)
			if !d17StatusPrimary199(section, guide) || !d17EntombOptional199(section) || !d17InitAfterArchiveClear199(section, guide) {
				t.Errorf("%s violates D-17 status-first optional-entomb archive-clear-before-init ordering", guide.path)
			}
		}
	})
}

func loadCurrentVocabulary199(t *testing.T, root string) currentVocabulary199Document {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(currentVocabulary199Path)))
	if err != nil {
		t.Fatalf("read inventory: %v", err)
	}
	var document currentVocabulary199Document
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("decode inventory: %v", err)
	}
	return document
}

func flattenCurrentVocabulary199(t *testing.T, document currentVocabulary199Document) []currentVocabulary199Entry {
	t.Helper()
	entries := []currentVocabulary199Entry{}
	seen := map[string]bool{}
	for _, bucket := range document.Buckets {
		for _, entry := range bucket.Entries {
			key := bucket.Name + "\x00" + entry.Path + "\x00" + entry.Family
			if seen[key] {
				t.Fatalf("duplicate inventory entry %q", key)
			}
			seen[key] = true
			entries = append(entries, entry)
		}
	}
	return entries
}

func validateCurrentVocabulary199Entries(t *testing.T, document currentVocabulary199Document, entries []currentVocabulary199Entry) {
	t.Helper()
	for _, entry := range entries {
		for field, value := range map[string]string{"path": entry.Path, "family": entry.Family, "locator": entry.Locator, "purpose": entry.Purpose, "host": entry.Host, "allowed_route": entry.AllowedRoute} {
			if strings.TrimSpace(value) == "" || strings.ContainsAny(value, "*") || strings.Contains(value, "other occurrence") {
				t.Errorf("inventory has non-narrow %s in %+v", field, entry)
			}
		}
		if entry.Count < 0 || strings.Contains(entry.Path, "*") || strings.HasSuffix(entry.Path, "/") {
			t.Errorf("inventory has invalid exact path/count: %+v", entry)
		}
	}
}

func trackedVocabularyOccurrences199(t *testing.T, root string) map[string]int {
	t.Helper()
	command := exec.Command("git", "ls-files", "-z")
	command.Dir = root
	raw, err := command.Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}
	occurrences := map[string]int{}
	for _, path := range bytes.Split(bytes.TrimSuffix(raw, []byte{0}), []byte{0}) {
		if len(path) == 0 {
			continue
		}
		content, readErr := os.ReadFile(filepath.Join(root, filepath.FromSlash(string(path))))
		if readErr != nil {
			t.Fatalf("read tracked %s: %v", path, readErr)
		}
		for family, token := range currentVocabularyFamilies199() {
			if count := bytes.Count(content, []byte(token)); count > 0 {
				occurrences[string(path)+"\x00"+family] = count
			}
		}
	}
	return occurrences
}

func currentVocabularyFamilies199() map[string]string {
	return map[string]string{
		"legacy_pause":        "pause" + "-colony",
		"legacy_resume":       "resume" + "-colony",
		"retired_recover":     "aether " + "recover",
		"retired_ant_recover": "/ant-" + "recover",
		"retired_apply":       "recover " + "--apply",
	}
}

func currentVocabularyRetiredTokens199() []string {
	families := currentVocabularyFamilies199()
	return []string{families["legacy_pause"], families["legacy_resume"], families["retired_recover"], families["retired_ant_recover"], families["retired_apply"]}
}

func requiredCurrentVocabularyPaths199() []string {
	return []string{
		".aether/rules/aether-colony.md", ".claude/rules/aether-colony.md", ".aether/docs/command-playbooks/continue-full.md", ".aether/docs/command-playbooks/continue-finalize.md",
		".aether/commands/organize.yaml", ".claude/commands/ant-organize.md", ".claude/commands/ant/organize.md", ".opencode/commands/ant/organize.md",
		".aether/skills/colony/context-management/SKILL.md", ".aether/templates/handoff-build-success.template.md", ".aether/templates/opencode-md-template.md", "cmd/recovery_snapshot.go", ".aether/CONTEXT.md",
		".aether/commands/help.yaml", ".claude/commands/ant-help.md", ".claude/commands/ant/help.md", ".opencode/commands/ant/help.md", ".aether/commands/pause.yaml", ".aether/commands/resume.yaml",
		".claude/commands/ant-pause.md", ".claude/commands/ant/pause.md", ".opencode/commands/ant/pause.md", ".claude/commands/ant-resume.md", ".claude/commands/ant/resume.md", ".opencode/commands/ant/resume.md",
		".aether/skills/colony/colony-lifecycle/SKILL.md", "cmd/command_guide.go", "README.md", "AGENTS.md", "cmd/.opencode/OPENCODE.md", "docs/phase3-section-commands.md",
		"bench/RUNBOOK.md", "bench/harness/permitted-inputs.md", "bench/tasks/03-interrupted-execution.md", "cmd/codex_visuals.go", "cmd/init_cmd.go", "cmd/entomb_cmd.go", "cmd/clash.go", "cmd/worktree_safety.go", "cmd/codex_build_worktree.go", "cmd/worktree_reap.go",
		".aether/docs/PARITY_CLASSIC_VS_GO.md", "cmd/hook_cmds.go", "cmd/recovery_engine.go", "cmd/recover_visuals.go", "cmd/recover_repair.go", "cmd/worktree.go", ".aether/templates/agents-md-template.md", "cmd/AGENTS.md",
	}
}

func mapsKeysCurrentVocabulary199(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

var _ = fmt.Sprintf
