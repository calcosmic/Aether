package codegraph

import "testing"

func TestShouldSkipDir(t *testing.T) {
	tests := []struct {
		dir  string
		want bool
	}{
		// Version control
		{".git", true},
		// Python
		{".venv", true},
		{"venv", true},
		{"__pycache__", true},
		{".mypy_cache", true},
		{".pytest_cache", true},
		{".tox", true},
		{".ruff_cache", true},
		{".pytype", true},
		{"site-packages", true},
		// Node.js
		{"node_modules", true},
		{".next", true},
		{".nuxt", true},
		{".svelte-kit", true},
		// Go
		{"vendor", true},
		// Rust
		{"target", true},
		{".cargo", true},
		// Build/artifacts
		{"dist", true},
		{"build", true},
		{"out", true},
		{"bin", true},
		{"coverage", true},
		// Caches
		{".cache", true},
		{".terraform", true},
		{".gradle", true},
		// Aether-managed
		{".aether", true},
		{".claude", true},
		{".codex", true},
		{".opencode", true},
		// IDE/editor
		{".idea", true},
		{".vscode", true},
		// Temp
		{"tmp", true},
		{"temp", true},
	}
	for _, tt := range tests {
		t.Run(tt.dir, func(t *testing.T) {
			if got := ShouldSkipDir(tt.dir); got != tt.want {
				t.Errorf("ShouldSkipDir(%q) = %v, want %v", tt.dir, got, tt.want)
			}
		})
	}
}

func TestShouldSkipDir_Negative(t *testing.T) {
	tests := []struct {
		dir  string
		want bool
	}{
		{"src", false},
		{"cmd", false},
		{"pkg", false},
		{"internal", false},
		{"env", false}, // Pitfall 8: bare "env" must NOT be excluded
		{"lib", false},
		{"docs", false},
		{"tests", false},
		{"app", false},
		{"main", false},
	}
	for _, tt := range tests {
		t.Run(tt.dir, func(t *testing.T) {
			if got := ShouldSkipDir(tt.dir); got != tt.want {
				t.Errorf("ShouldSkipDir(%q) = %v, want %v", tt.dir, got, tt.want)
			}
		})
	}
}

func TestShouldSkipDir_ExtendedCoverage(t *testing.T) {
	// Specifically assert the key Python/Node/Rust/Go noise dirs
	keyDirs := []string{
		".venv", "__pycache__", "site-packages", ".pytest_cache", ".tox", ".mypy_cache",
		"node_modules", ".gradle", ".cache", ".cargo", "target",
	}
	for _, dir := range keyDirs {
		if !ShouldSkipDir(dir) {
			t.Errorf("ShouldSkipDir(%q) = false, want true", dir)
		}
	}
}

func TestShouldSkipFile(t *testing.T) {
	tests := []struct {
		file string
		want bool
	}{
		{".pyc", true},
		{"foo.pyc", true},
		{"foo.pyo", true},
		{"foo.so", true},
		{"foo.dylib", true},
		{"foo.dll", true},
		{"foo.map", true},
		{"app.go", false},
		{"app.ts", false},
		{"app.py", false},
		{"app.js", false},
		{"README.md", false},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			if got := ShouldSkipFile(tt.file); got != tt.want {
				t.Errorf("ShouldSkipFile(%q) = %v, want %v", tt.file, got, tt.want)
			}
		})
	}
}

func TestShouldSkipFile_CompoundExtensions(t *testing.T) {
	tests := []struct {
		file string
		want bool
	}{
		{"app.min.js", true},
		{"app.min.css", true},
		{"app.bundle.js", true},
		{"app.d.ts", true},
		{"app.js", false},
		{"app.css", false},
		{"app.ts", false},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			if got := ShouldSkipFile(tt.file); got != tt.want {
				t.Errorf("ShouldSkipFile(%q) = %v, want %v", tt.file, got, tt.want)
			}
		})
	}
}

func TestShouldSkipDir_NoBareEnv(t *testing.T) {
	// Pitfall 8: "env" (without leading dot) must return false
	if ShouldSkipDir("env") {
		t.Error("ShouldSkipDir(\"env\") returned true, want false (prevents over-exclusion)")
	}
}

func TestNoiseDirCount(t *testing.T) {
	count := NoiseDirCount()
	if count < 25 {
		t.Errorf("NoiseDirCount() = %d, want >= 25", count)
	}
}
