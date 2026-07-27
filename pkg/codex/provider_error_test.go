package codex

import (
	"strings"
	"testing"
)

// Verbatim shape of the four opencode worker failures of 2026-05-05: the
// provider returned a structured 404 on stdout with exit code 0, and Aether
// reported "parse worker output: no JSON found in output" — both halves false.
const openCodeExpress404 = `{"type":"error","timestamp":1777942557487,"sessionID":"ses_20a5da0d5ffe9pIELS6EMr5pvj","error":{"name":"APIError","data":{"message":"Not Found: Cannot POST /messages","statusCode":404,"isRetryable":false,"metadata":{"url":"http://localhost:4000/messages"}}}}`

const openCodeUvicorn404 = `{"type":"error","timestamp":1777942544000,"sessionID":"ses_x","error":{"name":"APIError","data":{"message":"{\"detail\":\"Not Found\"}","statusCode":404,"isRetryable":false,"metadata":{"url":"http://localhost:4000/messages"}}}}`

func TestDetectProviderErrorEnvelope(t *testing.T) {
	cases := []struct {
		name       string
		output     string
		wantOK     bool
		wantStatus int
	}{
		{"express 404", openCodeExpress404, true, 404},
		{"uvicorn 404", openCodeUvicorn404, true, 404},
		{"jsonl stream with error line", "{\"type\":\"start\"}\n" + openCodeExpress404 + "\n", true, 404},
		{"successful claims payload", `{"ant_name":"x","caste":"scout","status":"completed","summary":"done"}`, false, 0},
		{"claims payload mentioning error", `{"ant_name":"x","status":"completed","summary":"fixed the error handling"}`, false, 0},
		{"error object without status code", `{"type":"error","error":{"name":"weird"}}`, false, 0},
		{"empty", "", false, 0},
		{"plain text", "something went wrong", false, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env, ok := detectProviderErrorEnvelope(tc.output)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && env.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d, want %d", env.StatusCode, tc.wantStatus)
			}
		})
	}
}

func TestProviderErrorEnvelopeMessageNamesProviderLayer(t *testing.T) {
	env, ok := detectProviderErrorEnvelope(openCodeExpress404)
	if !ok {
		t.Fatal("envelope not detected")
	}
	msg := env.Error()
	for _, want := range []string{"provider endpoint not found", "404", "http://localhost:4000/messages", "Cannot POST /messages"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("message %q missing %q", msg, want)
		}
	}
	if strings.Contains(msg, "parse worker output") {
		t.Fatalf("provider failure still labelled as a parse failure: %q", msg)
	}
}

// A genuine claims-parse failure must stay a parse failure — the provider
// detector must not absorb it.
func TestGenuineParseFailureStillReportsParse(t *testing.T) {
	output := `{"is_error":false,"subtype":"success","result":"not json at all"}`
	if _, ok := detectProviderErrorEnvelope(output); ok {
		t.Fatal("provider detector claimed a non-provider payload")
	}
	_, err := parseHostedWorkerOutput("claude", output)
	if err == nil {
		t.Fatal("expected a parse error")
	}
	if strings.Contains(err.Error(), "provider") {
		t.Fatalf("parse failure mislabelled as provider failure: %v", err)
	}
}

// The doubled "parse worker output: parse worker output:" prefix must not
// return, whatever the prefix is renamed to.
func TestNoDoubledErrorPrefix(t *testing.T) {
	for _, output := range []string{
		"",
		"no braces here",
		"{\"not\":\"claims\"}",
		"```json\n{\"broken\": }\n```",
	} {
		_, err := ParseWorkerOutput(output)
		if err == nil {
			continue
		}
		wrapped := classifyWorkerFinalMessageError("parse worker output", err, true)
		if got := strings.Count(wrapped.Error(), "parse worker output"); got > 1 {
			t.Fatalf("prefix appears %d times in %q", got, wrapped.Error())
		}
	}
}
