package codegraph

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// helper creates a temp dir with files and returns its path.
func createTestRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestGoImports(t *testing.T) {
	repo := createTestRepo(t, map[string]string{
		"cmd/main.go": `package main

import (
	"fmt"
	"myrepo/internal/handler"
	"myrepo/pkg/utils"
)

func main() {
	handler.Serve()
}`,
		"internal/handler/handler.go": `package handler

import "myrepo/pkg/utils"

func Serve() {}`,
		"pkg/utils/utils.go": `package utils

func Helper() {}`,
		"go.mod":             "module myrepo\n",
	})

	got := parseGoImports("cmd/main.go", mustRead(t, repo, "cmd/main.go"), repo)
	if len(got) == 0 {
		// Relative imports only — since these are module paths, they won't resolve
		// unless we handle module prefix resolution. For now, this is expected.
		t.Logf("Go module imports don't resolve as relative paths (expected)")
	}

	// Test with actual relative import
	repo2 := createTestRepo(t, map[string]string{
		"main.go":       `package main` + "\n" + `import "./lib"`,
		"lib/lib.go":    `package lib`,
	})
	got2 := parseGoImports("main.go", mustRead(t, repo2, "main.go"), repo2)
	if len(got2) != 1 || got2[0] != filepath.Clean("lib") {
		t.Errorf("relative Go import: got %v, want [lib]", got2)
	}
}

func TestTSImports(t *testing.T) {
	repo := createTestRepo(t, map[string]string{
		"src/app.ts": `import { User } from './types';
import { serve } from "./server";
const dyn = import('./dynamic');
const req = require("./legacy");
export { User } from './types';
`,
		"src/types.ts":   `export interface User { name: string }`,
		"src/server.ts":  `export function serve() {}`,
		"src/dynamic.ts": `export default {}`,
		"src/legacy.js":  `module.exports = {}`,
	})

	got := parseTSImports("src/app.ts", mustRead(t, repo, "src/app.ts"), repo)

	// Should find at least types.ts and server.ts
	found := map[string]bool{}
	for _, dep := range got {
		found[dep] = true
	}
	if !found[filepath.Clean("src/types.ts")] {
		t.Errorf("missing src/types.ts in deps: %v", got)
	}
	if !found[filepath.Clean("src/server.ts")] {
		t.Errorf("missing src/server.ts in deps: %v", got)
	}
	if !found[filepath.Clean("src/dynamic.ts")] {
		t.Errorf("missing src/dynamic.ts in deps: %v", got)
	}
	if !found[filepath.Clean("src/legacy.js")] {
		t.Errorf("missing src/legacy.js in deps: %v", got)
	}
}

func TestPythonImports(t *testing.T) {
	repo := createTestRepo(t, map[string]string{
		"app.py": `import utils
from models import User
from . import config
from .services import auth`,
		"utils.py":              `def helper(): pass`,
		"models.py":             `class User: pass`,
		"config.py":             `DEBUG = True`,
		"services/__init__.py":  ``,
		"services/auth.py":      `def login(): pass`,
	})

	got := parsePythonImports("app.py", mustRead(t, repo, "app.py"), repo)

	found := map[string]bool{}
	for _, dep := range got {
		found[dep] = true
	}
	if !found["utils.py"] {
		t.Errorf("missing utils.py: %v", got)
	}
	if !found["models.py"] {
		t.Errorf("missing models.py: %v", got)
	}
	if !found["services"] {
		t.Errorf("missing services package: %v", got)
	}
}

func TestRubyImports(t *testing.T) {
	repo := createTestRepo(t, map[string]string{
		"app.rb": `require_relative 'lib/helper'
require './config'`,
		"lib/helper.rb": `module Helper; end`,
		"config.rb":      `SETTINGS = {}`,
	})

	got := parseRubyImports("app.rb", mustRead(t, repo, "app.rb"), repo)

	found := map[string]bool{}
	for _, dep := range got {
		found[dep] = true
	}
	if !found[filepath.Clean("lib/helper.rb")] {
		t.Errorf("missing lib/helper.rb: %v", got)
	}
	if !found[filepath.Clean("config.rb")] {
		t.Errorf("missing config.rb: %v", got)
	}
}

func TestRustImports(t *testing.T) {
	repo := createTestRepo(t, map[string]string{
		"main.rs": `mod utils;
use crate::models::User;`,
		"utils.rs":       `pub fn help() {}`,
		"models/mod.rs":  `pub mod user;`,
		"models/user.rs": `pub struct User;`,
	})

	got := parseRustImports("main.rs", mustRead(t, repo, "main.rs"), repo)

	found := map[string]bool{}
	for _, dep := range got {
		found[dep] = true
	}
	if !found["utils.rs"] {
		t.Errorf("missing utils.rs: %v", got)
	}
	if !found[filepath.Clean("models/user.rs")] {
		t.Errorf("missing models/user.rs: %v", got)
	}
}

func TestCImports(t *testing.T) {
	repo := createTestRepo(t, map[string]string{
		"main.c": `#include "utils.h"
#include "lib/helper.h"`,
		"utils.h":          `void help();`,
		"lib/helper.h":     `void assist();`,
	})

	got := parseCImports("main.c", mustRead(t, repo, "main.c"), repo)

	found := map[string]bool{}
	for _, dep := range got {
		found[dep] = true
	}
	if !found["utils.h"] {
		t.Errorf("missing utils.h: %v", got)
	}
	if !found[filepath.Clean("lib/helper.h")] {
		t.Errorf("missing lib/helper.h: %v", got)
	}
}

func TestScan(t *testing.T) {
	repo := createTestRepo(t, map[string]string{
		"src/app.ts":   `import { User } from './types';`,
		"src/types.ts": `export interface User { name: string }`,
		"README.md":    `# test`,
	})

	graph, stats, err := Scan(repo, nil)
	if err != nil {
		t.Fatal(err)
	}
	if stats.FilesScanned != 2 {
		t.Errorf("files scanned = %d, want 2", stats.FilesScanned)
	}
	if stats.EdgesFound != 1 {
		t.Errorf("edges found = %d, want 1", stats.EdgesFound)
	}
	if len(graph.Langs) != 1 || graph.Langs[0] != "typescript" {
		t.Errorf("languages = %v, want [typescript]", graph.Langs)
	}
}

func TestScanLanguageFilter(t *testing.T) {
	repo := createTestRepo(t, map[string]string{
		"app.py":       `import utils`,
		"utils.py":     `def help(): pass`,
		"main.go":      `package main`,
		"go.mod":       `module test`,
	})

	graph, stats, err := Scan(repo, []string{"python"})
	if err != nil {
		t.Fatal(err)
	}
	if stats.FilesScanned != 2 {
		t.Errorf("files scanned = %d, want 2 (python only)", stats.FilesScanned)
	}
	if len(graph.Langs) != 1 || graph.Langs[0] != "python" {
		t.Errorf("languages = %v, want [python]", graph.Langs)
	}
}

func TestRelatedFiles(t *testing.T) {
	repo := createTestRepo(t, map[string]string{
		"a.ts": `import { b } from './b';`,
		"b.ts": `import { c } from './c';`,
		"c.ts": `export const c = 1;`,
		"d.ts": `export const d = 2;`,
	})

	graph, _, err := Scan(repo, nil)
	if err != nil {
		t.Fatal(err)
	}

	// a imports b, b imports c. 1 hop from a = [b], 2 hops = [b, c]
	related1 := graph.RelatedFiles("a.ts", 1)
	if len(related1) != 1 {
		t.Errorf("1 hop from a.ts: got %v, want 1 file", related1)
	}

	related2 := graph.RelatedFiles("a.ts", 2)
	if len(related2) != 2 {
		t.Errorf("2 hops from a.ts: got %v, want 2 files", related2)
	}

	// d.ts has no connections
	relatedD := graph.RelatedFiles("d.ts", 2)
	if len(relatedD) != 0 {
		t.Errorf("d.ts connections: got %v, want empty", relatedD)
	}
}

func TestFormatRelatedFiles(t *testing.T) {
	repo := createTestRepo(t, map[string]string{
		"app.ts":    `import { User } from './types';`,
		"types.ts":  `export interface User {}`,
	})

	graph, _, err := Scan(repo, nil)
	if err != nil {
		t.Fatal(err)
	}

	output := graph.FormatRelatedFiles([]string{"app.ts"}, 500)
	if output == "" {
		t.Fatal("expected non-empty output")
	}
	if !contains(output, "app.ts") {
		t.Errorf("output should mention app.ts: %s", output)
	}
	if !contains(output, "types.ts") {
		t.Errorf("output should mention types.ts: %s", output)
	}
}

func TestSaveAndLoad(t *testing.T) {
	repo := createTestRepo(t, map[string]string{
		"a.ts": `import { b } from './b';`,
		"b.ts": `export const b = 1;`,
	})

	graph, _, err := Scan(repo, nil)
	if err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(repo, "test-graph.json")
	if err := graph.Save(outPath); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.FileCount != graph.FileCount {
		t.Errorf("file count mismatch: loaded=%d, original=%d", loaded.FileCount, graph.FileCount)
	}
	if loaded.EdgeCount != graph.EdgeCount {
		t.Errorf("edge count mismatch: loaded=%d, original=%d", loaded.EdgeCount, graph.EdgeCount)
	}
	if len(loaded.Langs) != len(graph.Langs) {
		t.Errorf("language count mismatch")
	}
	sort.Strings(loaded.Langs)
	sort.Strings(graph.Langs)
	for i := range loaded.Langs {
		if loaded.Langs[i] != graph.Langs[i] {
			t.Errorf("lang mismatch at %d: %s vs %s", i, loaded.Langs[i], graph.Langs[i])
		}
	}
}

func TestDirsToSkip(t *testing.T) {
	repo := createTestRepo(t, map[string]string{
		"src/app.ts":                  `import { x } from './x';`,
		"src/x.ts":                    `export const x = 1;`,
		"node_modules/y.ts":           `export const y = 1;`,
		".git/objects/pack/test":      `data`,
		"vendor/z.ts":                 `export const z = 1;`,
	})

	graph, stats, err := Scan(repo, nil)
	if err != nil {
		t.Fatal(err)
	}
	if stats.FilesScanned != 2 {
		t.Errorf("files scanned = %d, want 2 (skipped node_modules, .git, vendor)", stats.FilesScanned)
	}
	if stats.SkippedDirs == 0 {
		t.Error("expected skipped dirs > 0")
	}
	// No edges from node_modules or vendor
	for _, e := range graph.Edges {
		if contains(e.Source, "node_modules") || contains(e.Target, "node_modules") {
			t.Errorf("should not have node_modules edge: %s -> %s", e.Source, e.Target)
		}
	}
}

func TestJavaImports(t *testing.T) {
	repo := createTestRepo(t, map[string]string{
		"src/main/java/com/example/App.java": `package com.example;
import com.example.models.User;
import com.example.utils.Helper;`,
		"src/main/java/com/example/models/User.java":   `package com.example.models; public class User {}`,
		"src/main/java/com/example/utils/Helper.java":  `package com.example.utils; public class Helper {}`,
	})

	got := parseJavaImports(
		"src/main/java/com/example/App.java",
		mustRead(t, repo, "src/main/java/com/example/App.java"),
		repo,
	)

	found := map[string]bool{}
	for _, dep := range got {
		found[dep] = true
	}
	if !found[filepath.Clean("src/main/java/com/example/models/User.java")] {
		t.Errorf("missing User.java: %v", got)
	}
	if !found[filepath.Clean("src/main/java/com/example/utils/Helper.java")] {
		t.Errorf("missing Helper.java: %v", got)
	}
}

func mustRead(t *testing.T, repo, relPath string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repo, relPath))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(len(s) > 0 && len(sub) > 0 && stringContains(s, sub)))
}

func stringContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
