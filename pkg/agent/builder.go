package agent

import (
	"context"
	"fmt"

	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/llm"
)

// BuilderAgent is a streaming-enabled agent that implements code changes.
// It demonstrates streaming execution by progressively reporting progress.
type BuilderAgent struct {
	name    string
	triggers []Trigger
}

// NewBuilderAgent creates a new builder agent with the given name.
func NewBuilderAgent(name string) *BuilderAgent {
	return &BuilderAgent{
		name: name,
		triggers: []Trigger{
			{Topic: "build.*"},
			{Topic: "task.execute"},
		},
	}
}

// Name returns the agent's unique identifier.
func (b *BuilderAgent) Name() string {
	return b.name
}

// Caste returns the builder caste.
func (b *BuilderAgent) Caste() Caste {
	return CasteBuilder
}

// Triggers returns the event patterns that activate this agent.
func (b *BuilderAgent) Triggers() []Trigger {
	return b.triggers
}

// Execute runs the builder agent's logic (non-streaming fallback).
// This maintains backward compatibility for callers that don't need streaming.
func (b *BuilderAgent) Execute(ctx context.Context, event events.Event) error {
	// Default non-streaming execution
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Simulate work - in real implementation this would execute build tasks
		return b.runBuild(ctx, event, nil)
	}
}

// ExecuteStreaming runs the builder agent with real-time progress updates.
// The handler receives callbacks as the agent progresses through its work.
// Context cancellation is propagated to stop the stream early if needed.
func (b *BuilderAgent) ExecuteStreaming(ctx context.Context, event events.Event, handler llm.StreamHandler) error {
	if handler != nil {
		handler.OnToken(fmt.Sprintf("[%s] Starting build task...\n", b.name))
	}

	select {
	case <-ctx.Done():
		if handler != nil {
			handler.OnError(ctx.Err())
		}
		return ctx.Err()
	default:
		return b.runBuild(ctx, event, handler)
	}
}

// runBuild performs the actual build work, streaming progress if handler is provided.
func (b *BuilderAgent) runBuild(ctx context.Context, event events.Event, handler llm.StreamHandler) error {
	// Check context before each major operation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Stream analysis phase
	if handler != nil {
		handler.OnToken(fmt.Sprintf("[%s] Analyzing task requirements...\n", b.name))
	}

	// Check context
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Stream implementation phase
	if handler != nil {
		handler.OnToken(fmt.Sprintf("[%s] Implementing changes...\n", b.name))
	}

	// Check context
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Stream verification phase
	if handler != nil {
		handler.OnToken(fmt.Sprintf("[%s] Running verification...\n", b.name))
	}

	// Check context
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Complete
	if handler != nil {
		handler.OnComplete(&llm.StreamResult{
			Text:       fmt.Sprintf("[%s] Build completed successfully", b.name),
			Role:       "assistant",
			StopReason: "end_turn",
		})
	}

	return nil
}

// Ensure BuilderAgent implements both Agent and StreamingAgent interfaces.
var _ Agent = (*BuilderAgent)(nil)
var _ StreamingAgent = (*BuilderAgent)(nil)
