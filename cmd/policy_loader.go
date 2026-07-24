package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// policyCache holds loaded policies in memory to avoid repeated file reads.
var policyCache = map[string]interface{}{}
var policyCacheMu sync.RWMutex

// policyPath returns the filesystem path for a named policy file.
func policyPath(name string) string {
	return filepath.Join("colony", "policies", name+".yaml")
}

// loadYAMLPolicy reads a YAML file, strips optional frontmatter delimiters (---),
// and unmarshals the content into out. If the file does not exist or is malformed,
// an error is returned so the caller can fall back to hardcoded defaults.
func loadYAMLPolicy(path string, out interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read policy file %s: %w", path, err)
	}
	content := string(data)
	content = strings.TrimSpace(content)
	// Strip YAML frontmatter delimiters if present.
	if strings.HasPrefix(content, "---") {
		content = strings.TrimPrefix(content, "---")
		content = strings.TrimSpace(content)
		if idx := strings.Index(content, "---"); idx >= 0 {
			content = strings.TrimSpace(content[idx+3:])
		}
	}
	if err := yaml.Unmarshal([]byte(content), out); err != nil {
		return fmt.Errorf("unmarshal policy file %s: %w", path, err)
	}
	return nil
}

// cachedPolicyLoad loads a policy through the given loader function and caches
// the result by path. It is safe for concurrent use.
func cachedPolicyLoad(path string, loader func() (interface{}, error)) (interface{}, error) {
	policyCacheMu.RLock()
	if cached, ok := policyCache[path]; ok {
		policyCacheMu.RUnlock()
		return cached, nil
	}
	policyCacheMu.RUnlock()

	loaded, err := loader()
	if err != nil {
		return nil, err
	}

	policyCacheMu.Lock()
	policyCache[path] = loaded
	policyCacheMu.Unlock()
	return loaded, nil
}
