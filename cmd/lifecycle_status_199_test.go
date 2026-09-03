package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestLifecycleStatusWrapper199(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("find repository root: %v", err)
	}
	const (
		source      = ".aether/commands/status.yaml"
		description = "Show the complete authoritative colony snapshot."
		runtime     = "AETHER_OUTPUT_MODE=visual aether status $ARGUMENTS"
	)
	spec, issues := readSourceCheckCommandSpec(root, source, "status")
	if len(issues) > 0 {
		t.Fatalf("status source is invalid: %#v", issues)
	}
	if spec.Description != description {
		t.Fatalf("canonical description = %q, want %q", spec.Description, description)
	}
	if spec.Runtime.Command != runtime {
		t.Fatalf("canonical runtime command = %q, want %q", spec.Runtime.Command, runtime)
	}

	canonical, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(source)))
	if err != nil {
		t.Fatalf("read canonical status source: %v", err)
	}
	canonicalText := string(canonical)
	for _, want := range []string{"default", "full", "--compact", "snapshot", "watch", "event stream"} {
		if !strings.Contains(strings.ToLower(canonicalText), strings.ToLower(want)) {
			t.Errorf("canonical status source missing %q semantics\n%s", want, canonicalText)
		}
	}

	for _, rel := range []string{
		".claude/commands/ant-status.md",
		".claude/commands/ant/status.md",
		".opencode/commands/ant/status.md",
	} {
		t.Run(rel, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
			if err != nil {
				t.Fatalf("read wrapper: %v", err)
			}
			firstLine := strings.SplitN(string(data), "\n", 2)[0]
			if want := "<!-- Aether-managed: runtime spec at " + source + ". Synced by aether update. -->"; firstLine != want {
				t.Fatalf("source linkage = %q, want %q", firstLine, want)
			}
			frontmatter, body, err := parseSourceCheckWrapper(data)
			if err != nil {
				t.Fatalf("parse wrapper: %v", err)
			}
			if frontmatter.Name != "ant-status" || frontmatter.Description != description {
				t.Fatalf("frontmatter = %#v, want ant-status and exact description", frontmatter)
			}
			for _, want := range []string{spec.SourceOfTruth, runtime, "default full", "--compact", "complete snapshot", "event stream"} {
				if !strings.Contains(strings.ToLower(body), strings.ToLower(want)) {
					t.Errorf("wrapper missing %q\n%s", want, body)
				}
			}
			for _, forbidden := range []string{
				".aether/data", "COLONY_STATE.json", "spawn-tree", "pending-decisions",
				"`ps ", "`pgrep ", "`git log", "`cat ", "`jq ", "cost ledger",
			} {
				if strings.Contains(strings.ToLower(body), strings.ToLower(forbidden)) {
					t.Errorf("wrapper performs or names host-side inference %q\n%s", forbidden, body)
				}
			}
		})
	}
}

func TestLifecycleStatus199FullOrder(t *testing.T) {
	projection := projectLifecycle(lifecycleStatus199Facts(), LifecycleViewFull, "codex")
	projection.Command = "status"

	wantIDs := []string{
		"identity", "progress", "actors", "signals", "research",
		"memory_evidence", "elapsed_cost", "history", "open_items", "next_action",
	}
	if got := lifecycleProjectionSectionIDs(projection.Sections); !reflect.DeepEqual(got, wantIDs) {
		t.Fatalf("full status section IDs = %#v, want %#v", got, wantIDs)
	}

	output := stripANSI(renderLifecycleStatus(projection, 100))
	wantHeadings := []string{
		"Colony", "Phase & Tasks", "Ants & Outcomes", "Pheromones", "Territory & Notes",
		"Memory, Findings & Gates", "Elapsed & Reported Cost", "Recent History", "Open Items", "Next Up",
	}
	last := -1
	for _, heading := range wantHeadings {
		index := strings.Index(output, heading)
		if index < 0 {
			t.Fatalf("full status missing %q\n%s", heading, output)
		}
		if index <= last {
			t.Fatalf("full status section %q is out of order\n%s", heading, output)
		}
		last = index
	}

	for _, want := range []string{
		"Atlas", "Ship the complete status", "Mason-1", "Queen → Mason-1", "FOCUS",
		"research/front-door.md", "dreams/2026-09-03-status.md", "Unverified local note",
		"1500 tokens", "gate-tests", "history-entry", "owner-decision", "aether continue",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("full status missing recorded fact %q\n%s", want, output)
		}
	}
}

func TestLifecycleStatus199CompactSubset(t *testing.T) {
	facts := lifecycleStatus199Facts()
	full := projectLifecycle(facts, LifecycleViewFull, "codex")
	compact := projectLifecycle(facts, LifecycleViewCompact, "codex")
	full.Command = "status"
	compact.Command = "status"

	fullSet := map[string]bool{}
	for _, section := range full.Sections {
		fullSet[section.ID] = true
	}
	for _, section := range compact.Sections {
		if !fullSet[section.ID] {
			t.Fatalf("compact section %q is not in full projection", section.ID)
		}
	}
	if len(compact.Sections) >= len(full.Sections) {
		t.Fatalf("compact sections = %d, full = %d; compact is not a strict subset", len(compact.Sections), len(full.Sections))
	}
	if !reflect.DeepEqual(compact.Identity, full.Identity) || !reflect.DeepEqual(compact.Phase, full.Phase) ||
		!reflect.DeepEqual(compact.NextAction, full.NextAction) {
		t.Fatal("compact status recomputed shared semantic facts")
	}

	output := stripANSI(renderLifecycleStatus(compact, 72))
	lines := lifecycleStatus199Lines(output)
	if len(lines) > 12 {
		t.Fatalf("compact status used %d lines, want at most 12\n%s", len(lines), output)
	}
	for _, forbidden := range []string{"Pheromones", "Memory, Findings & Gates", "Recent History"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("compact status rendered non-selected section %q\n%s", forbidden, output)
		}
	}
	for _, want := range []string{"Atlas", "Phase 1/1", "No ants are active", "1500 tokens", "aether continue"} {
		if !strings.Contains(output, want) {
			t.Errorf("compact status missing %q\n%s", want, output)
		}
	}
}

func TestLifecycleStatus199Responsive(t *testing.T) {
	facts := lifecycleStatus199Facts()
	facts.Identity.Value.Goal = strings.Repeat("responsive status wording ", 8)
	facts.Research.Value.Dreams = []string{".aether/dreams/2026-09-03-a-very-long-but-still-identifying-status-research-note.md"}

	for _, width := range []int{48, 72, 100} {
		projection := projectLifecycle(facts, LifecycleViewFull, "codex")
		projection.Command = "status"
		output := stripANSI(renderLifecycleStatus(projection, width))
		for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
			if got := utf8.RuneCountInString(line); got > width {
				t.Fatalf("width %d rendered %d-column line %q\n%s", width, got, line, output)
			}
		}
	}
}

func TestLifecycleStatus199OutputModes(t *testing.T) {
	projection := projectLifecycle(lifecycleStatus199Facts(), LifecycleViewFull, "claude")
	projection.Command = "status"

	t.Setenv("AETHER_FORCE_COLOR", "1")
	colored := renderLifecycleStatus(projection, 100)
	t.Setenv("AETHER_FORCE_COLOR", "")
	t.Setenv("NO_COLOR", "1")
	plain := renderLifecycleStatus(projection, 100)
	if got := stripANSI(colored); got != plain {
		t.Fatalf("visual and NO_COLOR changed semantic facts\nvisual:\n%s\nplain:\n%s", got, plain)
	}

	payload, err := json.Marshal(projection)
	if err != nil {
		t.Fatalf("marshal status projection: %v", err)
	}
	jsonText := string(payload)
	for _, field := range []string{
		`"schema_version"`, `"command":"status"`, `"outcome_kind"`, `"projection_revision"`,
		`"identity"`, `"goal"`, `"standing"`, `"phase"`, `"tasks"`, `"actors"`, `"lineage"`,
		`"signals"`, `"verification"`, `"warnings"`, `"debt"`, `"blockers"`, `"owner_decisions"`,
		`"elapsed"`, `"reported_cost"`, `"next_action"`, `"alternatives"`, `"state_effect"`, `"provenance"`,
	} {
		if !strings.Contains(jsonText, field) {
			t.Errorf("status JSON missing shared result field %s\n%s", field, jsonText)
		}
	}
	if strings.Contains(jsonText, "$0") || strings.Contains(plain, "$0") {
		t.Fatalf("missing cost was rendered as a zero-dollar claim\njson=%s\nvisual=%s", jsonText, plain)
	}
}

func TestLifecycleStatus199ReadOnly(t *testing.T) {
	for _, fixture := range []string{"valid", "missing", "malformed"} {
		t.Run(fixture, func(t *testing.T) {
			root, factStore, watched := seedLifecycleFactsFixture(t, fixture)
			before := fingerprintLifecycleFactSurfaces(t, root, watched)
			now, _ := time.Parse(time.RFC3339, lifecycleFactsNow)
			facts, err := loadLifecycleFacts(root, factStore, now)
			if err != nil {
				t.Fatalf("load lifecycle facts: %v", err)
			}
			for _, view := range []LifecycleView{LifecycleViewFull, LifecycleViewCompact} {
				projection := projectLifecycle(facts, view, "codex")
				projection.Command = "status"
				_ = renderLifecycleStatus(projection, 72)
			}
			after := fingerprintLifecycleFactSurfaces(t, root, watched)
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("%s status invocation mutated workspace or hub\nbefore: %#v\nafter: %#v", fixture, before, after)
			}
		})
	}
}

func TestLifecycleStatus199NoInventedEvidence(t *testing.T) {
	facts := projectionFacts(projectionState(colony.StateREADY, true, false))
	facts.Actors = LifecycleFact[[]LifecycleActorFact]{Source: lifecycleSource("actors", "spawn-tree.txt", LifecycleFactMissing, "not recorded")}
	facts.Signals = LifecycleFact[[]colony.PheromoneSignal]{Source: lifecycleSource("signals", "pheromones.json", LifecycleFactMissing, "not recorded")}
	facts.Verification = LifecycleFact[LifecycleVerificationFacts]{Source: lifecycleSource("verification", "build", LifecycleFactMissing, "not recorded")}
	facts.ReportedCost = LifecycleFact[LifecycleReportedCostFacts]{Source: lifecycleSource("reported cost", "spend", LifecycleFactMissing, "not reported")}
	facts.History = LifecycleFact[[]string]{Source: lifecycleSource("history", "COLONY_STATE.json", LifecycleFactMissing, "not recorded")}

	projection := projectLifecycle(facts, LifecycleViewFull, "codex")
	projection.Command = "status"
	output := stripANSI(renderLifecycleStatus(projection, 100))
	for _, want := range []string{"No ants are active", "Reported cost: Unreported", "Verification: Unknown"} {
		if !strings.Contains(output, want) {
			t.Errorf("status did not state absent evidence as %q\n%s", want, output)
		}
	}
	for _, invented := range []string{"$0", "Acknowledged by Queen", "Verified complete", "Recovered successfully"} {
		if strings.Contains(output, invented) {
			t.Fatalf("status invented %q\n%s", invented, output)
		}
	}
}

func lifecycleStatus199Facts() LifecycleFacts {
	state := projectionState(colony.StateBUILT, true, false)
	state.State = colony.StateBUILT
	state.Plan.Phases[0].SuccessCriteria = []string{"Status is truthful"}
	facts := projectionFacts(state)
	facts.Identity.Value.Name = "Atlas"
	facts.Identity.Value.Goal = "Ship the complete status"
	facts.Identity.Value.Scope = "project"
	facts.Identity.Value.Mode = "orchestrator"
	facts.Actors.Value = []LifecycleActorFact{
		{Timestamp: "2026-09-03T11:30:00Z", Parent: "Queen", Caste: "builder", Name: "Mason-1", Task: "render status", Status: "completed", Summary: "renderer committed"},
	}
	facts.Signals.Value = []colony.PheromoneSignal{{ID: "signal-focus", Type: "FOCUS", Active: true, Content: json.RawMessage(`{"text":"preserve truth"}`)}}
	facts.Research.Value = LifecycleResearchFacts{
		Docs:      []string{".aether/research/front-door.md"},
		Dreams:    []string{".aether/dreams/2026-09-03-status.md"},
		Territory: []string{".aether/data/survey/territory.json"},
	}
	facts.Memory.Value.Instincts = nil
	facts.Memory.Value.Observations = nil
	facts.Memory.Value.Findings = []colony.ReviewLedgerEntry{{ID: "finding-1", Status: "resolved", Severity: colony.ReviewSeverityLow, Description: "renderer uses facts"}}
	facts.Verification.Value.Gates = []colony.GateResultEntry{{Name: "gate-tests", Passed: true, Timestamp: "2026-09-03T11:45:00Z", Detail: "focused tests passed"}}
	facts.Verification.Value.Artifacts = []string{".aether/data/build/phase-1/verification.json"}
	facts.Timing.Value.Elapsed = 90 * time.Minute
	facts.ReportedCost = LifecycleFact[LifecycleReportedCostFacts]{
		Value:  LifecycleReportedCostFacts{Phase: 1, TotalTokens: 1500, Rows: 1},
		Source: lifecycleSource("reported cost", ".aether/data/spend", LifecycleFactConfirmed, ""),
	}
	facts.History.Value = []string{"history-entry"}
	facts.Blockers.Value = []colony.FlagEntry{{ID: "owner-decision", Type: "decision", Description: "owner-decision", Resolved: false}}
	return facts
}

func lifecycleStatus199Lines(output string) []string {
	raw := strings.Split(strings.TrimSpace(output), "\n")
	lines := make([]string, 0, len(raw))
	for _, line := range raw {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
