package learn

import (
	"context"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/memory"
	"github.com/calcosmic/Aether/pkg/storage"
)

// ObservationService wraps pkg/memory.ObservationService for cmd/ consumers.
// cmd/ code should call learn.NewObservationService() instead of memory.NewObservationService().
type ObservationService struct {
	inner *memory.ObservationService
}

// NewObservationService creates a new observation service.
func NewObservationService(store *storage.Store, bus *events.Bus) *ObservationService {
	return &ObservationService{inner: memory.NewObservationService(store, bus)}
}

// Capture captures a new observation with default source/evidence types.
func (s *ObservationService) Capture(ctx context.Context, content, wisdomType, colonyName string) (*ObservationResult, error) {
	return s.inner.Capture(ctx, content, wisdomType, colonyName)
}

// CaptureWithTrust captures a new observation with specified source and evidence types.
func (s *ObservationService) CaptureWithTrust(ctx context.Context, content, wisdomType, colonyName, sourceType, evidenceType string) (*ObservationResult, error) {
	return s.inner.CaptureWithTrust(ctx, content, wisdomType, colonyName, sourceType, evidenceType)
}

// ObservationResult wraps memory.ObservationResult.
type ObservationResult = memory.ObservationResult

// CheckPromotion checks if an observation is eligible for promotion.
// Delegates to memory.CheckPromotion.
func CheckPromotion(obs colony.Observation) (bool, string) {
	return memory.CheckPromotion(obs)
}

// PromoteService wraps pkg/memory.PromoteService for cmd/ consumers.
type PromoteService struct {
	inner *memory.PromoteService
}

// NewPromoteService creates a new promotion service.
func NewPromoteService(store *storage.Store, bus *events.Bus) *PromoteService {
	return &PromoteService{inner: memory.NewPromoteService(store, bus)}
}

// PromotionResult wraps memory.PromotionResult.
type PromotionResult = memory.PromotionResult

// Promote promotes an observation to an instinct.
func (s *PromoteService) Promote(ctx context.Context, obs colony.Observation, colonyName string) (*PromotionResult, error) {
	return s.inner.Promote(ctx, obs, colonyName)
}

// PipelineConfig mirrors memory.PipelineConfig for cmd/ consumers.
type PipelineConfig struct {
	ColonyName string
	QueenPath  string
}

// Pipeline wraps pkg/memory.Pipeline for cmd/ consumers.
type Pipeline struct {
	inner *memory.Pipeline
}

// NewPipeline creates a new pipeline with all services wired together.
func NewPipeline(store *storage.Store, bus *events.Bus, config PipelineConfig) *Pipeline {
	mc := memory.PipelineConfig{
		ColonyName: config.ColonyName,
		QueenPath:  config.QueenPath,
	}
	return &Pipeline{inner: memory.NewPipeline(store, bus, mc)}
}

// ConsolidationResult wraps memory.ConsolidationResult.
type ConsolidationResult = memory.ConsolidationResult

// RunConsolidation runs the full consolidation pipeline.
func (p *Pipeline) RunConsolidation(ctx context.Context) (*ConsolidationResult, error) {
	return p.inner.RunConsolidation(ctx)
}

// ConsolidationService wraps pkg/memory.ConsolidationService for cmd/ consumers.
type ConsolidationService struct {
	inner *memory.ConsolidationService
}

// NewConsolidationService creates a new consolidation service.
func NewConsolidationService(store *storage.Store, bus *events.Bus, queenPath string, colonyName string) *ConsolidationService {
	return &ConsolidationService{inner: memory.NewConsolidationService(store, bus, queenPath, colonyName)}
}

// Run executes the full consolidation pipeline.
func (s *ConsolidationService) Run(ctx context.Context) (*ConsolidationResult, error) {
	return s.inner.Run(ctx)
}

// QueenService wraps pkg/memory.QueenService for cmd/ consumers.
type QueenService struct {
	inner *memory.QueenService
}

// NewQueenService creates a new queen service.
func NewQueenService(store *storage.Store, bus *events.Bus) *QueenService {
	return &QueenService{inner: memory.NewQueenService(store, bus)}
}

// RecurrenceConfidence calculates confidence based on observation count.
// Delegates to memory.RecurrenceConfidence.
func RecurrenceConfidence(observationCount int) float64 {
	return memory.RecurrenceConfidence(observationCount)
}
