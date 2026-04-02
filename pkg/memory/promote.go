package memory

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/aether-colony/aether/pkg/colony"
	"github.com/aether-colony/aether/pkg/events"
	"github.com/aether-colony/aether/pkg/storage"
)

const triggerPrefixLen = 50
const instinctCap = 50

// PromotionResult holds the outcome of an instinct promotion.
type PromotionResult struct {
	Instinct   colony.InstinctEntry
	IsNew      bool
	WasDeduped bool
}

// PromoteService promotes observations to instincts with dedup and provenance.
type PromoteService struct {
	store *storage.Store
	bus   *events.Bus
}

// NewPromoteService creates a new promotion service.
func NewPromoteService(store *storage.Store, bus *events.Bus) *PromoteService {
	return &PromoteService{store: store, bus: bus}
}

// Promote promotes an observation to an instinct.
// Source: Shell instinct-store.sh lines 25-203
func (s *PromoteService) Promote(ctx context.Context, obs colony.Observation, colonyName string) (*PromotionResult, error) {
	now := time.Now().UTC()

	// Step 1: Generate instinct ID: inst_{unix}_{6hex}
	instID, err := generateInstinctID(now)
	if err != nil {
		return nil, fmt.Errorf("generate instinct ID: %w", err)
	}

	// Step 2: Calculate trust score
	daysSince := calcDaysSince(obs.FirstSeen)
	trustInput := TrustInput{
		SourceType: obs.SourceType,
		Evidence:   obs.EvidenceType,
		DaysSince:  daysSince,
	}
	trustResult := Calculate(trustInput)
	trustScore := trustResult.Score
	tierName, _ := Tier(trustScore)

	// Step 3: Calculate confidence
	confidence := RecurrenceConfidence(obs.ObservationCount)

	nowStr := events.FormatTimestamp(now)

	// Step 4: Load existing instincts
	var file colony.InstinctsFile
	if err := s.store.LoadJSON("instincts.json", &file); err != nil {
		file = colony.InstinctsFile{Version: "1.0", Instincts: []colony.InstinctEntry{}}
	}
	if file.Instincts == nil {
		file.Instincts = []colony.InstinctEntry{}
	}

	// Step 5: Check dedup - first 50 chars of trigger
	prefix := triggerPrefix(obs.Content)
	for i := range file.Instincts {
		existing := &file.Instincts[i]
		if existing.Archived {
			continue
		}
		existingPrefix := triggerPrefix(existing.Trigger)
		if existingPrefix == prefix {
			// Dedup: update existing entry
			existing.Confidence = math.Min(existing.Confidence+0.05, 0.9)
			existing.TrustScore = trustScore
			existing.TrustTier = tierName
			nowCopy := nowStr
			existing.Provenance.LastApplied = &nowCopy
			existing.Provenance.ApplicationCount++

			if err := s.store.SaveJSON("instincts.json", file); err != nil {
				return nil, fmt.Errorf("save instincts: %w", err)
			}

			return &PromotionResult{
				Instinct:   *existing,
				IsNew:      false,
				WasDeduped: true,
			}, nil
		}
	}

	// Step 6: Check capacity - if 50+ non-archived, evict lowest trust
	nonArchived := 0
	for _, inst := range file.Instincts {
		if !inst.Archived {
			nonArchived++
		}
	}

	if nonArchived >= instinctCap {
		lowestIdx := -1
		lowestScore := math.MaxFloat64
		for idx, inst := range file.Instincts {
			if !inst.Archived && inst.TrustScore < lowestScore {
				lowestScore = inst.TrustScore
				lowestIdx = idx
			}
		}
		if lowestIdx >= 0 {
			file.Instincts[lowestIdx].Archived = true
		}
	}

	// Step 7: Create new instinct entry
	action := fmt.Sprintf("When %s, apply observed pattern", truncateStr(obs.Content, 100))
	entry := colony.InstinctEntry{
		ID:                 instID,
		Trigger:            obs.Content,
		Action:             action,
		Domain:             obs.WisdomType,
		TrustScore:         trustScore,
		TrustTier:          tierName,
		Confidence:         confidence,
		Provenance: colony.InstinctProvenance{
			Source:           obs.ContentHash,
			SourceType:       obs.SourceType,
			Evidence:         obs.EvidenceType,
			CreatedAt:        nowStr,
			LastApplied:      nil,
			ApplicationCount: 0,
		},
		ApplicationHistory: []interface{}{},
		RelatedInstincts:   []interface{}{},
		Archived:           false,
	}

	file.Instincts = append(file.Instincts, entry)

	// Step 8: Save
	if err := s.store.SaveJSON("instincts.json", file); err != nil {
		return nil, fmt.Errorf("save instincts: %w", err)
	}

	// Step 9: Write graph edge
	s.writeGraphEdge(obs.ContentHash, instID, nowStr)

	// Step 10: Publish event
	s.publishPromoteEvent(ctx, instID, obs.ContentHash)

	return &PromotionResult{
		Instinct:   entry,
		IsNew:      true,
		WasDeduped: false,
	}, nil
}

func triggerPrefix(s string) string {
	if len(s) > triggerPrefixLen {
		return s[:triggerPrefixLen]
	}
	return s
}

func generateInstinctID(now time.Time) (string, error) {
	unix := now.Unix()
	rnd := make([]byte, 3)
	if _, err := rand.Read(rnd); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return fmt.Sprintf("inst_%d_%06x", unix, rnd), nil
}

func truncateStr(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen]
	}
	return s
}

type graphFile struct {
	Edges []graphEdge `json:"edges"`
}

type graphEdge struct {
	From      string `json:"from"`
	To        string `json:"to"`
	EdgeType  string `json:"edge_type"`
	CreatedAt string `json:"created_at"`
}

func (s *PromoteService) writeGraphEdge(from, to string, createdAt string) {
	var gf graphFile
	if err := s.store.LoadJSON("instinct-graph.json", &gf); err != nil {
		gf = graphFile{}
	}
	if gf.Edges == nil {
		gf.Edges = []graphEdge{}
	}

	gf.Edges = append(gf.Edges, graphEdge{
		From:      from,
		To:        to,
		EdgeType:  "promoted_from",
		CreatedAt: createdAt,
	})

	s.store.SaveJSON("instinct-graph.json", gf)
}

func (s *PromoteService) publishPromoteEvent(ctx context.Context, instinctID, sourceHash string) {
	payload, _ := json.Marshal(map[string]string{
		"instinct_id":  instinctID,
		"source_hash": sourceHash,
	})
	s.bus.Publish(ctx, "instinct.promote", payload, "promote")
}

func calcDaysSince(timestamp string) int {
	if timestamp == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return 0
	}
	days := int(time.Since(t).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}
