package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/aether-colony/aether/pkg/colony"
	"github.com/aether-colony/aether/pkg/events"
	"github.com/aether-colony/aether/pkg/storage"
)

// ConsolidationResult holds the outcome of a phase-end consolidation run.
type ConsolidationResult struct {
	InstinctsDecayed    int
	InstinctsArchived   int
	ObservationsDecayed int
	PromotionCandidates []string // content hashes of observations eligible for promotion
	QueenEligible       []string // instinct IDs eligible for QUEEN.md promotion
	Errors              []error
}

// ConsolidationService runs phase-end consolidation: decay, archive, and check promotions.
// Source: Shell consolidation.sh lines 23-127
type ConsolidationService struct {
	store      *storage.Store
	bus        *events.Bus
	queenPath  string
	colonyName string
}

// NewConsolidationService creates a new consolidation service.
func NewConsolidationService(store *storage.Store, bus *events.Bus, queenPath string, colonyName string) *ConsolidationService {
	return &ConsolidationService{
		store:      store,
		bus:        bus,
		queenPath:  queenPath,
		colonyName: colonyName,
	}
}

// Run executes the full consolidation pipeline: decay instincts, archive low-trust,
// decay observations, check promotion candidates, identify queen-eligible instincts.
// Each step is non-blocking: failures are recorded in result.Errors but do not stop the pipeline.
func (s *ConsolidationService) Run(ctx context.Context) (*ConsolidationResult, error) {
	result := &ConsolidationResult{
		PromotionCandidates: []string{},
		QueenEligible:       []string{},
	}

	// STEP 1 - Nurse: Recalculate trust scores using decay for all instincts.
	// Also track raw decay values (before 0.2 floor) for archival decision.
	var instincts colony.InstinctsFile
	instinctsLoaded := false
	rawDecayScores := map[int]float64{}
	if err := s.store.LoadJSON("instincts.json", &instincts); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("load instincts: %w", err))
	} else {
		instinctsLoaded = true
		result.InstinctsDecayed = s.decayInstincts(&instincts, rawDecayScores)
		if err := s.store.SaveJSON("instincts.json", instincts); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("save decayed instincts: %w", err))
		}
	}

	// STEP 2 - Janitor: Archive instincts whose raw decayed score < 0.2.
	if instinctsLoaded {
		result.InstinctsArchived = s.archiveBelowFloor(&instincts, rawDecayScores)
		if err := s.store.SaveJSON("instincts.json", instincts); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("save archived instincts: %w", err))
		}
	}

	// STEP 3 - Observation decay
	var obsFile colony.LearningFile
	obsLoaded := false
	if err := s.store.LoadJSON("learning-observations.json", &obsFile); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("load observations: %w", err))
	} else {
		obsLoaded = true
		result.ObservationsDecayed = s.decayObservations(&obsFile)
		if err := s.store.SaveJSON("learning-observations.json", obsFile); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("save decayed observations: %w", err))
		}
	}

	// STEP 4 - Check promotion candidates from observations
	if obsLoaded {
		for _, obs := range obsFile.Observations {
			eligible, _ := CheckPromotion(obs)
			if eligible {
				result.PromotionCandidates = append(result.PromotionCandidates, obs.ContentHash)
			}
		}
	}

	// STEP 5 - Check queen-eligible instincts
	if instinctsLoaded {
		for _, inst := range instincts.Instincts {
			if !inst.Archived && inst.Confidence >= 0.75 && inst.Provenance.ApplicationCount >= 3 {
				result.QueenEligible = append(result.QueenEligible, inst.ID)
			}
		}
	}

	// Publish consolidation.phase_end event with summary
	s.publishConsolidationEvent(ctx, result)

	return result, nil
}

// decayInstincts applies trust decay to all non-archived instincts.
// Populates rawDecayScores with the pre-floor decay values for archival decisions.
func (s *ConsolidationService) decayInstincts(file *colony.InstinctsFile, rawDecayScores map[int]float64) int {
	count := 0
	for i := range file.Instincts {
		inst := &file.Instincts[i]
		if inst.Archived {
			continue
		}
		days := daysSinceTimestamp(inst.Provenance.CreatedAt)
		// Store raw decay (before floor) for archival decision
		raw := rawDecay(inst.TrustScore, days)
		rawDecayScores[i] = raw
		// Apply floored decay for stored value
		decayed := Decay(inst.TrustScore, days)
		inst.TrustScore = decayed
		tierName, _ := Tier(decayed)
		inst.TrustTier = tierName
		count++
	}
	return count
}

// archiveBelowFloor marks instincts with raw decayed score below 0.2 as archived.
func (s *ConsolidationService) archiveBelowFloor(file *colony.InstinctsFile, rawDecayScores map[int]float64) int {
	count := 0
	for i := range file.Instincts {
		inst := &file.Instincts[i]
		if inst.Archived {
			continue
		}
		if raw, ok := rawDecayScores[i]; ok && raw < 0.2 {
			inst.Archived = true
			count++
		}
	}
	return count
}

// rawDecay applies half-life decay without the 0.2 floor.
func rawDecay(score float64, days int) float64 {
	return score * math.Pow(0.5, float64(days)/60.0)
}

// decayObservations applies trust decay to all observations with trust scores.
func (s *ConsolidationService) decayObservations(file *colony.LearningFile) int {
	count := 0
	for i := range file.Observations {
		obs := &file.Observations[i]
		if obs.TrustScore == nil {
			continue
		}
		days := daysSinceTimestamp(obs.FirstSeen)
		decayed := Decay(*obs.TrustScore, days)
		obs.TrustScore = &decayed
		count++
	}
	return count
}

// publishConsolidationEvent publishes the consolidation.phase_end event.
func (s *ConsolidationService) publishConsolidationEvent(ctx context.Context, result *ConsolidationResult) {
	payload, _ := json.Marshal(map[string]int{
		"instincts_decayed":    result.InstinctsDecayed,
		"instincts_archived":   result.InstinctsArchived,
		"observations_decayed": result.ObservationsDecayed,
		"promotion_candidates": len(result.PromotionCandidates),
		"queen_eligible":       len(result.QueenEligible),
	})
	s.bus.Publish(ctx, "consolidation.phase_end", payload, "consolidation")
}

// daysSinceTimestamp calculates days since an ISO-8601 timestamp.
func daysSinceTimestamp(timestamp string) int {
	if timestamp == "" {
		return 0
	}
	t, err := time.Parse("2006-01-02T15:04:05Z", timestamp)
	if err != nil {
		return 0
	}
	days := int(time.Since(t).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}
