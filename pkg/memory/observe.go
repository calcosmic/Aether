package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/aether-colony/aether/pkg/colony"
	"github.com/aether-colony/aether/pkg/events"
	"github.com/aether-colony/aether/pkg/storage"
)

// RecurrenceConfidence calculates the confidence based on observation count.
// Source: learning.sh lines 469-475
// Formula: min(0.7 + (observationCount - 1) * 0.05, 0.9)
func RecurrenceConfidence(observationCount int) float64 {
	if observationCount < 1 {
		observationCount = 1
	}
	v := 0.7 + float64(observationCount-1)*0.05
	return math.Min(v, 0.9)
}

// ObservationResult holds the result of an observation capture.
type ObservationResult struct {
	Observation       colony.Observation
	IsNew             bool
	PromotionEligible bool
	PromotionReason   string
}

// ObservationService captures and stores observations with trust scoring.
// Source: Shell learning.sh lines 97-381
type ObservationService struct {
	store *storage.Store
	bus   *events.Bus
}

// NewObservationService creates a new observation service.
func NewObservationService(store *storage.Store, bus *events.Bus) *ObservationService {
	return &ObservationService{store: store, bus: bus}
}

// Capture captures a new observation with default source/evidence types.
// Uses source_type="observation" and evidence_type="anecdotal" for new entries.
func (s *ObservationService) Capture(ctx context.Context, content, wisdomType, colonyName string) (*ObservationResult, error) {
	return s.CaptureWithTrust(ctx, content, wisdomType, colonyName, "observation", "anecdotal")
}

// CaptureWithTrust captures a new observation with specified source and evidence types.
func (s *ObservationService) CaptureWithTrust(ctx context.Context, content, wisdomType, colonyName, sourceType, evidenceType string) (*ObservationResult, error) {
	// Compute content hash for dedup: SHA-256 of content + ":" + wisdomType
	h := sha256.Sum256([]byte(content + ":" + wisdomType))
	contentHash := "sha256:" + hex.EncodeToString(h[:])

	now := time.Now().UTC()
	nowStr := events.FormatTimestamp(now)

	// Load existing observations
	var file colony.LearningFile
	if err := s.store.LoadJSON("learning-observations.json", &file); err != nil {
		// File does not exist - create empty
		file = colony.LearningFile{Observations: []colony.Observation{}}
	}
	if file.Observations == nil {
		file.Observations = []colony.Observation{}
	}

	// Backfill legacy entries
	backfillLegacy(file.Observations)

	// Search for existing entry by contentHash
	for i := range file.Observations {
		obs := &file.Observations[i]
		if obs.ContentHash == contentHash {
			// Found: increment count, update timestamps, recalculate trust
			obs.ObservationCount++
			obs.LastSeen = nowStr
			if !containsString(obs.Colonies, colonyName) {
				obs.Colonies = append(obs.Colonies, colonyName)
			}

			days := daysSinceFirstSeen(obs)
			trustInput := TrustInput{
				SourceType: obs.SourceType,
				Evidence:   obs.EvidenceType,
				DaysSince:  days,
			}
			trustResult := Calculate(trustInput)
			score := trustResult.Score
			obs.TrustScore = &score

			// Save updated file
			if err := s.store.SaveJSON("learning-observations.json", file); err != nil {
				return nil, fmt.Errorf("save observations: %w", err)
			}

			// Publish learning.observe event
			payload, _ := json.Marshal(map[string]string{
				"content":      content,
				"wisdom_type":  wisdomType,
				"colony_name":  colonyName,
			})
			s.bus.Publish(ctx, "learning.observe", payload, "observe")

			eligible, reason := CheckPromotion(*obs)
			return &ObservationResult{
				Observation:       *obs,
				IsNew:             false,
				PromotionEligible: eligible,
				PromotionReason:   reason,
			}, nil
		}
	}

	// New observation
	days := 0
	trustInput := TrustInput{
		SourceType: sourceType,
		Evidence:   evidenceType,
		DaysSince:  days,
	}
	trustResult := Calculate(trustInput)
	score := trustResult.Score

	newObs := colony.Observation{
		ContentHash:      contentHash,
		Content:          content,
		WisdomType:       wisdomType,
		ObservationCount: 1,
		FirstSeen:        nowStr,
		LastSeen:         nowStr,
		Colonies:         []string{colonyName},
		TrustScore:       &score,
		SourceType:       sourceType,
		EvidenceType:     evidenceType,
		CompressionLevel: 0,
	}

	file.Observations = append(file.Observations, newObs)

	// Save
	if err := s.store.SaveJSON("learning-observations.json", file); err != nil {
		return nil, fmt.Errorf("save observations: %w", err)
	}

	// Publish learning.observe event
	payload, _ := json.Marshal(map[string]string{
		"content":      content,
		"wisdom_type":  wisdomType,
		"colony_name":  colonyName,
	})
	s.bus.Publish(ctx, "learning.observe", payload, "observe")

	eligible, reason := CheckPromotion(newObs)
	return &ObservationResult{
		Observation:       newObs,
		IsNew:             true,
		PromotionEligible: eligible,
		PromotionReason:   reason,
	}, nil
}

// CheckPromotion checks if an observation should be promoted to an instinct.
func CheckPromotion(obs colony.Observation) (bool, string) {
	// Check trust threshold first (>= 0.50)
	if obs.TrustScore != nil && *obs.TrustScore >= 0.50 {
		return true, "trust_threshold"
	}
	// Check similar patterns (3+ observations)
	if obs.ObservationCount >= 3 {
		return true, "similar_patterns"
	}
	// Check wisdom type threshold
	if threshold, ok := WisdomThresholds[obs.WisdomType]; ok {
		if threshold.Auto > 0 && obs.ObservationCount >= threshold.Auto {
			return true, "wisdom_threshold"
		}
	}
	return false, ""
}

// WisdomThresholdEntry holds promotion thresholds for a wisdom type.
type WisdomThresholdEntry struct {
	Propose int
	Auto    int
}

// WisdomThresholds maps wisdom types to their promotion thresholds.
var WisdomThresholds = map[string]WisdomThresholdEntry{
	"build_learning": {0, 0},
	"instinct":       {0, 0},
	"philosophy":     {1, 3},
	"pattern":        {1, 1},
	"redirect":       {1, 2},
	"stack":          {1, 2},
	"decree":         {0, 0},
	"failure":        {1, 2},
}

func backfillLegacy(observations []colony.Observation) {
	for i := range observations {
		obs := &observations[i]
		if obs.TrustScore == nil {
			defaultTrust := 0.49
			obs.TrustScore = &defaultTrust
			if obs.SourceType == "" {
				obs.SourceType = "legacy"
			}
			if obs.EvidenceType == "" {
				obs.EvidenceType = "indirect"
			}
		}
	}
}

func daysSinceFirstSeen(obs *colony.Observation) int {
	t, err := time.Parse("2006-01-02T15:04:05Z", obs.FirstSeen)
	if err != nil {
		return 0
	}
	days := int(time.Since(t).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
