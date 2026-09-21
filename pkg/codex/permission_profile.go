package codex

import (
	"fmt"
	"sort"
	"strings"
)

const PermissionProfileSchemaVersion = 1

type PermissionProfileName string

const (
	PermissionRepositoryReadOnly PermissionProfileName = "repository_read_only"
	PermissionWorkspaceWrite     PermissionProfileName = "workspace_write"
	PermissionScopedWrite        PermissionProfileName = "scoped_write"
	PermissionTestWrite          PermissionProfileName = "test_write"
)

type FilesystemPermission string

const (
	FilesystemRepositoryReadOnly FilesystemPermission = "repository_read_only"
	FilesystemWorkspaceWrite     FilesystemPermission = "workspace_write"
	FilesystemScopedWrite        FilesystemPermission = "scoped_write"
	FilesystemTestWrite          FilesystemPermission = "test_write"
)

// PermissionProfile is the canonical permission request attached to a worker.
// BehavioralRestrictions are visible caste guidance, not host-enforced claims.
type PermissionProfile struct {
	SchemaVersion          int                   `json:"schema_version"`
	Name                   PermissionProfileName `json:"name"`
	Filesystem             FilesystemPermission  `json:"filesystem"`
	Shell                  string                `json:"shell"`
	Network                string                `json:"network"`
	Approval               string                `json:"approval"`
	WriteScopes            []string              `json:"write_scopes,omitempty"`
	BehavioralRestrictions []string              `json:"behavioral_restrictions,omitempty"`
}

// PermissionDecision records what the selected host will actually enforce.
type PermissionDecision struct {
	SchemaVersion int               `json:"schema_version"`
	Platform      Platform          `json:"platform"`
	Profile       PermissionProfile `json:"profile"`
	Allowed       bool              `json:"allowed"`
	Enforcement   CapabilityLevel   `json:"enforcement"`
	Mechanism     string            `json:"mechanism"`
	Limitations   []string          `json:"limitations,omitempty"`
}

var repositoryReadOnlyCastes = map[string]struct{}{
	"includer": {},
}

// PermissionProfileForCaste returns the smallest permission promise that the
// current provider adapters can enforce consistently. Narrower prompt rules are
// retained as visible behavioral restrictions until they have a real sandbox.
func PermissionProfileForCaste(caste string) PermissionProfile {
	caste = normalizePermissionCaste(caste)
	if _, ok := repositoryReadOnlyCastes[caste]; ok {
		return PermissionProfile{
			SchemaVersion: PermissionProfileSchemaVersion,
			Name:          PermissionRepositoryReadOnly,
			Filesystem:    FilesystemRepositoryReadOnly,
			Shell:         "within_filesystem_boundary",
			Network:       "provider_default",
			Approval:      "never",
		}
	}

	profile := PermissionProfile{
		SchemaVersion: PermissionProfileSchemaVersion,
		Name:          PermissionWorkspaceWrite,
		Filesystem:    FilesystemWorkspaceWrite,
		Shell:         "within_filesystem_boundary",
		Network:       "provider_default",
		Approval:      "never",
	}
	profile.BehavioralRestrictions = behavioralRestrictionsForCaste(caste)
	return profile
}

func behavioralRestrictionsForCaste(caste string) []string {
	switch normalizePermissionCaste(caste) {
	case "archaeologist", "auditor", "gatekeeper", "measurer", "tracker", "watcher":
		return []string{"do not modify project source or tests; persist only approved review evidence"}
	case "probe":
		return []string{"write test files only; do not modify project source"}
	case "surveyor_disciplines", "surveyor_nest", "surveyor_pathogens", "surveyor_provisions":
		return []string{"write survey artifacts only to the survey output paths named in your dispatch brief: .aether/data/survey, or the .aether/data/territory-candidates staging directory during a territory refresh"}
	case "scout":
		return []string{"write phase research artifacts under .aether/data/phase-research only"}
	case "chronicler":
		return []string{"write documentation artifacts only"}
	case "architect", "route_setter", "sage":
		return []string{"return structured analysis without repository writes"}
	case "oracle":
		return []string{"write escalated research findings under .aether/oracle and .aether/data/phase-research only"}
	default:
		return nil
	}
}

func normalizePermissionCaste(caste string) string {
	caste = strings.ToLower(strings.TrimSpace(caste))
	caste = strings.TrimPrefix(caste, "aether-")
	caste = strings.ReplaceAll(caste, "-", "_")
	return caste
}

// ResolvePermissionProfile binds an optional request to the canonical caste
// profile. An explicit request must match exactly so a stale or edited manifest
// cannot silently broaden a worker's access.
func ResolvePermissionProfile(caste string, requested PermissionProfile) (PermissionProfile, error) {
	canonical := PermissionProfileForCaste(caste)
	if requested.Name == "" && requested.SchemaVersion == 0 {
		return canonical, nil
	}
	requested = normalizePermissionProfile(requested)
	canonical = normalizePermissionProfile(canonical)
	if !permissionEnforcementEqual(requested, canonical) {
		return PermissionProfile{}, fmt.Errorf(
			"permission profile mismatch for caste %q: requested %q but canonical profile is %q",
			normalizePermissionCaste(caste), requested.Name, canonical.Name,
		)
	}
	return canonical, nil
}

func permissionEnforcementEqual(left, right PermissionProfile) bool {
	return left.SchemaVersion == right.SchemaVersion &&
		left.Name == right.Name &&
		left.Filesystem == right.Filesystem &&
		left.Shell == right.Shell &&
		left.Network == right.Network &&
		left.Approval == right.Approval &&
		strings.Join(left.WriteScopes, "\x00") == strings.Join(right.WriteScopes, "\x00")
}

func normalizePermissionProfile(profile PermissionProfile) PermissionProfile {
	profile.WriteScopes = compactPermissionStrings(profile.WriteScopes)
	profile.BehavioralRestrictions = compactPermissionStrings(profile.BehavioralRestrictions)
	return profile
}

func compactPermissionStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

// PermissionDecisionFor returns a fail-closed decision for the selected host.
func PermissionDecisionFor(platform Platform, profile PermissionProfile) PermissionDecision {
	profile = normalizePermissionProfile(profile)
	decision := PermissionDecision{
		SchemaVersion: PermissionProfileSchemaVersion,
		Platform:      platform,
		Profile:       profile,
		Allowed:       false,
		Enforcement:   CapabilityUnavailable,
	}

	switch profile.Name {
	case PermissionRepositoryReadOnly:
		switch platform {
		case PlatformCodex:
			decision.Allowed = true
			decision.Enforcement = CapabilityProven
			decision.Mechanism = "codex exec uses the read-only filesystem sandbox with approvals disabled"
		case PlatformClaude:
			decision.Allowed = true
			decision.Enforcement = CapabilityProven
			decision.Mechanism = "Claude Code uses plan permission mode, which permits reads but denies source edits"
		case PlatformOpenCode:
			decision.Allowed = true
			decision.Enforcement = CapabilityProven
			decision.Mechanism = "the Aether primary router and the selected read-only subagent deny edit, bash, and external-directory tools"
		case PlatformFake:
			decision.Allowed = true
			decision.Enforcement = CapabilityTestOnly
			decision.Mechanism = "deterministic test fixture launches no external process"
		default:
			decision.Mechanism = "the selected platform has no read-only enforcement contract"
		}
	case PermissionWorkspaceWrite:
		switch platform {
		case PlatformCodex:
			decision.Allowed = true
			decision.Enforcement = CapabilityProven
			decision.Mechanism = "codex exec uses the workspace-write filesystem sandbox with approvals disabled"
		case PlatformClaude:
			decision.Allowed = true
			decision.Enforcement = CapabilityProven
			decision.Mechanism = "Claude Code uses acceptEdits plus a fail-closed OS sandbox confined to the working directory"
		case PlatformOpenCode:
			decision.Allowed = true
			decision.Enforcement = CapabilityProven
			decision.Mechanism = "the Aether primary router delegates to the selected subagent with external-directory access denied"
		case PlatformFake:
			decision.Allowed = true
			decision.Enforcement = CapabilityTestOnly
			decision.Mechanism = "deterministic test fixture launches no external process"
		default:
			decision.Mechanism = "the selected platform has no workspace-write enforcement contract"
		}
		if len(profile.BehavioralRestrictions) > 0 {
			decision.Limitations = append(decision.Limitations,
				"caste-specific write scope is behavioral; the enforced boundary is the full workspace",
			)
		}
	case PermissionScopedWrite, PermissionTestWrite:
		decision.Mechanism = fmt.Sprintf("%s is not implemented by the production adapters", profile.Name)
		decision.Limitations = []string{"dispatch is blocked rather than treating prompt text or post-run validation as isolation"}
	default:
		decision.Mechanism = fmt.Sprintf("unknown permission profile %q", profile.Name)
	}
	if profile.Network == "provider_default" {
		decision.Limitations = append(decision.Limitations, "network access remains provider-managed and is not an Aether isolation guarantee")
	}
	return decision
}

func ResolvePermissionDecision(platform Platform, caste string, requested PermissionProfile) (PermissionDecision, error) {
	profile, err := ResolvePermissionProfile(caste, requested)
	if err != nil {
		return PermissionDecision{}, err
	}
	decision := PermissionDecisionFor(platform, profile)
	if !decision.Allowed {
		return decision, fmt.Errorf(
			"permission profile %q for caste %q is unavailable on %s: %s",
			profile.Name, normalizePermissionCaste(caste), platform, decision.Mechanism,
		)
	}
	return decision, nil
}

func RenderPermissionProfileSection(decision PermissionDecision) string {
	var b strings.Builder
	b.WriteString("## Enforced Permission Profile\n\n")
	fmt.Fprintf(&b, "- Profile: `%s`\n", decision.Profile.Name)
	fmt.Fprintf(&b, "- Filesystem: `%s`\n", decision.Profile.Filesystem)
	fmt.Fprintf(&b, "- Shell: `%s`\n", decision.Profile.Shell)
	fmt.Fprintf(&b, "- Network: `%s`\n", decision.Profile.Network)
	fmt.Fprintf(&b, "- Approval: `%s`\n", decision.Profile.Approval)
	fmt.Fprintf(&b, "- Host enforcement: `%s` via %s\n", decision.Enforcement, decision.Mechanism)
	if len(decision.Profile.BehavioralRestrictions) > 0 {
		b.WriteString("\nThe following caste restrictions are required behavior but are not a narrower host sandbox:\n")
		for _, restriction := range decision.Profile.BehavioralRestrictions {
			fmt.Fprintf(&b, "- %s\n", restriction)
		}
	}
	if decision.Profile.Name == PermissionRepositoryReadOnly {
		b.WriteString("\nDo not attempt to create or modify repository files. Return all findings through the structured worker response.\n")
	}
	return strings.TrimSpace(b.String())
}
