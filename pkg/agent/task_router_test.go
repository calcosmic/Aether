package agent

import (
	"testing"
)

func TestParseTypeHint_Implement(t *testing.T) {
	hint := ParseTypeHint("[implement] build the API")
	if hint != "implement" {
		t.Errorf("expected 'implement', got %q", hint)
	}
}

func TestParseTypeHint_Test(t *testing.T) {
	hint := ParseTypeHint("[test] verify auth")
	if hint != "test" {
		t.Errorf("expected 'test', got %q", hint)
	}
}

func TestParseTypeHint_NoHint(t *testing.T) {
	hint := ParseTypeHint("build the API")
	if hint != "" {
		t.Errorf("expected empty string, got %q", hint)
	}
}

func TestParseTypeHint_Multiple(t *testing.T) {
	hint := ParseTypeHint("[implement] [test] stuff")
	if hint != "implement" {
		t.Errorf("expected 'implement' (first match), got %q", hint)
	}
}

func TestRouteTask_TypeHint(t *testing.T) {
	caste := RouteTask("[test] verify auth")
	if caste != CasteWatcher {
		t.Errorf("expected watcher, got %q", caste)
	}
}

func TestRouteTask_KeywordTest(t *testing.T) {
	caste := RouteTask("verify the login works")
	if caste != CasteWatcher {
		t.Errorf("expected watcher, got %q", caste)
	}
}

func TestRouteTask_KeywordResearch(t *testing.T) {
	caste := RouteTask("investigate why tests fail")
	if caste != CasteScout {
		t.Errorf("expected scout, got %q", caste)
	}
}

func TestRouteTask_KeywordImplement(t *testing.T) {
	caste := RouteTask("create the new endpoint")
	if caste != CasteBuilder {
		t.Errorf("expected builder, got %q", caste)
	}
}

func TestRouteTask_KeywordDesign(t *testing.T) {
	caste := RouteTask("architect the data model")
	if caste != CasteArchitect {
		t.Errorf("expected architect, got %q", caste)
	}
}

func TestRouteTask_Default(t *testing.T) {
	caste := RouteTask("do the thing")
	if caste != CasteBuilder {
		t.Errorf("expected builder (default), got %q", caste)
	}
}

func TestHintToCaste_Review(t *testing.T) {
	caste := hintToCaste("review")
	if caste != CasteScout {
		t.Errorf("expected scout for review hint, got %q", caste)
	}
}

func TestHintToCaste_Unknown(t *testing.T) {
	caste := hintToCaste("unknown")
	if caste != CasteBuilder {
		t.Errorf("expected builder for unknown hint, got %q", caste)
	}
}
