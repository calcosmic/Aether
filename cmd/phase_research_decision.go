package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// phaseResearchExternalSignals identifies phrases suggesting a phase touches
// external technology or an integration surface. This list is intentionally
// separate from researchPhaseKeywords (cmd/plan_grounding.go), which serves
// the grounding-exemption gate and is a Phase 167 removal target (TYPED-05).
// Depending on that list here would mean un-wiring this work next phase.
var phaseResearchExternalSignals = []string{
	"api", "sdk", "webhook", "oauth", "oauth2", "protocol", "integration",
	"third-party", "external service", "http", "grpc", "websocket", "graphql",
	"provider", "vendor", "upstream", "migrate to", "upgrade to",
}

// phaseResearchReasons is a fixed lookup table of plain-English reason
// templates keyed by the signal that drove the recommendation. Mirrors the
// getSmartDefaultReason shape in cmd/review_depth.go. Only the matched signal
// token is interpolated -- never the raw phase description -- so a crafted
// phase description cannot inject instruction text into the reason line the
// Queen relays (T-164-04).
var phaseResearchReasons = map[string]string{
	"domain_gap":       "new external tech (%s) is absent from the territory survey",
	"no_survey":        "external tech (%s) mentioned, but the territory survey has no language/framework/dependency coverage yet",
	"discovery_mode":   "phase mode is discovery -- exploring unmapped territory before committing to an approach",
	"survey_covered":   "pure refactor -- the domain is already mapped by the territory survey",
	"no_external_tech": "no external technology signals found in the phase description -- the domain looks internal",
	"fresh_evidence":   "fresh applicable research evidence already covers the current gap",
	"preset_budget":    "the selected preset does not spend another research pass on this nonmaterial gap",
	"material_gap":     "the current material gap requires evidence before the shared owner-decision boundary",
	"refresh":          "the caller requested a fresh planning pass, so prior research is not silently reused",
}

// phaseResearchHint carries the cheap, deterministic signals computed for one
// candidate phase (D-02). PhaseMode is a best-effort signal: colony.Phase.Mode
// is omitempty and unpopulated for most phases until Phase 167 backfills it
// (164-RESEARCH.md Pitfall 3), so this field is frequently empty by design.
type phaseResearchHint struct {
	ExternalTech []string `json:"external_tech"`
	DomainGaps   []string `json:"domain_gaps"`
	PhaseMode    string   `json:"phase_mode"`
}

// phaseResearchRecommendation is the Queen's grounded research/skip call for
// one candidate phase, with a plain-English, non-empty reason.
type phaseResearchRecommendation struct {
	PhaseID   int               `json:"phase_id"`
	PhaseName string            `json:"phase_name"`
	Recommend string            `json:"recommend"` // "research" or "skip"
	Reason    string            `json:"reason"`
	Hints     phaseResearchHint `json:"hints"`
}

const phaseResearchAutomaticPolicySchemaVersion = "phase-research-policy/v1"

// phaseResearchEvidenceContract is the exact evidence shape an autonomous
// Scout must return. It grants research authority only: specification
// approval, candidate acceptance, and plan activation remain Go/owner-owned.
type phaseResearchEvidenceContract struct {
	SourceKind              colony.PlanningEvidenceKind `json:"source_kind"`
	ProducerCaste           planningStageWorkerCaste    `json:"producer_caste"`
	RequiredFields          []string                    `json:"required_fields"`
	MayApproveSpecification bool                        `json:"may_approve_specification"`
	MayAcceptCandidate      bool                        `json:"may_accept_candidate"`
	MayActivatePlan         bool                        `json:"may_activate_plan"`
}

// phaseResearchEvidenceAttribution binds a typed evidence ID to the worker
// caste that produced it without changing the authority-neutral evidence ref.
type phaseResearchEvidenceAttribution struct {
	EvidenceID    string                      `json:"evidence_id"`
	ProducerCaste planningStageWorkerCaste    `json:"producer_caste"`
	SourceKind    colony.PlanningEvidenceKind `json:"source_kind"`
}

type phaseResearchEvidenceCollection struct {
	Records      []planningEvidenceRecord
	Attributions []phaseResearchEvidenceAttribution
}

// phaseResearchAutomaticDecision is a deterministic per-phase research call.
// It is informational and never becomes a PendingDecision.
type phaseResearchAutomaticDecision struct {
	PhaseID        int               `json:"phase_id"`
	PhaseName      string            `json:"phase_name"`
	ResearchNeeded bool              `json:"research_needed"`
	Reason         string            `json:"reason"`
	Hints          phaseResearchHint `json:"hints"`
}

// phaseResearchAutomaticPolicy replaces the former routine approval card.
// The preset bounds cost, the weakest gap identifies value, and freshness
// prevents duplicate work. Product ambiguity is reported only after the Scout
// pass to the shared material-decision policy.
type phaseResearchAutomaticPolicy struct {
	SchemaVersion         string                           `json:"schema_version"`
	Preset                planningStagePreset              `json:"preset"`
	WeakestGapID          string                           `json:"weakest_gap_id,omitempty"`
	ResearchRequired      bool                             `json:"research_required"`
	Refresh               bool                             `json:"refresh"`
	RequiresOwnerPrompt   bool                             `json:"requires_owner_prompt"`
	OwnerDecisionBoundary string                           `json:"owner_decision_boundary"`
	FreshEvidenceIDs      []string                         `json:"fresh_evidence_ids,omitempty"`
	Phases                []phaseResearchAutomaticDecision `json:"phases,omitempty"`
	EvidenceContract      phaseResearchEvidenceContract    `json:"evidence_contract"`
}

func automaticPhaseResearchEvidenceContract() phaseResearchEvidenceContract {
	return phaseResearchEvidenceContract{
		SourceKind:    colony.PlanningEvidenceResearch,
		ProducerCaste: planningStageCasteScout,
		RequiredFields: []string{
			"origin", "source_revision", "observed_at", "applicable_dimensions", "content_hash",
		},
	}
}

// computeAutomaticPhaseResearchPolicy never consults or writes owner-decision
// state. A valid fresh research ref suppresses duplicate work unless refresh
// explicitly asks for a new observation.
func computeAutomaticPhaseResearchPolicy(preset planningStagePreset, weakestGap *colony.PlanningGap, survey codexSurveyContext, candidates []phaseResearchCandidate, phases []colony.Phase, evidence []planningEvidenceRecord, refresh bool) phaseResearchAutomaticPolicy {
	policy := phaseResearchAutomaticPolicy{
		SchemaVersion:         phaseResearchAutomaticPolicySchemaVersion,
		Preset:                preset,
		Refresh:               refresh,
		RequiresOwnerPrompt:   false,
		OwnerDecisionBoundary: "after_scout_pass",
		EvidenceContract:      automaticPhaseResearchEvidenceContract(),
		Phases:                make([]phaseResearchAutomaticDecision, 0, len(candidates)),
	}
	if weakestGap != nil {
		policy.WeakestGapID = strings.TrimSpace(weakestGap.ID)
	}
	policy.FreshEvidenceIDs = freshApplicablePhaseResearchEvidenceIDs(evidence, weakestGap)
	hasFreshEvidence := len(policy.FreshEvidenceIDs) > 0

	phaseByID := make(map[int]colony.Phase, len(phases))
	for _, phase := range phases {
		phaseByID[phase.ID] = phase
	}
	surveyEmpty := len(survey.Languages) == 0 && len(survey.Frameworks) == 0 && len(survey.Dependencies) == 0
	gapWarrantsResearch := automaticPhaseResearchGapWarrants(preset, weakestGap)

	for _, candidate := range candidates {
		hint := computePhaseResearchHint(candidate, survey, phaseByID[candidate.ID])
		candidateFresh := hasFreshApplicablePhaseResearchEvidence(evidence, weakestGap, candidate.ID)
		signalWarrantsResearch := len(hint.DomainGaps) > 0 ||
			(len(hint.ExternalTech) > 0 && surveyEmpty) ||
			hint.PhaseMode == string(colony.PhaseModeDiscovery)
		needed := !candidateFresh && automaticPhaseResearchCandidateWarrants(preset, weakestGap, gapWarrantsResearch, signalWarrantsResearch, len(candidates))
		if refresh && automaticPhaseResearchCandidateWarrants(preset, weakestGap, gapWarrantsResearch, signalWarrantsResearch, len(candidates)) {
			needed = true
		}
		decision := phaseResearchAutomaticDecision{
			PhaseID: candidate.ID, PhaseName: candidate.Name, ResearchNeeded: needed, Hints: hint,
			Reason: automaticPhaseResearchReason(refresh, candidateFresh, weakestGap, hint, needed),
		}
		policy.Phases = append(policy.Phases, decision)
		policy.ResearchRequired = policy.ResearchRequired || needed
	}

	// A first Scout pass may not have Route-Setter phase candidates yet. The
	// weakest-gap decision still governs whether that one Scout should gather
	// authoritative external evidence during its pass.
	if len(candidates) == 0 {
		policy.ResearchRequired = (refresh || !hasFreshEvidence) && gapWarrantsResearch
	}
	return policy
}

func automaticPhaseResearchGapWarrants(preset planningStagePreset, gap *colony.PlanningGap) bool {
	if gap != nil && gap.Materiality == colony.PlanningGapMaterial {
		return true
	}
	severity := 0
	if gap != nil {
		severity = gap.Severity
	}
	switch preset {
	case planningStagePresetFast:
		return false
	case planningStagePresetBalanced:
		return severity >= 70
	case planningStagePresetDeep:
		return severity >= 40
	case planningStagePresetExhaustive:
		return true
	default:
		return false
	}
}

func automaticPhaseResearchCandidateWarrants(preset planningStagePreset, gap *colony.PlanningGap, gapWarrants, signalWarrants bool, candidateCount int) bool {
	switch preset {
	case planningStagePresetFast:
		return gap != nil && gap.Materiality == colony.PlanningGapMaterial && (signalWarrants || gapWarrants)
	case planningStagePresetBalanced:
		return signalWarrants || (candidateCount == 1 && gapWarrants)
	case planningStagePresetDeep:
		return signalWarrants || (candidateCount == 1 && gapWarrants)
	case planningStagePresetExhaustive:
		return true
	default:
		return false
	}
}

func automaticPhaseResearchReason(refresh, fresh bool, gap *colony.PlanningGap, hint phaseResearchHint, needed bool) string {
	switch {
	case refresh && needed:
		return phaseResearchReasons["refresh"]
	case fresh:
		return phaseResearchReasons["fresh_evidence"]
	case needed && gap != nil && gap.Materiality == colony.PlanningGapMaterial:
		return phaseResearchReasons["material_gap"]
	case needed && len(hint.DomainGaps) > 0:
		return fmt.Sprintf(phaseResearchReasons["domain_gap"], hint.DomainGaps[0])
	case needed && len(hint.ExternalTech) > 0:
		return fmt.Sprintf(phaseResearchReasons["no_survey"], hint.ExternalTech[0])
	case needed && hint.PhaseMode == string(colony.PhaseModeDiscovery):
		return phaseResearchReasons["discovery_mode"]
	case !needed && len(hint.ExternalTech) == 0:
		return phaseResearchReasons["no_external_tech"]
	default:
		return phaseResearchReasons["preset_budget"]
	}
}

func freshApplicablePhaseResearchEvidenceIDs(records []planningEvidenceRecord, gap *colony.PlanningGap) []string {
	ids := make([]string, 0)
	for _, record := range records {
		ref := record.Reference
		if ref.Kind != colony.PlanningEvidenceResearch || !ref.Fresh || !ref.Admissible || ref.Validate() != nil {
			continue
		}
		if gap != nil && !planningDimensionIncluded(ref.ApplicableDimensions, gap.Dimension) {
			continue
		}
		ids = append(ids, ref.ID)
	}
	return uniqueSortedStrings(ids)
}

func hasFreshApplicablePhaseResearchEvidence(records []planningEvidenceRecord, gap *colony.PlanningGap, phaseID int) bool {
	for _, record := range records {
		ref := record.Reference
		if ref.Kind != colony.PlanningEvidenceResearch || !ref.Fresh || !ref.Admissible || ref.Validate() != nil {
			continue
		}
		if gap != nil && !planningDimensionIncluded(ref.ApplicableDimensions, gap.Dimension) {
			continue
		}
		if artifactPhaseID, phaseScoped := phaseResearchArtifactPhaseID(ref.RepositoryPath); phaseScoped && artifactPhaseID != phaseID {
			continue
		}
		return true
	}
	return false
}

func planningDimensionIncluded(values []colony.PlanningDimension, want colony.PlanningDimension) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func (p phaseResearchAutomaticPolicy) SelectedPhases() map[int]bool {
	selected := make(map[int]bool)
	for _, phase := range p.Phases {
		if phase.ResearchNeeded {
			selected[phase.PhaseID] = true
		}
	}
	return selected
}

// collectScoutPhaseResearchEvidence upgrades worker-authored phase research
// files into normal typed planning evidence and records Scout provenance next
// to their authority-neutral IDs. Paths are restricted to the phase-research
// directory, inspected before reading, and rejected if secret-bearing.
func collectScoutPhaseResearchEvidence(root string, docs []string, scope planningEvidenceScope, observedAt time.Time) (phaseResearchEvidenceCollection, error) {
	result := phaseResearchEvidenceCollection{}
	cleaned := uniqueSortedStrings(docs)
	if len(cleaned) == 0 {
		return result, nil
	}
	canonicalRoot, err := canonicalPlanningEvidenceRoot(root)
	if err != nil {
		return phaseResearchEvidenceCollection{}, err
	}
	const approvedRoot = ".aether/data/phase-research"
	sources := make([]planningEvidenceSource, 0, len(cleaned))
	for _, doc := range cleaned {
		rel, fullPath, err := resolvePlanningEvidencePath(canonicalRoot, []string{approvedRoot}, doc)
		if err != nil {
			return phaseResearchEvidenceCollection{}, err
		}
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return phaseResearchEvidenceCollection{}, &planningEvidenceRefusal{
				Code: planningEvidenceRefusalReadFailed, Kind: colony.PlanningEvidenceResearch,
				Origin: rel, Detail: "could not read the validated Scout research source",
			}
		}
		if scan := privacyScan(string(content)); scan.Blocked {
			return phaseResearchEvidenceCollection{}, fmt.Errorf("Scout research evidence %s refused: source contains secret-bearing material", rel)
		}
		sources = append(sources, planningEvidenceSource{
			Kind: colony.PlanningEvidenceResearch, Origin: rel, RepositoryPath: rel,
			Scope: scope, SourceRevision: planningEvidenceSourceRevision("scout-research", content),
			ObservedAt: observedAt, ApplicableDimensions: planningEvidenceDimensions(colony.PlanningEvidenceResearch),
			State: planningEvidenceSourceCurrent,
		})
	}
	records, err := collectPlanningEvidence(planningEvidenceCollectionRequest{
		RepositoryRoot: root,
		ApprovedRoots:  []string{approvedRoot},
		Sources:        sources,
	})
	if err != nil {
		return phaseResearchEvidenceCollection{}, err
	}
	result.Records = records
	result.Attributions = make([]phaseResearchEvidenceAttribution, 0, len(records))
	for _, record := range records {
		result.Attributions = append(result.Attributions, phaseResearchEvidenceAttribution{
			EvidenceID: record.Reference.ID, ProducerCaste: planningStageCasteScout, SourceKind: colony.PlanningEvidenceResearch,
		})
	}
	sort.Slice(result.Attributions, func(i, j int) bool {
		return result.Attributions[i].EvidenceID < result.Attributions[j].EvidenceID
	})
	return result, nil
}

func discoverScoutPhaseResearchDocs(root string) ([]string, error) {
	relDir := filepath.ToSlash(filepath.Join(".aether", "data", "phase-research"))
	dir := filepath.Join(root, filepath.FromSlash(relDir))
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read Scout phase research directory: %w", err)
	}
	docs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !phaseResearchArtifactName(entry.Name()) {
			continue
		}
		rel := filepath.ToSlash(filepath.Join(relDir, entry.Name()))
		if !hasWorkerAuthoredResearch(filepath.Join(dir, entry.Name())) {
			continue
		}
		docs = append(docs, rel)
	}
	sort.Strings(docs)
	return docs, nil
}

func phaseResearchArtifactName(name string) bool {
	const prefix = "phase-"
	const suffix = "-research.md"
	if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, suffix) {
		return false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(name, prefix), suffix)
	value, err := strconv.Atoi(id)
	return err == nil && value > 0 && strconv.Itoa(value) == id
}

func phaseResearchArtifactPhaseID(repositoryPath string) (int, bool) {
	path := filepath.ToSlash(strings.TrimSpace(repositoryPath))
	name := filepath.Base(path)
	if !phaseResearchArtifactName(name) || filepath.ToSlash(filepath.Dir(path)) != ".aether/data/phase-research" {
		return 0, false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(name, "phase-"), "-research.md")
	value, err := strconv.Atoi(id)
	return value, err == nil
}

func mergePlanningEvidenceRecords(groups ...[]planningEvidenceRecord) []planningEvidenceRecord {
	byID := make(map[string]planningEvidenceRecord)
	for _, records := range groups {
		for _, record := range records {
			if _, exists := byID[record.Reference.ID]; !exists {
				byID[record.Reference.ID] = record
			}
		}
	}
	merged := make([]planningEvidenceRecord, 0, len(byID))
	for _, record := range byID {
		merged = append(merged, record)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].Reference.ID < merged[j].Reference.ID })
	return merged
}

func renderAutomaticPhaseResearchPolicy(policy phaseResearchAutomaticPolicy) string {
	var b strings.Builder
	b.WriteString("\n\n## Autonomous Research Policy\n")
	fmt.Fprintf(&b, "Preset: %s. Research required in this Scout pass: %t.\n", policy.Preset, policy.ResearchRequired)
	if policy.WeakestGapID != "" {
		fmt.Fprintf(&b, "Weakest gap: %s.\n", policy.WeakestGapID)
	}
	b.WriteString("Routine repository and authoritative external research is pre-authorized by this manifest; do not request a phase-research approval.\n")
	for _, phase := range policy.Phases {
		fmt.Fprintf(&b, "- Phase %d research=%t: %s\n", phase.PhaseID, phase.ResearchNeeded, phase.Reason)
	}
	b.WriteString(renderPhaseResearchEvidenceContract(policy.EvidenceContract))
	b.WriteString("Report material product ambiguity as a decision candidate; the shared policy evaluates it only after this Scout pass completes.\n")
	return b.String()
}

func renderPhaseResearchEvidenceContract(contract phaseResearchEvidenceContract) string {
	return fmt.Sprintf("Every research result must become typed planning evidence with source kind %s and Scout attribution. Record origin, source revision, observation time/freshness, applicable dimensions, and content hash. Reject sources outside the repository scope, unavailable sources, and secret-bearing content. The Scout must not approve a specification, accept a candidate, or activate a plan.\n", contract.SourceKind)
}

// computePhaseResearchHint computes the deterministic, lowercase-normalised
// signals for one candidate phase (D-02). phase is the zero value when no
// matching colony.Phase was found by ID -- Mode.Valid() correctly returns
// false in that case, matching the guard checkPlanGrounding already uses.
func computePhaseResearchHint(candidate phaseResearchCandidate, survey codexSurveyContext, phase colony.Phase) phaseResearchHint {
	text := strings.ToLower(candidate.Name + " " + candidate.Description)

	var externalTech []string
	for _, signal := range phaseResearchExternalSignals {
		if strings.Contains(text, signal) {
			externalTech = append(externalTech, signal)
		}
	}

	var domainGaps []string
	for _, tech := range externalTech {
		if surveyContainsSignal(survey.Languages, tech) ||
			surveyContainsSignal(survey.Frameworks, tech) ||
			surveyContainsSignal(survey.Dependencies, tech) {
			continue
		}
		domainGaps = append(domainGaps, tech)
	}

	var mode string
	if phase.Mode.Valid() {
		mode = string(phase.Mode)
	}

	return phaseResearchHint{
		ExternalTech: externalTech,
		DomainGaps:   domainGaps,
		PhaseMode:    mode,
	}
}

// surveyContainsSignal reports whether signal appears as a substring in any
// lowercased entry of entries.
func surveyContainsSignal(entries []string, signal string) bool {
	for _, entry := range entries {
		if strings.Contains(strings.ToLower(entry), signal) {
			return true
		}
	}
	return false
}

// phaseResearchDecisionType remains solely so the deprecated
// plan-research-approve command can resolve legacy rows in old colonies. New
// planning runs never create this decision type.
const phaseResearchDecisionType = "research-decision"

// phaseResearchDecisionResolution builds the resolution string written when a
// research decision is resolved -- the durable record of the flip (T-164-06),
// so it carries the resulting direction, not just the fact of a flip.
func phaseResearchDecisionResolution(rec phaseResearchRecommendation, flipped bool, auto bool) string {
	var resolution string
	if flipped {
		resolution = fmt.Sprintf("user overrode: %s research on phase %d", oppositeRecommend(rec.Recommend), rec.PhaseID)
	} else {
		resolution = fmt.Sprintf("approved: %s phase %d", rec.Recommend, rec.PhaseID)
	}
	if auto {
		resolution = "auto-accepted (autopilot) -- " + resolution
	}
	return resolution
}

// oppositeRecommend returns the opposite research direction of recommend.
func oppositeRecommend(recommend string) string {
	if recommend == "skip" {
		return "research"
	}
	return "skip"
}
