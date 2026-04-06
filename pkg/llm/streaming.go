package llm

import (
	"fmt"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
)

// StreamResult holds the accumulated result from an SSE stream.
type StreamResult struct {
	Text       string
	Role       string
	Model      string
	StopReason string
	Usage      Usage
}

// StreamHandler is a callback interface for receiving streaming events.
// Implementations can provide real-time progress updates as tokens arrive.
// All methods are optional; the handler may choose to implement only
// the callbacks it cares about.
type StreamHandler interface {
	// OnToken is called for each text token received from the stream.
	OnToken(token string)
	// OnToolStart is called when a tool use block begins.
	OnToolStart(toolName string, toolID string)
	// OnToolEnd is called when a tool use block ends (with the result).
	OnToolEnd(toolName string, toolID string, result string)
	// OnComplete is called when the stream completes successfully.
	OnComplete(result *StreamResult)
	// OnError is called when an error occurs during streaming.
	OnError(err error)
}

// AccumulateStream consumes all events from an SSE stream and accumulates
// text content into a StreamResult. Returns an error if the stream fails.
// The optional handler parameter receives callbacks for streaming events;
// pass nil for silent accumulation.
func AccumulateStream(stream *ssestream.Stream[anthropic.MessageStreamEventUnion], handler StreamHandler) (*StreamResult, error) {
	var text strings.Builder
	var role string
	var model string
	var stopReason string
	var usage Usage

	for stream.Next() {
		event := stream.Current()

		switch variant := event.AsAny().(type) {
		case anthropic.MessageStartEvent:
			role = string(variant.Message.Role)
			model = string(variant.Message.Model)
			usage.InputTokens = variant.Message.Usage.InputTokens
		case anthropic.ContentBlockDeltaEvent:
			delta := variant.Delta
			if delta.Type == "text_delta" {
				text.WriteString(delta.Text)
				if handler != nil {
					handler.OnToken(delta.Text)
				}
			}
		case anthropic.MessageDeltaEvent:
			stopReason = string(variant.Delta.StopReason)
			usage.OutputTokens = variant.Usage.OutputTokens
		}
	}

	if err := stream.Err(); err != nil {
		if handler != nil {
			handler.OnError(err)
		}
		return nil, fmt.Errorf("llm: accumulate stream: %w", err)
	}

	result := &StreamResult{
		Text:       text.String(),
		Role:       role,
		Model:      model,
		StopReason: stopReason,
		Usage:      usage,
	}

	if handler != nil {
		handler.OnComplete(result)
	}

	return result, nil
}
