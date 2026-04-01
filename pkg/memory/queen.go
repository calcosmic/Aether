package memory

import (
	"context"
	"fmt"
	"strings"

	"github.com/aether-colony/aether/pkg/colony"
	"github.com/aether-colony/aether/pkg/events"
	"github.com/aether-colony/aether/pkg/storage"
)

// QueenPromotionResult holds the outcome of a QUEEN.md promotion.
type QueenPromotionResult struct {
	EntryID  string
	Section  string
	QueenPath string
}

// QueenService promotes instincts and patterns to QUEEN.md.
type QueenService struct {
	store *storage.Store
	bus   *events.Bus
}

// NewQueenService creates a new queen promotion service.
func NewQueenService(store *storage.Store, bus *events.Bus) *QueenService {
	return &QueenService{store: store, bus: bus}
}

// PromoteInstinct writes an instinct entry to QUEEN.md under the Patterns section.
func (s *QueenService) PromoteInstinct(ctx context.Context, queenPath string, instinct colony.InstinctEntry, colonyName string) (*QueenPromotionResult, error) {
	if queenPath == "" {
		return nil, fmt.Errorf("queen: queenPath is required")
	}

	var content string
	data, err := s.store.ReadFile(queenPath)
	if err != nil {
		// Create new QUEEN.md with standard structure
		content = fmt.Sprintf("# QUEEN.md\n\n## Patterns\n\n- %s\n", instinct.Trigger)
	} else {
		content = string(data)
		if !strings.Contains(content, "## Patterns") {
			content += "\n## Patterns\n\n"
		}
		// Append pattern after ## Patterns heading
		content = strings.Replace(content, "## Patterns\n", fmt.Sprintf("## Patterns\n\n- %s\n", instinct.Trigger), 1)
	}

	if err := s.store.AtomicWrite(queenPath, []byte(content)); err != nil {
		return nil, fmt.Errorf("queen: write QUEEN.md: %w", err)
	}

	return &QueenPromotionResult{
		EntryID:  instinct.ID,
		Section:  "Patterns",
		QueenPath: queenPath,
	}, nil
}

// PromotePattern writes a content string to QUEEN.md under the Patterns section.
func (s *QueenService) PromotePattern(ctx context.Context, queenPath string, content string, colonyName string) (*QueenPromotionResult, error) {
	if queenPath == "" {
		return nil, fmt.Errorf("queen: queenPath is required")
	}

	var existing string
	data, err := s.store.ReadFile(queenPath)
	if err != nil {
		existing = fmt.Sprintf("# QUEEN.md\n\n## Patterns\n\n- %s\n", content)
	} else {
		existing = string(data)
		if !strings.Contains(existing, "## Patterns") {
			existing += "\n## Patterns\n\n"
		}
		existing = strings.Replace(existing, "## Patterns\n", fmt.Sprintf("## Patterns\n\n- %s\n", content), 1)
	}

	if err := s.store.AtomicWrite(queenPath, []byte(existing)); err != nil {
		return nil, fmt.Errorf("queen: write QUEEN.md: %w", err)
	}

	return &QueenPromotionResult{
		EntryID:  "pattern",
		Section:  "Patterns",
		QueenPath: queenPath,
	}, nil
}
