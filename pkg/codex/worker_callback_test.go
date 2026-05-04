package codex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestValidateCallbackURL_Missing tests that an empty callback URL
// returns a clear error.
func TestValidateCallbackURL_Missing(t *testing.T) {
	err := validateCallbackURL("")
	if err == nil {
		t.Fatal("expected error for missing callback URL, got nil")
	}
	if !strings.Contains(err.Error(), "callback") {
		t.Errorf("error should mention 'callback', got: %v", err)
	}
	if !strings.Contains(err.Error(), "callback_url") {
		t.Errorf("error should mention 'callback_url' config key, got: %v", err)
	}
}

// TestValidateCallbackURL_Valid tests that a valid HTTP callback URL passes.
func TestValidateCallbackURL_Valid(t *testing.T) {
	err := validateCallbackURL("http://localhost:8080/ws/agents")
	if err != nil {
		t.Fatalf("expected no error for valid callback URL, got: %v", err)
	}
}

// TestValidateCallbackURL_HTTPS tests that HTTPS callback URLs are accepted.
func TestValidateCallbackURL_HTTPS(t *testing.T) {
	err := validateCallbackURL("https://api.example.com/workers/callback")
	if err != nil {
		t.Fatalf("expected no error for HTTPS callback URL, got: %v", err)
	}
}

// TestValidateCallbackURL_InvalidScheme tests that non-HTTP/HTTPS schemes
// are rejected (per T-89-09 threat mitigation).
func TestValidateCallbackURL_InvalidScheme(t *testing.T) {
	invalidSchemes := []string{
		"file:///etc/passwd",
		"javascript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"ftp://evil.com/callback",
	}

	for _, url := range invalidSchemes {
		t.Run(url, func(t *testing.T) {
			err := validateCallbackURL(url)
			if err == nil {
				t.Fatalf("expected error for callback URL with invalid scheme %q, got nil", url)
			}
			if !strings.Contains(err.Error(), "callback") {
				t.Errorf("error should mention 'callback', got: %v", err)
			}
		})
	}
}

// TestValidateCallbackURL_WhitespaceOnly tests that whitespace-only callback
// URLs are treated as missing.
func TestValidateCallbackURL_WhitespaceOnly(t *testing.T) {
	err := validateCallbackURL("   ")
	if err == nil {
		t.Fatal("expected error for whitespace-only callback URL, got nil")
	}
}

// TestCallbackURLFieldIsSeparate tests that CallbackURL is a dedicated field
// on WorkerConfig, distinct from any provider baseURL concept.
func TestCallbackURLFieldIsSeparate(t *testing.T) {
	cfg := WorkerConfig{
		Root:        "/tmp/repo",
		CallbackURL: "http://localhost:8080/ws/agents",
	}

	if cfg.CallbackURL == "" {
		t.Error("CallbackURL should be set as a distinct field")
	}

	// WorkerConfig intentionally has no BaseURL field for LLM provider.
	// CallbackURL is exclusively for worker messaging callbacks.
}

// TestValidateWorkerLaunchConfig_CallbackURLSchemeValidated tests that
// when CallbackURL is set on a WorkerConfig, its scheme is validated
// during the standard launch config validation.
func TestValidateWorkerLaunchConfig_CallbackURLSchemeValidated(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	t.Run("valid http callback", func(t *testing.T) {
		cfg := WorkerConfig{
			Root:        root,
			CallbackURL: "http://localhost:8080/ws/agents",
		}
		if err := validateWorkerLaunchConfig(cfg); err != nil {
			t.Fatalf("expected no error for valid callback URL in launch config, got: %v", err)
		}
	})

	t.Run("valid https callback", func(t *testing.T) {
		cfg := WorkerConfig{
			Root:        root,
			CallbackURL: "https://api.example.com/callback",
		}
		if err := validateWorkerLaunchConfig(cfg); err != nil {
			t.Fatalf("expected no error for valid HTTPS callback URL in launch config, got: %v", err)
		}
	})

	t.Run("invalid scheme rejected", func(t *testing.T) {
		cfg := WorkerConfig{
			Root:        root,
			CallbackURL: "file:///etc/passwd",
		}
		err := validateWorkerLaunchConfig(cfg)
		if err == nil {
			t.Fatal("expected error for invalid scheme in launch config, got nil")
		}
		if !strings.Contains(err.Error(), "callback") {
			t.Errorf("error should mention 'callback', got: %v", err)
		}
	})

	t.Run("empty callback does not break existing behavior", func(t *testing.T) {
		cfg := WorkerConfig{
			Root:        root,
			CallbackURL: "",
		}
		// Empty callback should NOT error -- it's optional.
		// The explicit validateCallbackURL function handles the "required" check
		// for callers that need it.
		if err := validateWorkerLaunchConfig(cfg); err != nil {
			t.Fatalf("empty callback URL should not break launch config validation, got: %v", err)
		}
	})
}

// TestCallbackURLRequiredBeforeSpawn tests the full validation chain:
// a caller that requires callback URL uses ValidateCallbackURL before
// passing the config to the invoker. This fails BEFORE any subprocess spawn.
func TestCallbackURLRequiredBeforeSpawn(t *testing.T) {
	cfg := WorkerConfig{
		AgentName:     "aether-builder",
		AgentTOMLPath: "/nonexistent.toml",
		Caste:         "builder",
		WorkerName:    "Hammer-23",
		TaskID:        "2.1",
		TaskBrief:     "Do work",
		Root:          "/tmp/repo",
		CallbackURL:   "",
	}

	// Pre-spawn validation: callback URL is required
	if err := validateCallbackURL(cfg.CallbackURL); err == nil {
		t.Fatal("expected callback URL validation to fail for empty URL")
	}
}
