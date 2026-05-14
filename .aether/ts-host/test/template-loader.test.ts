/**
 * Template loader unit tests.
 *
 * Verifies YAML frontmatter parsing, variable substitution with fallback,
 * disk loading, inline default fallback, and error handling.
 */

import { describe, it } from "node:test";
import assert from "node:assert/strict";

import {
  parseTemplate,
  substituteTemplate,
  loadTemplate,
  DEFAULT_TEMPLATES,
} from "../src/template-loader.js";

const REPO_ROOT = "/Users/callumcowie/repos/Aether";

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("template-loader", () => {
  it("parseTemplate splits YAML frontmatter and body", () => {
    const raw = "---\nfoo: bar\n---\nHello";
    const result = parseTemplate(raw);
    assert.equal(result.frontmatter.foo, "bar");
    assert.equal(result.body, "Hello");
  });

  it("parseTemplate throws on missing frontmatter", () => {
    assert.throws(
      () => parseTemplate("no frontmatter"),
      /missing YAML frontmatter/
    );
  });

  it("substituteTemplate replaces variables with values", () => {
    const result = substituteTemplate("Hello {name}", { name: "World" });
    assert.equal(result, "Hello World");
  });

  it("substituteTemplate uses fallback when variable missing", () => {
    const result = substituteTemplate("{missing:default}", {});
    assert.equal(result, "default");
  });

  it("substituteTemplate uses empty string when variable and fallback missing", () => {
    const result = substituteTemplate("{missing}", {});
    assert.equal(result, "");
  });

  it("loadTemplate reads banner-build-start from disk", () => {
    const result = loadTemplate(REPO_ROOT, "banner-build-start");
    assert.equal(result.frontmatter.figlet_font, "Standard");
    assert.equal(result.frontmatter.emoji, "🔨");
    assert.equal(result.frontmatter.title, "BUILD");
    assert.ok(result.body.includes("{figlet_banner}"));
    assert.ok(result.body.includes("{stage}"));
    assert.ok(result.body.includes("{content}"));
  });

  it("loadTemplate reads spawn-frame from disk", () => {
    const result = loadTemplate(REPO_ROOT, "spawn-frame");
    assert.deepEqual(result.frontmatter, {});
    assert.ok(result.body.includes("{emoji}"));
    assert.ok(result.body.includes("{label}"));
    assert.ok(result.body.includes("{name}"));
    assert.ok(result.body.includes("{task}"));
  });

  it("loadTemplate falls back to DEFAULT_TEMPLATES for missing file", () => {
    const result = loadTemplate("/non-existent-dir-12345", "spawn-frame");
    assert.ok(result.body.includes("{emoji}"));
    assert.equal(result.frontmatter.emoji, undefined);
  });

  it("loadTemplate throws for unknown template with no default", () => {
    assert.throws(
      () => loadTemplate(REPO_ROOT, "unknown-template-xyz"),
      /Template not found/
    );
  });

  it("DEFAULT_TEMPLATES contains all six ceremony templates", () => {
    const names = [
      "banner-build-start",
      "banner-seal-complete",
      "spawn-frame",
      "stage-separator",
      "build-summary",
      "closeout-ritual",
    ];
    for (const name of names) {
      assert.ok(DEFAULT_TEMPLATES[name], `Should have default for ${name}`);
      assert.ok(typeof DEFAULT_TEMPLATES[name].body === "string");
    }
  });
});
