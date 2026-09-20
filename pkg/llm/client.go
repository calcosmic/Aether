package llm

import (
	"context"
	"fmt"
	"os"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Client wraps an Anthropic SDK client with configured model and token limits.
type Client struct {
	apiClient *anthropic.Client
	model     string
	maxTokens int64
}

// ClientOption configures a Client during construction.
type ClientOption func(*Client)

// WithModel sets the model to use for API calls.
func WithModel(model string) ClientOption {
	return func(c *Client) {
		c.model = model
	}
}

// WithMaxTokens sets the maximum number of tokens for API responses.
func WithMaxTokens(n int64) ClientOption {
	return func(c *Client) {
		c.maxTokens = n
	}
}

// WithAPIKey sets the API key explicitly, overriding the ANTHROPIC_API_KEY env var.
func WithAPIKey(key string) ClientOption {
	return func(c *Client) {
		client := anthropic.NewClient(option.WithAPIKey(key))
		c.apiClient = &client
	}
}

// NewClient creates a new LLM client with the given options.
// If no model is specified, defaults to claude-sonnet-4-20250514.
// If no max tokens is specified, defaults to 4096.
// Returns an error if no API key is available (neither explicit nor env var).
func NewClient(opts ...ClientOption) (*Client, error) {
	c := &Client{
		model:     string(anthropic.ModelClaudeSonnet4_20250514),
		maxTokens: 4096,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.apiClient == nil {
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("llm: ANTHROPIC_API_KEY not set and no explicit key provided")
		}
		client := anthropic.NewClient()
		c.apiClient = &client
	}

	return c, nil
}

// MessageResponse holds the parsed response from an LLM call.
type MessageResponse struct {
	ID         string
	Role       string
	Content    []ContentBlock
	StopReason string
	Model      string
	Usage      Usage
}

// ContentBlock represents a single content block in a response.
type ContentBlock struct {
	Type string
	Text string
}

// Usage holds token usage information from an API response.
//
// All four columns Anthropic bills for are carried, not just input and output.
// The provider reports input, cache_read and cache_creation as DISJOINT
// counts; a value carrying only two of them under-reports by the whole cached
// prefix, which on a realistic run is the 186x undercount recorded in
// pkg/codex/usage.go. Phase 196 (D-05) closed that on this second accounting
// lane.
//
// This type deliberately declares NO method. In particular it must never gain
// one that sums its columns: the authoritative total is
// codex.WorkerUsage.BilledTotalTokens(), and a second implementation of that
// arithmetic in a second package is precisely how the two lanes drifted apart.
type Usage struct {
	InputTokens              int64
	CacheReadInputTokens     int64
	CacheCreationInputTokens int64
	OutputTokens             int64
}

// SendMessage sends messages to the LLM and returns the response.
func (c *Client) SendMessage(ctx context.Context, messages []anthropic.MessageParam) (*MessageResponse, error) {
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(c.model),
		MaxTokens: c.maxTokens,
		Messages:  messages,
	}

	msg, err := c.apiClient.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("llm: send message: %w", err)
	}

	return convertSDKMessage(msg), nil
}

// convertSDKMessage converts an Anthropic SDK Message to our MessageResponse.
func convertSDKMessage(msg *anthropic.Message) *MessageResponse {
	if msg == nil {
		return nil
	}

	var blocks []ContentBlock
	for _, block := range msg.Content {
		blocks = append(blocks, ContentBlock{
			Type: string(block.Type),
			Text: block.Text,
		})
	}

	return &MessageResponse{
		ID:         msg.ID,
		Role:       string(msg.Role),
		Content:    blocks,
		StopReason: string(msg.StopReason),
		Model:      string(msg.Model),
		Usage: Usage{
			InputTokens:              msg.Usage.InputTokens,
			CacheReadInputTokens:     msg.Usage.CacheReadInputTokens,
			CacheCreationInputTokens: msg.Usage.CacheCreationInputTokens,
			OutputTokens:             msg.Usage.OutputTokens,
		},
	}
}
