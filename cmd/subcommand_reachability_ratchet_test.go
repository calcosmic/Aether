package cmd

// WIRE-01 (172-02): the orphan reachability ratchet.
//
// This file answers one question for every registered cobra subcommand: does
// anything outside its own definition file actually execute it? "Works, and
// nothing calls it" is the precise failure mode CLAUDE.md's Definition of Done
// exists to catch (D-05) — a command can pass every test in its own file and
// still be reached by nobody. This test enumerates the real cobra tree,
// searches exactly the three permitted kinds of caller evidence (D-01), and
// fails naming every command it finds neither in that evidence nor in the
// committed, shrink-only allowlist.
//
// See .planning/phases/172-wiring-proof/172-CONTEXT.md for the full decision
// record (D-01 through D-15) this file implements.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// updateOrphanAllowlist regenerates testdata/orphan_allowlist.json from the
// scanner's real, honest output (D-07). It must NEVER touch
// testdata/orphan_allowlist_baseline.json — TestWiringGuardsHaveNoRuntimeEscapeHatch
// asserts that in this file's own source, not just in this comment (D-11).
var updateOrphanAllowlist = flag.Bool("update-orphan-allowlist", false, "regenerate testdata/orphan_allowlist.json from the scanner's real output")

// callerWrapperCorpora are the D-01(b) "platform wrapper the runtime actually
// loads" trees. Deliberately narrower than auditedCorpora in
// command_call_audit_test.go: that list also includes
// .aether/docs/command-playbooks and colony/playbooks, which D-02/D-06 rule
// out as caller evidence (a doc mention is not an execution) even though they
// stay in scope for the flag audit.
var callerWrapperCorpora = []string{
	filepath.Join(".claude", "commands", "ant"),
	filepath.Join(".opencode", "commands", "ant"),
	filepath.Join(".aether", "commands"),
}

// hookScriptCorpora are the D-01(c) "shipped hook or script" trees.
var hookScriptCorpora = []struct {
	dir string
	ext string
}{
	{filepath.Join(".aether", "utils", "hooks"), ".js"},
	{filepath.Join("scripts"), ".sh"},
}

// buildConstraintRe matches a real Go build-constraint directive, which is
// only recognised by the toolchain at column 0 of a line (never indented).
// Anchoring this way is what lets TestWiringGuardsHaveNoRuntimeEscapeHatch
// scan this file's own RAW source — including this doc comment's mention of
// the phrase — without self-triggering on prose that merely discusses it.
// The pattern is built by concatenation, rather than as one contiguous
// string literal, so this file's own source never contains the literal
// directive text on a single line — the exact thing
// TestWiringGuardsHaveNoRuntimeEscapeHatch's own verify command
// (`grep -c 'os.Getenv\|t\.Skip\|//go:build'`) greps for.
var buildConstraintRe = regexp.MustCompile(`(?m)^//` + `go:build`)

// subcommandNameShapeRe is the same command-name shape extractDocumentedCalls
// already enforces (^[a-z][a-z0-9-]*$), reused here so a credited token is
// never anything other than a real, lowercase, hyphenated command name.
var subcommandNameShapeRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// yamlRuntimeCommandRe extracts a .aether/commands/*.yaml `runtime.command`
// field's value. extractDocumentedCalls only looks inside backticks or fenced
// code blocks, and this field is a plain, unbackticked YAML string
// (`command: "AETHER_OUTPUT_MODE=visual aether status $ARGUMENTS"`), so it is
// otherwise invisible to the shared extractor. D-03 requires a menu entry to
// count as a caller, so this file reads the field directly.
var yamlRuntimeCommandRe = regexp.MustCompile(`(?m)^\s*command:\s*"(.*)"\s*$`)

// binPathAssignRe finds a shell variable assigned a path ending in `/aether`
// (`BIN="$WORK/aether"` in scripts/smoke-daily-driver.sh), so a later
// `"$BIN" status` invocation in the same file is recognised as calling the
// binary even though the literal token is not `aether`.
var binPathAssignRe = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)=.*/aether"?\s*$`)

// skillLifecycleDeletedCommands records SKILL-01's reviewed eight-command set
// (CONTEXT.md's "8 skill entries" — skill-index, skill-detect, skill-match,
// skill-inject, skill-list, skill-diff, skill-parse-frontmatter,
// skill-cache-rebuild), originally the set Phase 178's success criterion
// measured "reaching zero" via wiring. Phase 191 delivered SKILL-01 by
// deleting the entire reviewed set as dead CLI surface instead — the
// standalone cobra.Command wrappers are gone; matchSkillsForWorkflow and its
// live in-process caller (cmd/codex_build.go's composeBuildManifestBrief)
// were preserved unconditionally. There is no longer a "reaches zero" count
// to measure: the set reached zero by not existing. Kept leaf-keyed, as the
// original set was, for the same audit-trail reason it was reviewed by name.
var skillLifecycleDeletedCommands = []string{
	"skill-index",
	"skill-detect",
	"skill-match",
	"skill-inject",
	"skill-list",
	"skill-diff",
	"skill-parse-frontmatter",
	"skill-cache-rebuild",
}

// assertSkillLifecycleCommandsStayDeleted proves each of SKILL-01's eight
// reviewed leaf names no longer resolves to a real, registered command via
// rootCmd.Find — the reachability ratchet's own proof mechanism, inverted
// from "must resolve" (pre-191, when the set was orphaned-but-present) to
// "must never resolve again" (post-191, when the set was deleted outright).
// A leaf name that resolves again is a regression: either it was
// reintroduced by accident and must be deleted again, or it is a genuinely
// new command that happens to reuse a retired name and that reuse must be
// recorded on purpose, never silently allowed to pass.
func assertSkillLifecycleCommandsStayDeleted(t *testing.T) {
	t.Helper()
	for _, leaf := range skillLifecycleDeletedCommands {
		if target, _, err := rootCmd.Find([]string{leaf}); err == nil && target != nil && target != rootCmd {
			t.Errorf("skill-lifecycle command %q resolved to %q — Phase 191 deleted this entire reviewed set as dead CLI surface (SKILL-01); if it is back, either it has a new, real caller (update this ratchet with that evidence) or it must be deleted again", leaf, target.CommandPath())
		}
	}
}

// pathCollisionRevealedOrphans is 172-09's one deliberate, on-the-record
// D-07 disposition (2026-08-12): the nine command paths the path-keying
// migration in this plan newly reveals as orphaned, because the pre-migration
// name-keyed scanner collapsed each of them onto a same-leaf-name sibling
// that DOES have a real caller. D-07 requires every orphan the scan finds to
// go into the baseline honestly — these are recorded, not repaired (no
// caller is invented, no command is deleted). Each is a genuinely different,
// separately-registered command from the sibling that used to (incorrectly)
// vouch for it:
//
//   - aether colonize (cmd/codex_workflow_cmds.go:30) — previously credited by
//     a documented call to aether host colonize (cmd/host_cmd.go:32).
//   - aether closeout (cmd/ceremony_cmd.go:118) — previously credited by a
//     documented call to aether ceremony closeout.
//   - aether host (cmd/host_cmd.go:16) — the bare parent command with no
//     subcommand. Previously credited by ANY "aether host <subcommand>"
//     invocation, because the old scheme credited the first token ("host")
//     bare, regardless of which subcommand followed. Nothing calls bare
//     "aether host" with no subcommand.
//   - aether host build (cmd/host_cmd.go:56) — previously credited by a
//     documented call to the unrelated top-level aether build
//     (cmd/codex_workflow_cmds.go:106).
//   - aether host oracle (cmd/host_cmd.go:80) — previously credited by a
//     documented call to the unrelated top-level aether oracle
//     (cmd/compatibility_cmds.go:55).
//   - aether host swarm (cmd/host_cmd.go:96) — previously credited by a
//     documented call to the unrelated top-level aether swarm
//     (cmd/swarm_cmd.go:103).
//   - aether host watch (cmd/host_cmd.go:88) — previously credited by a
//     documented call to the unrelated top-level aether watch
//     (cmd/compatibility_cmds.go:31).
//   - aether export pheromones (cmd/exchange.go:46) — previously credited by
//     a documented call to aether import pheromones (cmd/exchange.go:344),
//     which shares the bare leaf "pheromones". Neither is actually called
//     directly today: /ant-export-signals and /ant-import-signals invoke the
//     separate flat commands aether export-signals / aether import-signals
//     (cmd/codex_signals_cmds.go), not the nested "export pheromones" /
//     "import pheromones" subcommands.
//   - aether import pheromones (cmd/exchange.go:344) — previously credited by
//     a documented call to aether export pheromones, the mirror image of the
//     entry above.
var pathCollisionRevealedOrphans = map[string]bool{
	"aether colonize":          true,
	"aether closeout":          true,
	"aether host":              true,
	"aether host build":        true,
	"aether host oracle":       true,
	"aether host swarm":        true,
	"aether host watch":        true,
	"aether export pheromones": true,
	"aether import pheromones": true,

	// Phase 187 (2026-08-18), a second, distinct kind of reviewed exemption:
	// "aether worktree-reap" is a brand-new command with no pre-migration
	// entry at all — it did not exist before this phase, so it cannot be one
	// of the 278 frozen leaves' legitimate path expansions. It is listed here
	// deliberately, not because of a path-collision, but because D-01/T-187-13
	// require it to have NO caller anywhere except a human typing it.
	// TestOrphanAllowlistOnlyShrinks carries the same rationale at its call
	// site; this is the other half of the same on-the-record decision.
	"aether worktree-reap": true,

	// Phase 191.1 (2026-08-21), the same kind of reviewed exemption as
	// worktree-reap immediately above: "aether verify-out-of-band" is a
	// brand-new command (FIELD-05,
	// .planning/todos/pending/2026-08-21-no-reentry-path-for-out-of-band-work.md)
	// with no pre-migration entry at all. It is listed here deliberately
	// because its own reachability ratchet
	// (TestVerifyOutOfBandHasNoLifecycleCaller,
	// cmd/verify_out_of_band_reachability_test.go) requires it to have NO
	// caller anywhere except a human typing it — an invented wrapper, hook,
	// or script call would itself violate that guarantee. This is the other
	// half of the same on-the-record decision as its entries in
	// testdata/orphan_allowlist.json and testdata/orphan_allowlist_baseline.json.
	"aether verify-out-of-band": true,
}

// loadPreMigrationReasonByLeaf reads the frozen, byte-identical,
// name-keyed pre-migration snapshot and returns leaf name -> (reason,
// owner_phase), so writeOrphanAllowlist can carry forward exactly the
// reason/owner_phase each pre-existing entry already had rather than
// re-deriving it (which could silently drift the D-08 count).
func loadPreMigrationReasonByLeaf(t *testing.T) map[string]orphanAllowlistEntry {
	t.Helper()
	pre := loadOrphanAllowlist(t, "testdata/orphan_allowlist_baseline_pre_path_migration.json")
	byLeaf := make(map[string]orphanAllowlistEntry, len(pre))
	for _, e := range pre {
		byLeaf[e.Name] = e
	}
	return byLeaf
}

// preMigrationSnapshotSHA256 is the SHA-256 of
// testdata/orphan_allowlist_baseline_pre_path_migration.json as committed —
// the frozen anchor every "the allowlist can only shrink" claim rests on
// (CR-05). Editing that file by even a single character — the exact defect
// the verifier reproduced, which passed all 30 named guard tests at the
// time — silently raises the tolerance ceiling for every future entry,
// because pathMigrationToleratedPaths trusts whatever leaves the file
// currently contains. Pinning the hash in Go source rather than leaving the
// anchor as a bare JSON file is what makes changing it an obvious,
// reviewable code edit instead of a line buried in a 278-entry list — D-11's
// "visible, on-the-record edit to a checked-in file." If this file genuinely
// must change, the change is a reviewed edit to this constant, verified by
// TestPreMigrationSnapshotIsFrozen, never a silent JSON edit.
const preMigrationSnapshotSHA256 = "873cad5a20e472af7050e69042be9faf8527a2c3e1d34907762b37e6a5845e0e"

// TestPreMigrationSnapshotIsFrozen is CR-05's fix: hash the frozen
// pre-migration anchor at test time and fail if it no longer matches
// preMigrationSnapshotSHA256. Uses no environment read, no skip, and no
// flag, so TestWiringGuardsHaveNoRuntimeEscapeHatch stays green.
func TestPreMigrationSnapshotIsFrozen(t *testing.T) {
	const path = "testdata/orphan_allowlist_baseline_pre_path_migration.json"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if got != preMigrationSnapshotSHA256 {
		t.Errorf("%s has SHA-256 %s, want %s (preMigrationSnapshotSHA256)\n"+
			"This file is the anchor every \"the list can only shrink\" guarantee rests on and must never be edited. "+
			"If it genuinely must change, that is a reviewed edit to the preMigrationSnapshotSHA256 constant in Go source, not a silent JSON edit.",
			path, got, preMigrationSnapshotSHA256)
	}
}

// orphanAllowlistEntry is one committed exemption. D-08: every entry carries
// a reason and an owning phase, so a later phase's "this list drops to 0"
// criterion stays measurable. D-09: this lives in its own dedicated file,
// never as a field on command_catalog.json, which scripts/classify_commands.py
// rewrites in place.
type orphanAllowlistEntry struct {
	Name       string `json:"name"`
	Reason     string `json:"reason"`
	OwnerPhase string `json:"owner_phase"`
}

// registeredCommandInfo is one node of the real, registered cobra tree.
//
// Path and AliasPaths carry the full resolved cobra command path
// (CommandPath(), e.g. "aether host colonize") rather than the bare leaf
// name. This is the fix for CR-04/GAP B: two commands sharing a leaf name at
// different parents (`aether colonize` vs `aether host colonize`) used to
// collapse onto the same bare-name key, so a documented call to one silently
// credited the other as wired. Name is kept only for the AST-based
// definition-file attribution check and for reason-tagging, both of which are
// leaf-based by design.
type registeredCommandInfo struct {
	Name       string
	Path       string
	AliasPaths []string
	Aliases    []string
	Hidden     bool
}

// ---------------------------------------------------------------------------
// Registration truth
// ---------------------------------------------------------------------------

// enumerateRegisteredCommands recursively walks the real, registered cobra
// tree. Cobra's own generated "help" and "completion" commands are skipped —
// they are not repo-owned and have no definition file to point at. Hidden
// commands are NOT skipped: a hidden orphan is still an orphan.
func enumerateRegisteredCommands(root *cobra.Command) []registeredCommandInfo {
	var out []registeredCommandInfo
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		for _, child := range c.Commands() {
			if child.Name() == "help" || child.Name() == "completion" {
				continue
			}
			path := child.CommandPath()
			aliasPaths := make([]string, 0, len(child.Aliases))
			for _, alias := range child.Aliases {
				aliasPaths = append(aliasPaths, strings.TrimSuffix(path, child.Name())+alias)
			}
			out = append(out, registeredCommandInfo{
				Name:       child.Name(),
				Path:       path,
				AliasPaths: aliasPaths,
				Aliases:    append([]string{}, child.Aliases...),
				Hidden:     child.Hidden,
			})
			walk(child)
		}
	}
	walk(root)
	return out
}

// buildCommandDefinitionIndex maps a command name (the first whitespace-
// delimited token of its Use string, matching cobra's own Name() derivation)
// to the basename of the file that declares it. It reuses declNameAndBody
// (cmd/visual_writer_discipline_test.go) verbatim, because a FuncDecl-only AST
// walk misses every cobra command — they are declared as
// `var xCmd = &cobra.Command{...}` or, for anonymous inline registrations
// (cmd/host_cmd.go), as a composite literal inside an init() function body.
//
// Errors reading or parsing cmdDir are tolerated (returns an empty index)
// rather than fatal: this function also runs against synthetic roots in
// TestCallerEvidenceCreditsCommandSubstitution's half B, which have no cmd/
// directory at all. A genuinely broken real cmd/ directory surfaces loudly
// anyway, via every registered command showing up as "unattributed" in
// TestNoRegisteredSubcommandIsUnreferenced.
func buildCommandDefinitionIndex(t *testing.T, cmdDir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(cmdDir)
	if err != nil {
		return map[string]string{}
	}

	fset := token.NewFileSet()
	index := map[string]string{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, filepath.Join(cmdDir, name), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}

		for _, decl := range file.Decls {
			_, body := declNameAndBody(decl)
			if body == nil {
				continue
			}
			ast.Inspect(body, func(inner ast.Node) bool {
				// Case 1: a direct `&cobra.Command{Use: "..."}` composite
				// literal — the common shape.
				if cl, ok := inner.(*ast.CompositeLit); ok {
					sel, ok := cl.Type.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					pkgIdent, ok := sel.X.(*ast.Ident)
					if !ok || pkgIdent.Name != "cobra" || sel.Sel.Name != "Command" {
						return true
					}
					for _, elt := range cl.Elts {
						kv, ok := elt.(*ast.KeyValueExpr)
						if !ok {
							continue
						}
						key, ok := kv.Key.(*ast.Ident)
						if !ok || key.Name != "Use" {
							continue
						}
						lit, ok := kv.Value.(*ast.BasicLit)
						if !ok || lit.Kind != token.STRING {
							continue
						}
						useVal, err := strconv.Unquote(lit.Value)
						if err != nil {
							continue
						}
						fields := strings.Fields(useVal)
						if len(fields) == 0 {
							continue
						}
						cmdName := fields[0]
						if _, exists := index[cmdName]; !exists {
							index[cmdName] = name
						}
					}
					return true
				}

				// Case 2: a factory call whose *cobra.Command body builds its
				// Use string from a parameter rather than a literal
				// (`var focusCmd = newSignalShortcutCommand("focus", ...)`,
				// codex_workflow_cmds.go). The literal command name only
				// exists at the CALL SITE, not inside the factory's own
				// composite literal, so it cannot be found by case 1 alone.
				// Scoped narrowly to this repo's one known factory rather
				// than any function whose name merely contains "Command",
				// to avoid over-attributing an unrelated call.
				if call, ok := inner.(*ast.CallExpr); ok {
					fn, ok := call.Fun.(*ast.Ident)
					if !ok || fn.Name != "newSignalShortcutCommand" || len(call.Args) == 0 {
						return true
					}
					lit, ok := call.Args[0].(*ast.BasicLit)
					if !ok || lit.Kind != token.STRING {
						return true
					}
					cmdName, err := strconv.Unquote(lit.Value)
					if err != nil || cmdName == "" {
						return true
					}
					if _, exists := index[cmdName]; !exists {
						index[cmdName] = name
					}
				}
				return true
			})
		}
	}
	return index
}

// ---------------------------------------------------------------------------
// Caller evidence (D-01: exactly three kinds, D-02/D-05/D-06: nothing else)
// ---------------------------------------------------------------------------

// extractYAMLRuntimeCommand reads a .aether/commands/*.yaml `runtime.command`
// field and returns the aether subcommand it names, plus its immediately
// following args. D-03: a menu entry counts as a caller.
func extractYAMLRuntimeCommand(path string) (string, []string, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, false
	}
	m := yamlRuntimeCommandRe.FindStringSubmatch(string(data))
	if m == nil {
		return "", nil, false
	}
	fields := tokenizeShellLike(m[1])
	idx := -1
	for j, f := range fields {
		nf := normalizeShellToken(f)
		if nf == "aether" || strings.HasSuffix(nf, "/aether") {
			idx = j
			break
		}
	}
	if idx == -1 || idx+1 >= len(fields) {
		return "", nil, false
	}
	name := fields[idx+1]
	if !subcommandNameShapeRe.MatchString(name) {
		return "", nil, false
	}
	return name, fields[idx+2:], true
}

// collectYAMLRuntimeCommandPaths returns every resolved command PATH
// reachable via a `runtime.command` field in dir, used only by the
// non-vacuity guard in TestNoRegisteredSubcommandIsUnreferenced to prove the
// evidence set is resolved at runtime rather than hardcoded. Renamed from
// collectYAMLRuntimeCommandNames and resolved through rootCmd.Find (the same
// truncation rule credit() uses) so its output vocabulary matches the
// path-keyed evidence set it is compared against — a bare-name comparison
// against a path-keyed set would report the non-vacuity check as broken for
// a vocabulary-mismatch reason that has nothing to do with a real regression.
func collectYAMLRuntimeCommandPaths(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		name, args, ok := extractYAMLRuntimeCommand(filepath.Join(dir, e.Name()))
		if !ok {
			continue
		}
		argv := []string{name}
		for _, a := range args {
			if isShellOperator(a) || isPlaceholder(a) {
				break
			}
			if strings.HasPrefix(a, "-") {
				break
			}
			if !subcommandNameShapeRe.MatchString(a) {
				break
			}
			argv = append(argv, a)
		}
		if target, _, ferr := rootCmd.Find(argv); ferr == nil && target != nil && target != rootCmd {
			paths = append(paths, target.CommandPath())
		}
	}
	return paths
}

// discoverBinaryPathVars finds shell variables assigned a value ending in
// `/aether` within content (`BIN="$WORK/aether"`), so a later `"$BIN" status`
// invocation is recognised as calling the binary.
func discoverBinaryPathVars(content string) map[string]bool {
	vars := map[string]bool{}
	for _, line := range strings.Split(content, "\n") {
		if m := binPathAssignRe.FindStringSubmatch(line); m != nil {
			vars[m[1]] = true
		}
	}
	return vars
}

// isBinaryToken reports whether tok names the aether binary: the literal
// `aether`, a path ending in `/aether`, or a shell variable known (via
// discoverBinaryPathVars) to hold a path ending in `/aether`.
func isBinaryToken(tok string, binVars map[string]bool) bool {
	nf := normalizeShellToken(tok)
	if nf == "aether" || strings.HasSuffix(nf, "/aether") {
		return true
	}
	bare := strings.Trim(tok, `"'`)
	bare = strings.TrimPrefix(bare, "$")
	bare = strings.Trim(bare, "{}")
	return binVars[bare]
}

// singleFileCallerNames returns every command name path treats as caller
// evidence, on its own — the unit both collectCallerEvidence (aggregating
// across every permitted file) and the single-caller search in
// TestDeletingACallerMakesTheRatchetNameIt (finding which ONE file credits a
// given command) are built from.
//
// TOKEN BOUNDARIES ARE LOAD-BEARING: every comparison here is against an
// exact token produced by tokenizeShellLike and normalizeShellToken, never a
// substring match. `skill-list` is a literal substring of
// `skill-list-lifecycle` (cmd/skill_lifecycle.go:125); a substring match
// would silently clear a real orphan.
func singleFileCallerNames(t *testing.T, path string) map[string]bool {
	t.Helper()
	names := map[string]bool{}

	// credit resolves command plus its immediately-following args through the
	// real cobra tree and records the resolved CommandPath() as the evidence
	// key — never the bare leaf name. This is the CR-04/GAP B fix: the old
	// version credited the bare command token PLUS every following bareword
	// (`aether host colonize` credited both the parent token "host" and the
	// bare trailing token "colonize"), which is exactly what let a documented
	// call to `aether host colonize` silently clear the unrelated top-level
	// `aether colonize`. rootCmd.Find consumes precisely the tokens that
	// resolve to a real command and returns the one node they resolve to, so
	// resolving once and keying on CommandPath() is the correct, narrower
	// replacement: `aether host colonize` now credits only
	// "aether host colonize", never bare "colonize".
	credit := func(command string, args []string) {
		argv := []string{command}
		for _, a := range args {
			if isShellOperator(a) || isPlaceholder(a) {
				break
			}
			if strings.HasPrefix(a, "-") {
				break
			}
			if !subcommandNameShapeRe.MatchString(a) {
				break
			}
			argv = append(argv, a)
		}
		if target, _, err := rootCmd.Find(argv); err == nil && target != nil && target != rootCmd {
			names[target.CommandPath()] = true
		}
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".md", ".yaml", ".yml":
		for _, c := range extractDocumentedCalls(t, path) {
			credit(c.Command, c.Args)
		}
		if ext == ".yaml" || ext == ".yml" {
			if name, args, ok := extractYAMLRuntimeCommand(path); ok {
				credit(name, args)
			}
		}
	case ".js", ".sh":
		data, err := os.ReadFile(path)
		if err != nil {
			return names
		}
		content := string(data)
		binVars := discoverBinaryPathVars(content)
		for _, line := range strings.Split(content, "\n") {
			fields := tokenizeShellLike(line)
			for j, f := range fields {
				if !isBinaryToken(f, binVars) {
					continue
				}
				if j+1 >= len(fields) {
					continue
				}
				name := normalizeShellToken(fields[j+1])
				if !subcommandNameShapeRe.MatchString(name) {
					continue
				}
				credit(name, fields[j+2:])
			}
		}
	}
	return names
}

// callerFileKey is the repo-root-relative, forward-slashed identity of a
// caller-corpus file, used both as a skipFiles key and as the human-readable
// name reported when TestDeletingACallerMakesTheRatchetNameIt names the file
// it suppressed.
func callerFileKey(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

// isAetherSelfArg reports whether arg is the literal "aether" or os.Args[0] —
// the two shapes D-01(a) recognises as the binary self-invoking itself.
func isAetherSelfArg(arg ast.Expr) bool {
	if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		if v, err := strconv.Unquote(lit.Value); err == nil && v == "aether" {
			return true
		}
	}
	if idx, ok := arg.(*ast.IndexExpr); ok {
		if sel, ok := idx.X.(*ast.SelectorExpr); ok {
			if pkgIdent, ok := sel.X.(*ast.Ident); ok && pkgIdent.Name == "os" && sel.Sel.Name == "Args" {
				return true
			}
		}
	}
	return false
}

// collectGoSelfInvocationCallers implements D-01(a): Go code outside a
// command's own definition file that executes the binary via
// exec.Command("aether", ...) or exec.Command(os.Args[0], ...). Expect zero
// hits today (verified: cmd/install_cmd.go's only exec.Command call builds
// the binary with `go build`, it does not invoke it) — implemented narrowly
// and honestly anyway, per the plan.
func collectGoSelfInvocationCallers(t *testing.T, root string, defIndex map[string]string, skipFiles map[string]bool) map[string]bool {
	t.Helper()
	evidence := map[string]bool{}
	fset := token.NewFileSet()
	for _, sub := range []string{"cmd", "pkg"} {
		dir := filepath.Join(root, sub)
		_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return nil
			}
			key := callerFileKey(root, p)
			if skipFiles[key] {
				return nil
			}
			file, perr := parser.ParseFile(fset, p, nil, 0)
			if perr != nil {
				return nil
			}
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkgIdent, ok := sel.X.(*ast.Ident)
				if !ok || pkgIdent.Name != "exec" || sel.Sel.Name != "Command" {
					return true
				}
				if len(call.Args) < 2 {
					return true
				}
				if !isAetherSelfArg(call.Args[0]) {
					return true
				}
				lit, ok := call.Args[1].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				name, uerr := strconv.Unquote(lit.Value)
				if uerr != nil {
					return true
				}
				if sub == "cmd" && defIndex[name] == filepath.Base(p) {
					// Self-reference from the command's own definition file
					// is not caller evidence (D-01: "outside its own
					// definition file"). Deliberately leaf-based: defIndex is
					// keyed by leaf name, and a command can only be defined
					// once, so no path ambiguity applies here.
					return true
				}
				if target, _, ferr := rootCmd.Find([]string{name}); ferr == nil && target != nil && target != rootCmd {
					evidence[target.CommandPath()] = true
				}
				return true
			})
			return nil
		})
	}
	return evidence
}

// collectCallerEvidence returns the set of command names with at least one
// caller among exactly the three permitted kinds (D-01), searching every file
// in callerWrapperCorpora and hookScriptCorpora plus a narrow Go
// self-invocation scan, skipping any file whose callerFileKey is set in
// skipFiles.
//
// This is parameterised — not an inline loop — so
// TestDeletingACallerMakesTheRatchetNameIt can call it again with one file
// suppressed, and TestCallerEvidenceCreditsCommandSubstitution's half B can
// call it against a synthetic root.
func collectCallerEvidence(t *testing.T, root string, skipFiles map[string]bool) map[string]bool {
	t.Helper()
	evidence := map[string]bool{}

	for _, corpus := range callerWrapperCorpora {
		dir := filepath.Join(root, corpus)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if ext != ".md" && ext != ".yaml" && ext != ".yml" {
				continue
			}
			p := filepath.Join(dir, e.Name())
			if skipFiles[callerFileKey(root, p)] {
				continue
			}
			for name := range singleFileCallerNames(t, p) {
				evidence[name] = true
			}
		}
	}

	for _, hd := range hookScriptCorpora {
		dir := filepath.Join(root, hd.dir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if strings.ToLower(filepath.Ext(e.Name())) != hd.ext {
				continue
			}
			p := filepath.Join(dir, e.Name())
			if skipFiles[callerFileKey(root, p)] {
				continue
			}
			for name := range singleFileCallerNames(t, p) {
				evidence[name] = true
			}
		}
	}

	defIndex := buildCommandDefinitionIndex(t, filepath.Join(root, "cmd"))
	for name := range collectGoSelfInvocationCallers(t, root, defIndex, skipFiles) {
		evidence[name] = true
	}

	return evidence
}

// listCallerCorpusFiles returns the callerFileKey of every file
// collectCallerEvidence's wrapper and hook/script corpora scan, used by
// TestDeletingACallerMakesTheRatchetNameIt to find a single-caller command.
func listCallerCorpusFiles(root string) []string {
	var files []string
	for _, corpus := range callerWrapperCorpora {
		dir := filepath.Join(root, corpus)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if ext != ".md" && ext != ".yaml" && ext != ".yml" {
				continue
			}
			files = append(files, callerFileKey(root, filepath.Join(dir, e.Name())))
		}
	}
	for _, hd := range hookScriptCorpora {
		dir := filepath.Join(root, hd.dir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if strings.ToLower(filepath.Ext(e.Name())) != hd.ext {
				continue
			}
			files = append(files, callerFileKey(root, filepath.Join(dir, e.Name())))
		}
	}
	sort.Strings(files)
	return files
}

// collectSubstitutionCallerNames re-runs the same extraction path as
// collectCallerEvidence, restricted to calls whose raw text is shaped like a
// command substitution (contains "$("). Used only by
// TestCallerEvidenceCreditsCommandSubstitution's half A.
func collectSubstitutionCallerNames(t *testing.T, root string) map[string]bool {
	t.Helper()
	names := map[string]bool{}

	for _, corpus := range callerWrapperCorpora {
		dir := filepath.Join(root, corpus)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if ext != ".md" && ext != ".yaml" && ext != ".yml" {
				continue
			}
			for _, c := range extractDocumentedCalls(t, filepath.Join(dir, e.Name())) {
				if !strings.Contains(c.Raw, "$(") {
					continue
				}
				argv := []string{c.Command}
				for _, a := range c.Args {
					if isShellOperator(a) || isPlaceholder(a) {
						break
					}
					if strings.HasPrefix(a, "-") {
						break
					}
					if !subcommandNameShapeRe.MatchString(a) {
						break
					}
					argv = append(argv, a)
				}
				if target, _, ferr := rootCmd.Find(argv); ferr == nil && target != nil && target != rootCmd {
					names[target.CommandPath()] = true
				}
			}
		}
	}

	for _, hd := range hookScriptCorpora {
		dir := filepath.Join(root, hd.dir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || strings.ToLower(filepath.Ext(e.Name())) != hd.ext {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			for _, line := range strings.Split(string(data), "\n") {
				if !strings.Contains(line, "$(") {
					continue
				}
				fields := tokenizeShellLike(line)
				for j, f := range fields {
					nf := normalizeShellToken(f)
					if nf != "aether" && !strings.HasSuffix(nf, "/aether") {
						continue
					}
					if j+1 >= len(fields) {
						continue
					}
					name := normalizeShellToken(fields[j+1])
					if !subcommandNameShapeRe.MatchString(name) {
						continue
					}
					argv := []string{name}
					for k := j + 2; k < len(fields); k++ {
						a := fields[k]
						if isShellOperator(a) || isPlaceholder(a) {
							break
						}
						if strings.HasPrefix(a, "-") {
							break
						}
						if !subcommandNameShapeRe.MatchString(a) {
							break
						}
						argv = append(argv, a)
					}
					if target, _, ferr := rootCmd.Find(argv); ferr == nil && target != nil && target != rootCmd {
						names[target.CommandPath()] = true
					}
				}
			}
		}
	}

	return names
}

// ---------------------------------------------------------------------------
// Orphan computation and allowlist I/O
// ---------------------------------------------------------------------------

// computeOrphanNames returns, sorted, every registered command whose full
// command path and every alias path have no caller evidence.
func computeOrphanNames(registered []registeredCommandInfo, evidence map[string]bool) []string {
	// seen is keyed by Path now, not by bare Name. Command paths are unique
	// by construction (cobra does not allow two commands to register the
	// same path), so nothing collapses any more — the old collapsing of two
	// differently-parented commands sharing a leaf name into one allowlist
	// entry was itself part of the CR-04/GAP B defect: `aether colonize` and
	// `aether host colonize` used to share one "colonize" entry, so crediting
	// either cleared both.
	seen := map[string]bool{}
	var orphans []string
	for _, c := range registered {
		credited := evidence[c.Path]
		if !credited {
			for _, a := range c.AliasPaths {
				if evidence[a] {
					credited = true
					break
				}
			}
		}
		if !credited {
			if seen[c.Path] {
				// Unreachable: c.Path is unique per registered command, so
				// no two loop iterations can ever produce the same Path.
				// Kept as a defensive no-op rather than deleted, so a future
				// change to enumerateRegisteredCommands that reintroduces
				// path collisions fails by silently skipping an orphan
				// rather than by a panic — visible in a shrinking orphan
				// count, not a crash.
				continue
			}
			seen[c.Path] = true
			orphans = append(orphans, c.Path)
		}
	}
	sort.Strings(orphans)
	return orphans
}

// formatOrphanFailureMessage is the ratchet's own failure-message formatter:
// one line per command, naming it directly rather than reporting "N orphans
// found" — CONTEXT.md requires the failure surface to be readable by a
// non-technical operator.
func formatOrphanFailureMessage(orphans []string) string {
	lines := make([]string, 0, len(orphans))
	for _, name := range orphans {
		lines = append(lines, fmt.Sprintf("%s is registered but nothing calls it (searched: wrappers, menu specs, hooks, scripts)", name))
	}
	return strings.Join(lines, "\n  ")
}

// loadOrphanAllowlist reads a committed allowlist JSON file.
func loadOrphanAllowlist(t *testing.T, path string) []orphanAllowlistEntry {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (run with -update-orphan-allowlist to create testdata/orphan_allowlist.json)", path, err)
	}
	var entries []orphanAllowlistEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return entries
}

// writeOrphanAllowlist writes the scanner's real, honest output (D-07) to
// testdata/orphan_allowlist.json ONLY — never the baseline (D-11).
func writeOrphanAllowlist(t *testing.T, orphans []string) {
	t.Helper()
	preByLeaf := loadPreMigrationReasonByLeaf(t)
	entries := make([]orphanAllowlistEntry, 0, len(orphans))
	for _, name := range orphans {
		reason, owner := "unreviewed-pre-existing", "RECLAIM"
		switch {
		// The skill-lifecycle / 178 case was removed in Phase 191: that
		// reviewed eight-command set was deleted outright (SKILL-01), so none
		// of its names can ever appear in a fresh orphan scan again -- there
		// is nothing left for this branch to match.
		case pathCollisionRevealedOrphans[name]:
			// 172-09's one deliberate, on-the-record disposition (D-07):
			// newly revealed by the path-keying migration, recorded honestly
			// rather than repaired.
			reason, owner = "path-collision-revealed", "RECLAIM"
		default:
			// Carried forward from before the migration: keep exactly the
			// reason/owner_phase the pre-migration snapshot already recorded
			// for this leaf, rather than re-deriving it.
			leaf := name[strings.LastIndex(name, " ")+1:]
			if pre, ok := preByLeaf[leaf]; ok {
				reason, owner = pre.Reason, pre.OwnerPhase
			}
		}
		entries = append(entries, orphanAllowlistEntry{Name: name, Reason: reason, OwnerPhase: owner})
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		t.Fatalf("marshal orphan allowlist: %v", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile("testdata/orphan_allowlist.json", data, 0644); err != nil {
		t.Fatalf("write testdata/orphan_allowlist.json: %v", err)
	}
}

// stripGoComments removes /* */ block comment contents and blanks any line
// whose trimmed content starts with "//", so a comment merely EXPLAINING a
// forbidden pattern (e.g. "// never call os.Getenv here") does not itself
// trip a check for that pattern. Callers that must still detect a genuine
// "//"-only directive (a real //go:build constraint) check the raw,
// unstripped source instead — see TestWiringGuardsHaveNoRuntimeEscapeHatch.
func stripGoComments(src string) string {
	var withoutBlocks strings.Builder
	runes := []rune(src)
	inBlock := false
	for i := 0; i < len(runes); i++ {
		if inBlock {
			if i+1 < len(runes) && runes[i] == '*' && runes[i+1] == '/' {
				inBlock = false
				i++
			}
			continue
		}
		if i+1 < len(runes) && runes[i] == '/' && runes[i+1] == '*' {
			inBlock = true
			i++
			continue
		}
		withoutBlocks.WriteRune(runes[i])
	}

	lines := strings.Split(withoutBlocks.String(), "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			lines[i] = ""
		}
	}
	return strings.Join(lines, "\n")
}

// ---------------------------------------------------------------------------
// Task 1: the ratchet itself
// ---------------------------------------------------------------------------

// TestNoRegisteredSubcommandIsUnreferenced is the standard the rest of
// milestone v1.26 is measured against (WIRE-01). It fails, naming each
// offending command on its own line, when a registered cobra subcommand has
// no caller among the three permitted kinds and is not in the committed
// allowlist.
func TestNoRegisteredSubcommandIsUnreferenced(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	registered := enumerateRegisteredCommands(rootCmd)
	if len(registered) < 100 {
		t.Fatalf("enumerated only %d registered commands, want >= 100 — the enumeration is broken and would pass vacuously forever", len(registered))
	}

	// Anti-vacuity: every enumerated Path must be a real, resolved cobra
	// command path with the "aether " prefix. A silent degradation back to
	// bare names (or an empty Path) would make every evidence-set comparison
	// miss and report all ~405 commands as orphans — this states that
	// specific failure mode in one line instead of a wall of 405 individually
	// unhelpful diffs.
	var badPaths []string
	for _, c := range registered {
		if c.Path == "" || !strings.HasPrefix(c.Path, "aether ") {
			badPaths = append(badPaths, fmt.Sprintf("%q (Path=%q)", c.Name, c.Path))
		}
	}
	if len(badPaths) > 0 {
		sort.Strings(badPaths)
		t.Fatalf("%d registered command(s) have an empty or malformed Path (want the \"aether \" prefix from CommandPath()): %s",
			len(badPaths), strings.Join(badPaths, ", "))
	}

	defIndex := buildCommandDefinitionIndex(t, ".")
	var unattributed []string
	for _, c := range registered {
		if _, ok := defIndex[c.Name]; !ok {
			unattributed = append(unattributed, c.Name)
		}
	}
	if len(unattributed) > 0 {
		sort.Strings(unattributed)
		t.Errorf("%d registered command(s) have no AST-resolved definition file (a hole in the ratchet, not a pass): %s",
			len(unattributed), strings.Join(unattributed, ", "))
	}

	evidence := collectCallerEvidence(t, root, nil)
	if len(evidence) == 0 {
		t.Fatal("caller-evidence set is empty — the scan is not looking at anything, which would pass vacuously forever")
	}

	// Non-vacuity: the evidence set must contain a path resolved at runtime
	// from a real .aether/commands/*.yaml runtime.command field, not a
	// hardcoded string.
	yamlPaths := collectYAMLRuntimeCommandPaths(filepath.Join(root, ".aether", "commands"))
	if len(yamlPaths) == 0 {
		t.Fatal("found zero .aether/commands/*.yaml runtime.command fields resolving to a real command — cannot prove caller evidence is resolved at runtime")
	}
	foundYAMLEvidence := false
	for _, p := range yamlPaths {
		if evidence[p] {
			foundYAMLEvidence = true
			break
		}
	}
	if !foundYAMLEvidence {
		t.Fatal("caller-evidence set contains none of the command paths named by a .aether/commands/*.yaml runtime.command field — the menu-spec scan is not working")
	}

	// T-172-07 / token-boundary proof: "skill-list" is a literal substring of
	// "skill-list-lifecycle" (cmd/skill_lifecycle.go:125). Prove in the test
	// itself — not by eye — that a documented call to one does not credit the
	// other, using a synthetic single-file scan.
	{
		boundaryDir := t.TempDir()
		boundaryFile := filepath.Join(boundaryDir, "boundary.md")
		content := "```bash\naether skill-list-lifecycle\n```\n"
		if err := os.WriteFile(boundaryFile, []byte(content), 0644); err != nil {
			t.Fatalf("write token-boundary fixture: %v", err)
		}
		boundaryNames := singleFileCallerNames(t, boundaryFile)
		if boundaryNames["aether skill-list"] {
			t.Error("a documented call to skill-list-lifecycle incorrectly credited aether skill-list as having a caller — this is the exact substring-match failure T-172-07 guards against")
		}
		if !boundaryNames["aether skill-list-lifecycle"] {
			t.Error("a documented call to skill-list-lifecycle was not credited at all")
		}
	}

	orphans := computeOrphanNames(registered, evidence)

	if *updateOrphanAllowlist {
		writeOrphanAllowlist(t, orphans)
		t.Logf("wrote testdata/orphan_allowlist.json with %d entries from the scanner's real output", len(orphans))
		return
	}

	t.Logf("enumerated %d registered commands, found %d orphans", len(registered), len(orphans))

	allowlist := loadOrphanAllowlist(t, "testdata/orphan_allowlist.json")
	allowed := map[string]bool{}
	for _, e := range allowlist {
		allowed[e.Name] = true
	}

	var unallowed []string
	for _, name := range orphans {
		if !allowed[name] {
			unallowed = append(unallowed, name)
		}
	}

	if len(unallowed) > 0 {
		t.Errorf("%d registered subcommand(s) have no caller and are not in testdata/orphan_allowlist.json:\n  %s",
			len(unallowed), formatOrphanFailureMessage(unallowed))
	}

	// D-08 (Phase 191 SKILL-01 ruling-by-deletion): the reviewed eight-item
	// skill-lifecycle set (CONTEXT.md's "8 skill entries", originally the set
	// Phase 178's success criterion measured "reaching zero" via wiring) was
	// instead deleted outright as dead CLI surface — skill-index, skill-detect,
	// skill-match, skill-inject, skill-list, skill-diff,
	// skill-parse-frontmatter, and skill-cache-rebuild no longer exist as
	// invocable commands. matchSkillsForWorkflow and its live in-process
	// caller (cmd/codex_build.go's composeBuildManifestBrief) were preserved
	// unconditionally — this was a CLI-surface deletion, not a logic deletion.
	// There is no longer an owner_phase "178" count to measure: the set
	// reached zero by not existing, not by being wired up. This is NOT "every
	// entry whose name happens to begin with skill-" — skill_lifecycle.go's
	// unrelated authoring commands (skill-archive, skill-patch, skill-pin,
	// skill-promote, skill-view, skill-list-lifecycle) are a different,
	// unreviewed subsystem and correctly fall through to
	// "unreviewed-pre-existing".
	assertSkillLifecycleCommandsStayDeleted(t)
	for _, e := range allowlist {
		if e.OwnerPhase == "178" {
			t.Errorf("%q still carries owner_phase \"178\", but Phase 191 deleted the entire reviewed skill-lifecycle set (SKILL-01) rather than wiring it — no live orphan should carry this owner_phase anymore", e.Name)
		}
	}
}

// TestCallerEvidenceIsNotSharedBetweenSameLeafNames is the permanent,
// hermetic regression test for CR-04/GAP B. It is deliberately NOT pinned to
// the two real commands the gap was discovered against (`aether colonize` vs
// `aether host colonize`, `aether closeout` vs `aether ceremony closeout`) —
// those are expected to be repaired later (given a real caller or removed),
// and a guard whose own failure condition includes its target's eventual
// success is a guard that gets edited under pressure the day that happens.
// Instead this registers a throwaway same-leaf-name collision entirely inside
// its own body, so the property it proves — caller evidence for one path
// never leaks to a same-named command at a different path — survives the
// repair of colonize and closeout indefinitely.
func TestCallerEvidenceIsNotSharedBetweenSameLeafNames(t *testing.T) {
	const leaf = "ratchet-selftest-collide"
	const parent = "ratchet-selftest-parent"

	topLevel := &cobra.Command{Use: leaf, Run: func(*cobra.Command, []string) {}}
	parentCmd := &cobra.Command{Use: parent, Run: func(*cobra.Command, []string) {}}
	child := &cobra.Command{Use: leaf, Run: func(*cobra.Command, []string) {}}
	parentCmd.AddCommand(child)

	rootCmd.AddCommand(topLevel)
	rootCmd.AddCommand(parentCmd)
	defer rootCmd.RemoveCommand(topLevel)
	defer rootCmd.RemoveCommand(parentCmd)

	wantChildPath := "aether " + parent + " " + leaf
	wantTopLevelPath := "aether " + leaf

	// Sanity-check cobra actually resolves the two fixtures the way this test
	// assumes, before trusting any assertion built on top of that — via the
	// same rootCmd.Find entry point credit() itself uses, not just via
	// CommandPath() directly.
	if got := child.CommandPath(); got != wantChildPath {
		t.Fatalf("fixture child.CommandPath() = %q, want %q", got, wantChildPath)
	}
	if got := topLevel.CommandPath(); got != wantTopLevelPath {
		t.Fatalf("fixture topLevel.CommandPath() = %q, want %q", got, wantTopLevelPath)
	}
	if resolved, _, err := rootCmd.Find([]string{parent, leaf}); err != nil || resolved != child {
		t.Fatalf("rootCmd.Find([%q, %q]) = %v, %v, want the child fixture command", parent, leaf, resolved, err)
	}
	if resolved, _, err := rootCmd.Find([]string{leaf}); err != nil || resolved != topLevel {
		t.Fatalf("rootCmd.Find([%q]) = %v, %v, want the top-level fixture command", leaf, resolved, err)
	}

	tmp := t.TempDir()
	fixtureFile := filepath.Join(tmp, "collision.md")
	content := "```bash\naether " + parent + " " + leaf + "\n```\n"
	if err := os.WriteFile(fixtureFile, []byte(content), 0644); err != nil {
		t.Fatalf("write collision fixture: %v", err)
	}

	names := singleFileCallerNames(t, fixtureFile)
	if !names[wantChildPath] {
		t.Errorf("singleFileCallerNames did not credit %q from a documented call to %q", wantChildPath, parent+" "+leaf)
	}
	if names[wantTopLevelPath] {
		t.Errorf("singleFileCallerNames credited %q (the unrelated top-level command) from a call to %q — caller evidence leaked across a same-leaf-name collision, exactly the CR-04/GAP B defect", wantTopLevelPath, parent+" "+leaf)
	}

	registered := enumerateRegisteredCommands(rootCmd)
	orphans := computeOrphanNames(registered, names)

	orphanSet := map[string]bool{}
	for _, o := range orphans {
		orphanSet[o] = true
	}
	if !orphanSet[wantTopLevelPath] {
		t.Errorf("computeOrphanNames did not report %q as an orphan; it has no caller of its own and must be named", wantTopLevelPath)
	}
	if orphanSet[wantChildPath] {
		t.Errorf("computeOrphanNames reported %q as an orphan, but it has a direct documented caller", wantChildPath)
	}
}

// TestCallerEvidenceCreditsCommandSubstitution proves the seeding scan is not
// blind to a `result=$(aether cmd)` caller — the exact blind spot 172-00
// closed in the shared extractor. Both halves are required: half A confirms
// no allowlisted name secretly already has a substitution-shaped caller in
// the real corpora, and half B proves the extraction path itself can see a
// synthetic one.
func TestCallerEvidenceCreditsCommandSubstitution(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	t.Run("half_a_real_corpora", func(t *testing.T) {
		// allowlist entries are (post-migration) path-keyed strings, matching
		// collectSubstitutionCallerNames's now path-keyed output directly —
		// no resolution needed on this half, since both sides are already in
		// the same vocabulary.
		allowlist := loadOrphanAllowlist(t, "testdata/orphan_allowlist.json")
		subNames := collectSubstitutionCallerNames(t, root)

		var violations []string
		for _, e := range allowlist {
			if subNames[e.Name] {
				violations = append(violations, e.Name)
			}
		}
		if len(violations) > 0 {
			sort.Strings(violations)
			t.Errorf("allowlisted command(s) actually have a command-substitution caller in a permitted corpus, so they gained a real caller and must leave the allowlist rather than stay exempted: %s",
				strings.Join(violations, ", "))
		}
	})

	t.Run("half_b_synthetic_fixture", func(t *testing.T) {
		allowlist := loadOrphanAllowlist(t, "testdata/orphan_allowlist.json")
		if len(allowlist) == 0 {
			t.Fatal("allowlist is empty — half B cannot pick a name to build the synthetic caller from, which would make this half vacuous")
		}
		// Resolve the chosen allowlist entry through rootCmd.Find — the same
		// entry point credit() uses — to get its real, canonical
		// CommandPath(). This tolerates the allowlist entry being either a
		// bare leaf name (pre-migration) or a full "aether ..." path
		// (post-migration): TrimPrefix is a no-op on a bare name, and Find
		// resolves either shape to the one real command it names. The
		// synthetic invocation text is built from that resolved path, and
		// the assertion below checks evidence for that same resolved path —
		// proving the synthetic caller shape is credited under the exact key
		// computeOrphanNames will look up, regardless of allowlist key format.
		entryName := allowlist[0].Name
		target, _, ferr := rootCmd.Find(strings.Fields(strings.TrimPrefix(entryName, "aether ")))
		if ferr != nil || target == nil || target == rootCmd {
			t.Fatalf("allowlist entry %q did not resolve via rootCmd.Find — half B cannot build a synthetic caller for it", entryName)
		}
		invocation := target.CommandPath()

		tmp := t.TempDir()
		wrapperDir := filepath.Join(tmp, ".claude", "commands", "ant")
		if err := os.MkdirAll(wrapperDir, 0755); err != nil {
			t.Fatalf("mkdir synthetic wrapper dir: %v", err)
		}
		content := "```bash\nresult=$(" + invocation + " --some-flag)\n```\n"
		if err := os.WriteFile(filepath.Join(wrapperDir, "substitution-fixture.md"), []byte(content), 0644); err != nil {
			t.Fatalf("write synthetic fixture: %v", err)
		}

		evidence := collectCallerEvidence(t, tmp, nil)
		if !evidence[invocation] {
			t.Errorf("collectCallerEvidence did not credit %q from a synthetic `result=$(%s --some-flag)` caller — the seeding scan would be blind to this shape", invocation, invocation)
		}
	})
}

// TestRatchetDoesNotConsultTheRegeneratedCatalog turns D-04/D-09's rejections
// into an assertion: this ratchet must never read command_catalog.json or
// parity_snapshot.json, both of which are rewritten in place by
// scripts/classify_commands.py and would make the allowlist's storage
// vulnerable to silent erasure.
func TestRatchetDoesNotConsultTheRegeneratedCatalog(t *testing.T) {
	data, err := os.ReadFile("subcommand_reachability_ratchet_test.go")
	if err != nil {
		t.Fatalf("read own source: %v", err)
	}
	stripped := stripGoComments(string(data))
	for i, line := range strings.Split(stripped, "\n") {
		if strings.Contains(line, "command_catalog") || strings.Contains(line, "parity_snapshot") {
			t.Errorf("line %d references command_catalog or parity_snapshot, which this ratchet must never consult (D-04, D-09): %s", i+1, line)
		}
	}
}

// ---------------------------------------------------------------------------
// Task 2: shrink-only guard and self-tests proving the ratchet fires
// ---------------------------------------------------------------------------

// TestOrphanAllowlistOnlyShrinks is D-10/D-11: a pure set-membership diff
// against the committed baseline. It deliberately does not compare counts (a
// one-out-one-in swap must fail) and does not consult git history (CI
// shallow-clones and history is rewritable).
//
// Phase 187's one deliberate, on-the-record addition (2026-08-18): "aether
// worktree-reap" is added to both the live allowlist and this baseline
// together, in the same change, following the pathCollisionRevealedOrphans
// precedent above. This is not an unreviewed drift — it is D-01/T-187-13's
// explicit requirement that worktree-reap have NO caller anywhere except a
// human typing it. Any documented wrapper, hook, or script invocation would
// itself violate TestWorktreeReapHasNoLifecycleCaller's guarantee that
// destruction stays operator-invoked only. The command being registered but
// uncalled is therefore the intended, permanent end state, not a gap to
// close by inventing a caller.
func TestOrphanAllowlistOnlyShrinks(t *testing.T) {
	live := loadOrphanAllowlist(t, "testdata/orphan_allowlist.json")
	baseline := loadOrphanAllowlist(t, "testdata/orphan_allowlist_baseline.json")

	baselineNames := map[string]bool{}
	for _, e := range baseline {
		baselineNames[e.Name] = true
	}

	var added []string
	for _, e := range live {
		if !baselineNames[e.Name] {
			added = append(added, e.Name)
		}
	}

	if len(added) > 0 {
		sort.Strings(added)
		t.Errorf("%d command(s) were added to the tolerated orphan list without being added to the committed baseline: %s\n"+
			"The allowlist may only shrink. Give the command a real caller, or delete its entry — do not edit the baseline to make this pass.",
			len(added), strings.Join(added, ", "))
	}
}

// TestOrphanAllowlistIsPathKeyed is 172-09's guard against a silent revert to
// leaf-name keys (or a stale entry for a command that no longer exists):
// every entry name in BOTH the live list and the baseline must contain a
// space, carry the "aether " prefix, and resolve through rootCmd.Find to a
// command whose CommandPath() equals the entry name exactly.
func TestOrphanAllowlistIsPathKeyed(t *testing.T) {
	check := func(t *testing.T, path, listPath string) {
		t.Helper()
		entries := loadOrphanAllowlist(t, listPath)
		var bad []string
		for _, e := range entries {
			if !strings.Contains(e.Name, " ") || !strings.HasPrefix(e.Name, "aether ") {
				bad = append(bad, fmt.Sprintf("%q (not a space-containing \"aether \"-prefixed path)", e.Name))
				continue
			}
			target, _, err := rootCmd.Find(strings.Fields(strings.TrimPrefix(e.Name, "aether ")))
			if err != nil || target == nil || target == rootCmd {
				bad = append(bad, fmt.Sprintf("%q (does not resolve via rootCmd.Find)", e.Name))
				continue
			}
			if got := target.CommandPath(); got != e.Name {
				bad = append(bad, fmt.Sprintf("%q (resolves to %q instead)", e.Name, got))
			}
		}
		if len(bad) > 0 {
			sort.Strings(bad)
			t.Errorf("%s has %d entry name(s) that are not real, path-keyed command paths: %s", path, len(bad), strings.Join(bad, ", "))
		}
	}
	t.Run("live", func(t *testing.T) { check(t, "testdata/orphan_allowlist.json", "testdata/orphan_allowlist.json") })
	t.Run("baseline", func(t *testing.T) {
		check(t, "testdata/orphan_allowlist_baseline.json", "testdata/orphan_allowlist_baseline.json")
	})
}

// pathMigrationExpansion is the FROZEN, one-time record of what each of the
// 278 pre-migration leaf names legitimately became once 172-09 switched the
// allowlist to whole-command-path keys. Every key here is a leaf that did
// NOT become the trivial "aether "+leaf: 4 leaves (get, set, registry,
// wisdom) had quietly been standing in for more than one real command and
// collapsed onto several distinct commands; 2 leaves (archive, lifecycle)
// resolved to a single command that lives under a different parent menu
// than the trivial form would predict.
//
// This map describes a one-time historical key-format change made by
// 172-09. A command created after that migration can never legitimately be
// added to it: doing so would raise the tolerance ceiling for a command
// nobody reviewed — the CR-04 defect this rewrite exists to close. Widening
// this map is exactly the widening TestPathMigrationDidNotWidenTolerance
// exists to prevent, so any addition here must be reviewed as carefully as
// an addition to the allowlist itself.
var pathMigrationExpansion = map[string][]string{
	"get":       {"aether colony-depth get", "aether parallel-mode get", "aether plan-granularity get"},
	"set":       {"aether colony-depth set", "aether parallel-mode set", "aether plan-granularity set"},
	"registry":  {"aether export registry", "aether import registry"},
	"wisdom":    {"aether export wisdom", "aether import wisdom"},
	"archive":   {"aether export archive"},
	"lifecycle": {"aether host lifecycle"},
}

// preMigrationSnapshotEntryCount pins the frozen pre-migration snapshot's
// entry count (278). This pin is not redundant with the file itself: it is
// what lets pathMigrationToleratedPaths detect a snapshot that has silently
// grown or shrunk (e.g. a hand-added or hand-removed line) even though the
// SHA-256 pin (preMigrationSnapshotSHA256, below) would already catch a
// single-character edit — the count check is what a reader sees explained
// in the error message, without needing to know what a hash mismatch means.
const preMigrationSnapshotEntryCount = 278

// pathMigrationExpandedPathCount pins the number of full command paths the
// 278 frozen leaves legitimately expand to (272 trivial "aether "+leaf +
// 2 relocated + 10 collapsed = 284). Widening pathMigrationExpansion
// therefore requires editing this second, clearly-labelled number in the
// same review — a two-place diff that cannot be mistaken for routine
// maintenance, and a set-size check that catches a silently vacuous
// (empty or over-grown) tolerated set the same way an empty allowlist would
// otherwise slip past a shrink-only comparison forever.
const pathMigrationExpandedPathCount = 284

// pathMigrationToleratedPaths expands the frozen 278 pre-migration leaves
// into the exact 284 full command paths they legitimately became. It is a
// pure function over its argument (no *testing.T, no file reads) so
// TestPathMigrationRejectsASameLeafNewcomer can feed it the real frozen
// snapshot and then test pathMigrationWideningProblems against synthetic
// baseline input.
//
// Every returned error is an anti-vacuity guard: a tolerance set that
// silently grows or silently empties would make TestPathMigrationDidNotWidenTolerance
// pass forever, which is the same failure shape as the bare-last-word rule
// this function replaces.
func pathMigrationToleratedPaths(pre []orphanAllowlistEntry) (map[string]bool, error) {
	if len(pre) != preMigrationSnapshotEntryCount {
		return nil, fmt.Errorf("frozen pre-migration snapshot has %d entries, want exactly %d (preMigrationSnapshotEntryCount) — "+
			"the frozen anchor must never change size", len(pre), preMigrationSnapshotEntryCount)
	}

	seenLeaves := make(map[string]bool, len(pre))
	tolerated := make(map[string]bool, pathMigrationExpandedPathCount)
	for _, e := range pre {
		if seenLeaves[e.Name] {
			return nil, fmt.Errorf("leaf %q appears twice in the frozen pre-migration snapshot — a duplicate leaf would silently inflate the tolerated path set", e.Name)
		}
		seenLeaves[e.Name] = true
		if paths, ok := pathMigrationExpansion[e.Name]; ok {
			for _, p := range paths {
				tolerated[p] = true
			}
			continue
		}
		tolerated["aether "+e.Name] = true
	}

	for leaf := range pathMigrationExpansion {
		if !seenLeaves[leaf] {
			return nil, fmt.Errorf("pathMigrationExpansion key %q is absent from the frozen pre-migration snapshot — a dead expansion entry would tolerate paths nothing in the frozen snapshot ever earned", leaf)
		}
	}

	if len(tolerated) != pathMigrationExpandedPathCount {
		return nil, fmt.Errorf("tolerated path set has %d entries, want exactly %d (pathMigrationExpandedPathCount) — "+
			"the expansion map and the frozen snapshot have drifted out of sync", len(tolerated), pathMigrationExpandedPathCount)
	}

	return tolerated, nil
}

// pathMigrationWideningProblems returns the full path of every baseline
// entry that is neither in tolerated nor in pathCollisionRevealedOrphans,
// sorted. A pure function over slices and maps with no *testing.T and no
// file reads, so TestPathMigrationRejectsASameLeafNewcomer can feed it
// synthetic input built entirely in the test body.
func pathMigrationWideningProblems(baseline []orphanAllowlistEntry, tolerated map[string]bool) []string {
	var problems []string
	for _, e := range baseline {
		if tolerated[e.Name] {
			continue
		}
		if pathCollisionRevealedOrphans[e.Name] {
			continue
		}
		problems = append(problems, e.Name)
	}
	sort.Strings(problems)
	return problems
}

// TestPathMigrationDidNotWidenTolerance is the invariant that makes "the
// allowlist may only shrink" (D-10) true ACROSS a key-format change. A pure
// set-membership diff (TestOrphanAllowlistOnlyShrinks) cannot do this on its
// own, because every one of the 278 pre-migration entries changed its own
// key text (leaf name -> full path) in this exact migration — a naive diff
// against the old baseline would report all 278 as "added" even though
// nothing was actually widened.
//
// This is the CR-04 fix: every baseline entry's FULL PATH must be one of the
// 284 paths pathMigrationToleratedPaths says the 278 frozen leaves
// legitimately expand to, or one of the 9 explicitly reviewed exemptions in
// pathCollisionRevealedOrphans. Comparing full paths (not bare last words)
// is what stops a brand-new, genuinely unwired command from being carried
// forward just because it happens to share a last word — like "get" or
// "setup" — with an older tolerated command. The converse is asserted too,
// so a reviewed exemption cannot silently linger after its command is
// deleted from the baseline.
func TestPathMigrationDidNotWidenTolerance(t *testing.T) {
	baseline := loadOrphanAllowlist(t, "testdata/orphan_allowlist_baseline.json")
	pre := loadOrphanAllowlist(t, "testdata/orphan_allowlist_baseline_pre_path_migration.json")

	tolerated, err := pathMigrationToleratedPaths(pre)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("tolerated path set size = %d (want %d); baseline size = %d", len(tolerated), pathMigrationExpandedPathCount, len(baseline))

	unreviewedWidening := pathMigrationWideningProblems(baseline, tolerated)
	if len(unreviewedWidening) > 0 {
		t.Errorf("%d command(s) were added to the tolerated list without ever having been tolerated before, by full path: %s\n"+
			"Sharing a last word with an older tolerated command does not carry a new command forward — that is exactly the gap this check exists to close. "+
			"Give the command a real caller, or delete its entry — never edit the frozen snapshot or the expansion map to make this pass.",
			len(unreviewedWidening), strings.Join(unreviewedWidening, ", "))
	}

	baselinePaths := make(map[string]bool, len(baseline))
	for _, e := range baseline {
		baselinePaths[e.Name] = true
	}
	var deadExemptions []string
	for path := range pathCollisionRevealedOrphans {
		if !baselinePaths[path] {
			deadExemptions = append(deadExemptions, path)
		}
	}
	if len(deadExemptions) > 0 {
		sort.Strings(deadExemptions)
		t.Errorf("%d entry/entries in pathCollisionRevealedOrphans no longer appear in the baseline: %s\n"+
			"Remove the dead exemption from pathCollisionRevealedOrphans — its command was already fixed or deleted.",
			len(deadExemptions), strings.Join(deadExemptions, ", "))
	}
}

// TestPathMigrationRejectsASameLeafNewcomer proves by construction that
// CR-04 is closed: a brand-new, genuinely unwired command sharing a last
// word with any of the 278 tolerated frozen leaves is rejected BY FULL
// PATH, even though the old bare-leaf rule would have accepted it. Hermetic
// and table-driven: pathMigrationToleratedPaths is called on the real
// frozen snapshot (which is frozen, so this stays deterministic), and
// pathMigrationWideningProblems is called on synthetic baseline slices
// built entirely in this test's own body. No file is written and no JSON
// is edited.
func TestPathMigrationRejectsASameLeafNewcomer(t *testing.T) {
	pre := loadOrphanAllowlist(t, "testdata/orphan_allowlist_baseline_pre_path_migration.json")
	tolerated, err := pathMigrationToleratedPaths(pre)
	if err != nil {
		t.Fatalf("%v", err)
	}

	assertRejected := func(t *testing.T, path string) {
		t.Helper()
		baseline := []orphanAllowlistEntry{{Name: path, Reason: "test", OwnerPhase: "test"}}
		problems := pathMigrationWideningProblems(baseline, tolerated)
		if len(problems) != 1 || problems[0] != path {
			t.Errorf("pathMigrationWideningProblems did not reject %q by full path; got %v", path, problems)
		}
	}
	assertAccepted := func(t *testing.T, path string) {
		t.Helper()
		baseline := []orphanAllowlistEntry{{Name: path, Reason: "test", OwnerPhase: "test"}}
		problems := pathMigrationWideningProblems(baseline, tolerated)
		if len(problems) != 0 {
			t.Errorf("pathMigrationWideningProblems wrongly rejected %q: %v", path, problems)
		}
	}

	// Rejection rows: a brand-new command sharing a last word with a
	// tolerated frozen leaf must never be carried forward, no matter which
	// kind of frozen leaf (collapsed, relocated, generic-but-trivial, or
	// absent from the snapshot entirely) it borrows from.
	t.Run("rejected", func(t *testing.T) {
		rows := []string{
			"aether newthing get",           // the review's own example; "get" is a collapsed leaf whose three real expansions are named
			"aether colony-depth setup",     // the verifier's live reproduction; "setup" is a generic frozen leaf
			"aether newthing archive",       // a relocated leaf whose single expansion is "aether export archive"
			"aether newthing set",           // collapsed leaf
			"aether newthing registry",      // collapsed leaf
			"aether newthing wisdom",        // collapsed leaf
			"aether newthing lifecycle",     // relocated leaf
			"aether newthing audit-catalog", // GENERALITY: a frozen leaf that is not one of the 14 generic single words and is not in the expansion map
			"aether totally-new-command",    // a leaf absent from the snapshot entirely — the case the old rule already caught, kept so a regression there is visible
		}
		for _, path := range rows {
			path := path
			t.Run(path, func(t *testing.T) { assertRejected(t, path) })
		}
	})

	// Acceptance rows: every path a frozen leaf legitimately expanded to
	// must NOT be reported, including a member of pathCollisionRevealedOrphans.
	// Without these rows a guard that rejects everything would look like a
	// working guard.
	t.Run("accepted", func(t *testing.T) {
		rows := []string{
			"aether colony-depth get", "aether parallel-mode get", "aether plan-granularity get",
			"aether colony-depth set", "aether parallel-mode set", "aether plan-granularity set",
			"aether export registry", "aether import registry",
			"aether export wisdom", "aether import wisdom",
			"aether export archive",
			"aether host lifecycle",
			"aether audit-catalog", // a trivial one-to-one path
			"aether host build",    // pathCollisionRevealedOrphans member
			"aether colonize",      // pathCollisionRevealedOrphans member
		}
		for _, path := range rows {
			path := path
			t.Run(path, func(t *testing.T) { assertAccepted(t, path) })
		}
	})

	// Generality loop: for EVERY leaf in the frozen snapshot (not just the
	// 14 generic single words picked out above), a newcomer command built
	// from that leaf under an unrelated new parent must be rejected. This
	// establishes the rejection holds for all 278 tolerated last words at
	// once — not because a hand-picked list of generic ones was named.
	t.Run("generality_loop_all_frozen_leaves", func(t *testing.T) {
		baseline := make([]orphanAllowlistEntry, 0, len(pre))
		for _, e := range pre {
			baseline = append(baseline, orphanAllowlistEntry{
				Name:       "aether zzz-unwired-newcomer " + e.Name,
				Reason:     "test",
				OwnerPhase: "test",
			})
		}
		problems := pathMigrationWideningProblems(baseline, tolerated)
		t.Logf("generality loop: built %d newcomer paths from all %d frozen leaves; %d reported as widening problems", len(baseline), len(pre), len(problems))
		if len(pre) != preMigrationSnapshotEntryCount {
			t.Fatalf("frozen snapshot has %d entries, want %d — the generality loop would silently walk the wrong number of iterations", len(pre), preMigrationSnapshotEntryCount)
		}
		if len(problems) != len(pre) {
			t.Errorf("generality loop: expected all %d newcomer paths (one per frozen leaf) to be rejected, got %d rejected", len(pre), len(problems))
		}
	})
}

// TestRatchetDetectsASyntheticOrphan is success criterion 1's fixture: a
// throwaway command registered and removed inside this test's own body — it
// must never live at package scope, or it becomes a real permanent orphan
// needing its own baseline entry.
func TestRatchetDetectsASyntheticOrphan(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	const orphanLeaf = "ratchet-selftest-orphan"
	const orphanPath = "aether " + orphanLeaf
	selftest := &cobra.Command{Use: orphanLeaf, Run: func(*cobra.Command, []string) {}}
	rootCmd.AddCommand(selftest)
	defer rootCmd.RemoveCommand(selftest)

	registered := enumerateRegisteredCommands(rootCmd)
	evidence := collectCallerEvidence(t, root, nil)
	orphans := computeOrphanNames(registered, evidence)

	found := false
	for _, o := range orphans {
		if o == orphanPath {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("the ratchet did not detect the synthetic orphan %q, which appears nowhere in the repo as a caller", orphanPath)
	}

	msg := formatOrphanFailureMessage(orphans)
	if !strings.Contains(msg, orphanPath) {
		t.Errorf("the ratchet's own failure-message formatter did not name %q: %s", orphanPath, msg)
	}
}

// TestDeletingACallerMakesTheRatchetNameIt is success criterion 4's
// behavioural half, made automated and hermetic: at runtime it finds a real
// command whose caller evidence comes from exactly one file, proves it is NOT
// an orphan with all files present, then calls collectCallerEvidence again
// with that one file suppressed and proves the ratchet now names it.
func TestDeletingACallerMakesTheRatchetNameIt(t *testing.T) {
	root, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	registered := enumerateRegisteredCommands(rootCmd)
	registeredPaths := map[string]bool{}
	for _, c := range registered {
		registeredPaths[c.Path] = true
	}

	// Cheap per-file pre-scan: which files mention which command paths.
	// Building this directly (rather than by repeatedly calling the
	// expensive, full collectCallerEvidence per candidate file) is what keeps
	// this test fast; the actual behavioural assertion below still goes
	// through the real parameterised function. singleFileCallerNames already
	// returns path-keyed evidence (credit() resolves through rootCmd.Find),
	// so this map is naturally path-keyed too.
	pathToFiles := map[string][]string{}
	files := listCallerCorpusFiles(root)
	for _, f := range files {
		abs := filepath.Join(root, filepath.FromSlash(f))
		for path := range singleFileCallerNames(t, abs) {
			pathToFiles[path] = append(pathToFiles[path], f)
		}
	}

	paths := make([]string, 0, len(pathToFiles))
	for path := range pathToFiles {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	var chosenPath, chosenFile string
	for _, path := range paths {
		if !registeredPaths[path] {
			continue
		}
		fs := pathToFiles[path]
		if len(fs) != 1 {
			continue
		}
		chosenPath = path
		chosenFile = fs[0]
		break
	}

	if chosenPath == "" {
		t.Fatal("no registered command has exactly one caller file across the permitted corpora — nothing to test, which would make this assertion vacuous")
	}

	before := collectCallerEvidence(t, root, nil)
	if !before[chosenPath] {
		t.Fatalf("pre-scan chose %q as single-caller via %q, but the real collectCallerEvidence does not credit it — pre-scan and real scan disagree", chosenPath, chosenFile)
	}
	orphansBefore := computeOrphanNames(registered, before)
	for _, o := range orphansBefore {
		if o == chosenPath {
			t.Fatalf("chosen command %q is already an orphan with all files present — the search picked a bad candidate", chosenPath)
		}
	}

	after := collectCallerEvidence(t, root, map[string]bool{chosenFile: true})
	orphansAfter := computeOrphanNames(registered, after)
	found := false
	for _, o := range orphansAfter {
		if o == chosenPath {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("suppressing %q — the only file that calls %q — did not make the ratchet name it as an orphan", chosenFile, chosenPath)
	}

	msg := formatOrphanFailureMessage(orphansAfter)
	if !strings.Contains(msg, chosenPath) {
		t.Errorf("failure message did not name %q after its only caller (%s) was removed", chosenPath, chosenFile)
	}

	t.Logf("suppressing caller file %q removed the only caller of command %q; the ratchet correctly named it as an orphan", chosenFile, chosenPath)
}

// TestWiringGuardsHaveNoRuntimeEscapeHatch is D-11 and threat T-172-06/
// T-172-33/T-172-34: none of this phase's guard files may contain an
// environment-variable bypass, a test-skip call, or a build-tag exclusion.
// A guard that can be switched off at runtime is not a guard.
//
// Iterates wiringGateGuardFiles (cmd/ci_wiring_gate_test.go) — the single
// shared inventory of every guard file this phase created — rather than a
// second, locally-declared list. A local three-entry list here previously
// omitted cmd/spawn_enforce_test.go and cmd/ci_wiring_gate_test.go itself,
// which meant the one guard that could be t.Skip'd with nothing noticing
// (ci_wiring_gate_test.go) was never scanned for exactly that.
func TestWiringGuardsHaveNoRuntimeEscapeHatch(t *testing.T) {
	// The inventory started at the five guard files phase 172 created, grew
	// to ten across 173-10, and reached eleven when phase 196 plan 02
	// registered the no-length-derivation ratchet. The floor is raised to the
	// live count each time a guard is added, so this is a ratchet in its own
	// right: a future edit that empties or trims the shared inventory must
	// fail loudly here rather than silently narrowing this scan.
	if len(wiringGateGuardFiles) < 11 {
		t.Fatalf("wiringGateGuardFiles has only %d entries — expected at least 11 (the guard files phases 172, 173 and 196 registered); "+
			"a shrunk inventory would silently narrow this escape-hatch scan", len(wiringGateGuardFiles))
	}

	// forbiddenRe covers every spelling of "read an environment variable",
	// "skip this test", or "declare a new flag" this package has needed to
	// reject, not just the first one found: os.Getenv, os.LookupEnv,
	// os.Environ, syscall.Getenv, t.Skip, t.SkipNow, testing.Short, and (WR-01)
	// flag.Bool, flag.String, flag.Int — a flag is a runtime-switchable
	// bypass exactly like an environment-variable read, and the escape-hatch
	// scan previously did not look for one. Each alternative is written with
	// the `\.` escape (matching buildConstraintRe's `go:build` concatenation
	// trick below) precisely so this declaration line does not match its own
	// pattern when the scan below reaches this file.
	forbiddenRe := regexp.MustCompile(`os\.Getenv|os\.LookupEnv|os\.Environ|syscall\.Getenv|t\.Skip|t\.SkipNow|testing\.Short|flag\.Bool|flag\.String|flag\.Int`)

	// exemptedFlagLineCount counts, across every scanned guard file, how many
	// lines were skipped by the single reviewed exemption below. WR-01's
	// -update-orphan-allowlist flag is the one, deliberately reviewed
	// regeneration switch this phase keeps (documented in
	// .aether/docs/orphan-allowlist-policy.md); asserting the count is
	// exactly 1 after the loop means a second flag can neither hide behind
	// the exemption nor silently retire it without this test noticing.
	exemptedFlagLineCount := 0

	for _, f := range wiringGateGuardFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		raw := string(data)

		// A real Go build constraint is itself comment syntax recognised only
		// at column 0 of a line (never indented — an indented mention, like
		// this doc comment's own, is prose, not a directive), so it must be
		// checked against the RAW source with a column-anchored regex rather
		// than the comment-stripped text: stripping "//"-prefixed lines
		// (done below for the other three patterns, so an explanatory
		// comment doesn't trip them) would make a strings.Contains version of
		// this specific check vacuously blind to the thing it exists to
		// catch, and an unanchored one would self-trigger on this very
		// sentence.
		if buildConstraintRe.MatchString(raw) {
			t.Errorf("%s contains a build constraint directive — a guard that can be excluded from a build is not a guard", f)
		}

		stripped := stripGoComments(raw)
		for i, line := range strings.Split(stripped, "\n") {
			if !forbiddenRe.MatchString(line) {
				continue
			}
			// The single reviewed exemption: the line declaring the
			// updateOrphanAllowlist flag itself. See
			// .aether/docs/orphan-allowlist-policy.md for why this one
			// regeneration flag is tolerated — it can rewrite the live
			// allowlist but (per the baselineWriteRe assertion below) can
			// never write the baseline, so it cannot silently widen
			// tolerance. Scoped to lines that already match forbiddenRe (not
			// every line mentioning the identifier) so this file's own later
			// prose about the exemption — including this test's own error
			// message — cannot inflate the count.
			if strings.Contains(line, "updateOrphanAllowlist") {
				exemptedFlagLineCount++
				continue
			}
			t.Errorf("%s:%d contains a runtime escape hatch (an environment-variable read, a test-skip call, or a flag declaration): %s", f, i+1, strings.TrimSpace(line))
		}
	}

	if exemptedFlagLineCount != 1 {
		t.Errorf("expected exactly 1 line exempted as the single reviewed regeneration flag's own declaration, found %d — "+
			"either a second flag is hiding behind the exemption, or the exempted one was deleted without updating this guard", exemptedFlagLineCount)
	}

	// The -update-orphan-allowlist flag must never be able to write the
	// baseline (D-11's "changing the baseline is a visible, on-the-record
	// edit to a checked-in file, not a runtime bypass").
	data, err := os.ReadFile("subcommand_reachability_ratchet_test.go")
	if err != nil {
		t.Fatalf("read own source: %v", err)
	}
	stripped := stripGoComments(string(data))
	baselineWriteRe := regexp.MustCompile(`(?s)(os\.WriteFile|os\.Create)\([^)]*orphan_allowlist_baseline\.json`)
	if baselineWriteRe.MatchString(stripped) {
		t.Error("subcommand_reachability_ratchet_test.go writes orphan_allowlist_baseline.json from an os.WriteFile/os.Create call — the update flag must never regenerate the baseline (D-11)")
	}
}
