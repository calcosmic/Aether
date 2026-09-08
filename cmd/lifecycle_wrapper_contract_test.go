package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// lifecycleWrapperVerbs are the four core lifecycle wrapper commands Phase 165
// rewrites. Codex is runtime-native and carries no wrapper markdown, so it is
// intentionally absent from this list.
var lifecycleWrapperVerbs = []string{"init", "plan", "build", "continue"}

// canonicalWrapperPaths returns the .claude and .opencode canonical wrapper
// paths for a verb, in that order.
func canonicalWrapperPaths(repoRoot, verb string) []string {
	return []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant", verb+".md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", verb+".md"),
	}
}

// flatMirrorPath returns the flat installed-consumer mirror path for a verb
// (the shape `aether install`/`aether update` write from the canonical
// nested file — see cmd/install_cmd.go).
func flatMirrorPath(repoRoot, verb string) string {
	return filepath.Join(repoRoot, ".claude", "commands", "ant-"+verb+".md")
}

// countMarkerLines returns the number of non-comment lines in text that
// contain at least one of the given markers. HTML-comment lines are stripped
// first via stripCommentLines so a comment mentioning a marker cannot
// satisfy a proportion assertion meant to catch a live instruction.
func countMarkerLines(text string, markers []string) int {
	stripped := stripCommentLines(text)
	count := 0
	for _, line := range strings.Split(stripped, "\n") {
		for _, marker := range markers {
			if strings.Contains(line, marker) {
				count++
				break
			}
		}
	}
	return count
}

// stageSkeletonMarkers are the four D-05 stage-skeleton bold markers every
// lifecycle wrapper stage carries: Purpose, Reads, Spawns, Stop conditions.
func stageSkeletonMarkers() []string {
	return []string{"**Purpose:**", "**Reads:**", "**Spawns:**", "**Stop conditions:**"}
}

// envelopeMechanicsMarkers are the phrases that mark envelope-parsing
// mechanics (field-by-field JSON parsing, temp-file bookkeeping) as opposed
// to method prose. D-03 relocates this class of detail to
// .aether/docs/wrapper-host-contract.md; Wave 3's CMD-02 proportion test
// uses this marker set to prove it stayed relocated.
func envelopeMechanicsMarkers() []string {
	return []string{
		"Parse `result.",
		"Save the full JSON envelope",
		"Save the JSON envelope",
		"temporary manifest file outside",
	}
}

// assertOrderedHeadingParity compares the ordered list of level-2 headings
// between a wrapper's .claude and .opencode bodies (comment lines already
// stripped by the caller) and fails with both heading lists on mismatch.
// Generalizes TestPlanWrapperCardsParity's ordered_heading_sets_are_identical
// subtest so build.md, continue.md, and init.md plans can call it directly.
func assertOrderedHeadingParity(t *testing.T, verb, claudeText, opencodeText string) {
	t.Helper()

	claudeHeadings := level2Headings(claudeText)
	opencodeHeadings := level2Headings(opencodeText)
	if !reflect.DeepEqual(claudeHeadings, opencodeHeadings) {
		t.Fatalf("%s: wrapper level-2 heading order differs:\nClaude:   %v\nOpenCode: %v", verb, claudeHeadings, opencodeHeadings)
	}
}

// assertStageSkeletonDensity is the D-09 proportion/invariant assertion: for
// every "## " heading in the comment-stripped body that is not listed in
// exemptHeadings, the section running from that heading to the next "## "
// (or end of file) must contain the literal "**Purpose:**". Unlike a
// named-section grep, this fails when a NEW stage is added without a
// Purpose line, whatever that new stage is named.
func assertStageSkeletonDensity(t *testing.T, path, text string, exemptHeadings []string) {
	t.Helper()

	stripped := stripCommentLines(text)
	lines := strings.Split(stripped, "\n")

	exempt := make(map[string]bool, len(exemptHeadings))
	for _, h := range exemptHeadings {
		exempt[h] = true
	}

	var headingIdx []int
	for i, line := range lines {
		if strings.HasPrefix(line, "## ") {
			headingIdx = append(headingIdx, i)
		}
	}

	for n, idx := range headingIdx {
		heading := lines[idx]
		if exempt[heading] {
			continue
		}
		end := len(lines)
		if n+1 < len(headingIdx) {
			end = headingIdx[n+1]
		}
		section := strings.Join(lines[idx:end], "\n")
		if !strings.Contains(section, "**Purpose:**") {
			t.Errorf("%s: section %q is missing a **Purpose:** line", path, heading)
		}
	}
}

// TestLifecycleFlatMirrorsMatchCanonical pins the flat installed-consumer
// mirrors (.claude/commands/ant-<verb>.md) as byte-identical to their
// canonical nested sources (.claude/commands/ant/<verb>.md). This is a
// standing invariant, not a point-in-time state: `aether install` and
// `aether update` copy the flat mirror from the canonical nested source and
// never hand-edit it, so any drift between the two is always a bug.
func TestLifecycleFlatMirrorsMatchCanonical(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	// Every wrapper, not just the lifecycle five: 41 of 60 mirrors had silently
	// drifted to a superseded generation that hand-read COLONY_STATE.json with
	// jq — the exact boundary violation wrappers are forbidden to commit —
	// because only the lifecycle verbs were guarded here.
	canonicalPaths, err := filepath.Glob(filepath.Join(repoRoot, ".claude", "commands", "ant", "*.md"))
	if err != nil {
		t.Fatalf("glob canonical wrappers: %v", err)
	}
	if len(canonicalPaths) == 0 {
		t.Fatal("no canonical wrappers found — this guard would pass vacuously")
	}

	for _, canonicalPath := range canonicalPaths {
		verb := strings.TrimSuffix(filepath.Base(canonicalPath), ".md")
		canonicalPath := canonicalPath
		t.Run(verb, func(t *testing.T) {
			mirrorPath := flatMirrorPath(repoRoot, verb)

			canonicalBytes, err := os.ReadFile(canonicalPath)
			if err != nil {
				t.Fatalf("read %s: %v", canonicalPath, err)
			}
			mirrorBytes, err := os.ReadFile(mirrorPath)
			if err != nil {
				t.Fatalf("read %s: %v", mirrorPath, err)
			}

			if !bytes.Equal(canonicalBytes, mirrorBytes) {
				t.Errorf(
					"flat mirror drifted from canonical source for verb %q:\n  canonical: %s (%d bytes)\n  mirror:    %s (%d bytes)\nThe flat mirror is the installed-consumer shape (`aether install` writes "+
						"`.claude/commands/ant/%s.md` to `~/.claude/commands/ant-%s.md`) and must be copied, not hand-edited.",
					verb, canonicalPath, len(canonicalBytes), mirrorPath, len(mirrorBytes), verb, verb,
				)
			}

			t.Run("a_mirror_content_change_would_fail", func(t *testing.T) {
				mutated := append(append([]byte{}, canonicalBytes...), []byte("\n<!-- sentinel drift -->\n")...)
				if bytes.Equal(mutated, mirrorBytes) {
					t.Fatal("expected a mutated copy of the canonical text to break byte-equality, but it did not")
				}
			})
		})
	}
}

// wrapperHostContractManifestKeys are the current manifest and completion
// field keys documented in
// .aether/docs/wrapper-host-contract.md's "Manifest and Completion Packet
// Shapes" section. Planning is deliberately stage-shaped: the retired depth
// and research proposal cards and the old whole-chain/next-iteration aliases
// are not part of this required inventory.
var wrapperHostContractManifestKeys = []string{
	"result.manifest.dispatch_manifest",
	"dispatch_manifest.execution_plan",
	"dispatch_manifest.context_capsule",
	"dispatch.brief",
	"dispatch.brief_path",
	"dispatch.skill_section",
	"dispatch.permission_profile",
	"dispatch_manifest.orchestrator_boundary_guidance",
	"result.manifest.continue_manifest",
	"result.plan_manifest.stage_manifest",
	"stage_manifest.expected_caste",
	"stage_manifest.expected_result_type",
	"result.stage_receipt",
	"result.route_stage_manifest",
	"result.route_stage_receipt",
	"result.scout_stage_manifest",
	"result.decision_cards",
	"result.iteration_card",
	"result.plan_candidate",
	"result.acceptance_command",
	"result.completion_path",
}

var retiredPlanningHostContractRequirements = []string{
	"result.planning_manifest",
	"result.depth_proposal_card",
	"result.research_proposal_card",
	"result.requires_next_iteration",
}

// TestWrapperHostContractDocumentsManifestShapes asserts
// .aether/docs/wrapper-host-contract.md carries the "Manifest and Completion
// Packet Shapes" heading and every manifest key Task 1 was required to
// document there, so wrapper prose can point at real content instead of an
// empty promise.
func TestWrapperHostContractDocumentsManifestShapes(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	contractPath := filepath.Join(repoRoot, ".aether", "docs", "wrapper-host-contract.md")
	content, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read %s: %v", contractPath, err)
	}
	text := string(content)

	if !strings.Contains(text, "## Manifest and Completion Packet Shapes") {
		t.Errorf("%s missing heading %q", contractPath, "## Manifest and Completion Packet Shapes")
	}

	for _, key := range wrapperHostContractManifestKeys {
		if !strings.Contains(text, key) {
			t.Errorf("%s missing manifest key %q", contractPath, key)
		}
	}

	for _, retired := range retiredPlanningHostContractRequirements {
		for _, required := range wrapperHostContractManifestKeys {
			if required == retired {
				t.Errorf("retired planning key %q remains a required wrapper-host manifest shape", retired)
			}
		}
	}
}

// retiredDepthVocabulary is the D-11/D-10 retired four-value colony_depth
// vocabulary (light/standard/deep/full). The live vocabulary is
// `--verification-depth <light|standard|heavy>`. This fence lives here
// rather than in platform_doc_hygiene_test.go deliberately: Wave 2 has four
// plans editing wrappers in parallel and a single shared forbidden-list file
// would serialize them.
var retiredDepthVocabulary = []string{
	"colony_depth",
	"--verification-depth deep",
	"--verification-depth full",
	"light, standard, deep, full",
}

// TestLifecycleWrappersAvoidRetiredDepthVocabulary asserts none of the 8
// canonical wrapper files (.claude and .opencode, all 4 verbs) contains the
// retired four-value depth vocabulary. GREEN today; must stay green through
// Wave 2's ceremony rewrites.
func TestLifecycleWrappersAvoidRetiredDepthVocabulary(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	for _, verb := range lifecycleWrapperVerbs {
		for _, path := range canonicalWrapperPaths(repoRoot, verb) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			text := string(content)
			for _, retired := range retiredDepthVocabulary {
				if strings.Contains(text, retired) {
					t.Errorf("%s still contains retired depth vocabulary %q", path, retired)
				}
			}
		}
	}
}

// wrapperMinMethodToEnvelopeRatio is the D-09/CMD-02 minimum ratio of
// stage-skeleton method marker lines to envelope-mechanics marker lines a
// lifecycle wrapper body must maintain.
const wrapperMinMethodToEnvelopeRatio = 3

// wrapperRatioOK reports whether methodCount is at least
// wrapperMinMethodToEnvelopeRatio times envelopeCount.
//
// This is a ratio rather than a forbidden-string list deliberately: a
// forbidden-string list is evaded by rewording ("save the envelope" becomes
// "persist the response body"), whereas a ratio fails whenever envelope
// prose grows back regardless of the words chosen, because it measures
// proportion of the whole body rather than the presence of specific
// phrases. This is the same proportion/invariant pattern
// TestBuildWorkerBriefIsMostlyTask established and CLAUDE.md's Definition
// of Done names as the preferred shape over a named-section grep.
func wrapperRatioOK(methodCount, envelopeCount int) bool {
	return methodCount >= wrapperMinMethodToEnvelopeRatio*envelopeCount
}

// wrapperHostContractPointer is the single manifest-shape reference lifecycle
// wrappers point at instead of restating envelope mechanics inline (D-03).
const wrapperHostContractPointer = ".aether/docs/wrapper-host-contract.md"

// TestLifecycleWrappersDoNotParseEnvelopeAsPrimaryJob is the CMD-02 proportion
// invariant: for every canonical lifecycle wrapper file, stage-skeleton
// method marker lines must outnumber envelope-mechanics marker lines by at
// least wrapperMinMethodToEnvelopeRatio to 1. GREEN after Waves 1 and 2;
// fails if a future edit re-inflates envelope prose or strips method prose.
func TestLifecycleWrappersDoNotParseEnvelopeAsPrimaryJob(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	for _, verb := range lifecycleWrapperVerbs {
		for _, path := range canonicalWrapperPaths(repoRoot, verb) {
			path := path
			t.Run(verb+"_"+filepath.Base(filepath.Dir(filepath.Dir(path))), func(t *testing.T) {
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read %s: %v", path, err)
				}
				text := string(content)
				methodCount := countMarkerLines(text, stageSkeletonMarkers())
				envelopeCount := countMarkerLines(text, envelopeMechanicsMarkers())
				if !wrapperRatioOK(methodCount, envelopeCount) {
					t.Errorf(
						"%s: method count %d is not at least %dx envelope count %d -- CMD-02 requires the wrapper's primary job to be method (the stage skeleton), not envelope parsing",
						path, methodCount, wrapperMinMethodToEnvelopeRatio, envelopeCount,
					)
				}
			})
		}
	}

	// contract_pointer_is_singular: build and continue each reference the
	// wrapper-host contract doc exactly once; init references it zero times
	// because it has no host-manifest step at all. Plan's managed projections
	// are generated from plan.yaml, whose runtime.contract owns the one pointer,
	// so the projection itself intentionally carries zero duplicate pointers.
	t.Run("contract_pointer_is_singular", func(t *testing.T) {
		for _, verb := range lifecycleWrapperVerbs {
			wantCount := 1
			if verb == "init" || verb == "plan" {
				wantCount = 0
			}
			for _, path := range canonicalWrapperPaths(repoRoot, verb) {
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read %s: %v", path, err)
				}
				gotCount := strings.Count(string(content), wrapperHostContractPointer)
				if gotCount != wantCount {
					t.Errorf(
						"%s: references %q %d time(s), want %d -- D-03 keeps envelope mechanics in the shared contract; plan points there once from canonical plan.yaml instead of duplicating it in managed projections",
						path, wrapperHostContractPointer, gotCount, wantCount,
					)
				}
			}
		}

		planSource, err := os.ReadFile(filepath.Join(repoRoot, ".aether", "commands", "plan.yaml"))
		if err != nil {
			t.Fatalf("read canonical plan wrapper source: %v", err)
		}
		if got := strings.Count(string(planSource), wrapperHostContractPointer); got != 1 {
			t.Errorf("canonical plan.yaml references %q %d time(s), want exactly 1", wrapperHostContractPointer, got)
		}
	})

	// a_wrapper_dominated_by_envelope_prose_would_fail: a synthetic in-memory
	// body with one method line and five envelope-mechanics lines must be
	// rejected by the ratio check, mirroring
	// cmd/plan_wrapper_cards_test.go's a_one_sided_heading_addition_would_fail
	// negative control and proving this assertion has teeth.
	t.Run("a_wrapper_dominated_by_envelope_prose_would_fail", func(t *testing.T) {
		synthetic := strings.Join([]string{
			"**Purpose:** Do the thing this stage exists for.",
			"Parse `result.manifest.dispatch_manifest` field by field before doing anything else.",
			"Parse `result.completion_path` the same way, one field at a time.",
			"Save the full JSON envelope to a temp file for later inspection.",
			"Save the JSON envelope again in a second location for safety.",
			"Write the manifest to a temporary manifest file outside the repo before reading it back.",
		}, "\n")
		methodCount := countMarkerLines(synthetic, stageSkeletonMarkers())
		envelopeCount := countMarkerLines(synthetic, envelopeMechanicsMarkers())
		if wrapperRatioOK(methodCount, envelopeCount) {
			t.Fatalf(
				"expected a synthetic body with %d method line(s) and %d envelope line(s) to fail the ratio check, but it passed",
				methodCount, envelopeCount,
			)
		}
	})
}

// requiredStructuredBlockMarkers are the three structured blocks plus the
// state-carry heading every canonical lifecycle wrapper must carry (CMD-01).
var requiredStructuredBlockMarkers = []string{
	"<success_criteria>",
	"<failure_modes>",
	"<read_only>",
	"## Required Cross-Stage State",
}

// retiredCrossStageVocabulary are the v5.4.0 state-carry names the D-06
// modernization replaced. A current state-carry section must name none of
// them.
var retiredCrossStageVocabulary = []string{
	"colony_depth",
	"visual_mode",
	"verbose_mode",
	"suggest_enabled",
	"synthesis_status",
}

// extractStateCarrySection returns the text running from the
// "## Required Cross-Stage State" heading (exclusive) to the next level-2
// heading (exclusive), or "" if the heading is not present.
func extractStateCarrySection(text string) string {
	const heading = "## Required Cross-Stage State"
	idx := strings.Index(text, heading)
	if idx == -1 {
		return ""
	}
	rest := text[idx+len(heading):]
	var out []string
	for _, line := range strings.Split(rest, "\n") {
		if strings.HasPrefix(line, "## ") {
			break
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// backtickedValuePattern matches a single backtick-quoted span, used to
// count how many concrete values a state-carry section names.
var backtickedValuePattern = regexp.MustCompile("`[^`]+`")

// TestLifecycleWrappersCarryStructuredBlocks asserts all eight canonical
// lifecycle wrapper files contain the three structured block markers and the
// state-carry heading (CMD-01), and that each state-carry section names
// current vocabulary rather than retired v5.4.0 terms.
func TestLifecycleWrappersCarryStructuredBlocks(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	for _, verb := range lifecycleWrapperVerbs {
		for _, path := range canonicalWrapperPaths(repoRoot, verb) {
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			text := string(content)
			for _, marker := range requiredStructuredBlockMarkers {
				if !strings.Contains(text, marker) {
					t.Errorf("%s missing structured block marker %q", path, marker)
				}
			}
		}
	}

	t.Run("state_carry_uses_current_vocabulary", func(t *testing.T) {
		for _, verb := range lifecycleWrapperVerbs {
			for _, path := range canonicalWrapperPaths(repoRoot, verb) {
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read %s: %v", path, err)
				}
				section := extractStateCarrySection(string(content))
				if section == "" {
					t.Errorf("%s: could not locate a '## Required Cross-Stage State' section", path)
					continue
				}
				if matches := backtickedValuePattern.FindAllString(section, -1); len(matches) < 4 {
					t.Errorf("%s: state-carry section names only %d backticked value(s), want at least 4", path, len(matches))
				}
				for _, retired := range retiredCrossStageVocabulary {
					if strings.Contains(section, retired) {
						t.Errorf("%s: state-carry section still names retired vocabulary %q", path, retired)
					}
				}
			}
		}
	})
}

// specialistCommandSurfaces are the CMD-04 protected surfaces: seven Claude
// wrappers, seven byte-identical OpenCode mirrors, and sage's three agent
// definitions. `sage` has NO slash command on any platform -- it is invoked
// only as an agent, so its "keeps working unchanged" surface is the agent
// definition trio (.claude/agents/ant/aether-sage.md,
// .opencode/agents/aether-sage.md, .codex/agents/aether-sage.toml), not a
// missing wrapper file. A future reader who notices no ant-sage.md should
// read this comment before assuming coverage is incomplete.
var specialistCommandSurfaces = []string{
	".claude/commands/ant/chaos.md",
	".claude/commands/ant/archaeology.md",
	".claude/commands/ant/dream.md",
	".claude/commands/ant/oracle.md",
	".claude/commands/ant/swarm.md",
	".claude/commands/ant/colonize.md",
	".claude/commands/ant/council.md",
	".opencode/commands/ant/chaos.md",
	".opencode/commands/ant/archaeology.md",
	".opencode/commands/ant/dream.md",
	".opencode/commands/ant/oracle.md",
	".opencode/commands/ant/swarm.md",
	".opencode/commands/ant/colonize.md",
	".opencode/commands/ant/council.md",
	".claude/agents/ant/aether-sage.md",
	".opencode/agents/aether-sage.md",
	".codex/agents/aether-sage.toml",
}

// specialistCommandSurfaceHashes pins each specialistCommandSurfaces path to
// the lowercase hex SHA-256 of its bytes, computed at the time this test was
// written (`shasum -a 256` over all seventeen paths). The ledger is
// deliberately brittle: CMD-04 requires these commands to keep working
// unchanged through milestone v1.25, so ANY edit -- even a whitespace fix --
// must update the recorded hash in the same commit with a stated reason.
// This is a change-detection fence, not an integrity control against an
// adversary (see threat T-165-06-05 in 165-06-PLAN.md).
var specialistCommandSurfaceHashes = map[string]string{
	// chaos.md hashes updated 2026-08-17 (critics-bring-solutions round):
	// every finding now carries a `suggested_hardening` — the concrete step
	// that closes the gap, shown per-finding in the report and in the JSON
	// schema. Content addition only; the 5-scenario investigation flow,
	// Tester's Law, and credit-resilience framing are unchanged.
	".claude/commands/ant/chaos.md":       "16f24da7ea50ec125ca5e754ea7f7c0c8a06aed3a8d2e7f8d4f128a1b71115a0",
	".claude/commands/ant/archaeology.md": "fada154f485134084ec39eccdf129f0bc2d4b0750e5d055e22620f060c3bc2ba",
	".claude/commands/ant/dream.md":       "06158585eeafe94748086c871b02f7a9c2aa8165a8cd0409dcc98663fc94609e",
	".claude/commands/ant/oracle.md":      "bdd9c4cbaae5bf81360d50f671dfef334b179f58b8e4ba19e9552353553e597f",
	// swarm.md hashes updated 2026-08-16 (reclaim sweep): documented
	// swarm-findings-read / swarm-cleanup as the inspection and housekeeping
	// affordances for the runtime-recorded swarm state (RECLAIM-07). Content
	// addition only; the swarm flow itself is unchanged.
	".claude/commands/ant/swarm.md": "a8f81e0474550915399d66a05effe03a41b5ed073e189242bc070770ae5f8529",
	// Phase 198.2 plan 01 (WIRE-01): colonize wrappers now prepend the colony
	// memory capsule from the manifest, on all three hand-maintained copies.
	// Behaviour, tools and flow are otherwise unchanged; the capsule block is
	// the only edit, and the two copies below stay byte-identical.
	".claude/commands/ant/colonize.md":      "f512474c4d81037d2c047b70d3a2aa9fb4dfc5d36cd55ab247f15d57ee35dd4e",
	".claude/commands/ant/council.md":       "c7fbb1923890e84687fb23c40d8b8e88d6543ef0bf9fb8d5c931b4d1d853e568",
	".opencode/commands/ant/chaos.md":       "16f24da7ea50ec125ca5e754ea7f7c0c8a06aed3a8d2e7f8d4f128a1b71115a0",
	".opencode/commands/ant/archaeology.md": "fada154f485134084ec39eccdf129f0bc2d4b0750e5d055e22620f060c3bc2ba",
	".opencode/commands/ant/dream.md":       "06158585eeafe94748086c871b02f7a9c2aa8165a8cd0409dcc98663fc94609e",
	".opencode/commands/ant/oracle.md":      "bdd9c4cbaae5bf81360d50f671dfef334b179f58b8e4ba19e9552353553e597f",
	".opencode/commands/ant/swarm.md":       "a8f81e0474550915399d66a05effe03a41b5ed073e189242bc070770ae5f8529",
	".opencode/commands/ant/colonize.md":    "f512474c4d81037d2c047b70d3a2aa9fb4dfc5d36cd55ab247f15d57ee35dd4e",
	".opencode/commands/ant/council.md":     "c7fbb1923890e84687fb23c40d8b8e88d6543ef0bf9fb8d5c931b4d1d853e568",
	// aether-sage hashes updated 2026-08-21 (no-change vocabulary round): the
	// worker response contract gained completed_no_change as a first-class
	// success (ruling D6), and every agent definition that spells out its own
	// status list was updated in lockstep -- a definition still offering only
	// completed|failed|blocked contradicts the runtime brief the same worker
	// receives, and pushes it back to the choose-between-failing-and-faking
	// bind the status exists to remove. Status list only; sage's behaviour,
	// tools and flow are unchanged.
	".claude/agents/ant/aether-sage.md": "912f971e37130449211b6db3520948c808d845fb40a9203efd9afd0568688353",
	".opencode/agents/aether-sage.md":   "e9bb127682e1cdf668e219ecb18168b7a62f9d18c67bf69e28ef672178708c70",
	".codex/agents/aether-sage.toml":    "307dbfece02d48abe20166579526d8d9a07590f6f48a671a56a8d800a08e71cb",
}

// specialistCommandGuideVerbs are the seven CMD-04 surfaces that also have a
// command-guide entry. `sage` is intentionally excluded: it has no
// command-guide entry on any platform, so the existence check and content
// ledger above are sage's only failable CMD-04 requirements.
var specialistCommandGuideVerbs = []string{
	"archaeology", "chaos", "colonize", "council", "dream", "oracle", "swarm",
}

// TestSpecialistCommandSurfacesUnchanged is the CMD-04 fence: it fails if a
// surface is dropped, if any of the seventeen files' bytes change, or if a
// command-guide-backed verb stops resolving -- rather than relying on
// nobody having touched the files.
func TestSpecialistCommandSurfacesUnchanged(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	if len(specialistCommandSurfaces) != 17 {
		t.Fatalf(
			"specialistCommandSurfaces has %d entries, want exactly 17 -- a dropped surface must fail this count rather than silently reducing CMD-04 coverage",
			len(specialistCommandSurfaces),
		)
	}

	for _, rel := range specialistCommandSurfaces {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			path := filepath.Join(repoRoot, rel)
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("specialist command surface %s does not exist: %v", rel, err)
			}

			wantHash, ok := specialistCommandSurfaceHashes[rel]
			if !ok {
				t.Fatalf("%s has no recorded SHA-256 in specialistCommandSurfaceHashes", rel)
			}
			sum := sha256.Sum256(content)
			gotHash := hex.EncodeToString(sum[:])
			if gotHash != wantHash {
				t.Errorf(
					"%s hash changed: got %s, want %s -- CMD-04 requires this command to keep working unchanged through milestone v1.25. The ledger is deliberately brittle: a legitimate future edit updates the recorded hash in the same commit with a stated reason",
					rel, gotHash, wantHash,
				)
			}
		})
	}

	t.Run("command_guide_reachability", func(t *testing.T) {
		for _, verb := range specialistCommandGuideVerbs {
			if _, err := buildCommandGuide(verb, "claude"); err != nil {
				t.Errorf("aether command-guide %s: %v -- CMD-04's runtime-reachability check requires this verb to resolve in-process", verb, err)
			}
		}
	})
}

// extractReadOnlyBlock returns the text between <read_only> and
// </read_only>, trimmed, or the empty string when the block is absent.
// Models its slicing on extractStateCarrySection above.
func extractReadOnlyBlock(text string) string {
	const openTag = "<read_only>"
	const closeTag = "</read_only>"
	start := strings.Index(text, openTag)
	if start == -1 {
		return ""
	}
	start += len(openTag)
	rest := text[start:]
	end := strings.Index(rest, closeTag)
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(rest[:end])
}

// permissiveReadOnlyReadPhrase is the WR-01 contradictory phrasing: it tells
// the model it may read state files, while a Guardrails "Do NOT read or
// write" bullet elsewhere in the same file forbids exactly that. plan.md
// already carries the reconciled formulation and never uses this phrase.
const permissiveReadOnlyReadPhrase = "may read but never write"

// noHandReadWriteMarkers are the two reconciled phrasings that assert a
// no-hand-read-or-write boundary: plan.md's prose formulation and init.md's
// bulleted-file-list formulation.
var noHandReadWriteMarkers = []string{
	"never reads or writes",
	"never writes these files by hand",
}

// guardrailsForbidsHandReadWrite is the literal Guardrails bullet fragment
// that, when present, requires the <read_only> block to agree with it.
const guardrailsForbidsHandReadWrite = "Do NOT read or write"

// lifecycleWrapperSurfaces returns every surface for a verb: both canonical
// paths (.claude, .opencode) plus the flat installed-consumer mirror.
func lifecycleWrapperSurfaces(repoRoot, verb string) []string {
	surfaces := canonicalWrapperPaths(repoRoot, verb)
	return append(surfaces, flatMirrorPath(repoRoot, verb))
}

// relWrapperPath returns path relative to repoRoot for use in subtest names
// and error messages, so a failure names the exact repo-relative file.
func relWrapperPath(repoRoot, path string) string {
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return path
	}
	return rel
}

// TestLifecycleWrapperReadOnlyBlocksAreConsistent is the WR-01 fence: no
// lifecycle wrapper may tell the model two different things about reading
// state files. It iterates all four verbs (build, continue, plan, init)
// across both canonical paths and the flat mirror -- 12 surfaces total.
func TestLifecycleWrapperReadOnlyBlocksAreConsistent(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	var surfaces []string
	for _, verb := range lifecycleWrapperVerbs {
		surfaces = append(surfaces, lifecycleWrapperSurfaces(repoRoot, verb)...)
	}

	t.Run("read_only_block_exists", func(t *testing.T) {
		for _, path := range surfaces {
			path := path
			t.Run(relWrapperPath(repoRoot, path), func(t *testing.T) {
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read %s: %v", path, err)
				}
				if extractReadOnlyBlock(string(content)) == "" {
					t.Errorf("%s: missing a non-empty <read_only> block", relWrapperPath(repoRoot, path))
				}
			})
		}
	})

	t.Run("no_permissive_read_phrasing", func(t *testing.T) {
		for _, path := range surfaces {
			path := path
			t.Run(relWrapperPath(repoRoot, path), func(t *testing.T) {
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read %s: %v", path, err)
				}
				if strings.Contains(string(content), permissiveReadOnlyReadPhrase) {
					t.Errorf(
						"%s: <read_only> block uses the permissive phrase %q, which contradicts a Guardrails bullet elsewhere in the same file forbidding hand reads/writes of the same state -- the model receives two conflicting instructions about the same files and will follow whichever it read last. Use the reconciled formulation plan.md already carries: this wrapper never reads or writes, by hand, colony state -- it reads runtime state only through runtime commands such as `aether status`, never by opening the JSON. See Phase 165 review WR-01.",
						relWrapperPath(repoRoot, path), permissiveReadOnlyReadPhrase,
					)
				}
			})
		}
	})

	t.Run("read_only_block_matches_guardrails", func(t *testing.T) {
		for _, path := range surfaces {
			path := path
			t.Run(relWrapperPath(repoRoot, path), func(t *testing.T) {
				content, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("read %s: %v", path, err)
				}
				text := string(content)
				if !strings.Contains(text, guardrailsForbidsHandReadWrite) {
					// Conditional: only surfaces whose Guardrails forbid hand
					// reads/writes must have a matching <read_only> body.
					// init.md's guardrails live in its YAML source, not a
					// "## Guardrails" heading, so it is skipped here and
					// covered instead by no_permissive_read_phrasing.
					return
				}
				block := extractReadOnlyBlock(text)
				for _, marker := range noHandReadWriteMarkers {
					if strings.Contains(block, marker) {
						return
					}
				}
				t.Errorf(
					"%s: Guardrails forbid hand reads/writes (%q) but the <read_only> block does not assert a matching no-hand-read-or-write boundary (expected one of %v) -- the wrapper tells the model two different things about the same files. See Phase 165 review WR-01.",
					relWrapperPath(repoRoot, path), guardrailsForbidsHandReadWrite, noHandReadWriteMarkers,
				)
			})
		}
	})
}

// buildWrapperTripletPaths returns the three hand-maintained build wrapper
// copies in canonical-first order: the canonical Claude wrapper, the flat
// installed-consumer Claude mirror, and the OpenCode copy. There is no
// generator for these — they are byte-identical by policy, and this ordering
// makes the canonical one the comparison base.
func buildWrapperTripletPaths(repoRoot string) []string {
	return []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant", "build.md"),
		flatMirrorPath(repoRoot, "build"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "build.md"),
	}
}

// planWrapperTripletPaths returns the three hand-maintained plan wrapper
// copies in the same canonical-first order buildWrapperTripletPaths uses
// (198.2-01): the canonical Claude wrapper, the flat installed-consumer
// Claude mirror (the file an installed Claude Code session actually runs),
// and the OpenCode copy.
func planWrapperTripletPaths(repoRoot string) []string {
	return []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant", "plan.md"),
		flatMirrorPath(repoRoot, "plan"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "plan.md"),
	}
}

// colonizeWrapperTripletPaths returns the three hand-maintained colonize
// wrapper copies in the same canonical-first order (198.2-01).
func colonizeWrapperTripletPaths(repoRoot string) []string {
	return []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant", "colonize.md"),
		flatMirrorPath(repoRoot, "colonize"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "colonize.md"),
	}
}

// TestLifecycleWrappersCarryCoherentJobContract asserts all three build
// wrapper copies describe the Phase 195 coherent-job, task-receipt, recovery
// and check-in contract exactly as the Go runtime implements it, and that
// they remain byte-identical to each other while doing so.
//
// The forbidden-anchor half is the load-bearing part: before Phase 195 the
// Team Check-In stage said `checkin_requested` is false because
// `--no-checkin` was passed, which the one-worker fast path (D-11) made
// untrue. A wrapper that skipped the stage on that stale reading would drop
// the runtime's compact summary entirely.
func TestLifecycleWrappersCarryCoherentJobContract(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	paths := buildWrapperTripletPaths(repoRoot)
	bodies := make([][]byte, 0, len(paths))

	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		bodies = append(bodies, content)

		rel, relErr := filepath.Rel(repoRoot, path)
		if relErr != nil {
			rel = path
		}
		t.Run(rel, func(t *testing.T) {
			assertBuildCoherentJobContract(t, rel, string(content))
		})
	}

	t.Run("triplet_is_byte_identical", func(t *testing.T) {
		for i := 1; i < len(bodies); i++ {
			if !bytes.Equal(bodies[0], bodies[i]) {
				t.Errorf(
					"build wrapper copies drifted: %s (%d bytes) != %s (%d bytes) — the three copies are hand-maintained and must be byte-identical",
					paths[0], len(bodies[0]), paths[i], len(bodies[i]),
				)
			}
		}
	})
}
