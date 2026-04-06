package llm

import (
	"context"
	"encoding/json"
	"fmt"

	anthropic "github.com/anthropics/anthropic-sdk-go"
)

// ToolFunc is the signature for a tool execution function.
type ToolFunc func(ctx context.Context, input map[string]interface{}) (string, error)

// ToolDef describes a tool that can be registered with a ToolRunner.
type ToolDef struct {
	Name        string
	Description string
	InputSchema interface{}
}

// ToolRunner executes an agentic tool-use loop against the Anthropic API.
// It sends messages, detects tool_use blocks in responses, executes the
// corresponding tool functions, returns results, and repeats until the
// model responds with text only or max iterations are exceeded.
type ToolRunner struct {
	client        *Client
	tools         map[string]ToolFunc
	toolDefs      []anthropic.ToolUnionParam
	maxIterations int
}

// ToolRunnerOption configures a ToolRunner.
type ToolRunnerOption func(*ToolRunner)

// WithToolRunnerMaxIterations sets the maximum number of tool-use loop iterations.
func WithToolRunnerMaxIterations(n int) ToolRunnerOption {
	return func(tr *ToolRunner) {
		tr.maxIterations = n
	}
}

// NewToolRunner creates a new ToolRunner with the given client and tool functions.
// Tool definitions are automatically created from the function names.
// Defaults to a maximum of 10 iterations.
func NewToolRunner(client *Client, funcs map[string]ToolFunc, opts ...ToolRunnerOption) *ToolRunner {
	tr := &ToolRunner{
		client:        client,
		tools:         make(map[string]ToolFunc),
		maxIterations: 10,
	}

	for name, fn := range funcs {
		tr.tools[name] = fn
		tr.toolDefs = append(tr.toolDefs, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        name,
				Description: anthropic.String(fmt.Sprintf("Tool: %s", name)),
			},
		})
	}

	for _, opt := range opts {
		opt(tr)
	}

	return tr
}

// RegisterTool adds a tool to the runner with an explicit schema and function.
func (tr *ToolRunner) RegisterTool(name string, schema interface{}, fn ToolFunc) error {
	if _, exists := tr.tools[name]; exists {
		return fmt.Errorf("llm: tool %q already registered", name)
	}

	tr.tools[name] = fn
	tr.toolDefs = append(tr.toolDefs, anthropic.ToolUnionParam{
		OfTool: &anthropic.ToolParam{
			Name:        name,
			Description: anthropic.String(fmt.Sprintf("Tool: %s", name)),
			InputSchema: anthropic.ToolInputSchemaParam{
				Type: "object",
			},
		},
	})

	return nil
}

// Run executes the tool-use loop using non-streaming API calls.
// It sends messages, checks for tool_use blocks, executes tools, and loops
// until the model responds with text-only content or max iterations are exceeded.
func (tr *ToolRunner) Run(ctx context.Context, messages []anthropic.MessageParam) (*MessageResponse, error) {
	msgs := make([]anthropic.MessageParam, len(messages))
	copy(msgs, messages)

	for i := 0; i < tr.maxIterations; i++ {
		params := anthropic.MessageNewParams{
			Model:     anthropic.Model(tr.client.model),
			MaxTokens: tr.client.maxTokens,
			Messages:  msgs,
		}
		if len(tr.toolDefs) > 0 {
			params.Tools = tr.toolDefs
		}

		msg, err := tr.client.apiClient.Messages.New(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("llm: tool runner iteration %d: %w", i, err)
		}

		// Check for tool_use blocks
		var toolUseBlocks []anthropic.ToolUseBlock
		for _, block := range msg.Content {
			if block.Type == "tool_use" {
				toolUseBlocks = append(toolUseBlocks, block.AsToolUse())
			}
		}

		// No tool use -- return the final response
		if len(toolUseBlocks) == 0 {
			return convertSDKMessage(msg), nil
		}

		// Append the assistant's response to messages
		assistantBlocks := make([]anthropic.ContentBlockParamUnion, 0, len(msg.Content))
		for _, block := range msg.Content {
			switch block.Type {
			case "text":
				tb := block.AsText()
				assistantBlocks = append(assistantBlocks, anthropic.NewTextBlock(tb.Text))
			case "tool_use":
				tu := block.AsToolUse()
				var input interface{}
				json.Unmarshal(tu.Input, &input)
				assistantBlocks = append(assistantBlocks, anthropic.NewToolUseBlock(tu.ID, input, tu.Name))
			}
		}
		msgs = append(msgs, anthropic.NewAssistantMessage(assistantBlocks...))

		// Execute each tool and collect results
		for _, tu := range toolUseBlocks {
			fn, ok := tr.tools[tu.Name]
			if !ok {
				return nil, fmt.Errorf("llm: tool not found: %q", tu.Name)
			}

			var input map[string]interface{}
			if err := json.Unmarshal(tu.Input, &input); err != nil {
				return nil, fmt.Errorf("llm: parse tool input for %q: %w", tu.Name, err)
			}

			result, err := fn(ctx, input)
			if err != nil {
				// Return error result to the model
				msgs = append(msgs, anthropic.NewUserMessage(
					anthropic.NewToolResultBlock(tu.ID, fmt.Sprintf("error: %v", err), true),
				))
			} else {
				msgs = append(msgs, anthropic.NewUserMessage(
					anthropic.NewToolResultBlock(tu.ID, result, false),
				))
			}
		}
	}

	return nil, fmt.Errorf("llm: max iterations (%d) exceeded in tool-use loop", tr.maxIterations)
}

// RunStreaming executes the tool-use loop with streaming for the final response.
// Tool execution uses non-streaming calls; only the last response is streamed
// for text accumulation via AccumulateStream.
func (tr *ToolRunner) RunStreaming(ctx context.Context, messages []anthropic.MessageParam) (*StreamResult, error) {
	msgs := make([]anthropic.MessageParam, len(messages))
	copy(msgs, messages)

	for i := 0; i < tr.maxIterations; i++ {
		// Use non-streaming to detect tool use
		params := anthropic.MessageNewParams{
			Model:     anthropic.Model(tr.client.model),
			MaxTokens: tr.client.maxTokens,
			Messages:  msgs,
		}
		if len(tr.toolDefs) > 0 {
			params.Tools = tr.toolDefs
		}

		msg, err := tr.client.apiClient.Messages.New(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("llm: tool runner stream iteration %d: %w", i, err)
		}

		// Check for tool_use blocks
		var toolUseBlocks []anthropic.ToolUseBlock
		for _, block := range msg.Content {
			if block.Type == "tool_use" {
				toolUseBlocks = append(toolUseBlocks, block.AsToolUse())
			}
		}

		// No tool use -- stream the final response for text accumulation
		if len(toolUseBlocks) == 0 {
			stream := tr.client.apiClient.Messages.NewStreaming(ctx, params)
			return AccumulateStream(stream, nil)
		}

		// Append assistant response and tool results, then continue loop
		assistantBlocks := make([]anthropic.ContentBlockParamUnion, 0, len(msg.Content))
		for _, block := range msg.Content {
			switch block.Type {
			case "text":
				assistantBlocks = append(assistantBlocks, anthropic.NewTextBlock(block.AsText().Text))
			case "tool_use":
				tu := block.AsToolUse()
				var input interface{}
				json.Unmarshal(tu.Input, &input)
				assistantBlocks = append(assistantBlocks, anthropic.NewToolUseBlock(tu.ID, input, tu.Name))
			}
		}
		msgs = append(msgs, anthropic.NewAssistantMessage(assistantBlocks...))

		for _, tu := range toolUseBlocks {
			fn, ok := tr.tools[tu.Name]
			if !ok {
				return nil, fmt.Errorf("llm: tool not found: %q", tu.Name)
			}

			var input map[string]interface{}
			if err := json.Unmarshal(tu.Input, &input); err != nil {
				return nil, fmt.Errorf("llm: parse tool input for %q: %w", tu.Name, err)
			}

			result, err := fn(ctx, input)
			if err != nil {
				msgs = append(msgs, anthropic.NewUserMessage(
					anthropic.NewToolResultBlock(tu.ID, fmt.Sprintf("error: %v", err), true),
				))
			} else {
				msgs = append(msgs, anthropic.NewUserMessage(
					anthropic.NewToolResultBlock(tu.ID, result, false),
				))
			}
		}
	}

	return nil, fmt.Errorf("llm: max iterations (%d) exceeded in tool-use loop", tr.maxIterations)
}
