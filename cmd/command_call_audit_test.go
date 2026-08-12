package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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

// auditedFiles are individual top-level markdown files, audited WITHOUT
// recursing into their directory. WIRE-03 (172-CONTEXT.md <corpus_scope>)
// resolves the ".aether markdown corpus" as the literal top-level
// `.aether/*.md` files, not `.aether/**/*.md` (~250 files under docs/,
// skills/, templates/, references/) — that broader recursive scope was
// explicitly rejected. `auditedCorpora` entries are directories walked
// recursively by collectDocumentedCalls via filepath.Walk, which cannot
// express "this one directory, non-recursively" — hence a second, disjoint
// list of exact file paths instead of adding ".aether" itself as a corpus
// entry.
//
// `.aether/HANDOFF.md` is deliberately excluded: it is gitignored
// (.gitignore:87), so it exists locally but not in a fresh CI checkout —
// auditing it would make the test's pass/fail behavior depend on whether a
// session-local file happens to be present, which must never differ between
// CI and a local run.
var auditedFiles = []string{
	filepath.Join(".aether", "CONTEXT.md"),
	filepath.Join(".aether", "CROWNED-ANTHILL.md"),
	filepath.Join(".aether", "QUEEN.md"),
	filepath.Join(".aether", "workers.md"),
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
	// assignmentPrefixRe matches a leading `VAR=` shell-assignment prefix,
	// e.g. the `result=` in `result=$(aether spawn-can-spawn 5 --enforce)`.
	assignmentPrefixRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
)

// normalizeShellToken strips a leading assignment-and-substitution prefix
// (`result=`, then any leading run of `$(`, a backtick, or a bare `(`) from
// tok and returns the remainder. tokenizeShellLike does not split `=`, `$`,
// `(`, or a backtick from the token that follows them, so
// `result=$(aether` — the exact shape at `.aether/workers.md:292` — is a
// single token that neither `f == "aether"` nor `strings.HasSuffix(f,
// "/aether")` can ever match. Both places that locate the `aether` binary
// token call this ONE helper so they cannot drift apart again the way the
// dead `prev != "$("` comparison below (a token tokenizeShellLike can never
// produce on its own) already did once.
func normalizeShellToken(tok string) string {
	t := tok
	if loc := assignmentPrefixRe.FindStringIndex(t); loc != nil {
		t = t[loc[1]:]
	}
	for {
		switch {
		case strings.HasPrefix(t, "$("):
			t = t[2:]
		case strings.HasPrefix(t, "`"):
			t = t[1:]
		case strings.HasPrefix(t, "("):
			t = t[1:]
		default:
			return t
		}
	}
}

// substitutionOpener reports the command-substitution opener tok itself
// begins with (after an optional `VAR=` assignment prefix) and the
// delimiter that closes it: `$(` closed by `)`, a backtick closed by a
// backtick, a bare `(` closed by `)`. It returns two empty strings when tok
// opens no substitution.
//
// This is the other half of the decision normalizeShellToken makes when
// stripping openers, and the two functions must recognise exactly the same
// three cases in the same order: normalizeShellToken strips three openers
// while the trim-the-closing-delimiter decision below used to recognise
// only one (`$(`), so a backtick substitution's command name carried a
// stray trailing backtick and a bare-subshell call's final token carried a
// stray trailing `)` — both silently dropped or misreported (CR-05).
// Adding an opener to one function without the other reproduces that gap;
// keep them adjacent and keep them in the same order.
func substitutionOpener(tok string) (open, closeDelim string) {
	t := tok
	if loc := assignmentPrefixRe.FindStringIndex(t); loc != nil {
		t = t[loc[1]:]
	}
	switch {
	case strings.HasPrefix(t, "$("):
		return "$(", ")"
	case strings.HasPrefix(t, "`"):
		return "`", "`"
	case strings.HasPrefix(t, "("):
		return "(", ")"
	default:
		return "", ""
	}
}

// substitutionDepth reports how many more `(` than `)` characters have been
// seen scanning toks left to right — i.e. whether a command substitution
// opened by an EARLIER token is still open. A candidate `aether` token
// reached while this is positive sits inside a substitution opened by a
// DIFFERENT word — `echo $(foo | aether cmd)` — which is a nested pipeline,
// not a genuine invocation, even though the token immediately before
// `aether` (`|`) looks like an ordinary, accepted pipe.
func substitutionDepth(toks []string) int {
	depth := 0
	for _, tok := range toks {
		depth += strings.Count(tok, "(") - strings.Count(tok, ")")
	}
	return depth
}

// extractDocumentedCalls parses `aether <cmd> [args…]` invocations out of a file.
//
// Fence-marker tolerance (T-172-26 / T-172-27): a closing (or opening) triple-
// backtick marker glued to the end of a content line — `  --ttl "30d"` ```` — is
// not on its own line, so a naive `strings.HasPrefix(strings.TrimSpace(line),
// "```")` check never sees it, desyncing in-fence/out-of-fence parity for the
// rest of the file. Three cases, in order of precedence:
//
//  1. No marker on the line at all: process the line under the current
//     inFence state, unchanged from before this tolerance existed.
//  2. The marker is the first non-whitespace content on the line (the
//     established, unambiguous case): toggle inFence and skip the line
//     entirely. Unchanged.
//  3. The marker appears after other content (the glued case): process the
//     substring BEFORE the marker as an ordinary line under the CURRENT
//     inFence state — running processDocumentedCallLine exactly as case 1
//     does — and only then toggle. The remainder of the line after the
//     marker is discarded: an opening fence's info string (`bash`, `yaml`)
//     never carries an invocation. A line carrying two markers toggles
//     exactly once, at the first; TestAuditedCorpusHasNoGluedFenceMarkers
//     forbids the glued shape outright, so that shape cannot accumulate in
//     the corpus unnoticed. This is a bound, not a general CommonMark parser.
func extractDocumentedCalls(t *testing.T, path string) []documentedCall {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var calls []documentedCall
	inFence := false
	for i, line := range strings.Split(string(raw), "\n") {
		lineNum := i + 1
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			// Case 2: marker is the line's only non-whitespace content.
			inFence = !inFence
			continue
		}
		if idx := strings.Index(line, "```"); idx != -1 {
			// Case 3: marker glued to trailing content. Process what comes
			// before it under the CURRENT fence state, then toggle.
			calls = append(calls, processDocumentedCallLine(path, lineNum, line[:idx], inFence)...)
			inFence = !inFence
			continue
		}
		// Case 1: no marker on this line.
		calls = append(calls, processDocumentedCallLine(path, lineNum, line, inFence)...)
	}
	return calls
}

// processDocumentedCallLine is the single per-line extraction body shared by
// both the ordinary and glued-marker paths in extractDocumentedCalls, so the
// backtick-span loop exists in exactly one place rather than two copies that
// could drift apart.
func processDocumentedCallLine(path string, lineNum int, line string, inFence bool) []documentedCall {
	if negatedCallRe.MatchString(line) {
		return nil
	}
	// Inside a fence the whole line is the command; outside, only
	// backticked spans are.
	if inFence {
		if c, ok := parseFencedInvocation(path, lineNum, line); ok {
			return []documentedCall{c}
		}
		return nil
	}
	var calls []documentedCall
	for _, m := range backtickCallRe.FindAllStringSubmatch(line, -1) {
		snippet := m[1]
		fields := tokenizeShellLike(snippet)

		// Find the `aether` token, skipping env-var prefixes and any
		// shell plumbing before it. normalizeShellToken lets this see
		// through command-substitution syntax glued to the same token,
		// e.g. `x=$(aether cmd)` tokenizes to one `x=$(aether` field.
		idx := -1
		for j, f := range fields {
			nf := normalizeShellToken(f)
			if nf == "aether" || strings.HasSuffix(nf, "/aether") {
				idx = j
				break
			}
		}
		if idx == -1 || idx+1 >= len(fields) {
			continue
		}
		// Reject `FOO=bar aether ...` only when the token before `aether`
		// is something other than an env assignment or a pipe/&&. A
		// substitution still open from an EARLIER token — `echo $(foo |
		// aether cmd)` — means `aether` is nested inside someone else's
		// pipeline, not a genuine invocation; substitutionDepth catches
		// that even though the immediately preceding token (`|`) looks
		// like an ordinary, accepted pipe. The dead `prev != "$("`
		// comparison (a token tokenizeShellLike can never produce on its
		// own) is replaced by the shared helpers rather than left beside
		// them.
		if idx > 0 {
			prev := fields[idx-1]
			nested := substitutionDepth(fields[:idx]) > 0
			if nested || (!envPrefixRe.MatchString(prev) && prev != "|" && prev != "&&" && prev != ";") {
				continue
			}
		}

		name := fields[idx+1]
		args := append([]string{}, fields[idx+2:]...)
		// Trim the command substitution's own closing delimiter off the
		// last token — `--enforce)` must be reported as `--enforce`, not
		// as a stray-character typo — BEFORE the name-shape check below,
		// so a zero-argument call like `$(aether status)` doesn't get its
		// command name rejected as `"status)"`.
		if _, closeDelim := substitutionOpener(fields[idx]); closeDelim != "" {
			if len(args) > 0 {
				args[len(args)-1] = strings.TrimSuffix(args[len(args)-1], closeDelim)
			} else {
				name = strings.TrimSuffix(name, closeDelim)
			}
		}
		if !regexp.MustCompile(`^[a-z][a-z0-9-]*$`).MatchString(name) {
			continue
		}
		calls = append(calls, documentedCall{
			File:    path,
			Line:    lineNum,
			Raw:     strings.TrimSpace(snippet),
			Command: name,
			Args:    args,
			InFence: false,
		})
	}
	return calls
}

// parseFencedInvocation parses one line inside a fenced code block.
func parseFencedInvocation(path string, line int, text string) (documentedCall, bool) {
	fields := tokenizeShellLike(text)
	// normalizeShellToken lets this see through command-substitution syntax
	// glued to the same token — `result=$(aether spawn-can-spawn {your_depth}
	// --enforce)`, the exact shape at `.aether/workers.md:292`, tokenizes to
	// one `result=$(aether` field that neither `f == "aether"` nor
	// `strings.HasSuffix(f, "/aether")` can ever match on their own.
	idx := -1
	for j, f := range fields {
		nf := normalizeShellToken(f)
		if nf == "aether" || strings.HasSuffix(nf, "/aether") {
			idx = j
			break
		}
	}
	if idx == -1 || idx+1 >= len(fields) {
		return documentedCall{}, false
	}
	if idx > 0 {
		prev := fields[idx-1]
		// A substitution still open from an EARLIER token means `aether` is
		// nested inside someone else's pipeline, not a genuine invocation —
		// see substitutionDepth's doc comment.
		nested := substitutionDepth(fields[:idx]) > 0
		if nested || (!envPrefixRe.MatchString(prev) && prev != "|" && prev != "&&" && prev != ";") {
			return documentedCall{}, false
		}
	}
	name := fields[idx+1]
	args := append([]string{}, fields[idx+2:]...)
	// Trim the command substitution's own closing delimiter off the last
	// token — `--enforce)` must be reported as `--enforce`, not as a
	// stray-character typo — BEFORE the name-shape check below, so a
	// zero-argument call like `$(aether status)` doesn't get its command
	// name rejected as `"status)"`.
	if _, closeDelim := substitutionOpener(fields[idx]); closeDelim != "" {
		if len(args) > 0 {
			args[len(args)-1] = strings.TrimSuffix(args[len(args)-1], closeDelim)
		} else {
			name = strings.TrimSuffix(name, closeDelim)
		}
	}
	if !regexp.MustCompile(`^[a-z][a-z0-9-]*$`).MatchString(name) {
		return documentedCall{}, false
	}
	return documentedCall{
		File:    path,
		Line:    line,
		Raw:     strings.TrimSpace(text),
		Command: name,
		Args:    args,
		InFence: true,
	}, true
}

// auditedFilePaths returns every file the audit reads: the recursive walk
// over auditedCorpora filtered to .md/.yaml/.yml, plus each existing path in
// auditedFiles. Both collectDocumentedCalls and the corpus-wide glued-marker
// sweep (TestAuditedCorpusHasNoGluedFenceMarkers) call this ONE enumeration,
// so the sweep can never drift from the set the audit actually reads — a
// hardcoded second file list would rot the moment a corpus changed shape.
func auditedFilePaths(t *testing.T, root string) []string {
	t.Helper()
	var paths []string
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
				paths = append(paths, p)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
	// auditedFiles is a second, non-recursive corpus: exact top-level
	// `.aether/*.md` files rather than a directory walk. See auditedFiles'
	// doc comment for why this can't just be another auditedCorpora entry.
	for _, f := range auditedFiles {
		p := filepath.Join(root, f)
		if _, err := os.Stat(p); err != nil {
			continue // listed file removed; matches the corpus-removal tolerance above
		}
		paths = append(paths, p)
	}
	return paths
}

func collectDocumentedCalls(t *testing.T, root string) []documentedCall {
	t.Helper()
	var all []documentedCall
	for _, p := range auditedFilePaths(t, root) {
		all = append(all, extractDocumentedCalls(t, p)...)
	}
	return all
}

// redirectionRe matches an attached shell redirection token: an optional
// leading file-descriptor number followed by `<` or `>` — `2>/dev/null`,
// `>out.txt`, `>>log`, `<in`. tokenizeShellLike emits these as one token
// since none of `<digit>`, `<`, or `>` is a split point, so once
// command-substitution invocations become visible (normalizeShellToken),
// a call like `$(aether skill-index 2>/dev/null)` would otherwise hand
// `2>/dev/null` to validateCallAgainstCobra as a positional argument —
// producing a confident, wrong failure on a correct call.
var redirectionRe = regexp.MustCompile(`^[0-9]*[<>]`)

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
	// A `<phase>`-shaped placeholder starts with the same `<` character as a
	// genuine redirection but closes its own bracket; isPlaceholder is what
	// tells them apart, so redirectionRe never fires on the ones that must
	// still count as positionals.
	if isPlaceholder(tok) {
		return false
	}
	return redirectionRe.MatchString(tok)
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

	// T-172-10 (anti-vacuity): a corpus that is listed but never actually read
	// produces a test that passes while the bug it exists to catch sits
	// inside its declared scope — precisely the state `.aether/workers.md`
	// was in before this corpus was wired up. The overall len(calls) != 0
	// check above cannot catch this: the other four corpora keep the total
	// non-zero even if auditedFiles silently contributed nothing. This
	// asserts the scan actually opened `.aether/workers.md`, not just that
	// something, somewhere, was extracted.
	foundWorkersMd := false
	for _, c := range calls {
		if strings.HasSuffix(filepath.ToSlash(c.File), ".aether/workers.md") {
			foundWorkersMd = true
			break
		}
	}
	if !foundWorkersMd {
		t.Fatal(".aether/workers.md contributed zero extracted calls — the corpus is listed but not being read, which is the exact vacuous-pass failure mode this test exists to catch")
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
	// +1: auditedFiles is a second, non-recursive corpus (the top-level
	// `.aether/*.md` file list) alongside the five directory trees in
	// auditedCorpora.
	t.Logf("audited %d documented invocations across %d corpora", len(calls), len(auditedCorpora)+1)
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

// TestAetherCorpusCatchesAnUnregisteredFlag is WIRE-03's permanent proof.
// Success criterion 3 required the audit to be "seeded to fail today against
// --enforce" — but once 172-01 registered --enforce on the real
// spawn-can-spawn, that seed is gone from the live tree. This test replaces
// the seed with something that runs forever: a function-local fixture
// mirroring the PRE-172-01 spawn-can-spawn contract (no --enforce, no
// positional depth), fed the REAL `.aether/workers.md:292` text (name
// swapped to the fixture's), proving the corpus, the extractor and the
// validator together still catch exactly the bug this phase was created for
// — on every CI run, without a red commit ever landing on this branch.
//
// The fixture is registered and removed inside this function body only
// (never at package scope): a package-scope registration would become a
// real, permanent orphan requiring an entry in 172-02's shrink-only
// allowlist, and "test fixture" is not debt.
func TestAetherCorpusCatchesAnUnregisteredFlag(t *testing.T) {
	preFixSpawnCanSpawn := &cobra.Command{
		Use:  "audit-selftest-preenforce-spawn-can-spawn",
		Args: cobra.NoArgs, // the pre-172-01 contract: no positional depth
		Run:  func(*cobra.Command, []string) {},
	}
	preFixSpawnCanSpawn.Flags().Int("depth", 0, "Spawn depth to check (required)")
	// Deliberately no --enforce flag: this is the exact absence 172-01 fixed.
	rootCmd.AddCommand(preFixSpawnCanSpawn)
	defer rootCmd.RemoveCommand(preFixSpawnCanSpawn)

	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	// Run the real extractor over the real file — not a hand-built call —
	// so this proves the live corpus and extractor, not just the validator.
	workersPath := filepath.Join(root, ".aether", "workers.md")
	calls := extractDocumentedCalls(t, workersPath)

	var real *documentedCall
	for i := range calls {
		if calls[i].Command == "spawn-can-spawn" {
			real = &calls[i]
			break
		}
	}
	if real == nil {
		t.Fatal("the extractor found no `spawn-can-spawn` invocation in .aether/workers.md — a fixture test that silently found nothing to validate is the vacuous pass this whole phase exists to make impossible")
	}

	// Swap the command name to the fixture's so resolution hits the pre-fix
	// contract instead of the real (now-fixed) spawn-can-spawn, then
	// re-parse through the real extractor rather than hand-constructing the
	// documentedCall struct — a hand-built struct would prove only that the
	// validator works, which was never in doubt.
	fixtureRaw := strings.Replace(real.Raw, "spawn-can-spawn", preFixSpawnCanSpawn.Use, 1)
	fixtureCall, ok := parseFencedInvocation(real.File, real.Line, fixtureRaw)
	if !ok {
		t.Fatalf("could not re-parse the name-swapped .aether/workers.md:292 text (%q) through the real extractor", fixtureRaw)
	}

	v := validateCallAgainstCobra(fixtureCall)
	if v == "" {
		t.Fatal("the corpus + extractor + validator chain did not flag the pre-172-01 fixture at all — the .aether/workers.md:292 shape must be caught as it was before 172-01 fixed the real command")
	}
	if !strings.Contains(v, "--enforce") {
		t.Errorf("violation = %q, want it to name --enforce", v)
	}
}

// The extractor is the part most likely to rot into vacuous success: if its
// regex stops matching, every other assertion here passes trivially.
//
// 172-00 / Task 2: pins the three command-substitution behaviours added in
// Task 1 — normalizeShellToken seeing through `x=$(aether ...)` in BOTH the
// fenced and backtick branches, the trailing `)` trim, and redirection
// tokens no longer posing as positional arguments — plus the two negative
// cases proving change 1 did not widen detection past what it was meant to
// fix.
func TestCommandCallExtractorSeesRealInvocationsAndSkipsProse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.md")
	content := strings.Join([]string{
		"Run `aether status --json` to inspect.",
		"> **Important:** This is a pure prompt command. Do NOT attempt to run `aether dream`.",
		"Use `AETHER_OUTPUT_MODE=visual aether pheromones --type FOCUS` for signals.",
		"The aether status command is nice in prose but not backticked.",
		"Backtick command substitution: `x=$(aether skill-detect)` runs the detector.",
		"The manual's own shape: `result=$(aether spawn-can-spawn 5 --enforce)` checks the depth.",
		"Redirected output: `$(aether skill-index 2>/dev/null)` is still a valid call.",
		"Nested pipeline, must NOT be extracted: `echo $(foo | aether cmd)`.",
		"Bare subshell form: `(aether colony-name)` returns just the name.",
		"Grouped invocation: `(aether midden-recent-failures --limit 5)` trims recent failures.",
		"```bash",
		"NAME=$(aether cmd --flag)",
		"RESULT=`aether skill-list`",
		"```",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	calls := extractDocumentedCalls(t, path)

	got := map[string]bool{}
	byCommand := map[string]documentedCall{}
	nestedPipelineCount := 0
	for _, c := range calls {
		got[c.Command] = true
		byCommand[c.Command] = c
		if strings.Contains(c.Raw, "foo | aether cmd") {
			nestedPipelineCount++
		}
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

	// Backticked `x=$(aether cmd)`: the backtick branch's binary-detection
	// call site must see through the substitution too, not just the fenced
	// branch.
	if sd, ok := byCommand["skill-detect"]; !ok {
		t.Error("extractor missed a backticked command-substitution invocation (`x=$(aether skill-detect)`)")
	} else if len(sd.Args) != 0 {
		t.Errorf("skill-detect call has unexpected args %v, want none", sd.Args)
	}

	// The verbatim shape from .aether/workers.md:292: the trailing `)` must
	// be trimmed off the last argument, not carried into the flag name.
	if sc, ok := byCommand["spawn-can-spawn"]; !ok {
		t.Error("extractor missed the verbatim manual invocation `result=$(aether spawn-can-spawn 5 --enforce)`")
	} else if len(sc.Args) != 2 || sc.Args[0] != "5" || sc.Args[1] != "--enforce" {
		t.Errorf("spawn-can-spawn args = %v, want [\"5\" \"--enforce\"] (trailing `)` must not survive)", sc.Args)
	}

	// `2>/dev/null` inside a command substitution must terminate the
	// invocation as a redirection, not be validated as a positional.
	if !isShellOperator("2>/dev/null") {
		t.Error("isShellOperator(\"2>/dev/null\") = false, want true")
	}
	if si, ok := byCommand["skill-index"]; !ok {
		t.Error("extractor missed the redirected command substitution `$(aether skill-index 2>/dev/null)`")
	} else if v := validateCallAgainstCobra(si); v != "" {
		t.Errorf("skill-index call flagged a violation: %s (the redirection must not be treated as a positional argument)", v)
	}

	// A genuinely nested pipeline — `aether` appears after a `|` that is
	// itself inside a DIFFERENT, still-open substitution — must not be
	// extracted as an invocation, even though the token immediately before
	// `aether` (`|`) looks like an ordinary, accepted pipe.
	if nestedPipelineCount != 0 {
		t.Errorf("extractor treated a nested pipeline (`echo $(foo | aether cmd)`) as a real invocation: %d matching call(s)", nestedPipelineCount)
	}

	// The fenced branch must resolve the same command-substitution syntax as
	// the backtick branch.
	if fc, ok := byCommand["cmd"]; !ok {
		t.Error("extractor missed the fenced command-substitution invocation `NAME=$(aether cmd --flag)`")
	} else if len(fc.Args) != 1 || fc.Args[0] != "--flag" || !fc.InFence {
		t.Errorf("fenced cmd call = %+v, want Args [\"--flag\"] and InFence true", fc)
	}

	// CR-05, form 1: a backtick-substitution assignment. Before the
	// substitutionOpener fix, normalizeShellToken strips the leading
	// backtick to find `aether`, but the name-shape check never gets a
	// trimmed trailing backtick off the command name, so `skill-list``
	// (with a stray backtick) fails `^[a-z][a-z0-9-]*$` and the whole call
	// is silently dropped.
	if sl, ok := byCommand["skill-list"]; !ok {
		t.Error("extractor missed the backtick-substitution invocation `RESULT=`aether skill-list``")
	} else {
		if len(sl.Args) != 0 {
			t.Errorf("skill-list call has unexpected args %v, want none", sl.Args)
		}
		if strings.Contains(sl.Command, "`") {
			t.Errorf("skill-list command name %q retains a stray backtick", sl.Command)
		}
		for _, a := range sl.Args {
			if strings.Contains(a, "`") {
				t.Errorf("skill-list arg %q retains a stray backtick", a)
			}
		}
	}

	// CR-05, form 2: a bare-subshell invocation, `(aether colony-name)`.
	// Before the fix, the trim decision recognises only `$(`, so the
	// trailing `)` is never trimmed and the command name is reported as
	// `colony-name)`, which fails the name-shape check and is dropped.
	if cn, ok := byCommand["colony-name"]; !ok {
		t.Error("extractor missed the bare-subshell invocation `(aether colony-name)`")
	} else if len(cn.Args) != 0 {
		t.Errorf("colony-name call has unexpected args %v, want none", cn.Args)
	}

	// CR-05, form 3: a bare-subshell invocation with a trailing flag value,
	// `(aether midden-recent-failures --limit 5)`. Before the fix, the
	// untrimmed trailing `)` produces a false-positive flag violation
	// against a call nobody wrote wrong: the last argument is reported as
	// `5)` rather than `5`.
	if mr, ok := byCommand["midden-recent-failures"]; !ok {
		t.Error("extractor missed the bare-subshell invocation `(aether midden-recent-failures --limit 5)`")
	} else {
		if len(mr.Args) != 2 || mr.Args[0] != "--limit" || mr.Args[1] != "5" {
			t.Errorf("midden-recent-failures args = %v, want [\"--limit\" \"5\"] (trailing `)` must not survive)", mr.Args)
		}
		for _, a := range mr.Args {
			if strings.Contains(a, ")") {
				t.Errorf("midden-recent-failures arg %q retains a stray closing paren", a)
			}
		}
		if v := validateCallAgainstCobra(mr); v != "" {
			t.Errorf("midden-recent-failures call flagged a violation: %s (it really does accept --limit, cmd/midden_cmds.go:492)", v)
		}
	}

	if len(calls) != 9 {
		t.Errorf("extracted %d calls, want 9 (%+v)", len(calls), calls)
	}
}

// TestExtractorDoesNotDesyncOnGluedFenceMarker pins T-172-26 / T-172-27: a
// triple-backtick marker glued to the end of a content line must not desync
// extractDocumentedCalls' in-fence/out-of-fence parity, and the content
// before a glued marker must still be processed rather than silently
// dropped by a toggle-and-skip shortcut. The fixture reproduces the exact
// corpus shapes at continue-full.md:1194 (marker glued to non-invocation
// content) and :1759 (marker glued directly to an invocation), plus a prose
// line that merely mentions a marker mid-sentence next to a genuine
// backticked call.
func TestExtractorDoesNotDesyncOnGluedFenceMarker(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "glued.md")
	content := strings.Join([]string{
		"Step X: emit feedback.",
		"```bash",
		"  --ttl \"30d\"```",
		"",
		"```bash",
		"aether midden-recent-failures --limit 50",
		"```",
		"",
		"```bash",
		"aether backup-prune-global```",
		"",
		"Here is a call: `aether status --json` — a marker looks like this: ```.",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	calls := extractDocumentedCalls(t, path)
	byCommand := map[string]documentedCall{}
	for _, c := range calls {
		byCommand[c.Command] = c
	}

	// Case: closing marker glued to non-invocation content (`  --ttl "30d"`
	// + marker). A toggle that only recognises a marker on its own line
	// never re-syncs, so the SECOND fenced block — the real invocation —
	// becomes invisible. This is the live shape of continue-full.md:1194
	// hiding continue-full.md:1243.
	mrf, ok := byCommand["midden-recent-failures"]
	if !ok {
		t.Fatalf("extractor did not see midden-recent-failures: a closing marker glued to non-invocation content (`  --ttl \"30d\"` + marker) desynced fence parity and hid the fenced block that follows it; calls = %+v", calls)
	}
	if !mrf.InFence {
		t.Errorf("midden-recent-failures call InFence = false, want true")
	}

	// Case: closing marker glued directly to an invocation line
	// (`aether backup-prune-global` + marker, continue-full.md:1759). A
	// toggle-and-skip shortcut — recognise the glued marker, toggle, then
	// discard the whole line — drops this call outright.
	bpg, ok := byCommand["backup-prune-global"]
	if !ok {
		t.Fatalf("extractor did not see backup-prune-global: a closing marker glued directly to an invocation line must still yield that invocation, not be silently discarded; calls = %+v", calls)
	}
	if !bpg.InFence || len(bpg.Args) != 0 {
		t.Errorf("backup-prune-global call = %+v, want InFence true and zero args", bpg)
	}

	// Case: a prose line outside any fence that merely mentions a marker
	// mid-sentence, alongside a genuine backticked call. The tolerance must
	// not make ordinary prose calls disappear.
	if _, ok := byCommand["status"]; !ok {
		t.Error("extractor lost a backticked call on a prose line that merely mentions a marker mid-sentence — the glued-marker tolerance must not make ordinary prose calls disappear")
	}

	if len(calls) != 3 {
		t.Errorf("extracted %d calls, want 3 (%+v)", len(calls), calls)
	}
}

// T-160-24: an unresolvable command name must be a reported violation, not a
// silent skip. validateCallAgainstCobra deliberately returns no violation for
// names cobra cannot resolve ("one failure per cause"), deferring to a
// registration test — but that test scans only three of the five audited
// corpora. This test closes the gap across ALL of auditedCorpora: a typo'd or
// stale command name in `.aether/commands/` or `colony/playbooks/` fails here
// as its own distinct violation category.
func TestDocumentedCommandNamesResolve(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	calls := collectDocumentedCalls(t, root)
	if len(calls) == 0 {
		t.Fatal("extracted zero documented calls — a vacuous pass proves nothing")
	}

	var violations []string
	for _, c := range calls {
		target, _, findErr := rootCmd.Find([]string{c.Command})
		if findErr != nil || target == nil || target == rootCmd {
			rel, _ := filepath.Rel(root, c.File)
			violations = append(violations, fmt.Sprintf("%s:%d: `%s` — subcommand %q is not a registered aether command", rel, c.Line, c.Raw, c.Command))
		}
	}
	if len(violations) > 0 {
		t.Errorf("%d documented call(s) name a subcommand that does not exist (unresolvable-command violations):\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
}

// knownEnrichmentSubcommands is the reviewed allowlist backing
// TestDocumentedSubcommandsAreSeverityClassified (T-160-23). Every subcommand
// documented anywhere in auditedCorpora must appear either in
// gateClassifiedCommands (its failure halts a run) or here (its failure warns
// and the run continues degraded — the D-01 enrichment tier). A new
// safety-relevant command must NOT be added here reflexively: the point of
// this list is that the gate-versus-enrichment judgement is made once, on
// purpose, in review — not inherited silently from the enrichment default.
var knownEnrichmentSubcommands = map[string]bool{
	"abandon":             true,
	"activity-log":        true,
	"assumption-list":     true,
	"assumption-validate": true,
	"assumptions-analyze": true,
	// backup-prune-global and temp-clean judged here, deliberately, at the
	// point the 172-06 fence repair (Task 3) makes them visible for the
	// first time: both are housekeeping/cleanup commands whose failure
	// degrades tidiness only — no verification result, security scan, or
	// gate outcome depends on either — so both are enrichment, not a gate.
	"backup-prune-global":         true,
	"behavior-observe":            true,
	"build-completion-stage":      true,
	"build-finalize":              true,
	"build":                       true,
	"bump-version":                true,
	"ceremony":                    true,
	"changelog-append":            true,
	"changelog-collect-plan-data": true,
	"colonize-finalize":           true,
	"colony-depth":                true,
	"colony-name":                 true,
	"colony-prime":                true,
	"command-guide":               true,
	"context-update":              true,
	"continue-finalize":           true,
	"continue":                    true,
	"council-advocate":            true,
	"council-budget-check":        true,
	"council-challenger":          true,
	"council-deliberate":          true,
	"council-history":             true,
	"council-sage":                true,
	"council":                     true,
	"data-clean":                  true,
	"discuss-analyze":             true,
	"discuss":                     true,
	"entomb":                      true,
	"error-add":                   true,
	"error-flag-pattern":          true,
	"eternal-init":                true,
	"export-signals":              true,
	"feedback":                    true,
	"flag-acknowledge":            true,
	"flag-add":                    true,
	"flag-auto-resolve":           true,
	"flag-check-blockers":         true,
	"flag-create":                 true,
	"flag-list":                   true,
	"flag-resolve":                true,
	"flag":                        true,
	"flags":                       true,
	"focus":                       true,
	"gate-recovery-template":      true,
	"gate-results-read":           true,
	"gate-results-write":          true,
	"generate-ant-name":           true,
	"generate-commit-message":     true,
	"generate-progress-bar":       true,
	"grave-add":                   true,
	"grave-check":                 true,
	"history":                     true,
	"hive-promote":                true,
	"hive-store":                  true,
	"host":                        true,
	"import-signals":              true,
	"init-research":               true,
	"init":                        true,
	"insert-phase":                true,
	"install":                     true,
	"instinct-create":             true,
	"lay-eggs":                    true,
	"learning-approve-proposals":  true,
	"learning-check-promotion":    true,
	"learning-extract-fallback":   true,
	"learning-promote-auto":       true,
	"load-state":                  true,
	"maturity":                    true,
	"medic-auto-spawn-check":      true,
	"medic":                       true,
	"memory-capture":              true,
	"memory-details":              true,
	"memory-metrics":              true,
	"midden-collect":              true,
	"midden-cross-pr-analysis":    true,
	"midden-recent-failures":      true,
	"midden-write":                true,
	"migrate-state":               true,
	"oracle":                      true,
	"parallel-mode":               true,
	"patrol-check":                true,
	"pause-colony":                true,
	"pending-decision-list":       true,
	"phase":                       true,
	"pheromone-display":           true,
	"pheromone-expire":            true,
	"pheromone-merge-back":        true,
	"pheromone-read":              true,
	"pheromone-write":             true,
	"pheromones":                  true,
	"plan-finalize":               true,
	"plan-research-approve":       true,
	"plan":                        true,
	"porter":                      true,
	"preferences":                 true,
	"print-next-up":               true,
	"profile-read":                true,
	"profile-update":              true,
	"publish":                     true,
	"queen-compose":               true,
	"queen-promote-instinct":      true,
	"queen-write-learnings":       true,
	"quick":                       true,
	"recipes":                     true,
	"recover":                     true,
	"redirect":                    true,
	"reference-index":             true,
	"reference-list":              true,
	"reference-match":             true,
	"resume-colony":               true,
	"resume-dashboard":            true,
	"resume":                      true,
	"run":                         true,
	"seal-finalize":               true,
	"seal":                        true,
	"session-update":              true,
	"shelf-add":                   true,
	"shelf-dismiss":               true,
	"shelf-list":                  true,
	"shelf-promote":               true,
	"should-skip-gate":            true,
	"signal-housekeeping":         true,
	"skill-cache-rebuild":         true,
	"skill-detect":                true,
	"skill-index":                 true,
	"skill-inject":                true,
	"skill-parse-frontmatter":     true,
	"spawn-can-spawn":             true,
	"spawn-complete":              true,
	"spawn-log":                   true,
	// spawn-orphans (173-09/SPAWN-08): a patrol/health-check listing of
	// stale spawn-tree entries, plus an operator --clear path. Its failure
	// degrades visibility into ghost helpers only -- no verification
	// result, security scan, or gate outcome depends on it -- so this is
	// enrichment, not a gate, judged deliberately here per this test's own
	// review requirement.
	"spawn-orphans":        true,
	"state-checkpoint":     true,
	"state-mutate":         true,
	"state-read":           true,
	"status":               true,
	"suggest-analyze":      true,
	"suggest-approve":      true,
	"survey-verify":        true,
	"swarm-finalize":       true,
	"swarm":                true,
	"swarm-display-update": true,
	// temp-clean judged here, deliberately, at the point the 172-06 fence
	// repair (Task 3) makes it visible for the first time: it is a
	// housekeeping/cleanup command whose failure degrades tidiness only —
	// no verification result, security scan, or gate outcome depends on it
	// — so it is enrichment, not a gate.
	"temp-clean":               true,
	"tunnels":                  true,
	"unblock":                  true,
	"unload-state":             true,
	"update":                   true,
	"validate-state":           true,
	"validate-worker-response": true,
	"verify-castes":            true,
	"version":                  true,
	"watch":                    true,
	"worktree-allocate":        true,
	"worktree-list":            true,
	"worktree-merge-back":      true,
}

// T-160-23: no documented subcommand may sit unclassified. The enrichment
// default in commandCallSeverityFor is deliberate for CODE (adding a command
// doesn't force an edit here), but a command that reaches the DOCUMENTED
// corpus is about to be executed by wrappers — at that point someone must have
// decided which D-01 tier it belongs to. This test is what forces the
// decision.
func TestDocumentedSubcommandsAreSeverityClassified(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	calls := collectDocumentedCalls(t, root)
	if len(calls) == 0 {
		t.Fatal("extracted zero documented calls — a vacuous pass proves nothing")
	}

	unclassified := map[string][]string{}
	for _, c := range calls {
		if target, _, findErr := rootCmd.Find([]string{c.Command}); findErr != nil || target == nil || target == rootCmd {
			continue // unresolvable names are TestDocumentedCommandNamesResolve's violation
		}
		if _, gate := gateClassifiedCommands[c.Command]; gate {
			continue
		}
		if knownEnrichmentSubcommands[c.Command] {
			continue
		}
		if len(unclassified[c.Command]) < 3 {
			rel, _ := filepath.Rel(root, c.File)
			unclassified[c.Command] = append(unclassified[c.Command], fmt.Sprintf("%s:%d", rel, c.Line))
		}
	}

	if len(unclassified) > 0 {
		names := make([]string, 0, len(unclassified))
		for name := range unclassified {
			names = append(names, name)
		}
		sort.Strings(names)
		var detail strings.Builder
		for _, name := range names {
			fmt.Fprintf(&detail, "  %q (e.g. %s)\n", name, strings.Join(unclassified[name], ", "))
		}
		t.Errorf(
			"%d documented subcommand(s) have no D-01 severity classification. Decide deliberately for each: if its failure must halt a run, add it to gateClassifiedCommands (command_call_severity.go) with a rationale; if a loud warning suffices, add it to knownEnrichmentSubcommands in this file:\n%s",
			len(names), detail.String(),
		)
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

// gluedFenceMarker is built by concatenation rather than as one contiguous
// string literal, following the buildConstraintRe precedent
// (subcommand_reachability_ratchet_test.go:72) — so this file's own source
// never carries the bare triple-backtick marker text on a single line.
var gluedFenceMarker = "`" + "`" + "`"

// TestAuditedCorpusHasNoGluedFenceMarkers is the corpus-wide guard for
// T-172-26 / T-172-27 / T-172-28: a triple-backtick fence marker glued to
// the end of a content line desyncs extractDocumentedCalls' in-fence
// tracking and hides every invocation after it from the audit — the live
// shape that let continue-full.md:1243's positional-argument violation go
// unaudited for the whole phase. This sweep fails, naming file:line and the
// offending text, for any line in the audited corpus where the marker is
// present but is NOT the line's first non-whitespace content.
//
// Shares auditedFilePaths with collectDocumentedCalls so this sweep can
// never drift from the set the audit actually reads — a hardcoded second
// file list would rot the moment a corpus changed shape, which is the whole
// point of "sweep the audited corpus" rather than a fixed directory list.
func TestAuditedCorpusHasNoGluedFenceMarkers(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	paths := auditedFilePaths(t, root)
	// Anti-vacuity (T-172-28): a sweep whose enumeration silently returns
	// nothing — a moved corpus, a wrong root, a swallowed walk error —
	// would pass forever while checking nothing. Measured today: 215.
	if len(paths) < 200 {
		t.Fatalf("auditedFilePaths returned %d path(s), want >= 200 (measured 215) — an enumeration that silently returns nothing would pass forever, checking nothing", len(paths))
	}

	var offenders []string
	for _, p := range paths {
		raw, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		for i, line := range strings.Split(string(raw), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), gluedFenceMarker) {
				continue // marker is the line's own content — the correct, unglued shape
			}
			if strings.Contains(line, gluedFenceMarker) {
				rel, relErr := filepath.Rel(root, p)
				if relErr != nil {
					rel = p
				}
				offenders = append(offenders, fmt.Sprintf("%s:%d: %s", rel, i+1, strings.TrimSpace(line)))
			}
		}
	}

	if len(offenders) > 0 {
		t.Errorf("%d line(s) glue a fence marker to trailing content — this desyncs extractDocumentedCalls' fence tracking and hides every invocation after it from the audit, silently, the way continue-full.md:1243 sat unaudited for this whole phase:\n  %s",
			len(offenders), strings.Join(offenders, "\n  "))
	}
	t.Logf("scanned %d files in the audited corpus", len(paths))
}
