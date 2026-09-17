package cmd

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestCodexNativeBuildGuidePlatformIsolation(t *testing.T) {
	baseline := map[string]commandGuideResult{}
	for _, platform := range []string{"codex", "claude", "opencode", "opencode", "codex", "claude", "codex"} {
		guide, err := buildCommandGuide("build", platform)
		if err != nil {
			t.Fatal(err)
		}
		if before, ok := baseline[platform]; ok && !reflect.DeepEqual(before, guide) {
			t.Fatalf("%s guide changed after another platform read", platform)
		}
		baseline[platform] = guide
		// Include every instruction-bearing field, including raw bypass text.
		all := strings.Join(append(append(append([]string{guide.Intent, guide.RunCommand, guide.RawBypass}, guide.PreSteps...), guide.PostSteps...), guide.DriftGuards...), "\n")
		if platform == "codex" {
			for _, required := range []string{"codex-native-worker reserve", "codex-native-worker bind", "codex-native-worker record", "codex-native-worker stage", "spawn_agent", "sole launcher"} {
				if !strings.Contains(all, required) {
					t.Errorf("Codex guide missing %q", required)
				}
			}
			if strings.Contains(all, "build-wave playbook") {
				t.Error("Codex guide still delegates authority to retired playbook")
			}
		} else {
			for _, forbidden := range []string{"codex-native-worker", "spawn_agent"} {
				if strings.Contains(all, forbidden) {
					t.Errorf("%s includes %s", platform, forbidden)
				}
			}
			if guide.SkillReference != "" || !strings.Contains(guide.PreSteps[0], "generated "+platform+" slash-command wrapper") {
				t.Errorf("%s lost wrapper entrypoint", platform)
			}
			for _, required := range []string{"visible live Task/subagent", "spawn-log", "spawn-complete", "build-completion-stage"} {
				if !strings.Contains(all, required) {
					t.Errorf("%s lost %q", platform, required)
				}
			}
			if guide.RunCommand != "AETHER_OUTPUT_MODE=json aether build-finalize <phase> --completion-file <Go-owned completion_path returned by build-completion-stage>" {
				t.Errorf("%s finalizer changed: %s", platform, guide.RunCommand)
			}
		}
	}
	// The adapter must not mutate caller-owned instruction slices either.
	def := commandGuideCatalog()["build"]
	before, _ := json.Marshal(def)
	_ = adaptCommandGuideDefinitionForPlatform("build", "codex", def)
	after, _ := json.Marshal(def)
	if string(before) != string(after) {
		t.Fatal("Codex adapter mutated shared definition")
	}
}
