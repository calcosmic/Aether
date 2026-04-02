// Package llm provides a client abstraction for LLM providers,
// starting with Anthropic SDK integration for colony worker interactions.
//
// The package offers three main capabilities:
//   - Client: wraps the Anthropic SDK for sending messages and receiving responses
//   - Streaming: accumulates SSE stream events into complete text responses
//   - Tools: implements an agentic tool-use loop with registered tool functions
package llm
