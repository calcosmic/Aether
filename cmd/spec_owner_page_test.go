package cmd

// The owner page: the project description ("specification") shown to the
// owner as a short plain-English page -- what you'll get, what's included,
// what you decided, what's off-limits, how you'll check it -- with no item
// codes, no fingerprints and no sentence said twice (owner's choice,
// 2026-09-26). The detailed listing with every stable ID stays reachable
// behind `aether spec --detail`; JSON output is untouched.

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

var (
	specOwnerPageItemIDRe = regexp.MustCompile(`[a-z]+(-[a-z0-9]+)+-[0-9a-f]{8}`)
	specOwnerPageHexRe    = regexp.MustCompile(`[0-9a-f]{64}`)
)

const (
	specOwnerPageFixtureQuestion = "Should a failed save keep the previous version?"
	specOwnerPageFixtureAnswer   = "Yes, always keep the previous version"
)

// specOwnerPageFixtureState is the same accepted-charter shape the settled
// discuss fixture uses: constraints, key risks, governance and a technology
// context, so every machine lead-in in buildSettledDiscussDraftRequest fires.
func specOwnerPageFixtureState() colony.ColonyState {
	goal := "Deliver the settled owner-page contract"
	sessionID := "session-owner-page"
	initializedAt := time.Date(2026, time.September, 7, 16, 0, 0, 0, time.UTC)
	return colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		State:         colony.StateREADY,
		SessionID:     &sessionID,
		InitializedAt: &initializedAt,
		AcceptedCharter: &colony.AcceptedCharter{
			SchemaVersion: colony.AcceptedCharterSchemaVersion,
			EpisodeID:     "episode-owner-page",
			Goal:          goal,
			Provenance:    "owner-approved owner-page fixture",
			AcceptedAt:    initializedAt,
			Charter: &colony.Charter{
				Intent:      "Ship only the accepted owner-page behavior.",
				Vision:      "The owner can review a precise contract before planning.",
				Governance:  "Only the owner may approve the exact specification revision.",
				Goals:       "Produce a readable and verifiable result.",
				TechStack:   "Reuse the current repository stack.",
				KeyRisks:    "Do not infer approval or lose the last valid state.",
				Constraints: "Do not expand beyond the accepted goal.",
			},
		},
	}
}

// specOwnerPageFixtureRequest builds the draft request exactly the way a
// settled discussion does -- through buildSettledDiscussDraftRequest -- with
// one resolved, hard-constraint owner clarification.
func specOwnerPageFixtureRequest(t *testing.T) specificationDraftRequest {
	t.Helper()
	state := specOwnerPageFixtureState()
	pending := PendingDecisionFile{Decisions: []PendingDecision{{
		ID:             "D1",
		Type:           clarificationDecisionType,
		Description:    specOwnerPageFixtureQuestion + " Options: Yes, always keep the previous version | No",
		Source:         "discuss:behavior",
		Resolution:     specOwnerPageFixtureAnswer,
		Resolved:       true,
		HardConstraint: true,
		CreatedAt:      "2026-09-07T16:00:00Z",
	}}}
	frontier := discussEvidenceFrontier{
		Scope: planningEvidenceScope{GoalID: "goal-owner-page", SessionID: "session-owner-page"},
		Items: []discussEvidenceItem{
			{Record: planningEvidenceRecord{Reference: colony.PlanningEvidenceRef{ID: "evidence-charter", Kind: colony.PlanningEvidenceCharter, Origin: "state:charter"}}},
			{Record: planningEvidenceRecord{Reference: colony.PlanningEvidenceRef{ID: "evidence-decision", Kind: colony.PlanningEvidenceDecision, Origin: "decision:D1"}}},
		},
	}
	request, err := buildSettledDiscussDraftRequest(state, codexSurveyContext{}, analyzeScanData{}, pending, frontier, nil,
		time.Date(2026, time.September, 7, 16, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("build settled draft request: %v", err)
	}
	return request
}

func specOwnerPageFixtureRevision(t *testing.T) colony.SpecRevision {
	t.Helper()
	_, revision, err := buildSpecificationDraft(specOwnerPageFixtureRequest(t))
	if err != nil {
		t.Fatalf("build specification draft: %v", err)
	}
	return revision
}

func specOwnerPageBodyFromRevision(revision colony.SpecRevision) discussSpecificationBody {
	return discussSpecificationBody{
		Outcomes:             revision.Outcomes,
		IncludedBehaviors:    revision.IncludedBehaviors,
		Exclusions:           revision.Exclusions,
		BindingDecisions:     revision.BindingDecisions,
		Requirements:         revision.Requirements,
		AcceptanceChecks:     revision.AcceptanceChecks,
		NegativeExpectations: revision.NegativeExpectations,
		RecoveryExpectations: revision.RecoveryExpectations,
		AffectedPublicPaths:  revision.AffectedPublicPaths,
	}
}

func init() {
	registerVoiceScreen("spec-owner-page", func(t *testing.T) string {
		t.Helper()
		revision := specOwnerPageFixtureRevision(t)
		return renderBanner("📜", "Specification") + renderSpecOwnerPage(specOwnerPageBodyFromRevision(revision), revision.Status)
	})
}

// specOwnerPageCodeLeaks reports every line carrying a stable item ID, a
// 64-character fingerprint or a sha256: prefix.
func specOwnerPageCodeLeaks(text string) []string {
	var leaks []string
	for _, line := range strings.Split(text, "\n") {
		if specOwnerPageItemIDRe.MatchString(line) || specOwnerPageHexRe.MatchString(line) || strings.Contains(line, "sha256:") {
			leaks = append(leaks, line)
		}
	}
	return leaks
}

// specOwnerPageRepeatedSentences splits every content line (after its
// symbol) into sentences and reports any sentence that appears twice,
// ignoring case, whitespace and closing punctuation.
func specOwnerPageRepeatedSentences(text string) []string {
	seen := map[string]bool{}
	var repeats []string
	splitter := regexp.MustCompile(`[.?!]\s+`)
	for _, raw := range strings.Split(stripANSI(text), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "──") || strings.HasPrefix(line, "━━") {
			continue
		}
		if index := strings.Index(line, " "); index > 0 {
			line = line[index+1:]
		}
		for _, sentence := range splitter.Split(line, -1) {
			key := strings.ToLower(strings.Join(strings.Fields(sentence), " "))
			key = strings.TrimRight(key, ".?! ")
			if key == "" {
				continue
			}
			if seen[key] {
				repeats = append(repeats, sentence)
			}
			seen[key] = true
		}
	}
	return repeats
}

func TestSpecOwnerPageHasNoCodesOrRepeats(t *testing.T) {
	revision := specOwnerPageFixtureRevision(t)
	body := specOwnerPageBodyFromRevision(revision)
	page := renderSpecOwnerPage(body, revision.Status)

	if leaks := specOwnerPageCodeLeaks(page); len(leaks) > 0 {
		t.Errorf("owner page shows codes or fingerprints:\n  %s\n\n%s", strings.Join(leaks, "\n  "), page)
	}
	if repeats := specOwnerPageRepeatedSentences(page); len(repeats) > 0 {
		t.Errorf("owner page says the same thing twice: %q\n\n%s", repeats, page)
	}
	for _, leadIn := range []string{
		"Deliver the accepted goal:", "Required result:", "Work outside these accepted constraints is excluded:",
		"The result must not violate these accepted constraints:", "The result must not silently realize this known risk:",
		"If the known risk occurs, preserve the last valid state and report it before continuing:",
		"Use the accepted implementation context:", "Settled behavior:", "Owner decision:",
		"The result must not violate this explicit owner constraint:", "Revision Impact", "Affected specification IDs",
	} {
		if strings.Contains(page, leadIn) {
			t.Errorf("owner page still carries the machine wording %q:\n%s", leadIn, page)
		}
	}
	wantDecision := specOwnerPageFixtureQuestion + " → " + specOwnerPageFixtureAnswer
	if !strings.Contains(page, wantDecision) {
		t.Errorf("owner page does not show the owner decision as %q:\n%s", wantDecision, page)
	}
	for _, heading := range []string{
		"What you'll get", "What's included", "What you decided", "Rules it must follow",
		"Off-limits", "If something goes wrong", "How you'll check it",
	} {
		if !strings.Contains(page, "── "+heading+" ──") {
			t.Errorf("owner page is missing the %q section:\n%s", heading, page)
		}
	}
	for _, want := range []string{
		"Deliver the settled owner-page contract",
		"Do not expand beyond the accepted goal.",
		"Do not infer approval or lose the last valid state.",
		"Reuse the current repository stack.",
		"waiting for your approval",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("owner page lost %q:\n%s", want, page)
		}
	}

	t.Run("the repeat check can fail", func(t *testing.T) {
		if repeats := specOwnerPageRepeatedSentences("🚫 Do not expand.\n👑 do not   EXPAND\n"); len(repeats) == 0 {
			t.Fatal("a planted repeated sentence was not caught")
		}
	})
	t.Run("the code check can fail", func(t *testing.T) {
		if leaks := specOwnerPageCodeLeaks("👑 outcome-charter-vision-2e793ec3  The owner can review\n"); len(leaks) == 0 {
			t.Fatal("a planted item ID was not caught")
		}
	})
}

func runSpecCommandVisual(t *testing.T, root string, args ...string) string {
	t.Helper()
	saveGlobals(t)
	t.Setenv("AETHER_ROOT", root)
	t.Setenv("COLONY_DATA_DIR", "")
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	var output bytes.Buffer
	stdout = &output
	stderr = &output
	renderedCommandExitCode.Store(0)
	t.Cleanup(func() { renderedCommandExitCode.Store(0) })
	command := newSpecCommand()
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs(args)
	if err := command.Execute(); err != nil {
		t.Fatalf("spec %v: %v", args, err)
	}
	return stripANSI(output.String())
}

// specOwnerPageAboveNextUp is the rendered screen above the "Next Up"
// banner (drawn spaced out, "N E X T   U P"); the exact approval command the owner must type lives in that block
// and legitimately carries the revision fingerprint.
func specOwnerPageAboveNextUp(screen string) string {
	if index := strings.Index(screen, spacedTitle("Next Up")); index >= 0 {
		return screen[:index]
	}
	return screen
}

func TestSpecVisualDefaultIsTheOwnerPage(t *testing.T) {
	root := newSpecificationTestRepository(t, specOwnerPageFixtureState())
	draft, err := createSpecificationDraft(root, specOwnerPageFixtureRequest(t), specificationMutationOptions{})
	if err != nil {
		t.Fatalf("create draft: %v", err)
	}
	page := renderSpecOwnerPage(specOwnerPageBodyFromRevision(draft.Revision), draft.Revision.Status)

	for _, args := range [][]string{nil, {"--inspect"}} {
		screen := runSpecCommandVisual(t, root, args...)
		if !strings.Contains(screen, stripANSI(page)) {
			t.Errorf("aether spec %v does not show the owner page:\n%s", args, screen)
		}
		above := specOwnerPageAboveNextUp(screen)
		if leaks := specOwnerPageCodeLeaks(above); len(leaks) > 0 {
			t.Errorf("aether spec %v shows codes above the next step:\n  %s\n\n%s", args, strings.Join(leaks, "\n  "), screen)
		}
		if strings.Contains(screen, "Revision Impact") || strings.Contains(screen, "Affected specification IDs") {
			t.Errorf("aether spec %v still shows the revision bookkeeping:\n%s", args, screen)
		}
		if !strings.Contains(screen, "--approve") {
			t.Errorf("aether spec %v lost the exact approval command:\n%s", args, screen)
		}
	}

	detail := runSpecCommandVisual(t, root, "--detail")
	for _, want := range []string{"Revision Impact", draft.Revision.Outcomes[0].ID, draft.Revision.ContentHash} {
		if !strings.Contains(detail, want) {
			t.Errorf("aether spec --detail lost %q from the full listing:\n%s", want, detail)
		}
	}

	t.Run("settled discuss shows the same page", func(t *testing.T) {
		saveGlobals(t)
		dataDir := setupBuildFlowTest(t)
		discussRoot := filepath.Dir(filepath.Dir(dataDir))
		seedSettledDiscussState(t, dataDir, "owner-page")
		result, err := runDiscuss(discussRoot, 3, false)
		if err != nil {
			t.Fatalf("run settled discuss: %v", err)
		}
		closeout, ok := discussSpecificationCloseoutFromResult(result)
		if !ok {
			t.Fatalf("settled discuss produced no specification closeout: %#v", result)
		}
		screen := stripANSI(renderDiscussVisual(result))
		ownerPage := stripANSI(renderSpecOwnerPage(closeout.Body, closeout.Status))
		if !strings.Contains(screen, ownerPage) {
			t.Errorf("settled discuss does not show the owner page:\n%s", screen)
		}
		if leaks := specOwnerPageCodeLeaks(screen); len(leaks) > 0 {
			t.Errorf("settled discuss shows codes:\n  %s\n\n%s", strings.Join(leaks, "\n  "), screen)
		}
		if strings.Contains(screen, "Content hash:") || strings.Contains(screen, "Revision ID:") {
			t.Errorf("settled discuss still shows revision bookkeeping:\n%s", screen)
		}
	})
}

func TestSpecJSONUnchangedByOwnerPage(t *testing.T) {
	root := newSpecificationTestRepository(t, specOwnerPageFixtureState())
	if _, err := createSpecificationDraft(root, specOwnerPageFixtureRequest(t), specificationMutationOptions{}); err != nil {
		t.Fatalf("create draft: %v", err)
	}
	plain, err := runSpecCommandRaw(t, root)
	if err != nil {
		t.Fatalf("spec: %v", err)
	}
	inspected, err := runSpecCommandRaw(t, root, "--inspect")
	if err != nil {
		t.Fatalf("spec --inspect: %v", err)
	}
	if plain != inspected {
		t.Fatalf("spec and spec --inspect JSON differ:\n%s\n---\n%s", plain, inspected)
	}
	detailed, err := runSpecCommandRaw(t, root, "--inspect", "--detail")
	if err != nil {
		t.Fatalf("spec --inspect --detail: %v", err)
	}
	if detailed != inspected {
		t.Fatalf("--detail changed the JSON output:\n%s\n---\n%s", detailed, inspected)
	}

	var envelope struct {
		OK     bool                       `json:"ok"`
		Result map[string]json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal([]byte(inspected), &envelope); err != nil {
		t.Fatalf("decode: %v\n%s", err, inspected)
	}
	got := make([]string, 0, len(envelope.Result))
	for key := range envelope.Result {
		got = append(got, key)
	}
	sort.Strings(got)
	want := []string{
		"acceptance_checks", "affected_public_paths", "affected_scope", "after_revision_id", "before_revision_id",
		"binding_decisions", "classified_delta", "command", "content_hash", "exclusions", "included_behaviors",
		"negative_expectations", "next_action", "operation", "outcome_kind", "outcomes", "projection",
		"projection_repaired", "recovery_expectations", "replayed", "requirements", "result_kind", "revision_number",
		"schema_version", "scope", "specification_id", "state_effect", "status",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("spec JSON keys changed:\n got %v\nwant %v", got, want)
	}
}
