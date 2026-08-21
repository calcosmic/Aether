package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// goHintFormatVerbRe normalises fmt verbs into the placeholder shape the
// shared audit already understands: `aether flag-resolve --id %s` is the same
// contract as `aether flag-resolve --id <id>`.
var goHintFormatVerbRe = regexp.MustCompile(`%[+#\-\d.]*[a-zA-Z]`)

// TestGoSourceHintsMatchCobraContracts audits the `aether ...` invocations the
// runtime prints at users from Go source. The markdown audit
// (TestCommandCallsMatchCobraContracts) never looked here, which is exactly
// how `aether flag <id> --resolve` — a flag that does not exist — shipped and
// was handed to a user at the moment they were already blocked.
func TestGoSourceHintsMatchCobraContracts(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	calls := collectGoSourceHintCalls(t, filepath.Join(root, "cmd"))
	if len(calls) == 0 {
		t.Fatal("extracted zero Go-source hints — the audit is looking at nothing and would pass vacuously forever")
	}

	var violations []string
	for _, c := range calls {
		if v := validateGoHintAgainstCobra(c); v != "" {
			rel, _ := filepath.Rel(root, c.File)
			violations = append(violations, rel+":"+itoaLine(c.Line)+": `"+c.Raw+"` — "+v)
		}
	}

	if len(violations) > 0 {
		t.Fatalf("%d Go-source hint(s) name commands or flags that do not exist —\nusers are told to run these:\n  %s",
			len(violations), strings.Join(violations, "\n  "))
	}
	t.Logf("audited %d runtime-emitted hints", len(calls))
}

// validateGoHintAgainstCobra checks the two things a runtime-emitted hint can
// get wrong in a way the user feels: a command name that does not resolve, and
// a flag that command does not have. It deliberately does NOT validate
// positional arity.
//
// Unlike markdown, where fenced blocks bound an invocation exactly, Go hint
// strings interleave commands with prose ("run `aether install` to populate
// the hub") and build flags by concatenation ("--skip-watchers%s"). Validating
// positionals there produces false failures on correct hints, and an audit
// that cries wolf gets deleted. Flags and command names are unambiguous, and
// a bad flag — `aether flag <id> --resolve` — is precisely what shipped.
func validateGoHintAgainstCobra(c documentedCall) string {
	// A command name is lowercase kebab-case. Anything else — "Go binary from
	// GitHub Releases", a `<arg>` placeholder — is prose or a template, not an
	// invocation this audit can reason about.
	if !plausibleCommandToken(c.Command) {
		return ""
	}
	// Find resolves the DEEPEST subcommand path, so `ceremony spawn-plan
	// --workflow` validates --workflow against spawn-plan, not ceremony.
	target, remaining, err := rootCmd.Find(append([]string{c.Command}, c.Args...))
	if err != nil || target == nil || target == rootCmd {
		// Only a backticked invocation is the runtime telling a user to run
		// something. Unbackticked prose ("copies the aether binary to <path>")
		// is not a hint, and reporting it trains people to ignore this audit.
		if !c.InFence {
			return ""
		}
		return "unknown command " + c.Command
	}
	for cmd := target; cmd != nil; cmd = cmd.Parent() {
		if cmd.DisableFlagParsing {
			return "" // `aether host *` forwards its raw command line
		}
	}
	for _, arg := range remaining {
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		name := strings.TrimPrefix(arg, "--")
		if eq := strings.Index(name, "="); eq >= 0 {
			name = name[:eq]
		}
		// A flag built by concatenation ("--skip-watchers%s") is not a literal
		// flag name; the audit cannot know what it renders to.
		if strings.Contains(name, "<arg>") || name == "" {
			continue
		}
		if target.Flags().Lookup(name) != nil ||
			target.InheritedFlags().Lookup(name) != nil ||
			rootCmd.PersistentFlags().Lookup(name) != nil {
			continue
		}
		return "unknown flag --" + name + " for `aether " + target.Name() + "`"
	}
	return ""
}

// plausibleCommandToken reports whether a token could be a real command name:
// lowercase letters, digits and hyphens only.
func plausibleCommandToken(tok string) bool {
	if tok == "" {
		return false
	}
	for _, r := range tok {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

func itoaLine(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// collectGoSourceHintCalls walks Go sources and extracts `aether ...`
// invocations from STRING LITERALS ONLY. Using the AST rather than a line
// regex is what keeps prose in comments — of which there is a great deal in
// this codebase — out of the audit.
func collectGoSourceHintCalls(t *testing.T, dir string) []documentedCall {
	t.Helper()
	var all []documentedCall

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		// parser.SkipObjectResolution keeps this fast; comments are omitted by
		// default, which is the point.
		file, parseErr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			t.Fatalf("parse %s: %v", path, parseErr)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			value := lit.Value
			if len(value) >= 2 {
				value = value[1 : len(value)-1] // strip quotes/backticks
			}
			if !strings.Contains(value, "aether ") {
				return true
			}
			line := fset.Position(lit.Pos()).Line
			all = append(all, extractHintCallsFromLiteral(path, line, value)...)
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return all
}

// extractHintCallsFromLiteral pulls every `aether <verb> ...` invocation out of
// one string literal.
func extractHintCallsFromLiteral(path string, line int, value string) []documentedCall {
	var calls []documentedCall
	// Literals frequently pack several sentences or lines together; treat each
	// line, and each segment after a sentence break, independently.
	value = strings.ReplaceAll(value, `\n`, "\n")
	value = strings.ReplaceAll(value, `\"`, `"`)
	for _, segment := range strings.Split(value, "\n") {
		idx := 0
		for {
			rel := strings.Index(segment[idx:], "aether ")
			if rel < 0 {
				break
			}
			start := idx + rel
			// Require a word boundary: skip `aether-dev`, `.aether `, and
			// identifiers that merely end in "aether".
			if start > 0 {
				prev := segment[start-1]
				if prev != ' ' && prev != '`' && prev != '"' && prev != '(' && prev != '\t' {
					idx = start + len("aether ")
					continue
				}
			}
			backticked := start > 0 && segment[start-1] == '`'
			rest := segment[start:]
			// A hint ends at a backtick, quote, or sentence terminator.
			if cut := strings.IndexAny(rest[len("aether "):], "`\"'"); cut >= 0 {
				rest = rest[:len("aether ")+cut]
			}
			raw := goHintFormatVerbRe.ReplaceAllString(strings.TrimSpace(rest), "<arg>")
			raw = strings.TrimRight(raw, ".,;:)")
			tokens := tokenizeShellLike(raw)
			// Everything from a shell operator onward belongs to another
			// command (`aether completion bash > $(brew --prefix)/...`).
			for i, tok := range tokens {
				if isShellOperator(tok) {
					tokens = tokens[:i]
					break
				}
			}
			if len(tokens) >= 2 && tokens[0] == "aether" {
				calls = append(calls, documentedCall{
					File:    path,
					Line:    line,
					Raw:     raw,
					Command: tokens[1],
					Args:    tokens[2:],
					// InFence marks a backticked invocation — the runtime
					// telling a user to run something, as opposed to prose
					// that merely contains the word ("the aether binary").
					InFence: backticked,
				})
			}
			idx = start + len("aether ")
		}
	}
	return calls
}
