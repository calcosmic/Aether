package cmd

import (
	"bytes"
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
// canonical nested sources (.claude/commands/ant/<verb>.md). This is
// currently RED for build and init (both proven drifted by research) and
// GREEN for plan and continue. Task 3 of this plan resyncs build and init
// and turns this fully green.
func TestLifecycleFlatMirrorsMatchCanonical(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("failed to find repo root: %v", err)
	}

	for _, verb := range lifecycleWrapperVerbs {
		verb := verb
		t.Run(verb, func(t *testing.T) {
			canonicalPath := filepath.Join(repoRoot, ".claude", "commands", "ant", verb+".md")
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

// wrapperHostContractManifestKeys are the 15 manifest and completion field
// keys Task 1 of this plan documents in
// .aether/docs/wrapper-host-contract.md's "Manifest and Completion Packet
// Shapes" section.
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
	"result.plan_manifest",
	"result.planning_manifest",
	"result.depth_proposal_card",
	"result.research_proposal_card",
	"result.completion_path",
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

	// contract_pointer_is_singular: build, plan, and continue each reference
	// the wrapper-host contract doc exactly once; init references it zero
	// times because it has no host-manifest step at all.
	t.Run("contract_pointer_is_singular", func(t *testing.T) {
		for _, verb := range lifecycleWrapperVerbs {
			wantCount := 1
			if verb == "init" {
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
						"%s: references %q %d time(s), want %d -- D-03 puts envelope mechanics in the contract doc referenced exactly once per host-backed wrapper, and init has no host-manifest step at all",
						path, wrapperHostContractPointer, gotCount, wantCount,
					)
				}
			}
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
