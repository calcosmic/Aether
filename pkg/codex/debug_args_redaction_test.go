package codex

import (
	"strings"
	"testing"
)

// Redaction must work by argument identity, not position. The M4L artifacts of
// 27 July 2026 recorded `"--permission-mode", "<prompt: 4 bytes>"` while the
// real 19 KB prompt sat at index 1 in full — a privacy leak and a diagnostic
// lie in the same file.
func TestSafeHostedWorkerArgsClaudeVector(t *testing.T) {
	sentinel := "## USER PREFERENCES\nsecret behavioural profile\n" + strings.Repeat("x", 400)
	args := []string{
		"-p", sentinel,
		"--output-format", "json",
		"--json-schema", `{"type":"object","properties":{}}`,
		"--agent", "aether-scout",
		"--permission-mode", "plan",
	}

	out := safeHostedWorkerArgs(args)
	joined := strings.Join(out, " ")

	if strings.Contains(joined, "USER PREFERENCES") {
		t.Fatal("prompt content leaked into debug args")
	}
	if !strings.Contains(out[1], "<prompt:") {
		t.Fatalf("prompt not redacted: %q", out[1])
	}
	if !strings.Contains(out[5], "<schema:") {
		t.Fatalf("schema not redacted: %q", out[5])
	}
	if out[9] != "plan" {
		t.Fatalf("permission mode destroyed: %q (this was the arg the old code mislabelled)", out[9])
	}
	if out[7] != "aether-scout" {
		t.Fatalf("agent name destroyed: %q", out[7])
	}
}

func TestSafeHostedWorkerArgsClaudeSettingsVector(t *testing.T) {
	args := []string{
		"-p", strings.Repeat("p", 300),
		"--output-format", "json",
		"--json-schema", "{}",
		"--agent", "aether-builder",
		"--permission-mode", "acceptEdits",
		"--settings", "/tmp/aether-settings-123.json",
	}
	out := safeHostedWorkerArgs(args)
	if out[len(out)-1] != "/tmp/aether-settings-123.json" {
		t.Fatalf("settings path destroyed: %q — operators need it to reproduce the permission profile", out[len(out)-1])
	}
}

// The OpenCode vector passes the prompt as a bare trailing positional; it must
// still be redacted.
func TestSafeHostedWorkerArgsOpenCodeVector(t *testing.T) {
	prompt := "colony prompt with sk-proj-fake-token-for-test " + strings.Repeat("y", 300)
	args := []string{"run", "--agent", "build", "--format", "json", prompt}
	out := safeHostedWorkerArgs(args)
	last := out[len(out)-1]
	if strings.Contains(last, "sk-proj") {
		t.Fatalf("trailing positional prompt leaked: %q", last)
	}
	if !strings.Contains(last, "<prompt:") {
		t.Fatalf("trailing positional prompt not redacted: %q", last)
	}
}

// Secrets in a NON-last arg must be caught by the sanitizer — the old code
// sanitized nothing.
func TestSafeHostedWorkerArgsSanitizesRetainedArgs(t *testing.T) {
	args := []string{"run", "--env", "token=sk-live-abcdef123456", "--format", "json", strings.Repeat("p", 300)}
	out := safeHostedWorkerArgs(args)
	for _, arg := range out {
		if strings.Contains(arg, "sk-live-abcdef123456") {
			t.Fatalf("secret survived in retained arg: %q", arg)
		}
	}
}

// Invariant per the repo's Definition of Done: no element of the recorded args
// may carry a large payload, whatever it is called and wherever it sits.
func TestSafeHostedWorkerArgsNoLargePayloadSurvives(t *testing.T) {
	big := strings.Repeat("z", 5000)
	for _, args := range [][]string{
		{"-p", big, "--permission-mode", "plan"},
		{"run", "--agent", "x", "--format", "json", big},
		{"-p", big, "--json-schema", big, "--agent", "a", "--permission-mode", "plan"},
	} {
		for i, arg := range safeHostedWorkerArgs(args) {
			if len(arg) > 1000 {
				t.Fatalf("arg %d retains %d bytes: %q...", i, len(arg), arg[:60])
			}
		}
	}
}
