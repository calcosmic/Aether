package memory

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/aether-colony/aether/pkg/colony"
	"github.com/aether-colony/aether/pkg/events"
	"github.com/aether-colony/aether/pkg/storage"
)

// PipelineConfig holds configuration for the wisdom pipeline.
type PipelineConfig struct {
	ColonyName string
	QueenPath  string // path to QUEEN.md relative to store
}

// Pipeline wires all wisdom services together via event bus subscriptions.
// Observations flow through: capture -> trust scoring -> auto-promotion -> instinct creation -> QUEEN.md writing.
type Pipeline struct {
	config      PipelineConfig
	store       *storage.Store
	bus         *events.Bus
	Observe     *ObservationService
	Promote     *PromoteService
	Queen       *QueenService
	Consolidate *ConsolidationService

	observeCh        <-chan events.Event
	consolidationCh  <-chan events.Event
	cancel           context.CancelFunc
	wg               sync.WaitGroup
}

// NewPipeline creates a new pipeline with all services wired together.
func NewPipeline(store *storage.Store, bus *events.Bus, config PipelineConfig) *Pipeline {
	return &Pipeline{
		config:      config,
		store:       store,
		bus:         bus,
		Observe:     NewObservationService(store, bus),
		Promote:     NewPromoteService(store, bus),
		Queen:       NewQueenService(store, bus),
		Consolidate: NewConsolidationService(store, bus, config.QueenPath, config.ColonyName),
	}
}

// Start subscribes to events and begins processing in background goroutines.
func (p *Pipeline) Start(ctx context.Context) error {
	// Subscribe to observation events for auto-promotion
	ch, err := p.bus.Subscribe("learning.observe")
	if err != nil {
		return err
	}
	p.observeCh = ch

	// Subscribe to consolidation events
	cch, err := p.bus.Subscribe("consolidation.*")
	if err != nil {
		return err
	}
	p.consolidationCh = cch

	ctx, p.cancel = context.WithCancel(ctx)

	// Start observation -> promotion loop
	p.wg.Add(1)
	go p.observeLoop(ctx)

	return nil
}

// observeLoop listens for observation events and triggers auto-promotion
// when an observation meets promotion criteria.
func (p *Pipeline) observeLoop(ctx context.Context) {
	defer p.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-p.observeCh:
			if !ok {
				return
			}
			p.handleObserveEvent(ctx, evt)
		}
	}
}

// handleObserveEvent processes an observation event and promotes if eligible.
func (p *Pipeline) handleObserveEvent(ctx context.Context, evt events.Event) {
	var payload struct {
		Content     string `json:"content"`
		WisdomType  string `json:"wisdom_type"`
		ColonyName  string `json:"colony_name"`
	}
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return
	}

	// Load the observation to check promotion eligibility
	var obsFile colony.LearningFile
	if err := p.store.LoadJSON("learning-observations.json", &obsFile); err != nil {
		return
	}

	// Find the observation by content
	for _, obs := range obsFile.Observations {
		if obs.Content == payload.Content {
			eligible, _ := CheckPromotion(obs)
			if !eligible {
				return
			}

			// Promote to instinct
			_, err := p.Promote.Promote(ctx, obs, payload.ColonyName)
			if err != nil {
				log.Printf("pipeline: auto-promote failed: %v", err)
				return
			}
			return
		}
	}
}

// Stop unsubscribes from events and waits for goroutines to finish.
func (p *Pipeline) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()

	if p.observeCh != nil {
		p.bus.Unsubscribe("learning.observe", p.observeCh)
	}
	if p.consolidationCh != nil {
		p.bus.Unsubscribe("consolidation.*", p.consolidationCh)
	}
	p.bus.Close()
}

// RunConsolidation runs the consolidation cycle and acts on the results:
// promotes eligible observations and queen-eligible instincts.
func (p *Pipeline) RunConsolidation(ctx context.Context) (*ConsolidationResult, error) {
	result, err := p.Consolidate.Run(ctx)
	if err != nil {
		return result, err
	}

	// Promote eligible observations to instincts
	for _, hash := range result.PromotionCandidates {
		obs := p.findObservation(hash)
		if obs == nil {
			continue
		}
		_, err := p.Promote.Promote(ctx, *obs, p.config.ColonyName)
		if err != nil {
			log.Printf("pipeline: consolidate promote %s failed: %v", hash, err)
			continue
		}
	}

	// Promote queen-eligible instincts to QUEEN.md
	for _, instID := range result.QueenEligible {
		inst := p.findInstinct(instID)
		if inst == nil {
			continue
		}
		_, err := p.Queen.PromoteInstinct(ctx, p.config.QueenPath, *inst, p.config.ColonyName)
		if err != nil {
			log.Printf("pipeline: queen promote %s failed: %v", instID, err)
			continue
		}
	}

	return result, nil
}

// findObservation looks up an observation by content hash.
func (p *Pipeline) findObservation(contentHash string) *colony.Observation {
	var obsFile colony.LearningFile
	if err := p.store.LoadJSON("learning-observations.json", &obsFile); err != nil {
		return nil
	}
	for i := range obsFile.Observations {
		if obsFile.Observations[i].ContentHash == contentHash {
			return &obsFile.Observations[i]
		}
	}
	return nil
}

// findInstinct looks up an instinct by ID.
func (p *Pipeline) findInstinct(id string) *colony.InstinctEntry {
	var instFile colony.InstinctsFile
	if err := p.store.LoadJSON("instincts.json", &instFile); err != nil {
		return nil
	}
	for i := range instFile.Instincts {
		if instFile.Instincts[i].ID == id {
			return &instFile.Instincts[i]
		}
	}
	return nil
}
