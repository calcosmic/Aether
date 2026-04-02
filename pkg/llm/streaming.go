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

// AccumulateStream consumes all events from an SSE stream and accumulates
// text content into a StreamResult. Returns an error if the stream fails.
func AccumulateStream(stream *ssestream.Stream[anthropic.MessageStreamEventUnion]) (*StreamResult, error) {
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
			}
		case anthropic.MessageDeltaEvent:
			stopReason = string(variant.Delta.StopReason)
			usage.OutputTokens = variant.Usage.OutputTokens
		}
	}

	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("llm: accumulate stream: %w", err)
	}

	return &StreamResult{
		Text:       text.String(),
		Role:       role,
		Model:      model,
		StopReason: stopReason,
		Usage:      usage,
	}, nil
}
