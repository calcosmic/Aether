package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jedib0t/go-pretty/v6/text"
)

func TestTeamApprovalCardNamesEveryWorkerAndAssignment(t *testing.T) {
	manifest, _ := teamCheckinManifestFixture()
	manifest["phase"] = 1
	manifest["dispatches"] = []interface{}{
		map[string]interface{}{"name": "Mason-12", "caste": "builder", "task": "Preserve the backup and rehearse Obsidian", "covered_task_ids": []interface{}{"1.1", "1.2"}, "execution_wave": 11},
		map[string]interface{}{"name": "Anvil-7", "caste": "builder", "task": "Prepare the migration map", "task_id": "1.3", "execution_wave": 21},
	}
	before, _ := json.Marshal(manifest)
	result, visual := renderCeremonyTeamCheckin("build", manifest, ceremonyDispatchesFromManifest(manifest))
	card, _ := result["approval_card"].(string)
	for _, want := range []string{"┌", "┘", "PHASE 1", "2 planned worker(s)", "Builder Mason-12", "Builder Anvil-7", "1.1, 1.2", "Tasks: 1.3", "build wave 11", "build wave 21", "Preserve the backup and rehearse Obsidian", "Prepare the migration map"} {
		if !strings.Contains(card, want) {
			t.Errorf("card missing %q:\n%s", want, card)
		}
	}
	if !strings.HasPrefix(visual, card) || card == "" {
		t.Fatal("JSON approval card must also lead the visual output")
	}
	if strings.Contains(card, "AFTER BUILD") {
		t.Fatal("card invented future reviewers")
	}
	after, _ := json.Marshal(manifest)
	if string(after) != string(before) {
		t.Fatal("rendering changed the manifest")
	}
}

func TestTeamApprovalCardSeparatesLiveAndDeclinedReviews(t *testing.T) {
	hits := riskSignalHitsFromRecords([]codexForcedReviewerRecord{
		{Caste: "auditor", Signals: []string{"database migration"}, Matches: []string{"migration"}, Sources: []string{"plan wording"}},
		{Caste: "gatekeeper", Signals: []string{"credentials/auth"}, Matches: []string{"credentials"}, Sources: []string{"plan wording"}},
	})
	if len(hits) != 2 {
		t.Fatalf("fixture needs two risk signals, got %d", len(hits))
	}
	manifest, dispatches := teamCheckinManifestFixture()
	card := renderTeamApprovalCard(manifest, dispatches[:1], map[string]bool{"builder": true}, nil, hits, nil)
	for _, want := range []string{"1 planned worker(s)", "AFTER BUILD", "Auditor — quality review", "Gatekeeper — security review", "migration", "credentials", "Reviewer names are assigned at verification."} {
		if !strings.Contains(card, want) {
			t.Errorf("card missing %q:\n%s", want, card)
		}
	}
	hits[1].WaiverReason = "Owner accepts the documented risk"
	card = renderTeamApprovalCard(manifest, dispatches[:1], nil, nil, hits[:1], hits[1:])
	if strings.Contains(card, "Gatekeeper — security review") || !strings.Contains(card, hits[1].WaiverReason) {
		t.Fatalf("declined reviewer still shown as running, or reason lost:\n%s", card)
	}
}

func TestTeamApprovalCardWrapsAndStripsTerminalControls(t *testing.T) {
	card := frameTeamApprovalCard([]string{"\x1b[31mBuilder\x1b[0m\t紅葉", strings.Repeat("long task ", 40), strings.Repeat("x", 160)})
	if strings.Contains(card, "\x1b") || strings.Contains(card, "\t") {
		t.Fatal("plain card contains terminal controls")
	}
	for _, row := range strings.Split(strings.TrimSuffix(card, "\n"), "\n") {
		if width := text.StringWidth(row); width != 76 {
			t.Errorf("frame row width %d, want 76: %q", width, row)
		}
	}
}

func TestBuildApprovalCardVisibleAcrossGuidanceSurfaces(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		".aether/commands/build.yaml", ".aether/skills/colony/aether-colony-build-cycle/SKILL.md",
		".claude/commands/ant/build.md", ".claude/commands/ant-build.md", ".opencode/commands/ant/build.md", "cmd/command_guide.go",
	} {
		raw, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"aether ceremony team-checkin", "result.approval_card", "visible conversation immediately before the approval choices", "older runtime"} {
			if !strings.Contains(string(raw), want) {
				t.Errorf("%s missing %q", rel, want)
			}
		}
	}
}
