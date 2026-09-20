---
phase: 73-rich-init-research
reviewed: 2026-04-29T00:00:00Z
depth: standard
files_reviewed: 2
files_reviewed_list:
  - cmd/init_research.go
  - cmd/init_research_test.go
findings:
  critical: 0
  warning: 4
  info: 2
  total: 6
status: issues_found
---

# Phase 73: Code Review Report

**Reviewed:** 2026-04-29
**Depth:** standard
**Files Reviewed:** 2
**Status:** issues_found

## Summary

Reviewed `cmd/init_research.go` (1969 lines) and `cmd/init_research_test.go` (1801 lines). This implements a rich init-research command that scans a target directory for project metadata: detected languages, frameworks, governance tooling, dependency trees, directory classification, pheromone suggestions, git history, and a colony charter. The implementation is thorough with ~25 parsers covering multiple ecosystems. Tests are comprehensive with good coverage of happy paths, edge cases, and backward compatibility.

No critical security vulnerabilities found. Several warnings around correctness edge cases and one UTF-8 safety issue.

## Warnings

### WR-01: UTF-8-unsafe README truncation

**File:** `cmd/init_research.go:1875-1876`
**Issue:** The README summary truncation uses byte slicing (`data[:500]`) which can split multi-byte UTF-8 sequences. For projects with non-ASCII content (CJK, emoji, accented characters), this produces invalid UTF-8 in the output. The `readmeSummary` field would contain corrupted text.

**Fix:**
```go
// Replace byte slicing with rune-aware truncation
if len(data) > 500 {
    runes := bytes.Runes(data)
    if len(runes) > 500 {
        data = []byte(string(runes[:500]))
    }
}
```

### WR-02: `.eslintrc` (no extension) silently drops YAML/JS config

**File:** `cmd/init_research.go:370-398`
**Issue:** When parsing `.eslintrc` (no file extension), the code unconditionally treats the content as JSON via `gjson.ParseBytes`. ESLint allows `.eslintrc` files in JSON, YAML, or JS format. If the file is YAML, `gjson.ParseBytes` silently returns empty results (gjson is lenient with invalid JSON). The function returns a non-nil `governanceDetail` indicating ESLint was found, but with empty `Rules` and `Extends`, making it appear as if ESLint has no configuration when it actually does.

**Fix:** Add YAML fallback parsing when gjson finds no rules in `.eslintrc`:
```go
if strings.HasSuffix(name, ".json") || name == ".eslintrc" {
    parsed := gjson.ParseBytes(data)
    hasRules := false
    if rules := parsed.Get("rules"); rules.IsObject() {
        hasRules = true
        // ... existing parsing ...
    }
    // Fallback: try YAML for bare .eslintrc
    if !hasRules && name == ".eslintrc" {
        var raw map[string]interface{}
        if err := yaml.Unmarshal(data, &raw); err == nil {
            if rules, ok := raw["rules"].(map[string]interface{}); ok {
                detail.Rules = rules
            }
            // ... parse extends similarly ...
        }
    }
    return detail
}
```

### WR-03: `parsePytestDeep` processes comments as config entries

**File:** `cmd/init_research.go:616-622`
**Issue:** For `pytest.ini` files, the parser processes any line containing `=` as a config key-value pair, including comment lines. A line like `# see = documentation` would be parsed as `{"see": "documentation"}`. The `inPytestSection` guard on line 607 correctly handles section-scoped parsing for `setup.cfg`, but the unconditional `pytest.ini` path on line 616 has no comment filtering.

**Fix:** Skip comment lines before processing:
```go
if name == "pytest.ini" && strings.Contains(line, "=") && !strings.HasPrefix(line, "#") {
    parts := strings.SplitN(line, "=", 2)
    if len(parts) == 2 {
        key := strings.TrimSpace(parts[0])
        val := strings.TrimSpace(parts[1])
        detail.Config[key] = val
    }
}
```

### WR-04: `classifyDirectory` misclassifies lib-only projects as `standard_app`

**File:** `cmd/init_research.go:333-341`
**Issue:** The `appDirs` list includes `"lib"`, so a project with only a `lib/` directory (and no `src/`, `cmd/`, or `app/`) is classified as `standard_app`. This prevents the `library` classification (lines 343-351) from ever being reached when `lib/` exists. For example, a Go library with `lib/` + `main.go` in root is classified as `standard_app` rather than `library`, which could mislead colony context.

**Fix:** Remove `"lib"` from `appDirs` so that a lone `lib/` directory does not trigger the `standard_app` classification:
```go
appDirs := []string{"src", "cmd", "app"}
```

Alternatively, if `lib/` should remain a standard_app signal, add `"lib"` to the library exclusion check on line 344 for consistency.

## Info

### IN-01: Unused function `hasSuffix`

**File:** `cmd/init_research.go:1952-1959`
**Issue:** The `hasSuffix` function is defined but never called within this file or any other file in the codebase (confirmed via grep). This is dead code.

**Fix:** Remove the function entirely.

### IN-02: Variable shadows type name `techStackDetail`

**File:** `cmd/init_research.go:1913`
**Issue:** The local variable `techStackDetail := parseDependencyFiles(target)` shadows the struct type `techStackDetail` defined on line 63. While valid Go, it reduces readability and makes it harder to distinguish the type from the variable in the same scope.

**Fix:** Rename the local variable to `techStackDetails` (plural):
```go
techStackDetails := parseDependencyFiles(target)
```

---

_Reviewed: 2026-04-29_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
