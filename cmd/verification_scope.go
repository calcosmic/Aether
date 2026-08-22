package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

const (
	// verificationScopeTargeted means the tests command was rewritten to
	// cover only the packages this phase's changed files touched.
	verificationScopeTargeted = "targeted"
	// verificationScopeFull means the tests command ran exactly as
	// configured, over the whole project -- either because this phase
	// requires it (the plan's final phase) or because a smaller scope could
	// not be honestly derived.
	verificationScopeFull = "full"
	// verificationScopeNone means no tests command resolved for this
	// project at all (D-01) -- distinct from "full", which means a real
	// command ran over everything.
	verificationScopeNone = "none"
)

// verificationScope records how much of the project's tests this
// verification pass actually ran, and why -- the plain-English data the
// closing card (Phases 197/198) reads to say, for example, "targeted: 3
// packages" or "full suite" (D-01, D-07). Only the tests command is ever
// scoped -- see deriveVerificationScope's doc comment for why build, type
// and lint checks are left alone.
type verificationScope struct {
	// Mode is one of verificationScopeTargeted, verificationScopeFull, or
	// verificationScopeNone.
	Mode string `json:"mode"`
	// Packages lists the package paths a targeted run covered. Empty for
	// full or none.
	Packages []string `json:"packages,omitempty"`
	// PackageCount is len(Packages), carried as its own field so the closing
	// card never has to count a slice just to say "targeted: 3 packages".
	PackageCount int `json:"package_count,omitempty"`
	// Reason is a plain-English clause explaining why this mode was chosen.
	// It must be readable by someone who has never opened this repository
	// (CLAUDE.md's communication rule) -- no repo-invented vocabulary.
	Reason string `json:"reason"`
}

// deriveVerificationScope decides how much of the project's tests this
// verification pass runs (D-07): scoped to the packages the phase's claimed
// changed files actually touched, or the full suite whenever that scope
// cannot be honestly derived -- a phase reporting no changed files, files
// that don't map to a runnable package, an ecosystem with no scoped test
// runner, or the plan's final phase, which always runs everything regardless
// of what could have been scoped, as the last honest check before the plan
// closes. A project with no tests command at all resolves to mode "none"
// rather than "full" -- there is nothing to scope or to run.
//
// Only the tests command is ever rewritten here. Build, type and lint
// commands run exactly as configured in every mode: narrowing a compiler or
// a linter changes what it can see (a type error in a file outside the
// scoped set would go unreported), which is a materially different decision
// this task deliberately does not make.
//
// commands is returned unchanged except for a possibly-rewritten Test field;
// every other field passes through verbatim.
func deriveVerificationScope(root string, phase colony.Phase, isFinalPhase bool, claims codexBuildClaims, commands codexVerificationCommands) (verificationScope, codexVerificationCommands) {
	testCommand := strings.TrimSpace(commands.Test)
	if testCommand == "" {
		return verificationScope{Mode: verificationScopeNone, Reason: "there are no tests to run in this project"}, commands
	}

	full := func(reason string) (verificationScope, codexVerificationCommands) {
		return verificationScope{Mode: verificationScopeFull, Reason: reason}, commands
	}

	if isFinalPhase {
		return full("this is the last phase of the plan, so the full suite runs as the final check before the plan closes")
	}

	changed := changedFilesFromBuildClaims(claims)
	if len(changed) == 0 {
		return full("no changed files were reported for this phase, so a smaller scope could not be honestly derived")
	}

	packages := goPackagePathsForChangedFiles(changed)
	if len(packages) == 0 {
		return full("the files this phase changed don't map to a runnable part of the test suite, so the full run covers it")
	}
	if containsString(packages, "./...") {
		// A changed file directly at the repository root maps to the same
		// "./..." pattern the full run already uses (goPackagePathsForChangedFiles),
		// so the derived scope is not narrower than a full run at all --
		// report it honestly as full rather than as "targeted to 1
		// package(s)", which would understate how much of the suite
		// actually ran (IN-01, 193-REVIEW.md).
		return full("a file at the top level of the project was changed, so there's no smaller area to limit the run to -- the full run covers it")
	}

	scopedTest, ok := scopedGoTestCommand(testCommand, packages)
	if !ok {
		return full("this project's test command has no way to run only part of the suite, so the full run covers it")
	}

	scopedCommands := commands
	scopedCommands.Test = scopedTest
	return verificationScope{
		Mode:         verificationScopeTargeted,
		Packages:     packages,
		PackageCount: len(packages),
		Reason:       fmt.Sprintf("targeted to the %d package(s) this phase's changed files touched", len(packages)),
	}, scopedCommands
}

// loadRawBuildClaimsForScope loads the phase's build claims file (the same
// file verifyCodexBuildClaims reads) purely to learn which files changed --
// deriveVerificationScope's only use for it. A missing or unreadable claims
// file returns a zero value, which deriveVerificationScope treats as "no
// changed files reported" and falls back to the full run -- the same
// conservative default verifyCodexBuildClaims itself applies when it cannot
// read the file.
func loadRawBuildClaimsForScope(manifest codexContinueManifest) codexBuildClaims {
	if store == nil {
		return codexBuildClaims{}
	}
	claimsRel := "last-build-claims.json"
	if manifest.Present && strings.TrimSpace(manifest.Data.ClaimsPath) != "" {
		claimsRel = strings.TrimPrefix(strings.TrimSpace(manifest.Data.ClaimsPath), ".aether/data/")
	}
	var claims codexBuildClaims
	if err := store.LoadJSON(claimsRel, &claims); err != nil {
		return codexBuildClaims{}
	}
	return claims
}

// changedFilesFromBuildClaims normalizes the union of everything a phase's
// build claims say changed (created, modified, and newly written tests) into
// repository-relative, slash-separated paths -- reusing the same path
// normalization normalizeCriterionArtifactPath applies elsewhere, so a path
// this function accepts is a path the rest of the codebase already trusts.
func changedFilesFromBuildClaims(claims codexBuildClaims) []string {
	raw := append(append(append([]string{}, claims.FilesCreated...), claims.FilesModified...), claims.TestsWritten...)
	changed := make([]string, 0, len(raw))
	for _, path := range raw {
		normalized, err := normalizeCriterionArtifactPath(path)
		if err != nil {
			continue
		}
		changed = append(changed, normalized)
	}
	return uniqueSortedStrings(changed)
}

// goPackagePathsForChangedFiles maps changed files to the Go package
// directories they belong to, expressed as recursive "./dir/..." test
// patterns. Only .go files carry a derivable Go package -- a changed
// markdown or JSON file names no package a `go test` invocation can run, so
// it contributes nothing here (an all-non-.go changed-file set is exactly
// the "no derivable package" case that falls back to full).
func goPackagePathsForChangedFiles(changed []string) []string {
	seen := map[string]bool{}
	var packages []string
	for _, rel := range changed {
		if !strings.HasSuffix(rel, ".go") {
			continue
		}
		dir := filepath.ToSlash(filepath.Dir(filepath.FromSlash(rel)))
		pattern := "./..."
		if dir != "." {
			pattern = "./" + dir + "/..."
		}
		if seen[pattern] {
			continue
		}
		seen[pattern] = true
		packages = append(packages, pattern)
	}
	sort.Strings(packages)
	return packages
}

// scopedGoTestCommand rewrites a configured `go test` command's package
// pattern to the derived scoped set, keeping every other token (flags,
// timeouts) unchanged. It only rewrites the conventional "./..." pattern --
// a hand-authored command naming something else (a specific package, a build
// tag) is left alone and reported as having no scoped runner (D-07's
// explicit fallback), rather than guessed at.
func scopedGoTestCommand(testCommand string, packages []string) (string, bool) {
	fields := strings.Fields(testCommand)
	if len(fields) < 3 || fields[0] != "go" || fields[1] != "test" {
		return "", false
	}
	replaced := false
	rewritten := make([]string, 0, len(fields)+len(packages))
	rewritten = append(rewritten, fields[0], fields[1])
	for _, field := range fields[2:] {
		if !replaced && field == "./..." {
			rewritten = append(rewritten, packages...)
			replaced = true
			continue
		}
		rewritten = append(rewritten, field)
	}
	if !replaced {
		return "", false
	}
	return strings.Join(rewritten, " "), true
}
