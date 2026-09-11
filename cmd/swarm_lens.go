package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/events"
)

// Lens identifiers (SYN-202-05, 202-CLASSIC-SYNTHESIS.md). These are the
// four genuinely distinct, evidence-producing angles Swarm's investigation
// wave attacks a defect from -- the modern re-expression of Classic's
// Archaeologist / Pattern Hunter / Error Analyst / Web Researcher quartet
// (OLD-01). Selecting four castes is not, on its own, what satisfies
// LIVE-03: the structured swarmHypothesis comparison below (SYN-202-06) is.
const (
	swarmLensErrorPath        = "error-path"
	swarmLensPattern          = "pattern"
	swarmLensHistory          = "history"
	swarmLensExternalEvidence = "external-evidence"
)

// swarmLensDef names one of Swarm's four investigation lenses: its stable
// identifier, its human label, the caste that runs it, and the brief text
// that caste's worker is given. Brief is swarmTaskForCaste's own text for
// the same caste, kept verbatim rather than duplicated (202-PATTERNS.md).
type swarmLensDef struct {
	ID    string
	Label string
	Caste string
	Brief string
}

// swarmLenses is the fixed, ordered set of Swarm's four investigation
// lenses. Never mutated at runtime. TestFourSwarmLensesProduceDistinctEvidence
// asserts every field here is genuinely distinct across all four -- a lens
// sharing a brief, a caste, or an identifier with another fails that test by
// name.
var swarmLenses = []swarmLensDef{
	{ID: swarmLensErrorPath, Label: "Error Path", Caste: "tracker", Brief: swarmTaskForCaste("tracker")},
	{ID: swarmLensPattern, Label: "Repository Pattern", Caste: "scout", Brief: swarmTaskForCaste("scout")},
	{ID: swarmLensHistory, Label: "Git History", Caste: "archaeologist", Brief: swarmTaskForCaste("archaeologist")},
	{ID: swarmLensExternalEvidence, Label: "External Evidence", Caste: "oracle", Brief: swarmTaskForCaste("oracle")},
}

// swarmLensForCaste returns the lens definition a caste runs, if any. Not
// every dispatchable Swarm caste is a lens (gatekeeper and medic are
// additional, Queen-selected investigation castes, not lenses).
func swarmLensForCaste(caste string) (swarmLensDef, bool) {
	for _, lens := range swarmLenses {
		if lens.Caste == caste {
			return lens, true
		}
	}
	return swarmLensDef{}, false
}

// swarmLensByID looks up a lens by its stable identifier.
func swarmLensByID(id string) (swarmLensDef, bool) {
	for _, lens := range swarmLenses {
		if lens.ID == id {
			return lens, true
		}
	}
	return swarmLensDef{}, false
}

// distinctSwarmLensViolations reports every way a lens set fails to be
// genuinely distinct -- a reused identifier, a reused caste, or a reused
// brief. An empty result means the set is distinct. Used both to assert the
// production swarmLenses slice is distinct and, in tests, to prove the check
// itself can fail against a deliberately duplicated fixture.
func distinctSwarmLensViolations(lenses []swarmLensDef) []string {
	var issues []string
	seenID := map[string]string{}
	seenCaste := map[string]string{}
	seenBrief := map[string]string{}
	for _, lens := range lenses {
		if prior, ok := seenID[lens.ID]; ok {
			issues = append(issues, fmt.Sprintf("lens id %q reused by %q and %q", lens.ID, prior, lens.Label))
		} else {
			seenID[lens.ID] = lens.Label
		}
		if prior, ok := seenCaste[lens.Caste]; ok {
			issues = append(issues, fmt.Sprintf("caste %q reused by lens %q and %q", lens.Caste, prior, lens.Label))
		} else {
			seenCaste[lens.Caste] = lens.Label
		}
		if prior, ok := seenBrief[lens.Brief]; ok {
			issues = append(issues, fmt.Sprintf("brief reused by lens %q and %q", prior, lens.Label))
		} else {
			seenBrief[lens.Brief] = lens.Label
		}
	}
	return issues
}

// swarmHypothesis is Swarm's own sibling type to Oracle's
// oracleWorkerResponse (cmd/oracle_loop.go) -- same field shape, never an
// extension or import of it (202-PATTERNS.md "Sibling type over
// cross-package extension"). One hypothesis is produced per lens that
// reported a usable claim.
type swarmHypothesis struct {
	Lens           string                    `json:"lens"`
	LensLabel      string                    `json:"lens_label"`
	Caste          string                    `json:"caste"`
	WorkerName     string                    `json:"worker_name"`
	Claim          string                    `json:"claim"`
	Confidence     *int                      `json:"confidence,omitempty"`
	Evidence       []swarmHypothesisEvidence `json:"evidence,omitempty"`
	Contradictions []string                  `json:"contradictions,omitempty"`
	ProposedRepair string                    `json:"proposed_repair,omitempty"`
}

// swarmHypothesisEvidence preserves where a claim came from -- a file, a
// commit, or an external source -- so it can be traced back, not just
// asserted.
type swarmHypothesisEvidence struct {
	Title    string `json:"title"`
	Location string `json:"location"`
	Kind     string `json:"kind"`
}

// swarmNonReportingLens names one of Swarm's four lenses that produced no
// hypothesis in this investigation wave, and why -- either it never ran, or
// it ran but returned no usable claim.
type swarmNonReportingLens struct {
	Lens   string `json:"lens"`
	Label  string `json:"label"`
	Reason string `json:"reason"`
}

// swarmLensRawResponse captures the fields a worker's response file may
// carry that swarmWorkerResponse's own contract (cmd/swarm_cmd.go) does not
// -- confidence, richer structured evidence, and self-reported
// contradictions. Reading it is entirely additive: swarmWorkerResponse's own
// fields and JSON contract are never touched by this file, per Task 1's
// constraint that the external completion contract stays untouched.
type swarmLensRawResponse struct {
	Confidence     *int                      `json:"confidence,omitempty"`
	Evidence       []swarmHypothesisEvidence `json:"structured_evidence,omitempty"`
	Contradictions []string                  `json:"contradictions,omitempty"`
}

// loadSwarmLensRawResponse re-reads a lens worker's already-written response
// file for the confidence/structured-evidence/contradiction fields
// swarmWorkerResponse's contract does not carry. A missing or unparsable
// file yields a zero value (unstated confidence, no structured evidence, no
// self-reported contradictions) -- never an error that could block the
// comparison step.
func loadSwarmLensRawResponse(path string) swarmLensRawResponse {
	path = strings.TrimSpace(path)
	if path == "" {
		return swarmLensRawResponse{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return swarmLensRawResponse{}
	}
	var raw swarmLensRawResponse
	if err := json.Unmarshal(data, &raw); err != nil {
		return swarmLensRawResponse{}
	}
	return raw
}

// hypothesesFromSwarmRuns maps the investigation wave's own
// swarmWorkerExecution/swarmWorkerResponse values -- built by
// executeSwarmWave/loadSwarmWorkerResponse, the runtime's real
// response-loading path -- into structured hypotheses, one per lens that
// reported a usable claim. It also returns every declared lens that did not
// run in this investigation wave, or that ran but reported no usable claim,
// paired with a reason.
func hypothesesFromSwarmRuns(runs []swarmWorkerExecution) ([]swarmHypothesis, []swarmNonReportingLens) {
	reported := make(map[string]bool, len(swarmLenses))
	var hypotheses []swarmHypothesis
	var missing []swarmNonReportingLens

	for _, run := range runs {
		lens, ok := swarmLensForCaste(run.Caste)
		if !ok {
			continue
		}
		reported[lens.ID] = true

		claim := strings.TrimSpace(run.Response.RootCause)
		if claim == "" {
			claim = strings.TrimSpace(run.Response.Summary)
		}
		if claim == "" {
			claim = strings.TrimSpace(run.Summary)
		}
		if claim == "" {
			missing = append(missing, swarmNonReportingLens{
				Lens:  lens.ID,
				Label: lens.Label,
				Reason: fmt.Sprintf("%s worker %s returned no usable claim (status %s)",
					lens.Label, emptyFallback(run.Name, run.Caste), emptyFallback(run.Status, "unknown")),
			})
			continue
		}

		raw := loadSwarmLensRawResponse(run.ResponsePath)
		evidence := raw.Evidence
		if len(evidence) == 0 {
			for _, item := range run.Response.Evidence {
				item = strings.TrimSpace(item)
				if item == "" {
					continue
				}
				evidence = append(evidence, swarmHypothesisEvidence{
					Title:    item,
					Location: item,
					Kind:     lens.ID,
				})
			}
		}

		hypotheses = append(hypotheses, swarmHypothesis{
			Lens:           lens.ID,
			LensLabel:      lens.Label,
			Caste:          run.Caste,
			WorkerName:     run.Name,
			Claim:          claim,
			Confidence:     raw.Confidence,
			Evidence:       evidence,
			Contradictions: append([]string{}, raw.Contradictions...),
			ProposedRepair: strings.TrimSpace(run.Response.ProposedFix),
		})
	}

	for _, lens := range swarmLenses {
		if !reported[lens.ID] {
			missing = append(missing, swarmNonReportingLens{
				Lens:   lens.ID,
				Label:  lens.Label,
				Reason: fmt.Sprintf("%s lens did not run in this investigation wave", lens.Label),
			})
		}
	}

	return hypotheses, missing
}

// swarmSharedCause is one root cause two or more lenses independently named
// -- the "agreement" half of Classic's cross-compare step (OLD-02).
type swarmSharedCause struct {
	Cause  string
	Lenses []string
}

// swarmContradiction is one pair of lenses whose claims a lens itself
// flagged as conflicting -- the "disagreement" half of Classic's
// cross-compare step (OLD-02).
type swarmContradiction struct {
	LensA  string
	ClaimA string
	LensB  string
	ClaimB string
	Note   string
}

// swarmRankedRepair is one candidate repair, corroborated by the lenses that
// proposed it, ordered by compareSwarmHypotheses' ranking.
type swarmRankedRepair struct {
	Repair     string
	Lenses     []string
	Confidence *int
	// Synthesized is true when no lens proposed this exact repair text --
	// it was built from the most corroborated finding because at least one
	// lens reported evidence but none proposed a specific fix (see
	// compareSwarmHypotheses).
	Synthesized bool
}

// swarmComparison is the single, first-class artifact SYN-202-06 requires:
// every hypothesis the investigation wave produced, where they agree, where
// they disagree, the ranked candidate repairs, and the one selected repair
// with a stated reason -- rendered as one card by renderSwarmHypothesisCard
// and consumed by the fix wave via swarmFixWaveBrief.
type swarmComparison struct {
	Hypotheses      []swarmHypothesis
	SharedCauses    []swarmSharedCause
	Contradictions  []swarmContradiction
	Ranked          []swarmRankedRepair
	Selected        *swarmRankedRepair
	SelectionReason string
	MissingLenses   []swarmNonReportingLens
}

// compareSwarmHypotheses is Swarm's structured replacement for
// renderSwarmFindingSummary's free-text concatenation as the fix wave's
// input (SYN-202-06). It surfaces agreement, surfaces disagreement, and
// ranks candidate repairs by cross-lens corroboration first and stated
// confidence second.
//
// When at least one lens reported evidence but none proposed a specific
// repair text, one synthesized candidate is ranked from the most
// corroborated finding, so genuine investigation evidence is never thrown
// away merely because no lens happened to word a fix. Only when NO lens
// reported any usable evidence at all does the comparison return no ranked
// repair -- the honest "nothing to act on" case.
func compareSwarmHypotheses(hypotheses []swarmHypothesis, missing []swarmNonReportingLens) swarmComparison {
	comparison := swarmComparison{
		Hypotheses:    append([]swarmHypothesis{}, hypotheses...),
		MissingLenses: append([]swarmNonReportingLens{}, missing...),
	}
	comparison.SharedCauses = detectSwarmSharedCauses(hypotheses)
	comparison.Contradictions = detectSwarmContradictions(hypotheses)
	comparison.Ranked = rankSwarmRepairs(hypotheses, comparison.SharedCauses)

	if len(comparison.Ranked) == 0 {
		comparison.SelectionReason = "No lens produced usable evidence, so no repair was ranked or selected."
		return comparison
	}

	selected := comparison.Ranked[0]
	comparison.Selected = &selected
	comparison.SelectionReason = swarmRepairSelectionReason(comparison.Ranked)
	return comparison
}

// detectSwarmSharedCauses groups hypotheses whose claims match once
// trimmed and case-folded -- two or more lenses independently naming the
// same root cause.
func detectSwarmSharedCauses(hypotheses []swarmHypothesis) []swarmSharedCause {
	type group struct {
		cause  string
		lenses []string
	}
	order := []string{}
	groups := map[string]*group{}
	for _, h := range hypotheses {
		key := strings.ToLower(strings.TrimSpace(h.Claim))
		if key == "" {
			continue
		}
		g, ok := groups[key]
		if !ok {
			g = &group{cause: h.Claim}
			groups[key] = g
			order = append(order, key)
		}
		g.lenses = append(g.lenses, h.Lens)
	}
	var out []swarmSharedCause
	for _, key := range order {
		g := groups[key]
		if len(g.lenses) < 2 {
			continue
		}
		out = append(out, swarmSharedCause{Cause: g.cause, Lenses: g.lenses})
	}
	return out
}

// detectSwarmContradictions surfaces a contradiction between two lenses when
// one lens's own self-reported contradiction note names another lens (by
// its lens identifier or its label). This mirrors Classic's Queen
// cross-compare step (OLD-02) -- "flagged disagreement" -- as a structural,
// testable check rather than free-text prose nobody weighs.
func detectSwarmContradictions(hypotheses []swarmHypothesis) []swarmContradiction {
	var out []swarmContradiction
	seenPairs := map[string]bool{}
	for _, h := range hypotheses {
		for _, note := range h.Contradictions {
			lower := strings.ToLower(note)
			for _, other := range hypotheses {
				if other.Lens == h.Lens {
					continue
				}
				if !strings.Contains(lower, strings.ToLower(other.Lens)) && !strings.Contains(lower, strings.ToLower(other.LensLabel)) {
					continue
				}
				key := swarmContradictionPairKey(h.Lens, other.Lens)
				if seenPairs[key] {
					continue
				}
				seenPairs[key] = true
				out = append(out, swarmContradiction{
					LensA:  h.Lens,
					ClaimA: h.Claim,
					LensB:  other.Lens,
					ClaimB: other.Claim,
					Note:   note,
				})
			}
		}
	}
	return out
}

// swarmContradictionPairKey returns an order-independent key for a pair of
// lens identifiers, so A-vs-B and B-vs-A never both surface as separate
// contradiction entries.
func swarmContradictionPairKey(a, b string) string {
	if a > b {
		a, b = b, a
	}
	return a + "|" + b
}

// rankSwarmRepairs groups hypotheses by their proposed repair text and
// orders the groups by cross-lens corroboration first, then by whether a
// stated confidence exists, then by the stated confidence value. When no
// hypothesis proposed a repair text but at least one lens reported evidence,
// a single synthesized candidate is built from the most corroborated shared
// cause (or, absent one, the first hypothesis) so genuine investigation
// findings still reach the fix wave.
func rankSwarmRepairs(hypotheses []swarmHypothesis, sharedCauses []swarmSharedCause) []swarmRankedRepair {
	type group struct {
		repair         string
		lenses         []string
		bestConfidence *int
	}
	order := []string{}
	groups := map[string]*group{}
	for _, h := range hypotheses {
		repair := strings.TrimSpace(h.ProposedRepair)
		if repair == "" {
			continue
		}
		key := strings.ToLower(repair)
		g, ok := groups[key]
		if !ok {
			g = &group{repair: repair}
			groups[key] = g
			order = append(order, key)
		}
		g.lenses = append(g.lenses, h.Lens)
		if h.Confidence != nil {
			if g.bestConfidence == nil || *h.Confidence > *g.bestConfidence {
				val := *h.Confidence
				g.bestConfidence = &val
			}
		}
	}

	ranked := make([]swarmRankedRepair, 0, len(order))
	for _, key := range order {
		g := groups[key]
		ranked = append(ranked, swarmRankedRepair{Repair: g.repair, Lenses: g.lenses, Confidence: g.bestConfidence})
	}
	sortSwarmRankedRepairs(ranked)

	if len(ranked) > 0 || len(hypotheses) == 0 {
		return ranked
	}

	// No lens proposed a repair text, but at least one lens reported
	// evidence -- synthesize one candidate from the most corroborated
	// shared cause (falling back to the first hypothesis) rather than
	// discarding genuine investigation findings.
	claim := hypotheses[0].Claim
	lenses := []string{hypotheses[0].Lens}
	if len(sharedCauses) > 0 {
		claim = sharedCauses[0].Cause
		lenses = append([]string{}, sharedCauses[0].Lenses...)
	}
	return []swarmRankedRepair{{
		Repair:      fmt.Sprintf("Determine and apply the smallest safe fix for: %s", claim),
		Lenses:      lenses,
		Synthesized: true,
	}}
}

func sortSwarmRankedRepairs(ranked []swarmRankedRepair) {
	sort.SliceStable(ranked, func(i, j int) bool {
		a, b := ranked[i], ranked[j]
		if len(a.Lenses) != len(b.Lenses) {
			return len(a.Lenses) > len(b.Lenses)
		}
		aHas := a.Confidence != nil
		bHas := b.Confidence != nil
		if aHas != bHas {
			return aHas
		}
		if aHas && bHas && *a.Confidence != *b.Confidence {
			return *a.Confidence > *b.Confidence
		}
		return false
	})
}

// swarmRepairSelectionReason renders the plain-language reason the winning
// repair was chosen, naming what it beat when a runner-up exists.
func swarmRepairSelectionReason(ranked []swarmRankedRepair) string {
	winner := ranked[0]
	reason := fmt.Sprintf("%d lens(es) (%s) corroborate this repair", len(winner.Lenses), strings.Join(labelsForLenses(winner.Lenses), ", "))
	if winner.Confidence != nil {
		reason += fmt.Sprintf(" at %d%% confidence", *winner.Confidence)
	}
	if winner.Synthesized {
		reason += " -- no lens proposed a specific fix text, so the most corroborated finding was used"
	}
	if len(ranked) > 1 {
		runnerUp := ranked[1]
		reason += fmt.Sprintf(", ahead of %q which %d lens(es) support", runnerUp.Repair, len(runnerUp.Lenses))
	}
	return reason
}

// swarmFixWaveBrief renders the comparison's selected repair and reasoning
// as the fix wave's prior context (SYN-202-06): the structured comparison,
// not renderSwarmFindingSummary's free-text concatenation, is what the fix
// wave now consumes.
func swarmFixWaveBrief(comparison swarmComparison) string {
	if comparison.Selected == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("Selected repair (from the four-lens hypothesis comparison):\n")
	b.WriteString(comparison.Selected.Repair)
	b.WriteString("\n\nWhy this repair was selected:\n")
	b.WriteString(comparison.SelectionReason)
	if len(comparison.SharedCauses) > 0 {
		b.WriteString("\n\nCauses multiple lenses agreed on:\n")
		for _, cause := range comparison.SharedCauses {
			b.WriteString("- " + cause.Cause + " (" + strings.Join(labelsForLenses(cause.Lenses), ", ") + ")\n")
		}
	}
	if len(comparison.Contradictions) > 0 {
		b.WriteString("\nContradictions between lenses to be aware of:\n")
		for _, c := range comparison.Contradictions {
			b.WriteString(fmt.Sprintf("- %s says %q, but %s says %q\n", labelForLens(c.LensA), c.ClaimA, labelForLens(c.LensB), c.ClaimB))
		}
	}
	return strings.TrimSpace(b.String())
}

func labelForLens(lensID string) string {
	if lens, ok := swarmLensByID(lensID); ok {
		return lens.Label
	}
	return lensID
}

func labelsForLenses(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, labelForLens(id))
	}
	return out
}

func confidenceText(c *int) string {
	if c == nil {
		return "not stated"
	}
	return fmt.Sprintf("%d%%", *c)
}

func findHypothesisForLens(hyps []swarmHypothesis, lensID string) (swarmHypothesis, bool) {
	for _, h := range hyps {
		if h.Lens == lensID {
			return h, true
		}
	}
	return swarmHypothesis{}, false
}

func missingReasonForLens(missing []swarmNonReportingLens, lensID string) string {
	for _, m := range missing {
		if m.Lens == lensID {
			return m.Reason
		}
	}
	return "no reason recorded"
}

// renderSwarmHypothesisCard renders the single end-of-investigation card
// D-06 requires: one section per lens with its caste glyph and identity
// drawn from the shared caste-identity functions (cmd/codex_visuals.go),
// then shared causes, then contradictions, then the ranked repairs with the
// winner and the reason it won. Framing reuses renderBanner/renderStageMarker
// so the card sits in the same house style as every other ceremony screen.
func renderSwarmHypothesisCard(comparison swarmComparison) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji("swarm-display"), "Swarm Hypothesis Comparison"))
	b.WriteString(visualDividerStr())

	b.WriteString(renderStageMarker("Lenses"))
	for _, lens := range swarmLenses {
		b.WriteString("  ")
		b.WriteString(casteIdentity(lens.Caste))
		b.WriteString(" -- " + lens.Label + " (" + lens.ID + ")\n")
		if h, ok := findHypothesisForLens(comparison.Hypotheses, lens.ID); ok {
			b.WriteString("    Claim: " + h.Claim + "\n")
			b.WriteString("    Confidence: " + confidenceText(h.Confidence) + "\n")
		} else {
			b.WriteString("    Did not report: " + missingReasonForLens(comparison.MissingLenses, lens.ID) + "\n")
		}
	}

	b.WriteString(renderStageMarker("Where They Agree"))
	if len(comparison.SharedCauses) == 0 {
		b.WriteString("  No two lenses named the same cause.\n")
	} else {
		for _, cause := range comparison.SharedCauses {
			b.WriteString("  - " + cause.Cause + " (agreed by " + strings.Join(labelsForLenses(cause.Lenses), ", ") + ")\n")
		}
	}

	b.WriteString(renderStageMarker("Where They Disagree"))
	if len(comparison.Contradictions) == 0 {
		b.WriteString("  No contradictions were found between the lenses.\n")
	} else {
		for _, c := range comparison.Contradictions {
			b.WriteString(fmt.Sprintf("  - %s says %q, but %s says %q\n", labelForLens(c.LensA), c.ClaimA, labelForLens(c.LensB), c.ClaimB))
		}
	}

	b.WriteString(renderStageMarker("Ranked Repairs"))
	if len(comparison.Ranked) == 0 {
		b.WriteString("  No lens produced usable evidence, so no repair was ranked or applied.\n")
	} else {
		for i, r := range comparison.Ranked {
			marker := "  "
			if i == 0 {
				marker = "> "
			}
			b.WriteString(fmt.Sprintf("%s#%d %s -- supported by %s\n", marker, i+1, r.Repair, strings.Join(labelsForLenses(r.Lenses), ", ")))
		}
		if comparison.Selected != nil {
			b.WriteString("\n  Selected: " + comparison.Selected.Repair + "\n")
			b.WriteString("  Why: " + comparison.SelectionReason + "\n")
		}
	}

	return b.String()
}

// emitSwarmHypothesisEvents publishes one live.finding.recorded event per
// hypothesis, at the moment the investigation wave's responses were parsed
// into structured hypotheses -- so the live dashboard can show a hypothesis
// forming, not just a worker finishing (D-06).
func emitSwarmHypothesisEvents(swarmID string, hypotheses []swarmHypothesis) {
	for _, h := range hypotheses {
		emitColonyLive(events.LiveTopicFindingRecorded, events.ColonyLivePayload{
			EpisodeID:   swarmID,
			EpisodeKind: events.EpisodeKindSwarm,
			Wave:        1,
			WorkerID:    h.WorkerName,
			Caste:       h.Caste,
			WorkerName:  h.WorkerName,
			Lens:        h.Lens,
			Findings:    []string{h.Claim},
			Status:      "hypothesis_formed",
		})
	}
}

// emitSwarmContradictionEvents publishes one live.contradiction.found event
// per contradiction the comparison surfaced, at the point the comparison
// itself was computed (D-06).
func emitSwarmContradictionEvents(swarmID string, contradictions []swarmContradiction) {
	for _, c := range contradictions {
		emitColonyLive(events.LiveTopicContradictionFound, events.ColonyLivePayload{
			EpisodeID:   swarmID,
			EpisodeKind: events.EpisodeKindSwarm,
			Wave:        1,
			Lens:        c.LensA,
			Contradictions: []string{
				fmt.Sprintf("%s vs %s: %s / %s", labelForLens(c.LensA), labelForLens(c.LensB), c.ClaimA, c.ClaimB),
			},
			Status: "contradiction_found",
		})
	}
}
