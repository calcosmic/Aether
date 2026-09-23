package colony

import (
	"strings"
	"testing"
)

// TestRedactSecretValuesCatchesCommonTokenShapes is CR-03's own guard
// (208-REVIEW.md): a secret VALUE genuinely printed by a failing build or
// test must never survive into an externally-shared bundle. Each case
// proves one named shape is caught and the redaction label is what remains
// in its place -- never the secret itself.
func TestRedactSecretValuesCatchesCommonTokenShapes(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		secret string
	}{
		{"AWS access key", "failed: AWS_ACCESS_KEY_ID=AKIAABCDEFGHIJKLMNOP rejected", "AKIAABCDEFGHIJKLMNOP"},
		{"GitHub token", "remote: ghp_1234567890abcdefghijklmnopqrstuvwxyz invalid", "ghp_1234567890abcdefghijklmnopqrstuvwxyz"},
		{"sk- secret key", "stripe error using sk-live_reportbundlefakekey12345", "sk-live_reportbundlefakekey12345"},
		{"bearer token", `curl failed: Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.fake.token`, "eyJhbGciOiJIUzI1NiJ9.fake.token"},
		{"api key header", "response 401 X-Api-Key: super-secret-value-here", "super-secret-value-here"},
		{"env assignment", "printenv shows DATABASE_PASSWORD=hunter2hunter2hunter2", "hunter2hunter2hunter2"},
		{"credentialed URL", "clone failed: https://user:sup3rSecretPW@example.com/repo.git", "sup3rSecretPW"},
		{"base64-ish run", "token dump: " + strings.Repeat("aB3", 12), strings.Repeat("aB3", 12)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			redacted := RedactSecretValues(tc.input)
			if strings.Contains(redacted, tc.secret) {
				t.Fatalf("%s: secret value survived redaction: input=%q got=%q", tc.name, tc.input, redacted)
			}
			if !strings.Contains(redacted, secretRedactionLabel) {
				t.Fatalf("%s: expected redaction label %q in output, got=%q", tc.name, secretRedactionLabel, redacted)
			}
		})
	}
}

// TestRedactSecretValuesLeavesOrdinaryTextAlone proves the redaction pass
// is not a sledgehammer: ordinary failure prose with no secret-shaped span
// passes through completely unchanged.
func TestRedactSecretValuesLeavesOrdinaryTextAlone(t *testing.T) {
	input := "go build failed: undefined: fmt.Sprintff in cmd/report_cmd.go:42"
	if got := RedactSecretValues(input); got != input {
		t.Fatalf("expected ordinary failure text unchanged, got=%q want=%q", got, input)
	}
}

// TestRedactSecretValuesEmptyInput proves the function is safe to call on
// an empty string (the shape reportBundleFailuresSection would pass for a
// midden entry with no recorded Message).
func TestRedactSecretValuesEmptyInput(t *testing.T) {
	if got := RedactSecretValues(""); got != "" {
		t.Fatalf("expected empty input to stay empty, got=%q", got)
	}
}
