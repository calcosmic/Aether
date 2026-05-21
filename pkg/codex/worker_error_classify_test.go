package codex

import (
	"errors"
	"strings"
	"testing"
)

func TestClassifyProviderError_AuthFailures(t *testing.T) {
	cases := []struct {
		name string
		text string
		want string
	}{
		{"authentication failed", "Error: authentication failed for API key", "provider auth failure"},
		{"unauthorized", "401 Unauthorized: invalid credentials", "provider auth failure"},
		{"API key invalid", "Your API key is invalid or expired", "provider auth failure"},
		{"401 status", "HTTP 401: authentication required", "provider auth failure"},
		{"access denied", "Access denied: insufficient permissions", "provider auth failure"},
		{"invalid key", "Invalid key provided", "provider auth failure"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyProviderError(tc.text)
			if got != tc.want {
				t.Errorf("classifyProviderError(%q) = %q, want %q", tc.text, got, tc.want)
			}
		})
	}
}

func TestClassifyProviderError_RateLimits(t *testing.T) {
	cases := []struct {
		name string
		text string
		want string
	}{
		{"rate limit exceeded", "Error: rate limit exceeded", "rate limited"},
		{"429 status", "HTTP 429: too many requests", "rate limited"},
		{"throttled", "Request throttled by provider", "rate limited"},
		{"quota exceeded", "Monthly quota exceeded", "rate limited"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyProviderError(tc.text)
			if got != tc.want {
				t.Errorf("classifyProviderError(%q) = %q, want %q", tc.text, got, tc.want)
			}
		})
	}
}

func TestClassifyProviderError_Unavailable(t *testing.T) {
	cases := []struct {
		name string
		text string
		want string
	}{
		{"model not found", "Error: model not found", "provider unavailable"},
		{"model unavailable", "The model is currently unavailable", "provider unavailable"},
		{"503 status", "HTTP 503: service unavailable", "provider unavailable"},
		{"overloaded", "Provider is overloaded, try again later", "provider unavailable"},
		{"temporary error", "Temporary error from upstream", "provider unavailable"},
		{"internal server error", "Internal server error", "provider unavailable"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyProviderError(tc.text)
			if got != tc.want {
				t.Errorf("classifyProviderError(%q) = %q, want %q", tc.text, got, tc.want)
			}
		})
	}
}

func TestClassifyProviderError_ContextExceeded(t *testing.T) {
	cases := []struct {
		name string
		text string
		want string
	}{
		{"context length", "Error: context length exceeded", "context exceeded"},
		{"token limit", "Token limit reached for this model", "context exceeded"},
		{"maximum context", "Input exceeds maximum context length", "context exceeded"},
		{"too long for model", "Prompt too long for model", "context exceeded"},
		{"exceeds maximum", "Request exceeds maximum token count", "context exceeded"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyProviderError(tc.text)
			if got != tc.want {
				t.Errorf("classifyProviderError(%q) = %q, want %q", tc.text, got, tc.want)
			}
		})
	}
}

func TestClassifyProviderError_Unknown(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{"random text", "Something completely unrelated went wrong"},
		{"empty string", ""},
		{"json parse error", "unexpected token at position 42"},
		{"go panic", "panic: runtime error: index out of range"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyProviderError(tc.text)
			if got != "" {
				t.Errorf("classifyProviderError(%q) = %q, want empty string", tc.text, got)
			}
		})
	}
}

func TestClassifyProviderError_CaseInsensitive(t *testing.T) {
	// Verify that classification is case-insensitive
	got := classifyProviderError("RATE LIMIT EXCEEDED")
	if got != "rate limited" {
		t.Errorf("classifyProviderError(uppercase) = %q, want %q", got, "rate limited")
	}
}

func TestProviderErrorMessage(t *testing.T) {
	cases := []struct {
		class string
		want  string
	}{
		{"provider auth failure", "Provider authentication failed. Check your API key and try again."},
		{"rate limited", "Rate limited by provider. Wait a moment and retry."},
		{"provider unavailable", "Provider temporarily unavailable. Try again later."},
		{"context exceeded", "Input too long for provider context window. Reduce prompt size and retry."},
		{"unknown class", ""},
		{"", ""},
	}

	for _, tc := range cases {
		t.Run(tc.class, func(t *testing.T) {
			got := providerErrorMessage(tc.class)
			if got != tc.want {
				t.Errorf("providerErrorMessage(%q) = %q, want %q", tc.class, got, tc.want)
			}
		})
	}
}

func TestClassifyWorkerExecutionError_ProviderAuth(t *testing.T) {
	stderr := "Error: authentication failed for API key xyz"
	err := errors.New("exit status 1")

	got := classifyWorkerExecutionError(err, stderr, true)
	if got == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(got.Error(), "provider error") {
		t.Errorf("expected 'provider error' in message, got: %v", got)
	}
	if !strings.Contains(got.Error(), "auth failure") {
		t.Errorf("expected 'auth failure' in message, got: %v", got)
	}
}

func TestClassifyWorkerExecutionError_Fallback(t *testing.T) {
	// Unknown error should fall back to existing behavior
	stderr := "Some random failure text"
	err := errors.New("exit status 1")

	got := classifyWorkerExecutionError(err, stderr, true)
	if got == nil {
		t.Fatal("expected error, got nil")
	}
	if strings.Contains(got.Error(), "provider error") {
		t.Errorf("unexpected 'provider error' in fallback message: %v", got)
	}
	if !strings.Contains(got.Error(), "codex exec failed") {
		t.Errorf("expected 'codex exec failed' in fallback message, got: %v", got)
	}
}
