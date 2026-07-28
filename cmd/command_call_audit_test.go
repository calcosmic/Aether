package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// LOUD-03 / LOUD-04 / LOUD-05.
//
// The blind spot this closes: cmd/cli_flag_audit_test.go matches only `--flag`
// tokens with a regex and never consults cobra's Args validator, so a call that
// passes a bare positional argument to a `cobra.NoArgs` command is structurally
// invisible to it. All seven of the phase's confirmed-broken calls were of
// exactly that shape. A regex that only knows about flags cannot see them.
//
// This audit resolves each documented invocation against the real cobra command
// tree and validates it with cobra's OWN machinery — ValidateArgs and the real
// flag set — rather than matching text. It deliberately stops short of invoking
// RunE: running ~1,200 documented invocations for real would mutate colony
// state, delete data, and publish to the hub. Validating the argument contract
// through cobra is the strongest check available that is also safe to run in CI
// on every commit. What it cannot catch is a runtime failure inside RunE with a
// well-formed command line; that limitation is stated here rather than implied.

// auditedCorpora are the file trees whose documented `aether ...` invocations
// are contract-checked.
//
// Scope decision (RESEARCH.md, LOUD-06): `.aether/docs/command-playbooks/` and
// `colony/playbooks/` are included even though the runtime does not currently
// load them. They are the corpus the seven broken calls lived in, and a future
// phase that reconnects them must not inherit stale calls. `.claude/` and
// `.opencode/` are the live surfaces where a broken call reaches a user today.
var auditedCorpora = []string{
	filepath.Join(".claude", "commands", "ant"),
	filepath.Join(".opencode", "commands", "ant"),
	filepath.Join(".aether", "commands"),
	filepath.Join(".aether", "docs", "command-playbooks"),
	filepath.Join("colony", "playbooks"),
}

type documentedCall struct {
	File    string
	Line    int
	Raw     string
	Command string
	Args    []string
	// InFence marks a call inside a fenced code block. Prose frequently names
	// a command without invoking it ("recommend follow-up commands such as
	// `aether focus`, `aether redirect`"). Those are references, not calls;
	// auditing them as zero-argument invocations produces a violation for
	// every command that requires an argument. A bare `aether status` inside
	// a fenced block, by contrast, really is an invocation.
	InFence bool
}

var (
	// Invocations are extracted from inside backticks or fenced code only —
	// prose mentioning a command name is not a call.
	backtickCallRe = regexp.MustCompile("`([^`\n]*\\baether [a-z][a-z0-9-]*[^`\n]*)`")
	// Lines that explicitly tell the reader NOT to run the command. Several
	// wrappers are pure prompt commands ("Do NOT attempt to run `aether dream`
	// — you ARE the Dreamer"); treating those as invocations would produce
	// confident, wrong failures.
	negatedCallRe = regexp.MustCompile(`(?i)do not (attempt to )?run|never run|pure prompt command|do NOT call`)
	envPrefixRe   = regexp.MustCompile(`^[A-Z][A-Z0-9_]*=\S*$`)
)

// extractDocumentedCalls parses `aether <cmd> [args…]` invocations out of a file.
func extractDocumentedCalls(t *testing.T, path string) []documentedCall {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var calls []documentedCall
	inFence := false
	for i, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if negatedCallRe.MatchString(line) {
			continue
		}
		// Inside a fence the whole line is the command; outside, only
		// backticked spans are.
		if inFence {
			if c, ok := parseFencedInvocation(path, i+1, line); ok {
				calls = append(calls, c)
			}
			continue
		}
		for _, m := range backtickCallRe.FindAllStringSubmatch(line, -1) {
			snippet := m[1]
			fields := tokenizeShellLike(snippet)

			// Find the `aether` token, skipping env-var prefixes and any
			// shell plumbing before it.
			idx := -1
			for j, f := range fields {
				if f == "aether" || strings.HasSuffix(f, "/aether") {
					idx = j
					break
				}
			}
			if idx == -1 || idx+1 >= len(fields) {
				continue
			}
			// Reject `FOO=bar aether ...` only when the token before `aether`
			// is something other than an env assignment or a pipe/&&.
			if idx > 0 {
				prev := fields[idx-1]
				if !envPrefixRe.MatchString(prev) && prev != "|" && prev != "&&" && prev != ";" && prev != "$(" {
					continue
				}
			}

			name := fields[idx+1]
			if !regexp.MustCompile(`^[a-z][a-z0-9-]*$`).MatchString(name) {
				continue
			}
			calls = append(calls, documentedCall{
				File:    path,
				Line:    i + 1,
				Raw:     strings.TrimSpace(snippet),
				Command: name,
				Args:    fields[idx+2:],
				InFence: false,
			})
		}
	}
	return calls
}

// parseFencedInvocation parses one line inside a fenced code block.
func parseFencedInvocation(path string, line int, text string) (documentedCall, bool) {
	fields := tokenizeShellLike(text)
	idx := -1
	for j, f := range fields {
		if f == "aether" || strings.HasSuffix(f, "/aether") {
			idx = j
			break
		}
	}
	if idx == -1 || idx+1 >= len(fields) {
		return documentedCall{}, false
	}
	if idx > 0 {
		prev := fields[idx-1]
		if !envPrefixRe.MatchString(prev) && prev != "|" && prev != "&&" && prev != ";" {
			return documentedCall{}, false
		}
	}
	name := fields[idx+1]
	if !regexp.MustCompile(`^[a-z][a-z0-9-]*$`).MatchString(name) {
		return documentedCall{}, false
	}
	return documentedCall{
		File:    path,
		Line:    line,
		Raw:     strings.TrimSpace(text),
		Command: name,
		Args:    fields[idx+2:],
		InFence: true,
	}, true
}

func collectDocumentedCalls(t *testing.T, root string) []documentedCall {
	t.Helper()
	var all []documentedCall
	for _, corpus := range auditedCorpora {
		dir := filepath.Join(root, corpus)
		if _, err := os.Stat(dir); err != nil {
			continue // corpus removed by a later phase; not this test's business
		}
		err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			switch strings.ToLower(filepath.Ext(p)) {
			case ".md", ".yaml", ".yml":
				all = append(all, extractDocumentedCalls(t, p)...)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
	return all
}

// shellOperators end an invocation: everything after them belongs to another
// command, not to this one. Note that `<phase>` and `"$ARGUMENTS"` are NOT
// operators — they are argument placeholders, and treating them as redirects
// (an easy mistake, since both contain shell metacharacters) makes the audit
// report a missing positional on every correctly-written call.
func isShellOperator(tok string) bool {
	switch tok {
	case "|", "||", "&&", ";", ">", ">>", "<", "&", "2>&1",
		// A trailing backslash is a line continuation: the rest of the
		// invocation is on the next line and is not a positional argument.
		`\`,
		// Shell keywords that can share a line with a command inside a
		// fenced block (`aether suggest-approve    fi`).
		"fi", "done", "then", "else", "elif", "do", "esac", "}":
		return true
	}
	return strings.HasPrefix(tok, "2>") || strings.HasPrefix(tok, "1>")
}

// isPlaceholder reports whether a token is a documentation placeholder that
// stands in for a real positional value: <phase>, {name}, $ARGUMENTS, "$X".
// These must count as positionals — substituting a real value is exactly what
// the runtime does, so the arity contract has to hold for them.
func isPlaceholder(tok string) bool {
	t := strings.Trim(tok, `"'`)
	if t == "" {
		return false
	}
	if strings.HasPrefix(t, "$") {
		return true
	}
	if strings.HasPrefix(t, "<") && strings.HasSuffix(t, ">") {
		return true
	}
	if strings.HasPrefix(t, "{") && strings.HasSuffix(t, "}") {
		return true
	}
	return false
}

// tokenizeShellLike splits on whitespace while keeping quoted runs together, so
// `--content '{"text":"x y"}'` is one token rather than three.
func tokenizeShellLike(s string) []string {
	var out []string
	var cur strings.Builder
	var quote rune
	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
			cur.WriteRune(r)
		case r == '\'' || r == '"':
			quote = r
			cur.WriteRune(r)
		// Documentation placeholders may contain spaces: `<temp completion>`,
		// `<selected colony|orchestrator>`. Splitting those on whitespace turns
		// one argument into two and reports a bogus arity violation.
		case r == '<':
			quote = '>'
			cur.WriteRune(r)
		case r == '{':
			quote = '}'
			cur.WriteRune(r)
		case r == ' ' || r == '\t':
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return out
}

// validateCallAgainstCobra returns a violation string, or "" when the call
// satisfies the command's real argument contract.
func validateCallAgainstCobra(call documentedCall) string {
	// Trim at the first shell operator: `aether x | jq .` is a call to `x`.
	args := append([]string{call.Command}, call.Args...)
	for i, a := range args {
		if isShellOperator(a) {
			args = args[:i]
			break
		}
	}
	if len(args) == 0 {
		return ""
	}

	// cobra's own Find resolves the DEEPEST matching subcommand path and hands
	// back the remaining args — so `host build --dry-run` correctly resolves to
	// the `build` subcommand of `host`, not to `host` with a stray positional.
	target, remaining, err := rootCmd.Find(args)
	if err != nil || target == nil || target == rootCmd {
		// Unresolvable command names are reported by the dedicated
		// registration test, not here — keep one failure per cause.
		return ""
	}

	// `aether host *` sets DisableFlagParsing: cobra deliberately forwards the
	// raw command line to the TypeScript host, which is the authoritative
	// parser. Validating those calls against cobra's (empty) flag set reports
	// every real flag as unknown. Cobra is not the contract owner here, so it
	// is not the thing to check against.
	for c := target; c != nil; c = c.Parent() {
		if c.DisableFlagParsing {
			return ""
		}
	}

	var positionals []string
	variadicPlaceholder := false
	for i := 0; i < len(remaining); i++ {
		a := remaining[i]
		switch {
		case a == "--":
			continue
		case strings.HasPrefix(a, "--"):
			name := strings.TrimPrefix(a, "--")
			if eq := strings.Index(name, "="); eq >= 0 {
				name = name[:eq]
			}
			f := target.Flags().Lookup(name)
			if f == nil {
				f = target.InheritedFlags().Lookup(name)
			}
			if f == nil {
				f = rootCmd.PersistentFlags().Lookup(name)
			}
			if f == nil {
				return fmt.Sprintf("unknown flag --%s", name)
			}
			// Consume this flag's value when it takes one and was not given
			// in --flag=value form.
			if !strings.Contains(a, "=") && f.Value.Type() != "bool" &&
				i+1 < len(remaining) && !strings.HasPrefix(remaining[i+1], "-") {
				i++
			}
		case strings.HasPrefix(a, "-") && len(a) > 1:
			continue // shorthand; the existing flag audit covers these
		default:
			// `$ARGUMENTS` is the user's raw argument string: it expands to
			// ZERO OR MORE arguments, and is very often empty. `<phase>` and
			// `{name}` expand to exactly one. Conflating the two reports a
			// violation on every correct `aether <cmd> $ARGUMENTS` wrapper.
			if strings.HasPrefix(strings.Trim(a, `"'`), "$") {
				variadicPlaceholder = true
				continue
			}
			positionals = append(positionals, a)
		}
	}

	if target.Args == nil {
		return ""
	}
	if err := target.Args(target, positionals); err != nil {
		if variadicPlaceholder {
			// A variadic placeholder can expand to whatever arity the command
			// wants; only a violation that holds for EVERY expansion is real.
			if withOne := target.Args(target, append(append([]string{}, positionals...), "x")); withOne == nil {
				return ""
			}
		}
		return fmt.Sprintf("positional args %v rejected by the command's own Args validator: %v", positionals, err)
	}
	return ""
}

// LOUD-03 / LOUD-05: every documented invocation satisfies the real contract.
func TestCommandCallsMatchCobraContracts(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	calls := collectDocumentedCalls(t, root)
	if len(calls) == 0 {
		t.Fatal("extracted zero documented calls — the audit is not looking at anything, which would pass vacuously forever")
	}

	var violations []string
	for _, c := range calls {
		// A bare `aether focus` in prose is naming the command, not calling it.
		if len(c.Args) == 0 && !c.InFence {
			continue
		}
		if v := validateCallAgainstCobra(c); v != "" {
			rel, _ := filepath.Rel(root, c.File)
			violations = append(violations, fmt.Sprintf("%s:%d: `%s` — %s", rel, c.Line, c.Raw, v))
		}
	}

	if len(violations) > 0 {
		t.Errorf("%d documented CLI call(s) violate the command's real argument contract:\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
	t.Logf("audited %d documented invocations across %d corpora", len(calls), len(auditedCorpora))
}

// LOUD-04: the audit must actually detect positional-argument drift. Without
// this, an audit that silently extracted nothing — or that ignored positionals
// the way cli_flag_audit_test.go does — would pass forever and prove nothing.
func TestAuditDetectsPositionalDrift(t *testing.T) {
	// A command that takes no positional arguments, called with one. This is
	// the exact shape of all seven of the phase's confirmed-broken calls.
	noArgsCmd := &cobra.Command{Use: "audit-selftest-noargs", Args: cobra.NoArgs, Run: func(*cobra.Command, []string) {}}
	rootCmd.AddCommand(noArgsCmd)
	defer rootCmd.RemoveCommand(noArgsCmd)

	bad := documentedCall{
		File: "selftest", Line: 1,
		Raw:     "aether audit-selftest-noargs some-positional",
		Command: "audit-selftest-noargs",
		Args:    []string{"some-positional"},
	}
	if v := validateCallAgainstCobra(bad); v == "" {
		t.Error("the audit did not flag a positional argument passed to a cobra.NoArgs command — this is the exact blind spot in cli_flag_audit_test.go that let seven broken calls survive (LOUD-04)")
	}

	good := documentedCall{
		File: "selftest", Line: 1,
		Raw:     "aether audit-selftest-noargs",
		Command: "audit-selftest-noargs",
		Args:    nil,
	}
	if v := validateCallAgainstCobra(good); v != "" {
		t.Errorf("the audit flagged a correct call: %s", v)
	}

	// And an unknown flag must still be caught, so closing the positional gap
	// did not open a flag-shaped one.
	badFlag := documentedCall{
		File: "selftest", Line: 1,
		Raw:     "aether audit-selftest-noargs --no-such-flag",
		Command: "audit-selftest-noargs",
		Args:    []string{"--no-such-flag"},
	}
	if v := validateCallAgainstCobra(badFlag); v == "" {
		t.Error("the audit did not flag an unknown flag")
	}
}

// The extractor is the part most likely to rot into vacuous success: if its
// regex stops matching, every other assertion here passes trivially.
func TestCommandCallExtractorSeesRealInvocationsAndSkipsProse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.md")
	content := strings.Join([]string{
		"Run `aether status --json` to inspect.",
		"> **Important:** This is a pure prompt command. Do NOT attempt to run `aether dream`.",
		"Use `AETHER_OUTPUT_MODE=visual aether pheromones --type FOCUS` for signals.",
		"The aether status command is nice in prose but not backticked.",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	calls := extractDocumentedCalls(t, path)

	got := map[string]bool{}
	for _, c := range calls {
		got[c.Command] = true
	}
	if !got["status"] {
		t.Error("extractor missed a plain backticked invocation")
	}
	if !got["pheromones"] {
		t.Error("extractor missed an invocation behind an env-var prefix")
	}
	if got["dream"] {
		t.Error("extractor treated a 'Do NOT attempt to run' line as an invocation — pure prompt commands are not CLI calls")
	}
	if len(calls) != 2 {
		t.Errorf("extracted %d calls, want 2 (%+v)", len(calls), calls)
	}
}

// D-01: the gate/enrichment classification must name real commands, and the
// safety gates the phase cares about must be classified as gates rather than
// quietly defaulting to enrichment.
func TestGateClassifiedCallsHaveGateWiring(t *testing.T) {
	if len(gateClassifiedCommands) == 0 {
		t.Fatal("no commands are classified as safety gates — D-01 requires the gate/enrichment split to be encoded where a test can assert it")
	}

	for name, rationale := range gateClassifiedCommands {
		target, _, err := rootCmd.Find([]string{name})
		if err != nil || target == nil || target == rootCmd {
			t.Errorf("%q is classified as a safety gate but is not a registered command — a gate that does not exist cannot halt anything", name)
			continue
		}
		if strings.TrimSpace(rationale) == "" {
			t.Errorf("%q is classified as a gate with no recorded rationale; a future reader cannot re-make the judgement", name)
		}
		if commandCallSeverityFor(name) != severityGate {
			t.Errorf("%q is in gateClassifiedCommands but commandCallSeverityFor reports %q", name, commandCallSeverityFor(name))
		}
	}

	// check-antipattern is the specific gate this phase wired into the live
	// continue pipeline (LOUD-02). If it ever silently becomes enrichment, a
	// build could pass with its security scan unexecuted — the original bug.
	if commandCallSeverityFor("check-antipattern") != severityGate {
		t.Error("check-antipattern must be gate-classified: a phase must not pass with its security scan unexecuted (D-01, ROADMAP SC#2)")
	}

	// A command with no special safety role defaults to enrichment.
	if commandCallSeverityFor("status") != severityEnrichment {
		t.Error("status should default to enrichment; halting a run because a status render failed would be its own bug")
	}
}
