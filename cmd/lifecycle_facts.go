package cmd

// This file is the read-only side of the Phase 199 lifecycle boundary.
// Orientation commands may inspect these facts, but they may not repair,
// normalize on disk, acquire write-backed locks, or infer evidence that was
// never recorded. Each domain therefore carries the path that was inspected,
// a four-state provenance verdict, and a diagnostic when the source could not
// be used.

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// LifecycleFactProvenance is about the availability of one fact source. It is
// intentionally separate from colony.RecoveryProvenance, which describes how
// strongly durable evidence supports a recovered claim.
type LifecycleFactProvenance string

const (
	LifecycleFactConfirmed   LifecycleFactProvenance = "confirmed"
	LifecycleFactMissing     LifecycleFactProvenance = "missing"
	LifecycleFactMalformed   LifecycleFactProvenance = "malformed"
	LifecycleFactUnavailable LifecycleFactProvenance = "unavailable"
)

// LifecycleFactSource records exactly where a fact came from. Path may contain
// a comma-separated, stable list when one domain is assembled from several
// files or directories.
type LifecycleFactSource struct {
	Domain     string                  `json:"domain"`
	Path       string                  `json:"path"`
	Provenance LifecycleFactProvenance `json:"provenance"`
	Diagnostic string                  `json:"diagnostic,omitempty"`
}

// LifecycleFact pairs a value with the evidence boundary that produced it.
// A zero Value is meaningful only together with Source: it is never silently
// promoted from "not observed" into a confirmed empty result.
type LifecycleFact[T any] struct {
	Value  T                   `json:"value"`
	Source LifecycleFactSource `json:"source"`
}

type LifecycleIdentityFacts struct {
	Name      string `json:"name,omitempty"`
	Goal      string `json:"goal,omitempty"`
	Episode   string `json:"episode,omitempty"`
	Standing  string `json:"standing,omitempty"`
	Milestone string `json:"milestone,omitempty"`
	Scope     string `json:"scope,omitempty"`
	Mode      string `json:"mode,omitempty"`
}

type LifecycleProgressFacts struct {
	CurrentPhase int            `json:"current_phase"`
	Phases       []colony.Phase `json:"phases,omitempty"`
}

type LifecycleActorFact struct {
	Timestamp string `json:"timestamp,omitempty"`
	Parent    string `json:"parent,omitempty"`
	Caste     string `json:"caste,omitempty"`
	Name      string `json:"name,omitempty"`
	Task      string `json:"task,omitempty"`
	Depth     int    `json:"depth,omitempty"`
	Status    string `json:"status,omitempty"`
	Summary   string `json:"summary,omitempty"`
}

type LifecycleResearchFacts struct {
	Docs      []string `json:"docs,omitempty"`
	Dreams    []string `json:"dreams,omitempty"`
	Territory []string `json:"territory,omitempty"`
}

type LifecycleMemoryFacts struct {
	State        colony.Memory              `json:"state"`
	Instincts    []colony.InstinctEntry     `json:"instincts,omitempty"`
	Observations []colony.Observation       `json:"observations,omitempty"`
	Findings     []colony.ReviewLedgerEntry `json:"findings,omitempty"`
}

type LifecycleVerificationFacts struct {
	Gates     []colony.GateResultEntry `json:"gates,omitempty"`
	Artifacts []string                 `json:"artifacts,omitempty"`
}

type LifecycleTimingFacts struct {
	CapturedAt    time.Time     `json:"captured_at"`
	InitializedAt *time.Time    `json:"initialized_at,omitempty"`
	BuildStarted  *time.Time    `json:"build_started_at,omitempty"`
	Elapsed       time.Duration `json:"elapsed"`
}

// LifecycleReportedCostFacts carries only figures already recorded by a
// provider/session ledger. It never estimates a missing row.
type LifecycleReportedCostFacts struct {
	Phase       int   `json:"phase"`
	TotalTokens int64 `json:"total_tokens"`
	Rows        int   `json:"rows"`
}

type LifecycleEvidenceFacts struct {
	Receipt  *colony.LifecycleReceipt      `json:"receipt,omitempty"`
	Handoff  *colony.PauseHandoffReference `json:"handoff,omitempty"`
	Recovery *colony.RecoveryProvenance    `json:"recovery,omitempty"`
	Seal     *colony.SealOutcome           `json:"seal,omitempty"`
	Archive  *colony.ArchiveReference      `json:"archive,omitempty"`
}

// LifecycleFacts is the complete, immutable-by-convention snapshot passed to
// the pure lifecycle projection. Consumers receive it by value and do not get
// access to the store or any loader callbacks.
type LifecycleFacts struct {
	Root         string                                    `json:"root"`
	CapturedAt   time.Time                                 `json:"captured_at"`
	State        LifecycleFact[colony.ColonyState]         `json:"state"`
	Identity     LifecycleFact[LifecycleIdentityFacts]     `json:"identity"`
	Progress     LifecycleFact[LifecycleProgressFacts]     `json:"progress"`
	Actors       LifecycleFact[[]LifecycleActorFact]       `json:"actors"`
	Signals      LifecycleFact[[]colony.PheromoneSignal]   `json:"signals"`
	Research     LifecycleFact[LifecycleResearchFacts]     `json:"research"`
	Memory       LifecycleFact[LifecycleMemoryFacts]       `json:"memory"`
	Verification LifecycleFact[LifecycleVerificationFacts] `json:"verification"`
	Timing       LifecycleFact[LifecycleTimingFacts]       `json:"timing"`
	ReportedCost LifecycleFact[LifecycleReportedCostFacts] `json:"reported_cost"`
	History      LifecycleFact[[]string]                   `json:"history"`
	Blockers     LifecycleFact[[]colony.FlagEntry]         `json:"blockers"`
	Evidence     LifecycleFact[LifecycleEvidenceFacts]     `json:"evidence"`
	Session      LifecycleFact[colony.SessionFile]         `json:"session"`
}

// Sources returns one provenance record for every fact domain in stable order.
func (f LifecycleFacts) Sources() []LifecycleFactSource {
	return []LifecycleFactSource{
		f.State.Source,
		f.Identity.Source,
		f.Progress.Source,
		f.Actors.Source,
		f.Signals.Source,
		f.Research.Source,
		f.Memory.Source,
		f.Verification.Source,
		f.Timing.Source,
		f.ReportedCost.Source,
		f.History.Source,
		f.Blockers.Source,
		f.Evidence.Source,
		f.Session.Source,
	}
}

func lifecycleSource(domain, path string, provenance LifecycleFactProvenance, diagnostic string) LifecycleFactSource {
	return LifecycleFactSource{
		Domain:     domain,
		Path:       filepath.ToSlash(path),
		Provenance: provenance,
		Diagnostic: diagnostic,
	}
}

func lifecycleDerivedSource(domain string, source LifecycleFactSource) LifecycleFactSource {
	source.Domain = domain
	return source
}

func lifecycleString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func lifecycleUnavailableSource(domain, path, diagnostic string) LifecycleFactSource {
	if strings.TrimSpace(diagnostic) == "" {
		diagnostic = "source is unavailable"
	}
	return lifecycleSource(domain, path, LifecycleFactUnavailable, diagnostic)
}

func lifecycleCombinedSource(domain string, sources ...LifecycleFactSource) LifecycleFactSource {
	paths := make([]string, 0, len(sources))
	diagnostics := make([]string, 0, len(sources))
	provenance := LifecycleFactMissing
	confirmed := false
	for _, source := range sources {
		if source.Path != "" {
			paths = append(paths, source.Path)
		}
		if source.Diagnostic != "" {
			diagnostics = append(diagnostics, source.Domain+": "+source.Diagnostic)
		}
		switch source.Provenance {
		case LifecycleFactUnavailable:
			provenance = LifecycleFactUnavailable
		case LifecycleFactMalformed:
			if provenance != LifecycleFactUnavailable {
				provenance = LifecycleFactMalformed
			}
		case LifecycleFactConfirmed:
			confirmed = true
			if provenance == LifecycleFactMissing {
				provenance = LifecycleFactConfirmed
			}
		}
	}
	if confirmed && provenance == LifecycleFactMissing {
		provenance = LifecycleFactConfirmed
	}
	if len(paths) == 0 {
		paths = append(paths, "(unavailable)")
	}
	return lifecycleSource(domain, strings.Join(paths, ","), provenance, strings.Join(diagnostics, "; "))
}

func readLifecycleJSON[T any](domain, path string) (T, LifecycleFactSource) {
	var value T
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return value, lifecycleSource(domain, path, LifecycleFactMissing, "source does not exist")
		}
		return value, lifecycleUnavailableSource(domain, path, err.Error())
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return value, lifecycleSource(domain, path, LifecycleFactMalformed, err.Error())
	}
	return value, lifecycleSource(domain, path, LifecycleFactConfirmed, "")
}

func listLifecycleFiles(domain, dir string) ([]string, LifecycleFactSource) {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, lifecycleSource(domain, dir, LifecycleFactMissing, "source directory does not exist")
		}
		return nil, lifecycleUnavailableSource(domain, dir, err.Error())
	}
	if !info.IsDir() {
		return nil, lifecycleSource(domain, dir, LifecycleFactMalformed, "expected a directory")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, lifecycleUnavailableSource(domain, dir, err.Error())
	}
	_ = entries // ReadDir above distinguishes an unreadable directory before walking.
	var paths []string
	err = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type().IsRegular() {
			paths = append(paths, filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		return nil, lifecycleUnavailableSource(domain, dir, err.Error())
	}
	sort.Strings(paths)
	return paths, lifecycleSource(domain, dir, LifecycleFactConfirmed, "")
}

func readLifecycleState(path string) (colony.ColonyState, LifecycleFactSource) {
	state, repaired, err := loadColonyStateWithCompatibilityRepairReadOnlyFromPath(path)
	if err != nil {
		if os.IsNotExist(err) {
			return colony.ColonyState{}, lifecycleSource("state", path, LifecycleFactMissing, "source does not exist")
		}
		var syntaxError *json.SyntaxError
		if strings.Contains(err.Error(), "cannot unmarshal") || strings.Contains(err.Error(), "invalid character") || strings.Contains(err.Error(), "invalid ") || syntaxError != nil {
			return colony.ColonyState{}, lifecycleSource("state", path, LifecycleFactMalformed, err.Error())
		}
		return colony.ColonyState{}, lifecycleUnavailableSource("state", path, err.Error())
	}
	diagnostic := ""
	if repaired {
		diagnostic = "legacy numeric current_phase normalized in memory; source bytes were not changed"
	}
	return normalizeLegacyColonyState(state), lifecycleSource("state", path, LifecycleFactConfirmed, diagnostic)
}

func readLifecycleActors(path string) ([]LifecycleActorFact, LifecycleFactSource) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, lifecycleSource("actors", path, LifecycleFactMissing, "source does not exist")
		}
		return nil, lifecycleUnavailableSource("actors", path, err.Error())
	}
	actors := make([]LifecycleActorFact, 0)
	byName := map[string]int{}
	for lineNumber, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		fields := strings.Split(line, "|")
		switch len(fields) {
		case 7:
			depth, err := strconv.Atoi(fields[5])
			if err != nil {
				return nil, lifecycleSource("actors", path, LifecycleFactMalformed, fmt.Sprintf("line %d has a non-numeric depth", lineNumber+1))
			}
			byName[fields[3]] = len(actors)
			actors = append(actors, LifecycleActorFact{
				Timestamp: fields[0], Parent: fields[1], Caste: fields[2], Name: fields[3],
				Task: fields[4], Depth: depth, Status: fields[6],
			})
		case 4:
			if index, ok := byName[fields[1]]; ok {
				actors[index].Timestamp = fields[0]
				actors[index].Status = fields[2]
				actors[index].Summary = fields[3]
			}
		default:
			return nil, lifecycleSource("actors", path, LifecycleFactMalformed, fmt.Sprintf("line %d has %d fields; expected 7 or 4", lineNumber+1, len(fields)))
		}
	}
	return actors, lifecycleSource("actors", path, LifecycleFactConfirmed, "")
}

func readLifecycleResearch(root, dataDir string) (LifecycleResearchFacts, LifecycleFactSource) {
	researchDir := filepath.Join(root, ".aether", "research")
	dreamsDir := filepath.Join(root, ".aether", "dreams")
	territoryDir := filepath.Join(dataDir, "survey")
	docs, docsSource := listLifecycleFiles("research docs", researchDir)
	dreams, dreamsSource := listLifecycleFiles("dreams", dreamsDir)
	territory, territorySource := listLifecycleFiles("territory", territoryDir)
	return LifecycleResearchFacts{Docs: docs, Dreams: dreams, Territory: territory}, lifecycleCombinedSource("research", docsSource, dreamsSource, territorySource)
}

func readLifecycleFindings(dir string) ([]colony.ReviewLedgerEntry, LifecycleFactSource) {
	paths, dirSource := listLifecycleFiles("findings", dir)
	if dirSource.Provenance != LifecycleFactConfirmed {
		return nil, dirSource
	}
	var findings []colony.ReviewLedgerEntry
	for _, path := range paths {
		if filepath.Base(path) != "ledger.json" {
			continue
		}
		ledger, source := readLifecycleJSON[colony.ReviewLedgerFile]("findings", path)
		if source.Provenance != LifecycleFactConfirmed {
			return nil, source
		}
		findings = append(findings, ledger.Entries...)
	}
	return findings, lifecycleSource("findings", dir, LifecycleFactConfirmed, "")
}

func readLifecycleMemory(dataDir string, state colony.ColonyState, stateSource LifecycleFactSource) (LifecycleMemoryFacts, LifecycleFactSource) {
	instincts, instinctSource := readLifecycleJSON[colony.InstinctsFile]("instincts", filepath.Join(dataDir, "instincts.json"))
	learnings, learningSource := readLifecycleJSON[colony.LearningFile]("observations", filepath.Join(dataDir, "learning-observations.json"))
	findings, findingSource := readLifecycleFindings(filepath.Join(dataDir, "reviews"))
	return LifecycleMemoryFacts{
		State: state.Memory, Instincts: instincts.Instincts,
		Observations: learnings.Observations, Findings: findings,
	}, lifecycleCombinedSource("memory", lifecycleDerivedSource("state memory", stateSource), instinctSource, learningSource, findingSource)
}

func readLifecycleVerification(dataDir string, state colony.ColonyState, stateSource LifecycleFactSource) (LifecycleVerificationFacts, LifecycleFactSource) {
	buildDir := filepath.Join(dataDir, "build")
	paths, artifactSource := listLifecycleFiles("verification artifacts", buildDir)
	var artifacts []string
	if artifactSource.Provenance == LifecycleFactConfirmed {
		for _, path := range paths {
			name := strings.ToLower(filepath.Base(path))
			if !strings.Contains(name, "verification") && !strings.Contains(name, "gates") && !strings.Contains(name, "review") && !strings.Contains(name, "continue") {
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				artifactSource = lifecycleUnavailableSource("verification artifacts", path, err.Error())
				break
			}
			var value any
			if err := json.Unmarshal(data, &value); err != nil {
				artifactSource = lifecycleSource("verification artifacts", path, LifecycleFactMalformed, err.Error())
				break
			}
			artifacts = append(artifacts, filepath.ToSlash(path))
		}
	}
	return LifecycleVerificationFacts{Gates: state.GateResults, Artifacts: artifacts}, lifecycleCombinedSource("verification", lifecycleDerivedSource("state gates", stateSource), artifactSource)
}

func readLifecycleCost(dataDir string) (LifecycleReportedCostFacts, LifecycleFactSource) {
	dir := filepath.Join(dataDir, "spend")
	paths, source := listLifecycleFiles("reported cost", dir)
	if source.Provenance != LifecycleFactConfirmed {
		return LifecycleReportedCostFacts{}, source
	}
	var result LifecycleReportedCostFacts
	matched := false
	for _, path := range paths {
		if !strings.HasPrefix(filepath.Base(path), "phase-") || filepath.Ext(path) != ".json" {
			continue
		}
		matched = true
		ledger, ledgerSource := readLifecycleJSON[spendLedger]("reported cost", path)
		if ledgerSource.Provenance != LifecycleFactConfirmed {
			return LifecycleReportedCostFacts{}, ledgerSource
		}
		if ledger.SchemaVersion != spendLedgerSchemaVersion {
			return LifecycleReportedCostFacts{}, lifecycleSource("reported cost", path, LifecycleFactMalformed, fmt.Sprintf("unsupported schema_version %d", ledger.SchemaVersion))
		}
		if result.Phase == 0 || ledger.Phase > result.Phase {
			result.Phase = ledger.Phase
		}
		for _, row := range ledger.Rows {
			tokens := row.Usage.TotalTokens
			if tokens == 0 {
				continue
			}
			result.TotalTokens += tokens
			result.Rows++
		}
	}
	if !matched {
		return LifecycleReportedCostFacts{}, lifecycleSource("reported cost", dir, LifecycleFactMissing, "no recorded phase spend ledger exists")
	}
	return result, lifecycleSource("reported cost", dir, LifecycleFactConfirmed, "")
}

func lifecycleTiming(state colony.ColonyState, source LifecycleFactSource, now time.Time) LifecycleFact[LifecycleTimingFacts] {
	value := LifecycleTimingFacts{CapturedAt: now, InitializedAt: state.InitializedAt, BuildStarted: state.BuildStartedAt}
	if state.BuildStartedAt != nil && !now.Before(*state.BuildStartedAt) {
		value.Elapsed = now.Sub(*state.BuildStartedAt)
	} else if state.InitializedAt != nil && !now.Before(*state.InitializedAt) {
		value.Elapsed = now.Sub(*state.InitializedAt)
	}
	return LifecycleFact[LifecycleTimingFacts]{Value: value, Source: lifecycleDerivedSource("timing", source)}
}

func lifecycleEvidence(state colony.ColonyState, stateSource LifecycleFactSource, session colony.SessionFile, sessionSource LifecycleFactSource) LifecycleFact[LifecycleEvidenceFacts] {
	value := LifecycleEvidenceFacts{
		Receipt: state.LifecycleReceipt, Handoff: state.PauseHandoff,
		Recovery: state.RecoveryProvenance, Seal: state.SealOutcome, Archive: state.ArchiveReference,
	}
	if value.Receipt == nil {
		value.Receipt = session.LifecycleReceipt
	}
	if value.Handoff == nil {
		value.Handoff = session.PauseHandoff
	}
	if value.Recovery == nil {
		value.Recovery = session.RecoveryProvenance
	}
	if value.Seal == nil {
		value.Seal = session.SealOutcome
	}
	if value.Archive == nil {
		value.Archive = session.ArchiveReference
	}
	return LifecycleFact[LifecycleEvidenceFacts]{Value: value, Source: lifecycleCombinedSource("evidence", lifecycleDerivedSource("state evidence", stateSource), lifecycleDerivedSource("session evidence", sessionSource))}
}

func unavailableLifecycleFacts(root string, now time.Time, diagnostic string) LifecycleFacts {
	unavailable := func(domain, path string) LifecycleFactSource {
		return lifecycleUnavailableSource(domain, path, diagnostic)
	}
	base := filepath.Join(root, ".aether", "data")
	return LifecycleFacts{
		Root: root, CapturedAt: now,
		State:        LifecycleFact[colony.ColonyState]{Source: unavailable("state", filepath.Join(base, "COLONY_STATE.json"))},
		Identity:     LifecycleFact[LifecycleIdentityFacts]{Source: unavailable("identity", filepath.Join(base, "COLONY_STATE.json"))},
		Progress:     LifecycleFact[LifecycleProgressFacts]{Source: unavailable("progress", filepath.Join(base, "COLONY_STATE.json"))},
		Actors:       LifecycleFact[[]LifecycleActorFact]{Source: unavailable("actors", filepath.Join(base, "spawn-tree.txt"))},
		Signals:      LifecycleFact[[]colony.PheromoneSignal]{Source: unavailable("signals", filepath.Join(base, "pheromones.json"))},
		Research:     LifecycleFact[LifecycleResearchFacts]{Source: unavailable("research", filepath.Join(root, ".aether", "research"))},
		Memory:       LifecycleFact[LifecycleMemoryFacts]{Source: unavailable("memory", filepath.Join(base, "instincts.json"))},
		Verification: LifecycleFact[LifecycleVerificationFacts]{Source: unavailable("verification", filepath.Join(base, "build"))},
		Timing:       LifecycleFact[LifecycleTimingFacts]{Value: LifecycleTimingFacts{CapturedAt: now}, Source: unavailable("timing", filepath.Join(base, "COLONY_STATE.json"))},
		ReportedCost: LifecycleFact[LifecycleReportedCostFacts]{Source: unavailable("reported cost", filepath.Join(base, "spend"))},
		History:      LifecycleFact[[]string]{Source: unavailable("history", filepath.Join(base, "COLONY_STATE.json"))},
		Blockers:     LifecycleFact[[]colony.FlagEntry]{Source: unavailable("blockers", filepath.Join(base, "pending-decisions.json"))},
		Evidence:     LifecycleFact[LifecycleEvidenceFacts]{Source: unavailable("evidence", filepath.Join(base, "COLONY_STATE.json"))},
		Session:      LifecycleFact[colony.SessionFile]{Source: unavailable("session", filepath.Join(base, "session.json"))},
	}
}

// loadActiveRecoveryGuidanceReadOnly reads the optional continue report
// without acquiring storage.Store's write-backed lock. It preserves the
// report's exact recovery detail for closeout copy; lifecycle routing itself
// remains owned by projectLifecycle.
func loadActiveRecoveryGuidanceReadOnly(state colony.ColonyState, dataDir string) *activeRecoveryGuidance {
	state = normalizeLegacyColonyState(state)
	if state.CurrentPhase < 1 || (state.State != colony.StateEXECUTING && state.State != colony.StateBUILT) {
		return nil
	}
	rel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", state.CurrentPhase), "continue.json"))
	data, err := os.ReadFile(filepath.Join(dataDir, filepath.FromSlash(rel)))
	if err != nil {
		return nil
	}
	var report codexContinueReport
	if err := json.Unmarshal(data, &report); err != nil || report.Phase != state.CurrentPhase || report.Advanced || report.Completed {
		return nil
	}
	if state.BuildStartedAt != nil {
		generatedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(report.GeneratedAt))
		if err == nil && generatedAt.Before(state.BuildStartedAt.UTC()) {
			return nil
		}
	}
	next := strings.TrimSpace(report.Next)
	if next == "" {
		if fallback := continueNextCommandForAssessment(codexContinueAssessment{Recovery: report.Recovery}); fallback != "aether continue" {
			next = fallback
		}
	}
	return &activeRecoveryGuidance{
		Summary:          strings.TrimSpace(report.Summary),
		Next:             next,
		ReportPath:       displayDataPath(rel),
		GeneratedAt:      strings.TrimSpace(report.GeneratedAt),
		PartialSuccess:   report.PartialSuccess,
		Recovery:         report.Recovery,
		HasTargetedRoute: next != "" && next != "aether continue",
	}
}

// lifecycleFactsFromStateSnapshot adapts a state a mutating command already
// holds to the same aggregate used by disk-backed orientation. It records the
// state as confirmed in-memory evidence while leaving actors, spend, research,
// and other unobserved domains explicitly unavailable; it never invents them.
func lifecycleFactsFromStateSnapshot(state colony.ColonyState, noColony bool, now time.Time) LifecycleFacts {
	state = normalizeLegacyColonyState(state)
	stateSource := lifecycleSource("state", "(in-memory state)", LifecycleFactConfirmed, "")
	if noColony {
		stateSource = lifecycleSource("state", "(in-memory state)", LifecycleFactMissing, "no active colony state was supplied")
	}
	identity := LifecycleIdentityFacts{
		Name: strings.TrimSpace(lifecycleString(state.ColonyName)), Goal: strings.TrimSpace(lifecycleString(state.Goal)),
		Episode:  lifecycleAcceptedEpisode(state),
		Standing: string(state.State), Milestone: state.Milestone,
		Scope: string(state.EffectiveScope()), Mode: string(state.EffectiveColonyMode()),
	}
	unobserved := func(domain string) LifecycleFactSource {
		return lifecycleUnavailableSource(domain, "(not observed by in-memory state caller)", "the caller supplied state but did not load this fact domain")
	}
	facts := LifecycleFacts{
		CapturedAt: now,
		State:      LifecycleFact[colony.ColonyState]{Value: state, Source: stateSource},
		Identity:   LifecycleFact[LifecycleIdentityFacts]{Value: identity, Source: lifecycleDerivedSource("identity", stateSource)},
		Progress:   LifecycleFact[LifecycleProgressFacts]{Value: LifecycleProgressFacts{CurrentPhase: state.CurrentPhase, Phases: state.Plan.Phases}, Source: lifecycleDerivedSource("progress", stateSource)},
		Actors:     LifecycleFact[[]LifecycleActorFact]{Source: unobserved("actors")},
		Signals:    LifecycleFact[[]colony.PheromoneSignal]{Source: unobserved("signals")},
		Research:   LifecycleFact[LifecycleResearchFacts]{Source: unobserved("research")},
		Memory: LifecycleFact[LifecycleMemoryFacts]{
			Value: LifecycleMemoryFacts{State: state.Memory}, Source: lifecycleDerivedSource("memory", stateSource),
		},
		Verification: LifecycleFact[LifecycleVerificationFacts]{
			Value: LifecycleVerificationFacts{Gates: append([]colony.GateResultEntry(nil), state.GateResults...)}, Source: lifecycleDerivedSource("verification", stateSource),
		},
		ReportedCost: LifecycleFact[LifecycleReportedCostFacts]{Source: unobserved("reported cost")},
		History:      LifecycleFact[[]string]{Value: append([]string(nil), state.Events...), Source: lifecycleDerivedSource("history", stateSource)},
		Blockers:     LifecycleFact[[]colony.FlagEntry]{Source: unobserved("blockers")},
		Evidence: LifecycleFact[LifecycleEvidenceFacts]{
			Value:  LifecycleEvidenceFacts{Receipt: state.LifecycleReceipt, Handoff: state.PauseHandoff, Recovery: state.RecoveryProvenance, Seal: state.SealOutcome, Archive: state.ArchiveReference},
			Source: lifecycleDerivedSource("evidence", stateSource),
		},
		Session: LifecycleFact[colony.SessionFile]{Source: unobserved("session")},
	}
	facts.Timing = lifecycleTiming(state, stateSource, now)
	return facts
}

// loadLifecycleFacts performs one causally read-only orientation load. It uses
// storage.Store only to obtain the already-established data path; every byte is
// read through os.ReadFile/os.ReadDir, avoiding lock creation and every repair,
// registry, session-mirror, timestamp, or Git mutation path.
func loadLifecycleFacts(root string, factStore *storage.Store, now time.Time) (LifecycleFacts, error) {
	root = filepath.Clean(root)
	if factStore == nil {
		return unavailableLifecycleFacts(root, now, "store is not initialized"), nil
	}
	dataDir := factStore.BasePath()
	state, stateSource := readLifecycleState(filepath.Join(dataDir, "COLONY_STATE.json"))
	session, sessionSource := readLifecycleJSON[colony.SessionFile]("session", filepath.Join(dataDir, "session.json"))
	actors, actorSource := readLifecycleActors(filepath.Join(dataDir, "spawn-tree.txt"))
	pheromones, signalSource := readLifecycleJSON[colony.PheromoneFile]("signals", filepath.Join(dataDir, "pheromones.json"))
	research, researchSource := readLifecycleResearch(root, dataDir)
	memory, memorySource := readLifecycleMemory(dataDir, state, stateSource)
	verification, verificationSource := readLifecycleVerification(dataDir, state, stateSource)
	cost, costSource := readLifecycleCost(dataDir)
	flags, blockerSource := readLifecycleJSON[colony.FlagsFile]("blockers", filepath.Join(dataDir, "pending-decisions.json"))

	identity := LifecycleIdentityFacts{
		Goal: strings.TrimSpace(lifecycleString(state.Goal)), Standing: string(state.State),
		Name: strings.TrimSpace(lifecycleString(state.ColonyName)), Milestone: state.Milestone,
		Episode: lifecycleAcceptedEpisode(state),
		Scope:   string(state.EffectiveScope()), Mode: string(state.EffectiveColonyMode()),
	}
	facts := LifecycleFacts{
		Root: root, CapturedAt: now,
		State:        LifecycleFact[colony.ColonyState]{Value: state, Source: stateSource},
		Identity:     LifecycleFact[LifecycleIdentityFacts]{Value: identity, Source: lifecycleDerivedSource("identity", stateSource)},
		Progress:     LifecycleFact[LifecycleProgressFacts]{Value: LifecycleProgressFacts{CurrentPhase: state.CurrentPhase, Phases: state.Plan.Phases}, Source: lifecycleDerivedSource("progress", stateSource)},
		Actors:       LifecycleFact[[]LifecycleActorFact]{Value: actors, Source: actorSource},
		Signals:      LifecycleFact[[]colony.PheromoneSignal]{Value: pheromones.Signals, Source: signalSource},
		Research:     LifecycleFact[LifecycleResearchFacts]{Value: research, Source: researchSource},
		Memory:       LifecycleFact[LifecycleMemoryFacts]{Value: memory, Source: memorySource},
		Verification: LifecycleFact[LifecycleVerificationFacts]{Value: verification, Source: verificationSource},
		Timing:       lifecycleTiming(state, stateSource, now),
		ReportedCost: LifecycleFact[LifecycleReportedCostFacts]{Value: cost, Source: costSource},
		History:      LifecycleFact[[]string]{Value: append([]string(nil), state.Events...), Source: lifecycleDerivedSource("history", stateSource)},
		Blockers:     LifecycleFact[[]colony.FlagEntry]{Value: flags.Decisions, Source: blockerSource},
		Session:      LifecycleFact[colony.SessionFile]{Value: session, Source: sessionSource},
	}
	facts.Evidence = lifecycleEvidence(state, stateSource, session, sessionSource)
	return facts, nil
}

func lifecycleAcceptedEpisode(state colony.ColonyState) string {
	if state.AcceptedCharter == nil {
		return ""
	}
	return strings.TrimSpace(state.AcceptedCharter.EpisodeID)
}
