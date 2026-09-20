package cmd

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
	"gopkg.in/yaml.v3"
)

// sealCasteDisplayNames maps the lowercase caste identifier to the
// capitalised display name a wrapper sentence would use to name it, limited
// to the castes casteAllowedForFlow permits at seal (cmd/caste_relevance.go).
// Any other capitalised word appearing on the wrapper's `dispatches` line
// cannot be a caste-dispatch claim, so it is never mistaken for one.
var sealCasteDisplayNames = map[string]string{
	"gatekeeper": "Gatekeeper",
	"auditor":    "Auditor",
	"probe":      "Probe",
	"porter":     "Porter",
	"chronicler": "Chronicler",
	"keeper":     "Keeper",
	"sage":       "Sage",
	"measurer":   "Measurer",
	"includer":   "Includer",
}

// representativeSealColonyStates builds finished-colony fixtures that span
// the three review depths sealReviewDepthForColony actually derives from
// real phase-mode data (heavy, standard, light) -- not a hand-typed literal
// standing in for what a real project looks like. Each phase mirrors the
// colony.Phase shape production reads from disk (ID/Name/Description/Mode/
// Status), exercised through the exact call chain runSealPlanOnly and
// seal-finalize use: sealReviewRequiredCastes -> sealReviewDepthForColony +
// queenSealReviewSpecs -> queenOrchestrate.
func representativeSealColonyStates() map[string]colony.ColonyState {
	phase := func(mode colony.PhaseMode, name, description string) colony.Phase {
		return colony.Phase{ID: 1, Name: name, Description: description, Mode: mode, Status: colony.PhaseCompleted}
	}
	return map[string]colony.ColonyState{
		"shipped production work, ordinary feature": {
			Plan: colony.Plan{Phases: []colony.Phase{
				phase(colony.PhaseModeProduction, "Add user dashboard widgets", "Implement dashboard cards showing usage stats"),
			}},
		},
		"shipped production work, security-themed": {
			Plan: colony.Plan{Phases: []colony.Phase{
				phase(colony.PhaseModeProduction, "Harden auth tokens", "Add token rotation, secrets management, and audit logging for compliance"),
			}},
		},
		"maintenance only, no production phase": {
			Plan: colony.Plan{Phases: []colony.Phase{
				phase(colony.PhaseModeMaintenance, "Refactor billing module", "Clean up billing code, remove dead branches"),
			}},
		},
		"discovery only, nothing shipped": {
			Plan: colony.Plan{Phases: []colony.Phase{
				phase(colony.PhaseModeDiscovery, "Research caching options", "Investigate caching strategies for the API layer"),
			}},
		},
	}
}

// sealCastesAlwaysReturned intersects the caste sets sealReviewRequiredCastes
// actually returns for every representative state: the castes present in
// every one of them, and therefore the only castes a wrapper sentence may
// honestly name as an unconditional seal dispatch. It also returns the raw
// per-state observation so a failing assertion can show its work.
func sealCastesAlwaysReturned(states map[string]colony.ColonyState) ([]string, map[string][]string) {
	perState := make(map[string][]string, len(states))
	counts := make(map[string]int)
	total := 0
	for label, state := range states {
		castes := sealReviewRequiredCastes(state)
		perState[label] = castes
		total++
		for _, c := range castes {
			counts[c]++
		}
	}
	var always []string
	for c, n := range counts {
		if n == total {
			always = append(always, c)
		}
	}
	sort.Strings(always)
	return always, perState
}

// extractSealDispatchClaimLine finds the wrapper's `dispatches` bullet under
// "Expected manifest" -- the exact line the pre-205-04 text used to assert a
// flat "Gatekeeper, Auditor, and Probe final-review workers" roster.
func extractSealDispatchClaimLine(t *testing.T, wrapperText string) string {
	t.Helper()
	line := extractLineContaining(wrapperText, "`dispatches`")
	if line == "" {
		t.Fatalf("seal wrapper missing the `dispatches` bullet under \"Expected manifest\"")
	}
	return line
}

// extractSealDispatchClaimCastes derives the claim under test BY PARSING the
// wrapper's own dispatch line -- it never re-types a caste list inline. Any
// known seal caste display name appearing on that line is read as a claim
// that the runtime always dispatches it.
func extractSealDispatchClaimCastes(line string) []string {
	var claimed []string
	for lower, display := range sealCasteDisplayNames {
		if strings.Contains(line, display) {
			claimed = append(claimed, lower)
		}
	}
	sort.Strings(claimed)
	return claimed
}

// TestSealWrapperReviewClaimMatchesTheRuntime holds the seal wrapper's
// review-team claim to queenSealReviewSpecs's real output (via
// sealReviewRequiredCastes, the same function the seal command itself
// calls). It fails if the wrapper names a caste the runtime does not always
// return for a representative colony, and fails if the runtime always
// returns a caste the wrapper's claim omits.
func TestSealWrapperReviewClaimMatchesTheRuntime(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	wrapperPath := filepath.Join(repoRoot, ".claude", "commands", "ant", "seal.md")
	wrapperBytes, err := os.ReadFile(wrapperPath)
	if err != nil {
		t.Fatalf("read %s: %v", wrapperPath, err)
	}

	dispatchLine := extractSealDispatchClaimLine(t, string(wrapperBytes))
	claimedCastes := extractSealDispatchClaimCastes(dispatchLine)

	states := representativeSealColonyStates()
	alwaysReturned, perState := sealCastesAlwaysReturned(states)

	for _, claimed := range claimedCastes {
		if !containsString(alwaysReturned, claimed) {
			t.Errorf("seal.md's `dispatches` bullet names %q as part of the review team, but the runtime does not always dispatch it for a representative colony -- observed per colony: %+v\nbullet: %s", claimed, perState, dispatchLine)
		}
	}
	for _, always := range alwaysReturned {
		if !containsString(claimedCastes, always) {
			t.Errorf("the runtime always dispatches %q at seal across every representative colony tested, but seal.md's `dispatches` bullet omits it -- observed per colony: %+v\nbullet: %s", always, perState, dispatchLine)
		}
	}
}

// TestSealWrapperTripletStaysIdentical asserts the Claude and OpenCode
// copies of the seal wrapper are byte-identical, and that the runtime
// source YAML carries the same review-team sentence the two platform
// copies display -- the wrapper triplets are hand-maintained with no
// generator, so nothing but a named test keeps them in sync.
func TestSealWrapperTripletStaysIdentical(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	claudeBytes, err := os.ReadFile(filepath.Join(repoRoot, ".claude", "commands", "ant", "seal.md"))
	if err != nil {
		t.Fatalf("read claude seal.md: %v", err)
	}
	opencodeBytes, err := os.ReadFile(filepath.Join(repoRoot, ".opencode", "commands", "ant", "seal.md"))
	if err != nil {
		t.Fatalf("read opencode seal.md: %v", err)
	}
	if string(claudeBytes) != string(opencodeBytes) {
		t.Fatalf("seal.md drifted between the Claude and OpenCode copies -- these two files are hand-maintained with no generator and must stay byte-identical")
	}

	claudeLine := extractSealDispatchClaimLine(t, string(claudeBytes))

	yamlPath := filepath.Join(repoRoot, ".aether", "commands", "seal.yaml")
	yamlBytes, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read %s: %v", yamlPath, err)
	}
	var spec struct {
		WrapperContract struct {
			ReviewTeam string `yaml:"review_team"`
		} `yaml:"wrapper_contract"`
	}
	if err := yaml.Unmarshal(yamlBytes, &spec); err != nil {
		t.Fatalf("parse %s: %v", yamlPath, err)
	}
	reviewTeamSentence := strings.TrimSpace(spec.WrapperContract.ReviewTeam)
	if reviewTeamSentence == "" {
		t.Fatalf("%s: wrapper_contract.review_team is empty -- the runtime source of truth must carry the review-team sentence the platform wrappers mirror", yamlPath)
	}
	if !strings.Contains(claudeLine, reviewTeamSentence) {
		t.Errorf("seal.md's `dispatches` bullet does not carry the exact review-team sentence from %s\nyaml:    %s\nwrapper: %s", yamlPath, reviewTeamSentence, claudeLine)
	}
}

// relayCardSentence is the exact instruction that must follow every step
// where a wrapper runs the runtime in its picture-drawing (AETHER_OUTPUT_
// MODE=visual) output mode -- the moment the field report found nothing
// told the assistant to show that output to the owner. It grants the
// wrapper no computation, gating, or verification duty: it only says the
// card the runtime drew must reach the owner's chat.
const relayCardSentence = "Show this output to the owner in your own reply, unchanged — you are only passing along what the command already produced, not deciding, checking, or changing anything yourself."

// visualModeLinesWithoutNearbyRelay parses wrapperText for every step that
// runs the runtime in picture-drawing (AETHER_OUTPUT_MODE=visual) mode and
// returns the 1-based line numbers of any such step with no relay
// instruction within the following few lines -- covering both an inline
// numbered-list step (relay appended to the same line) and a fenced code
// block followed by a relay sentence a couple of lines later.
func visualModeLinesWithoutNearbyRelay(wrapperText string) []int {
	const proximity = 6
	lines := strings.Split(wrapperText, "\n")
	var missing []int
	for i, line := range lines {
		if !strings.Contains(line, "AETHER_OUTPUT_MODE=visual") {
			continue
		}
		found := false
		for j := i; j < len(lines) && j < i+proximity; j++ {
			if strings.Contains(lines[j], relayCardSentence) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, i+1)
		}
	}
	return missing
}

// yamlGuardrails reads the top-level `guardrails` list from a wrapper source
// YAML file.
func yamlGuardrails(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var spec struct {
		Guardrails []string `yaml:"guardrails"`
	}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return spec.Guardrails
}

// TestSealAndEntombWrappersRelayTheCard closes field-report defect 3: the
// wrappers ran the runtime's picture-drawing output mode through a shell and
// never told the assistant to show what came back, so the owner saw
// nothing. For each of the four platform wrapper files, this asserts every
// step that runs AETHER_OUTPUT_MODE=visual is followed by a relay
// instruction, and that all four files' relay sentences are byte-identical
// to each other and to both runtime source YAMLs.
func TestSealAndEntombWrappersRelayTheCard(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	wrapperPaths := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant", "seal.md"),
		filepath.Join(repoRoot, ".claude", "commands", "ant", "entomb.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "seal.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "entomb.md"),
	}
	for _, path := range wrapperPaths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(data)
		if !strings.Contains(text, relayCardSentence) {
			t.Errorf("%s never relays the runtime's picture-drawing output to the owner: missing %q", path, relayCardSentence)
			continue
		}
		if missing := visualModeLinesWithoutNearbyRelay(text); len(missing) > 0 {
			t.Errorf("%s runs AETHER_OUTPUT_MODE=visual at line(s) %v with no relay instruction nearby", path, missing)
		}
	}

	sealGuardrails := yamlGuardrails(t, filepath.Join(repoRoot, ".aether", "commands", "seal.yaml"))
	if !containsString(sealGuardrails, relayCardSentence) {
		t.Errorf(".aether/commands/seal.yaml guardrails do not carry the exact relay sentence: %q", relayCardSentence)
	}
	entombGuardrails := yamlGuardrails(t, filepath.Join(repoRoot, ".aether", "commands", "entomb.yaml"))
	if !containsString(entombGuardrails, relayCardSentence) {
		t.Errorf(".aether/commands/entomb.yaml guardrails do not carry the exact relay sentence: %q", relayCardSentence)
	}
}

// TestEntombWrapperTripletStaysIdentical mirrors the seal parity assertion
// for the archive command: the Claude and OpenCode copies of entomb.md are
// hand-maintained with no generator and must stay byte-identical.
func TestEntombWrapperTripletStaysIdentical(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	claudeBytes, err := os.ReadFile(filepath.Join(repoRoot, ".claude", "commands", "ant", "entomb.md"))
	if err != nil {
		t.Fatalf("read claude entomb.md: %v", err)
	}
	opencodeBytes, err := os.ReadFile(filepath.Join(repoRoot, ".opencode", "commands", "ant", "entomb.md"))
	if err != nil {
		t.Fatalf("read opencode entomb.md: %v", err)
	}
	if string(claudeBytes) != string(opencodeBytes) {
		t.Fatalf("entomb.md drifted between the Claude and OpenCode copies -- these two files are hand-maintained with no generator and must stay byte-identical")
	}
}
