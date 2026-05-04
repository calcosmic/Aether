// Package codegraph provides a lightweight, language-agnostic file dependency
// scanner. It walks a repository's source tree, parses import/require/include
// statements with regex patterns, and produces a queryable dependency graph.
//
// Design: file-level imports only, no AST, no external dependencies.
// The 80% case is a worker needing to know "which files should I read first?"
package codegraph

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// FileNode represents a source file in the scanned repository.
type FileNode struct {
	Path     string `json:"path"`
	Language string `json:"language"`
}

// DepEdge represents a directed dependency from one file to another.
type DepEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // "import", "require", "include", etc.
}

// CodeGraph holds the scanned file dependency graph.
type CodeGraph struct {
	Files    []FileNode `json:"files"`
	Edges    []DepEdge  `json:"edges"`
	Langs    []string   `json:"languages"`
	FileCount int       `json:"file_count"`
	EdgeCount int       `json:"edge_count"`
}

// Stats returns a human-readable summary of the scan.
type Stats struct {
	FilesScanned int     `json:"files_scanned"`
	EdgesFound   int     `json:"edges_found"`
	Languages    []string `json:"languages"`
	SkippedDirs  int     `json:"skipped_dirs"`
}

// languageDetectors maps file extensions to language names.
var languageDetectors = map[string]string{
	".go":   "go",
	".ts":   "typescript",
	".tsx":  "typescript",
	".js":   "javascript",
	".jsx":  "javascript",
	".mjs":  "javascript",
	".cjs":  "javascript",
	".py":   "python",
	".rb":   "ruby",
	".java": "java",
	".rs":   "rust",
	".c":    "c",
	".cpp":  "cpp",
	".cc":   "cpp",
	".cxx":  "cpp",
	".h":    "c",
	".hpp":  "cpp",
}

// dirsToSkip are directories never scanned.
var dirsToSkip = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, ".aether": true,
	"dist": true, "build": true, "out": true, "__pycache__": true,
	".next": true, ".nuxt": true, "target": true, "bin": true,
	".cache": true, ".terraform": true,
}

// importParser extracts file dependencies from source content.
// Returns resolved file paths (relative to repo root).
type importParser func(relPath, content, root string) []string

// parsers maps languages to their import parsers.
var parsers map[string]importParser

func init() {
	parsers = map[string]importParser{
		"go":         parseGoImports,
		"typescript": parseTSImports,
		"javascript": parseTSImports,
		"python":     parsePythonImports,
		"ruby":       parseRubyImports,
		"java":       parseJavaImports,
		"rust":       parseRustImports,
		"c":          parseCImports,
		"cpp":        parseCImports,
	}
}

// --- Go ---

var (
	goImportSingle = regexp.MustCompile(`(?m)^\s*import\s+"([^"]+)"`)
	goImportBlock  = regexp.MustCompile(`"([^"]+)"`)
	goImportGroup  = regexp.MustCompile(`(?s)import\s*\((.*?)\)`)
)

func parseGoImports(relPath, content, root string) []string {
	var deps []string
	seen := map[string]bool{}

	// Single imports
	for _, m := range goImportSingle.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			dep := resolveGoImport(m[1], relPath, root)
			if dep != "" && !seen[dep] {
				deps = append(deps, dep)
				seen[dep] = true
			}
		}
	}

	// Group imports
	for _, block := range goImportGroup.FindAllStringSubmatch(content, -1) {
		if len(block) < 2 {
			continue
		}
		for _, m := range goImportBlock.FindAllStringSubmatch(block[1], -1) {
			if len(m) > 1 {
				dep := resolveGoImport(m[1], relPath, root)
				if dep != "" && !seen[dep] {
					deps = append(deps, dep)
					seen[dep] = true
				}
			}
		}
	}

	return deps
}

func resolveGoImport(importPath, fromFile, root string) string {
	// Check for relative import first
	if strings.HasPrefix(importPath, ".") || strings.HasPrefix(importPath, "./") {
		dir := filepath.Dir(fromFile)
		resolved := filepath.Join(dir, importPath)
		cleaned := filepath.Clean(resolved)
		if fileExists(root, cleaned) {
			info, err := os.Stat(filepath.Join(root, cleaned))
			if err == nil && !info.IsDir() {
				return cleaned
			}
			return cleaned
		}
		if fileExists(root, cleaned+".go") {
			return cleaned + ".go"
		}
		return ""
	}

	// Module-relative import: resolve using go.mod module prefix
	moduleName := readGoModule(root)
	if moduleName == "" {
		return ""
	}
	if !strings.HasPrefix(importPath, moduleName+"/") {
		return "" // external package
	}
	// Strip module prefix to get local path
	localPath := strings.TrimPrefix(importPath, moduleName+"/")
	if fileExists(root, localPath) {
		info, err := os.Stat(filepath.Join(root, localPath))
		if err == nil && !info.IsDir() {
			return localPath
		}
		return localPath // package directory
	}
	return ""
}

// readGoModule reads the module name from go.mod. Cached per scan.
var goModuleCache map[string]string

func readGoModule(root string) string {
	if goModuleCache == nil {
		goModuleCache = make(map[string]string)
	}
	if name, ok := goModuleCache[root]; ok {
		return name
	}
	modPath := filepath.Join(root, "go.mod")
	data, err := os.ReadFile(modPath)
	if err != nil {
		goModuleCache[root] = ""
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "module "))
			goModuleCache[root] = name
			return name
		}
	}
	goModuleCache[root] = ""
	return ""
}

// --- TypeScript / JavaScript ---

var (
	tsStaticImport  = regexp.MustCompile(`(?:import|export)\s+(?:.+?\s+from\s+|{[^}]+}\s+from\s+|type\s+.{1,80}?\s+from\s+)['"]([^'"]+)['"]`)
	tsDynamicImport = regexp.MustCompile(`import\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	tsRequire       = regexp.MustCompile(`require\s*\(\s*['"]([^'"]+)['"]\s*\)`)
)

func parseTSImports(relPath, content, root string) []string {
	var deps []string
	seen := map[string]bool{}

	patterns := []*regexp.Regexp{tsStaticImport, tsDynamicImport, tsRequire}
	for _, pat := range patterns {
		for _, m := range pat.FindAllStringSubmatch(content, -1) {
			if len(m) > 1 && strings.HasPrefix(m[1], ".") {
				dep := resolveRelativeImport(m[1], relPath, root, tsExtensions)
				if dep != "" && !seen[dep] {
					deps = append(deps, dep)
					seen[dep] = true
				}
			}
		}
	}

	return deps
}

var tsExtensions = []string{"", ".ts", ".tsx", ".js", ".jsx", ".mjs", "/index.ts", "/index.tsx", "/index.js"}

// --- Python ---

var (
	pyImport     = regexp.MustCompile(`^\s*import\s+(\S+)`)
	pyFromImport = regexp.MustCompile(`^\s*from\s+(\S+)\s+import`)
)

func parsePythonImports(relPath, content, root string) []string {
	var deps []string
	seen := map[string]bool{}
	dir := filepath.Dir(relPath)

	for _, line := range strings.Split(content, "\n") {
		for _, m := range pyFromImport.FindAllStringSubmatch(line, -1) {
			if len(m) > 1 {
				dep := resolvePythonModule(m[1], dir, root)
				if dep != "" && !seen[dep] {
					deps = append(deps, dep)
					seen[dep] = true
				}
			}
		}
		for _, m := range pyImport.FindAllStringSubmatch(line, -1) {
			if len(m) > 1 {
				dep := resolvePythonModule(m[1], dir, root)
				if dep != "" && !seen[dep] {
					deps = append(deps, dep)
					seen[dep] = true
				}
			}
		}
	}

	return deps
}

func resolvePythonModule(mod, dir, root string) string {
	// Convert dots to paths
	parts := strings.Split(mod, ".")
	relPath := filepath.Join(parts...)

	// Try as package (directory with __init__.py)
	pkgDir := relPath
	if fileExists(root, filepath.Join(pkgDir, "__init__.py")) {
		return pkgDir
	}
	// Try as file
	for _, ext := range []string{".py", ".pyx"} {
		if fileExists(root, relPath+ext) {
			return relPath + ext
		}
	}
	// Try relative to the importing file's directory
	localPath := filepath.Join(dir, relPath)
	if fileExists(root, filepath.Join(localPath, "__init__.py")) {
		return localPath
	}
	for _, ext := range []string{".py"} {
		if fileExists(root, localPath+ext) {
			return localPath + ext
		}
	}
	return ""
}

// --- Ruby ---

var (
	rubyRequire         = regexp.MustCompile(`require\s+['"]([^'"]+)['"]`)
	rubyRequireRelative = regexp.MustCompile(`require_relative\s+['"]([^'"]+)['"]`)
)

func parseRubyImports(relPath, content, root string) []string {
	var deps []string
	seen := map[string]bool{}
	dir := filepath.Dir(relPath)

	for _, m := range rubyRequireRelative.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			dep := filepath.Join(dir, m[1])
			if !strings.HasSuffix(dep, ".rb") {
				dep += ".rb"
			}
			dep = filepath.Clean(dep)
			if fileExists(root, dep) && !seen[dep] {
				deps = append(deps, dep)
				seen[dep] = true
			}
		}
	}

	for _, m := range rubyRequire.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			if strings.HasPrefix(m[1], ".") {
				dep := filepath.Join(dir, m[1])
				if !strings.HasSuffix(dep, ".rb") {
					dep += ".rb"
				}
				dep = filepath.Clean(dep)
				if fileExists(root, dep) && !seen[dep] {
					deps = append(deps, dep)
					seen[dep] = true
				}
			}
			// Absolute require paths are gems — skip
		}
	}

	return deps
}

// --- Java ---

var javaImport = regexp.MustCompile(`(?m)^\s*import\s+(?:static\s+)?([^;]+);`)

func parseJavaImports(relPath, content, root string) []string {
	var deps []string
	seen := map[string]bool{}

	// Map this file's package to a directory
	thisPkg := javaPackageFromPath(relPath)

	for _, m := range javaImport.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			// Convert com.example.Foo -> com/example/Foo.java
			importPath := strings.ReplaceAll(m[1], ".", "/")
			// Strip wildcard
			importPath = strings.TrimSuffix(importPath, "/*")

			// Try to resolve within the repo
			for _, srcDir := range []string{"src/main/java", "src/test/java", "src"} {
				fullPath := filepath.Join(srcDir, importPath+".java")
				if fileExists(root, fullPath) && !seen[fullPath] {
					deps = append(deps, fullPath)
					seen[fullPath] = true
				}
			}
		}
	}

	_ = thisPkg
	return deps
}

func javaPackageFromPath(relPath string) string {
	dir := filepath.Dir(relPath)
	return strings.ReplaceAll(dir, "/", ".")
}

// --- Rust ---

var (
	rustUse  = regexp.MustCompile(`(?m)^\s*use\s+([^;]+);`)
	rustMod  = regexp.MustCompile(`(?m)^\s*mod\s+(\w+)`)
)

func parseRustImports(relPath, content, root string) []string {
	var deps []string
	seen := map[string]bool{}
	dir := filepath.Dir(relPath)

	for _, m := range rustMod.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			// Try mod.rs in subdirectory
			modDir := filepath.Join(dir, m[1])
			modFile := filepath.Join(dir, m[1]+".rs")
			if fileExists(root, filepath.Join(modDir, "mod.rs")) && !seen[modDir] {
				deps = append(deps, filepath.Join(modDir, "mod.rs"))
				seen[modDir] = true
			} else if fileExists(root, modFile) && !seen[modFile] {
				deps = append(deps, modFile)
				seen[modFile] = true
			}
		}
	}

	for _, m := range rustUse.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			path := strings.TrimPrefix(m[1], "crate::")
			path = strings.TrimPrefix(path, "super::")
			pathParts := strings.Split(path, "::")
			if len(pathParts) >= 1 {
				// Lowercase the last segment (Rust convention: mod MyMod -> my_mod.rs)
				last := strings.ToLower(pathParts[len(pathParts)-1])
				parts := make([]string, len(pathParts))
				copy(parts, pathParts)
				parts[len(parts)-1] = last

				// Try as file path
				relFile := filepath.Join(parts...) + ".rs"
				if fileExists(root, relFile) && !seen[relFile] {
					deps = append(deps, relFile)
					seen[relFile] = true
				}
				// Try as directory with mod.rs
				relDir := filepath.Join(pathParts...)
				modPath := filepath.Join(relDir, "mod.rs")
				if fileExists(root, modPath) && !seen[modPath] {
					deps = append(deps, modPath)
					seen[modPath] = true
				}
			}
		}
	}

	return deps
}

// --- C/C++ ---

var cInclude = regexp.MustCompile(`(?m)^\s*#\s*include\s+"([^"]+)"`)

func parseCImports(relPath, content, root string) []string {
	var deps []string
	seen := map[string]bool{}
	dir := filepath.Dir(relPath)

	for _, m := range cInclude.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			includePath := m[1]
			// Try relative to current file
			relFile := filepath.Join(dir, includePath)
			if fileExists(root, relFile) && !seen[relFile] {
				deps = append(deps, relFile)
				seen[relFile] = true
				continue
			}
			// Try from root
			if fileExists(root, includePath) && !seen[includePath] {
				deps = append(deps, includePath)
				seen[includePath] = true
			}
		}
	}

	return deps
}

// --- Shared helpers ---

func resolveRelativeImport(importPath, fromFile, root string, extensions []string) string {
	dir := filepath.Dir(fromFile)
	resolved := filepath.Join(dir, importPath)

	for _, ext := range extensions {
		candidate := filepath.Clean(resolved + ext)
		if fileExists(root, candidate) {
			return candidate
		}
	}
	return ""
}

func fileExists(root, relPath string) bool {
	_, err := os.Stat(filepath.Join(root, relPath))
	return err == nil
}

// Scan walks the repo root and builds a dependency graph.
// langs filters which languages to parse; if empty, all detected languages are parsed.
func Scan(root string, langs []string) (*CodeGraph, *Stats, error) {
	// Reset caches for fresh scan
	goModuleCache = nil

	graph := &CodeGraph{}
	stats := &Stats{}
	seenLangs := map[string]bool{}
	langFilter := map[string]bool{}
	for _, l := range langs {
		langFilter[l] = true
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if d.IsDir() {
			if dirsToSkip[d.Name()] {
				stats.SkippedDirs++
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(d.Name())
		lang, ok := languageDetectors[ext]
		if !ok {
			return nil
		}

		// Skip if language not in filter
		if len(langFilter) > 0 && !langFilter[lang] {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		graph.Files = append(graph.Files, FileNode{Path: rel, Language: lang})
		stats.FilesScanned++

		if !seenLangs[lang] {
			seenLangs[lang] = true
		}

		parser, ok := parsers[lang]
		if !ok {
			return nil
		}

		deps := parser(rel, string(content), root)
		for _, dep := range deps {
			graph.Edges = append(graph.Edges, DepEdge{
				Source: rel,
				Target: dep,
				Type:   "import",
			})
			stats.EdgesFound++
		}

		return nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("scan failed: %w", err)
	}

	for lang := range seenLangs {
		graph.Langs = append(graph.Langs, lang)
	}
	sort.Strings(graph.Langs)
	graph.FileCount = len(graph.Files)
	graph.EdgeCount = len(graph.Edges)
	stats.Languages = graph.Langs

	return graph, stats, nil
}

// RelatedFiles returns files connected to the given file within maxDepth hops.
func (g *CodeGraph) RelatedFiles(filePath string, maxDepth int) []string {
	// Build adjacency list
	adj := make(map[string][]string)
	for _, e := range g.Edges {
		adj[e.Source] = append(adj[e.Source], e.Target)
		adj[e.Target] = append(adj[e.Target], e.Source) // bidirectional for discovery
	}

	visited := map[string]bool{filePath: true}
	queue := []string{filePath}
	var result []string

	for depth := 0; depth < maxDepth && len(queue) > 0; depth++ {
		var nextQueue []string
		for _, node := range queue {
			for _, neighbor := range adj[node] {
				if !visited[neighbor] {
					visited[neighbor] = true
					result = append(result, neighbor)
					nextQueue = append(nextQueue, neighbor)
				}
			}
		}
		queue = nextQueue
	}

	return result
}

// FilesForTask finds related files for a set of task target files.
// Returns a deduplicated, sorted list of files within 2 hops of any target.
func (g *CodeGraph) FilesForTask(targetFiles []string) []string {
	allDeps := map[string]bool{}
	for _, f := range targetFiles {
		allDeps[f] = true
		for _, dep := range g.RelatedFiles(f, 2) {
			allDeps[dep] = true
		}
	}

	var result []string
	for f := range allDeps {
		result = append(result, f)
	}
	sort.Strings(result)
	return result
}

// Save writes the graph as JSON to the given path.
func (g *CodeGraph) Save(path string) error {
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal codegraph: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// Load reads a graph from a JSON file.
func Load(path string) (*CodeGraph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read codegraph: %w", err)
	}
	var g CodeGraph
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("parse codegraph: %w", err)
	}
	return &g, nil
}

// FormatRelatedFiles produces a compact summary of related files for worker context.
func (g *CodeGraph) FormatRelatedFiles(targetFiles []string, maxChars int) string {
	if len(g.Edges) == 0 || len(targetFiles) == 0 {
		return ""
	}

	// Build adjacency for direct imports only (1 hop)
	directDeps := make(map[string][]string)
	for _, e := range g.Edges {
		directDeps[e.Source] = append(directDeps[e.Source], e.Target)
	}

	var sb strings.Builder
	sb.WriteString("## Codebase Dependencies\n\n")

	for _, f := range targetFiles {
		deps := directDeps[f]
		if len(deps) == 0 {
			continue
		}
		line := fmt.Sprintf("- %s ← imports: %s\n", f, strings.Join(deps, ", "))
		if sb.Len()+len(line) > maxChars {
			break
		}
		sb.WriteString(line)
	}

	if sb.Len() <= len("## Codebase Dependencies\n\n") {
		return ""
	}
	return sb.String()
}
