package cmd

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
)

// rawWrapperCommandRe finds an `aether <verb>` mention that survived
// translation. It deliberately ignores an env-prefixed invocation
// (AETHER_OUTPUT_MODE=visual aether ...) because that form is a literal shell
// command a wrapper executes, never something a user types.
var rawWrapperCommandRe = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*=\S*\s+)?\baether ([a-z][a-z0-9-]*)`)

// rawWrapperCommandsIn returns the wrapped verbs still named in their raw CLI
// form in text.
func rawWrapperCommandsIn(text string) []string {
	var found []string
	for _, groups := range rawWrapperCommandRe.FindAllStringSubmatch(text, -1) {
		if strings.TrimSpace(groups[1]) != "" {
			continue
		}
		if !wrapperCommandNames[groups[2]] {
			continue
		}
		found = append(found, groups[2])
	}
	return found
}

// TestVisualOutputNeverLeaksRawWrapperCommands is the structural guard behind
// translating command names inside writeVisualOutput.
//
// Before that change renderNextUp translated its own hints while every other
// producer of visual text did not, so a Claude Code user was told to run
// `aether continue`, `aether init "your goal"` and `aether patrol` — none of
// which exist for them. Fixing the individual strings would not have held:
// there are roughly 500 such mentions across cmd/, and each new one would have
// had to remember. This asserts the property at the exit instead, so any future
// prose is covered whether or not its author knew about the rule.
func TestVisualOutputNeverLeaksRawWrapperCommands(t *testing.T) {
	// Text mixing every shape the codebase actually emits: a prose hint, a
	// column-aligned banner line, an error hint, a bare mention, an unwrapped
	// verb that must survive, and a literal env-prefixed shell invocation.
	sample := strings.Join([]string{
		"Run `aether continue` to verify worker claims and advance.",
		"  aether lay-eggs          Set up Aether in this repo",
		"Run `aether patrol` for diagnostics or `aether status` to check colony health.",
		"No project plan. Run `aether plan` first.",
		"Publish with `aether publish` and verify with `aether integrity`.",
		"AETHER_OUTPUT_MODE=visual aether build 1",
		// Recovery hints carry the flags that make them work. The slash
		// wrappers forward $ARGUMENTS, so the flag variants are real commands a
		// user can type and must be named the same way.
		"Run `aether build 2 --force` to regenerate the build manifest.",
		"Run `aether continue --skip-watchers --reconcile-task 2.1` to recover.",
		"Run `aether update --force --download-binary` to refresh the runtime.",
		"Run `aether maintenance recovery-inspect` to inspect recovery evidence.",
		// skip-phase has no wrapper, so its flagged form must stay raw.
		"Run `aether skip-phase 2 --force` only to abandon the phase.",
	}, "\n")

	for _, platform := range []string{"claude", "opencode"} {
		t.Run(platform, func(t *testing.T) {
			t.Setenv("AETHER_OUTPUT_MODE", "visual")
			t.Setenv("AETHER_PLATFORM", platform)

			var buf bytes.Buffer
			writeVisualOutput(&buf, sample)
			got := buf.String()

			if leaked := rawWrapperCommandsIn(got); len(leaked) > 0 {
				t.Errorf("raw wrapper commands reached %s visual output: %v\n%s", platform, leaked, got)
			}
			for _, want := range []string{
				"/ant-continue", "/ant-lay-eggs", "/ant-patrol", "/ant-status", "/ant-plan",
				// Flags must survive the rewrite intact — a recovery hint that
				// loses its --force is worse than one that was never translated.
				"/ant-build 2 --force",
				"/ant-continue --skip-watchers --reconcile-task 2.1",
				"/ant-update --force --download-binary",
				// Recovery guidance is shown at the worst possible moment to
				// hand someone a command they cannot type.
				"/ant-maintenance recovery-inspect",
			} {
				if !strings.Contains(got, want) {
					t.Errorf("expected %s in %s output, got:\n%s", want, platform, got)
				}
			}
			// A verb with no wrapper keeps the raw form even when flagged.
			if !strings.Contains(got, "aether skip-phase 2 --force") {
				t.Errorf("expected unwrapped flagged command to survive on %s, got:\n%s", platform, got)
			}
			// Commands with no wrapper must survive verbatim: rewriting them
			// would invent `/ant-publish` and `/ant-integrity`, which is worse
			// than showing the real CLI form.
			for _, want := range []string{"aether publish", "aether integrity"} {
				if !strings.Contains(got, want) {
					t.Errorf("expected unwrapped %q to survive on %s, got:\n%s", want, platform, got)
				}
			}
			// A literal shell invocation a wrapper executes must survive too.
			if !strings.Contains(got, "AETHER_OUTPUT_MODE=visual aether build 1") {
				t.Errorf("env-prefixed shell invocation was rewritten on %s, got:\n%s", platform, got)
			}
		})
	}

	t.Run("codex", func(t *testing.T) {
		t.Setenv("AETHER_OUTPUT_MODE", "visual")
		t.Setenv("AETHER_PLATFORM", "codex")

		var buf bytes.Buffer
		writeVisualOutput(&buf, sample)
		got := buf.String()

		// Codex exposes only nine public skills. Other actions and literal
		// shell invocations must keep their executable CLI spelling.
		if strings.Contains(got, "/ant-") {
			t.Errorf("codex output must not name slash wrappers, got:\n%s", got)
		}
		for _, want := range []string{
			"Run `$ant-continue`", "Run `$ant-plan`",
			"$ant-build 2 --force", "$ant-continue --skip-watchers --reconcile-task 2.1",
			"aether lay-eggs", "aether patrol", "aether status", "aether publish", "aether integrity",
			"aether update --force --download-binary", "aether maintenance recovery-inspect",
			"aether skip-phase 2 --force", "AETHER_OUTPUT_MODE=visual aether build 1",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("codex output missing %q:\n%s", want, got)
			}
		}
		for _, forbidden := range []string{
			"Run `aether continue", "Run `aether plan", "Run `aether build",
			"$ant-status", "$ant-patrol", "$ant-maintenance", "$ant-update", "$ant-skip-phase",
		} {
			if strings.Contains(got, forbidden) {
				t.Errorf("codex output advertises incorrect spelling %q:\n%s", forbidden, got)
			}
		}
	})
}

// TestOutputErrorTranslatesCommandNames pins the error path specifically. An
// error is the moment a user most needs a command they can actually type, and
// it was the path that leaked longest: outputError wrote to stderr with
// fmt.Fprint, bypassing the only function that translated anything.
func TestOutputErrorTranslatesCommandNames(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("AETHER_PLATFORM", "claude")

	var buf bytes.Buffer
	oldStderr := stderr
	stderr = &buf
	defer func() { stderr = oldStderr }()

	outputError(1, "No project plan. Run `aether plan` first.", nil)

	got := buf.String()
	if leaked := rawWrapperCommandsIn(got); len(leaked) > 0 {
		t.Errorf("raw wrapper commands reached error output: %v\n%s", leaked, got)
	}
	if !strings.Contains(got, "/ant-plan") {
		t.Errorf("expected /ant-plan in error output, got:\n%s", got)
	}
}

// TestJSONOutputKeepsRawCommandNames pins the other half of the contract. JSON
// is the machine surface: wrappers and the TypeScript host parse it and execute
// the commands it names. Translating there would hand a wrapper `/ant-continue`
// to exec, which is not a binary.
func TestJSONOutputKeepsRawCommandNames(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	t.Setenv("AETHER_PLATFORM", "claude")

	var buf bytes.Buffer
	oldStderr := stderr
	stderr = &buf
	defer func() { stderr = oldStderr }()

	outputError(1, "No project plan. Run `aether plan` first.", nil)

	got := buf.String()
	if strings.Contains(got, "/ant-") {
		t.Errorf("JSON error envelope must stay in raw CLI form, got: %s", got)
	}
	if !strings.Contains(got, "aether plan") {
		t.Errorf("expected raw command in JSON envelope, got: %s", got)
	}
}

// TestTranslationIsIdempotent guards the overlap created by translating at the
// exit: renderNextUp already translated its own text, so that text now passes
// through translation twice. A second pass must be a no-op.
func TestTranslationIsIdempotent(t *testing.T) {
	for _, platform := range []string{"claude", "opencode", "codex"} {
		t.Run(platform, func(t *testing.T) {
			in := "Run `aether continue` to verify, or `/ant-status` to check."
			once := translateHintCommandsForPlatform(in, platform)
			twice := translateHintCommandsForPlatform(once, platform)
			if once != twice {
				t.Errorf("translation not idempotent on %s:\n once: %s\ntwice: %s", platform, once, twice)
			}
		})
	}
}

// TestWrapperCommandNamesCoverLifecycleVerbs is a cheap smoke check that the
// allowlist has not lost the commands a user types most. The full corpus check
// against .claude/commands/ant/*.md lives in
// TestWrapperCommandNamesMatchCanonicalCorpus.
func TestWrapperCommandNamesCoverLifecycleVerbs(t *testing.T) {
	for _, verb := range []string{"init", "plan", "build", "continue", "status", "resume", "seal", "lay-eggs"} {
		if !wrapperCommandNames[verb] {
			t.Errorf("lifecycle verb %q missing from wrapperCommandNames", verb)
		}
	}
	// Commands with no wrapper must stay out, or hints will invent them.
	for _, verb := range []string{"publish", "install", "host", "integrity", "version", "build-finalize"} {
		if wrapperCommandNames[verb] {
			t.Errorf("%q has no slash wrapper but is in wrapperCommandNames", verb)
		}
	}
}
