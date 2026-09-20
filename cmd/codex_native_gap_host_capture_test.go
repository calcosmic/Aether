package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// Aether-owned normalized evidence contract, not a purported client export
// format. A normalized envelope can never authorize its own capture method.
type nativeGapHostCapture struct {
	SchemaVersion string                     `json:"schema_version"`
	Kind          string                     `json:"kind"`
	SessionID     string                     `json:"session_id"`
	Executable    nativeEvidenceFile         `json:"executable"`
	Version       string                     `json:"version"`
	Args          []string                   `json:"args"`
	Model         string                     `json:"model"`
	Effort        *string                    `json:"effort"`
	Method        string                     `json:"capture_method"`
	Provenance    string                     `json:"export_provenance"`
	Complete      bool                       `json:"complete"`
	Unavailable   string                     `json:"unavailable,omitempty"`
	Raw           nativeEvidenceFile         `json:"raw"`
	Settings      map[string]json.RawMessage `json:"effective_settings,omitempty"`
	Tools         []nativeGapToolDefinition  `json:"tools,omitempty"`
}

type nativeGapToolDefinition struct {
	Name       string                     `json:"name"`
	Parameters map[string]json.RawMessage `json:"parameters"`
}

func nativeGapCollectHostCapture(t *testing.T, r *codexNativeLiveReceipt, root string) {
	t.Helper()
	p := nativeGapHostProvenance{Version: r.ClientVersion, Model: r.Model, Args: r.Args, Executable: nativeEvidenceFile{r.ClientPath, r.ClientSHA256}}
	for _, kind := range []string{"configuration", "tool-definitions"} {
		c := nativeGapHostCapture{SchemaVersion: "aether-host-capture/v1", Kind: kind, SessionID: r.SessionID, Executable: p.Executable, Version: r.ClientVersion, Args: r.Args, Model: r.Model, Effort: r.HostEffort, Method: "unavailable", Provenance: "Actual invocation metadata and retained events only; no complete supported effective configuration/tool definition export established", Complete: false, Unavailable: "Retained client events do not establish complete effective settings and full model-facing tool definitions. Observed names and app-server protocol schemas cannot substitute."}
		if raw, err := os.ReadFile(r.RawEvents); err == nil {
			c.Raw = nativeEvidenceFile{r.RawEvents, lifecycleDigest(raw)}
		}
		path := filepath.Join(root, "host-"+kind+"-capture.json")
		liveSkillWriteJSON(t, path, c)
		ref := nativeEvidenceFile{path, liveSkillFileDigest(t, path)}
		if kind == "configuration" {
			p.Configuration = ref
		} else {
			p.ToolSchema = ref
		}
	}
	r.HostProvenance = &p
}

// Schema compilation is offline. A capture must not cause network requests or
// reads of unrelated local paths through a parameter schema's $ref.
type nativeGapNoSchemaLoader struct{}

func (nativeGapNoSchemaLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("external schema reference refused: %s", url)
}

func nativeGapDecodeCapture(raw []byte) (nativeGapHostCapture, error) {
	var c nativeGapHostCapture
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return c, err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return c, fmt.Errorf("capture has trailing data")
	}
	return c, nil
}

func nativeGapCaptureIdentity(c nativeGapHostCapture, kind string, r codexNativeLiveReceipt) error {
	if c.SchemaVersion != "aether-host-capture/v1" || c.Kind != kind || (kind != "configuration" && kind != "tool-definitions") {
		return fmt.Errorf("unknown host capture schema/kind")
	}
	if c.SessionID == "" || c.SessionID != r.SessionID || c.Model != r.Model || c.Model == "" || c.Version != r.ClientVersion || c.Executable.Path != r.ClientPath || c.Executable.SHA256 != r.ClientSHA256 || !reflect.DeepEqual(c.Args, r.Args) || !reflect.DeepEqual(c.Effort, r.HostEffort) {
		return fmt.Errorf("capture invocation identity differs")
	}
	return nil
}

func nativeGapCaptureSemantics(c nativeGapHostCapture, kind string, r codexNativeLiveReceipt) error {
	if err := nativeGapCaptureIdentity(c, kind, r); err != nil {
		return err
	}
	if !c.Complete || c.Unavailable != "" || c.Method == "" || c.Provenance == "" {
		return fmt.Errorf("host export unavailable or incomplete")
	}
	if kind == "configuration" {
		var model string
		if len(c.Tools) != 0 || json.Unmarshal(c.Settings["model"], &model) != nil || model != r.Model {
			return fmt.Errorf("effective model absent/different")
		}
		value, exists := c.Settings["effort"]
		if exists != (c.Effort != nil) {
			return fmt.Errorf("unreported effort defaulted or reported effort missing")
		}
		if exists {
			var effort string
			if json.Unmarshal(value, &effort) != nil || effort != *c.Effort {
				return fmt.Errorf("effective effort differs")
			}
		}
	} else {
		if len(c.Settings) != 0 || len(c.Tools) == 0 {
			return fmt.Errorf("full definitions absent")
		}
		names := map[string]bool{}
		for _, tool := range c.Tools {
			encoded, err := json.Marshal(tool.Parameters)
			if err != nil {
				return err
			}
			var document any
			if err = json.Unmarshal(encoded, &document); err != nil {
				return err
			}
			compiler := jsonschema.NewCompiler()
			compiler.UseLoader(nativeGapNoSchemaLoader{})
			if err = compiler.AddResource("urn:aether:tool-parameters", document); err != nil {
				return err
			}
			if _, err = compiler.Compile("urn:aether:tool-parameters"); err != nil {
				return fmt.Errorf("invalid tool parameter schema: %w", err)
			}
			var typ string
			if tool.Name == "" || names[tool.Name] || len(tool.Parameters) == 0 || json.Unmarshal(tool.Parameters["type"], &typ) != nil || typ != "object" {
				return fmt.Errorf("tool name/parameter object invalid")
			}
			var properties map[string]json.RawMessage
			if json.Unmarshal(tool.Parameters["properties"], &properties) != nil || properties == nil {
				return fmt.Errorf("parameter properties missing")
			}
			for _, schema := range properties {
				var object map[string]json.RawMessage
				if json.Unmarshal(schema, &object) != nil || len(object) == 0 {
					return fmt.Errorf("parameter schema invalid")
				}
			}
			if required, ok := tool.Parameters["required"]; ok {
				var list []string
				if json.Unmarshal(required, &list) != nil {
					return fmt.Errorf("required parameters invalid")
				}
				seen := map[string]bool{}
				for _, name := range list {
					if _, ok := properties[name]; !ok || seen[name] {
						return fmt.Errorf("required parameter undefined/duplicate")
					}
					seen[name] = true
				}
			}
			names[tool.Name] = true
		}
	}
	return nil
}

func nativeGapValidateCapture(ref nativeEvidenceFile, kind string, r codexNativeLiveReceipt) error {
	raw, err := nativeEvidenceBytes(ref)
	if err != nil {
		return err
	}
	c, err := nativeGapDecodeCapture(raw)
	if err != nil {
		return err
	}
	if err = nativeGapCaptureSemantics(c, kind, r); err != nil {
		return err
	}
	// replayArtifacts is the immutable inventory captured before replay. Never
	// accept artifacts newly discovered or inserted by a derived validator.
	inventory := r.Artifacts
	if r.replaying {
		inventory = r.replayArtifacts
	}
	if inventory[ref.Path] != ref.SHA256 || c.Raw.Path == ref.Path || inventory[c.Raw.Path] != c.Raw.SHA256 || c.Raw.SHA256 == "" {
		return fmt.Errorf("capture/raw source absent from original inventory")
	}
	if _, err = nativeEvidenceBytes(c.Raw); err != nil {
		return err
	}
	// No installed-client export or recorded full tool boundary has been
	// established. Schema validation is necessary but cannot make a locally
	// written envelope an actual host export. A future supported adapter must
	// decode raw bytes and corroborate the SAME invocation before admission.
	return fmt.Errorf("unsupported actual host export acquisition: %s", c.Method)
}

// The prospective contract admits an honest absence record, never an export.
// Actual worker/context/recovery proof is checked separately. Historical
// receipts retain the strict export requirement and cannot inherit this branch.
func nativeGapValidateHostMetadata(ref nativeEvidenceFile, kind string, r codexNativeLiveReceipt) error {
	prospective, err := nativeGapProofIdentity(r.ProofContract, r.ProofAmendmentSHA256)
	if err != nil {
		return err
	}
	if !prospective {
		return nativeGapValidateCapture(ref, kind, r)
	}
	raw, err := nativeEvidenceBytes(ref)
	if err != nil {
		return err
	}
	c, err := nativeGapDecodeCapture(raw)
	if err != nil {
		return err
	}
	if c.Method != "unavailable" {
		return nativeGapValidateCapture(ref, kind, r)
	}
	if err := nativeGapCaptureIdentity(c, kind, r); err != nil {
		return err
	}
	if c.Complete || strings.TrimSpace(c.Unavailable) == "" || strings.TrimSpace(c.Provenance) == "" || len(c.Settings) != 0 || len(c.Tools) != 0 {
		return fmt.Errorf("unavailable metadata contains a positive or unexplained export claim")
	}
	inventory := r.Artifacts
	if r.replaying {
		inventory = r.replayArtifacts
	}
	if c.Raw.Path == "" || c.Raw.Path != r.RawEvents || c.Raw.Path == ref.Path || c.Raw.SHA256 == "" || inventory[ref.Path] != ref.SHA256 || inventory[c.Raw.Path] != c.Raw.SHA256 {
		return fmt.Errorf("unavailable metadata absent from original invocation inventory")
	}
	_, err = nativeEvidenceBytes(c.Raw)
	return err
}

// These are deterministic adversarial bytes, never actual host exports.
func TestCodexNativeGapHostCapture(t *testing.T) {
	t.Run("prospective-unavailable-metadata", func(t *testing.T) {
		for _, kind := range []string{"configuration", "tool-definitions"} {
			for _, mutation := range []string{"none", "legacy", "unknown-contract", "stale-amendment", "complete", "empty-reason", "empty-provenance", "settings", "tools", "wrong-session", "wrong-model", "wrong-argv", "wrong-effort", "wrong-kind", "positive-export", "substituted-raw", "changed-raw", "post-hoc-envelope", "post-hoc-raw"} {
				t.Run(kind+"/"+mutation, func(t *testing.T) {
					dir := t.TempDir()
					path := filepath.Join(dir, "events.jsonl")
					liveSkillWrite(t, path, []byte("deterministic invocation events, not actual host evidence\n"))
					r := codexNativeLiveReceipt{ProofContract: nativeCapabilityProofContract, ProofAmendmentSHA256: nativeCapabilityProofAmendmentSHA256, SessionID: "deterministic-session", ClientVersion: "deterministic-version", ClientPath: "/deterministic/client", ClientSHA256: "deterministic-hash", Args: []string{"exec"}, Model: "model", RawEvents: path, Artifacts: map[string]string{path: liveSkillFileDigest(t, path)}}
					nativeGapCollectHostCapture(t, &r, dir)
					ref := r.HostProvenance.Configuration
					if kind == "tool-definitions" {
						ref = r.HostProvenance.ToolSchema
					}
					raw, err := nativeEvidenceBytes(ref)
					if err != nil {
						t.Fatal(err)
					}
					c, err := nativeGapDecodeCapture(raw)
					if err != nil {
						t.Fatal(err)
					}
					switch mutation {
					case "legacy":
						r.ProofContract, r.ProofAmendmentSHA256 = "", ""
					case "unknown-contract":
						r.ProofContract = "unknown/v1"
					case "stale-amendment":
						r.ProofAmendmentSHA256 = "sha256:stale"
					case "complete":
						c.Complete = true
					case "empty-reason":
						c.Unavailable = "  "
					case "empty-provenance":
						c.Provenance = "  "
					case "settings":
						c.Settings = map[string]json.RawMessage{"model": json.RawMessage(`"model"`)}
					case "tools":
						c.Tools = []nativeGapToolDefinition{{Name: "pretend-export"}}
					case "wrong-session":
						c.SessionID = "other"
					case "wrong-model":
						c.Model = "other"
					case "wrong-argv":
						c.Args = []string{"other"}
					case "wrong-effort":
						c.Effort = new(string)
					case "wrong-kind":
						c.Kind = "other"
					case "positive-export":
						c.Complete, c.Unavailable, c.Method = true, "", "pretend-export"
					case "substituted-raw":
						other := filepath.Join(dir, "other-invocation.jsonl")
						liveSkillWrite(t, other, []byte("other invocation"))
						c.Raw = nativeEvidenceFile{other, liveSkillFileDigest(t, other)}
						r.Artifacts[other] = c.Raw.SHA256
					case "changed-raw":
						liveSkillWrite(t, path, []byte("changed events"))
					}
					liveSkillWriteJSON(t, ref.Path, c)
					ref.SHA256 = liveSkillFileDigest(t, ref.Path)
					r.Artifacts[ref.Path] = ref.SHA256
					r.replaying = true
					r.replayArtifacts = make(map[string]string, len(r.Artifacts))
					for path, digest := range r.Artifacts {
						r.replayArtifacts[path] = digest
					}
					if mutation == "post-hoc-envelope" {
						delete(r.replayArtifacts, ref.Path)
					}
					if mutation == "post-hoc-raw" {
						delete(r.replayArtifacts, c.Raw.Path)
					}
					err = nativeGapValidateHostMetadata(ref, kind, r)
					if mutation == "none" {
						if err != nil {
							t.Fatal(err)
						}
						if err := nativeGapValidateCapture(ref, kind, r); err == nil {
							t.Fatal("unavailable metadata became a successful export")
						}
					} else if err == nil {
						t.Fatal("invalid unavailable metadata accepted")
					}
				})
			}
		}
	})
	t.Run("collector-unavailable", func(t *testing.T) {
		dir := t.TempDir()
		r := codexNativeLiveReceipt{SessionID: "deterministic-session", ClientVersion: "deterministic-version", ClientPath: "/deterministic/client", ClientSHA256: "deterministic-hash", Args: []string{"exec"}, Model: "model", Artifacts: map[string]string{}}
		nativeGapCollectHostCapture(t, &r, dir)
		if r.HostProvenance == nil {
			t.Fatal("collector omitted unavailable record")
		}
		for _, ref := range []nativeEvidenceFile{r.HostProvenance.Configuration, r.HostProvenance.ToolSchema} {
			raw, err := nativeEvidenceBytes(ref)
			if err != nil {
				t.Fatal(err)
			}
			c, err := nativeGapDecodeCapture(raw)
			if err != nil {
				t.Fatal(err)
			}
			if c.Complete || c.Unavailable == "" || c.SessionID != r.SessionID {
				t.Fatal("unavailable collector masquerades as complete export")
			}
		}
		if err := nativeGapHostIdentity(*r.HostProvenance, r); err == nil {
			t.Fatal("unavailable collector qualified")
		}
	})
	t.Run("deterministic-semantic-contract-only", func(t *testing.T) {
		r := codexNativeLiveReceipt{SessionID: "deterministic-session", ClientVersion: "deterministic-version", ClientPath: "/deterministic/client", ClientSHA256: "deterministic-hash", Args: []string{"exec", "-m", "model"}, Model: "model"}
		base := nativeGapHostCapture{SchemaVersion: "aether-host-capture/v1", Kind: "configuration", SessionID: r.SessionID, Executable: nativeEvidenceFile{r.ClientPath, r.ClientSHA256}, Version: r.ClientVersion, Args: r.Args, Model: r.Model, Method: "deterministic-test-only", Provenance: "deterministic fixture; not an installed client export", Complete: true, Settings: map[string]json.RawMessage{"model": json.RawMessage(`"model"`)}}
		if err := nativeGapCaptureSemantics(base, "configuration", r); err != nil {
			t.Fatal(err)
		}
		for _, mutation := range []string{"version", "kind", "session", "model", "argv", "effort", "unavailable", "incomplete", "effective-model", "default-effort"} {
			t.Run(mutation, func(t *testing.T) {
				encoded, _ := json.Marshal(base)
				c, _ := nativeGapDecodeCapture(encoded)
				switch mutation {
				case "version":
					c.SchemaVersion = "unknown/v2"
				case "kind":
					c.Kind = "tool-definitions"
				case "session":
					c.SessionID = "other"
				case "model":
					c.Model = "other"
				case "argv":
					c.Args = []string{"other"}
				case "effort":
					value := "high"
					c.Effort = &value
				case "unavailable":
					c.Unavailable = "no supported export"
				case "incomplete":
					c.Complete = false
				case "effective-model":
					c.Settings["model"] = json.RawMessage(`"other"`)
				case "default-effort":
					c.Settings["effort"] = json.RawMessage(`"medium"`)
				}
				if err := nativeGapCaptureSemantics(c, "configuration", r); err == nil {
					t.Fatal("invalid semantics accepted")
				}
			})
		}
		base.Kind = "tool-definitions"
		base.Settings = nil
		base.Tools = []nativeGapToolDefinition{{Name: "deterministic-tool", Parameters: map[string]json.RawMessage{"type": json.RawMessage(`"object"`), "properties": json.RawMessage(`{"text":{"type":"string"}}`), "required": json.RawMessage(`["text"]`)}}}
		if err := nativeGapCaptureSemantics(base, "tool-definitions", r); err != nil {
			t.Fatal(err)
		}
		for _, mutation := range []string{"names-only", "empty-schema", "missing-schema", "invalid-properties", "undefined-required", "empty-tools", "duplicate-tools"} {
			t.Run(mutation, func(t *testing.T) {
				encoded, _ := json.Marshal(base)
				c, _ := nativeGapDecodeCapture(encoded)
				switch mutation {
				case "names-only":
					c.Tools[0].Parameters = nil
				case "empty-schema":
					c.Tools[0].Parameters = map[string]json.RawMessage{}
				case "missing-schema":
					delete(c.Tools[0].Parameters, "properties")
				case "invalid-properties":
					c.Tools[0].Parameters["properties"] = json.RawMessage(`{"text":null}`)
				case "undefined-required":
					c.Tools[0].Parameters["required"] = json.RawMessage(`["other"]`)
				case "empty-tools":
					c.Tools = nil
				case "duplicate-tools":
					c.Tools = append(c.Tools, c.Tools[0])
				}
				if err := nativeGapCaptureSemantics(c, "tool-definitions", r); err == nil {
					t.Fatal("incomplete definitions accepted")
				}
			})
		}
		t.Run("post-hoc-inventory-and-unsupported-source", func(t *testing.T) {
			dir := t.TempDir()
			rawPath := filepath.Join(dir, "boundary.json")
			os.WriteFile(rawPath, []byte(`{"names":["deterministic-tool"]}`), 0600)
			raw, _ := os.ReadFile(rawPath)
			base.Raw = nativeEvidenceFile{rawPath, lifecycleDigest(raw)}
			encoded, _ := json.Marshal(base)
			path := filepath.Join(dir, "envelope.json")
			os.WriteFile(path, encoded, 0600)
			ref := nativeEvidenceFile{path, lifecycleDigest(encoded)}
			r.Artifacts = map[string]string{path: ref.SHA256, rawPath: base.Raw.SHA256}
			if err := nativeGapValidateCapture(ref, "tool-definitions", r); err == nil {
				t.Fatal("deterministic envelope became actual host export")
			}
			r.replaying = true
			r.replayArtifacts = map[string]string{}
			if err := nativeGapValidateCapture(ref, "tool-definitions", r); err == nil {
				t.Fatal("post-hoc inventory insertion accepted")
			}
		})
		for _, value := range []string{`{"schema_version":"aether-host-capture/v1","unknown":true}`, `{} {}`} {
			if _, err := nativeGapDecodeCapture([]byte(value)); err == nil {
				t.Fatal("unknown fields/trailing data accepted")
			}
		}
	})
	for _, value := range []string{"", "a prompt", "arbitrary bytes", `{"tools":["exec_command"]}`, `{"available":false}`} {
		t.Run(value, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "substitution")
			raw := []byte(value)
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			ref := nativeEvidenceFile{path, lifecycleDigest(raw)}
			r := codexNativeLiveReceipt{ClientVersion: "measured", Model: "model", ClientPath: path, ClientSHA256: ref.SHA256, Args: []string{"exec"}, Artifacts: map[string]string{path: ref.SHA256}}
			p := nativeGapHostProvenance{Version: r.ClientVersion, Model: r.Model, Args: r.Args, Executable: ref, Configuration: ref, ToolSchema: ref}
			if err := nativeGapHostIdentity(p, r); err == nil {
				t.Fatal("same-hash unrelated bytes accepted as configuration and full tool schema")
			}
		})
	}
}
