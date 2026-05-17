package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/calcosmic/Aether/pkg/codex"
)

func dispatchAgentPath(root string, invoker codex.WorkerInvoker, agentName string) string {
	platform := codex.PlatformFromInvoker(invoker)
	if platform == codex.PlatformUnknown {
		platform = codex.DetectActivePlatform()
	}
	return dispatchAgentPathForPlatform(root, platform, agentName)
}

func dispatchAgentPathForPlatform(root string, platform codex.Platform, agentName string) string {
	if platform == codex.PlatformUnknown || platform == codex.PlatformFake {
		return ""
	}
	return codex.AgentDefinitionPath(root, platform, agentName)
}

func dispatchAvailabilityMessage(invoker codex.WorkerInvoker) string {
	ctx := context.Background()
	if message := dispatchAvailabilityDiagnosticMessage(invoker, ctx); message != "" {
		return message
	}
	message := strings.TrimSpace(codex.DescribeInvokerAvailability(invoker, ctx))
	if message == "" || message == "worker dispatcher availability unknown" {
		return "no authenticated worker platform is available"
	}
	return message
}

func dispatchProviderDiagnostics(invoker codex.WorkerInvoker) string {
	if invoker == nil {
		return ""
	}
	if _, ok := invoker.(*codex.FakeInvoker); ok {
		return ""
	}
	message := dispatchAvailabilityDiagnosticMessage(invoker, context.Background())
	if strings.HasPrefix(message, "using ") {
		return ""
	}
	return strings.TrimSpace(message)
}

func dispatchUnavailableError(invoker codex.WorkerInvoker) error {
	return fmt.Errorf("worker dispatcher is unavailable: %s", dispatchAvailabilityMessage(invoker))
}

type dispatchAvailabilityReporter interface {
	Availability(context.Context) codex.AvailabilityStatus
}

type dispatchAvailabilityCandidateReporter interface {
	ActivePlatform() codex.Platform
	CandidateStatuses() []codex.AvailabilityStatus
}

func dispatchAvailabilityDiagnosticMessage(invoker codex.WorkerInvoker, ctx context.Context) string {
	if meta, ok := invoker.(dispatchAvailabilityCandidateReporter); ok {
		if reporter, ok := invoker.(dispatchAvailabilityReporter); ok {
			if status := reporter.Availability(ctx); status.Available {
				return strings.TrimSpace(codex.DescribeInvokerAvailability(invoker, ctx))
			}
		}
		if message := formatDispatchAvailabilityCandidates(meta.ActivePlatform(), meta.CandidateStatuses()); message != "" {
			return message
		}
	}
	reporter, ok := invoker.(dispatchAvailabilityReporter)
	if !ok {
		return ""
	}
	status := reporter.Availability(ctx)
	if status.Available {
		return strings.TrimSpace(codex.DescribeInvokerAvailability(invoker, ctx))
	}
	return formatDispatchAvailabilityStatus(status)
}

func formatDispatchAvailabilityCandidates(active codex.Platform, statuses []codex.AvailabilityStatus) string {
	parts := make([]string, 0, len(statuses)+1)
	if active != codex.PlatformUnknown {
		parts = append(parts, fmt.Sprintf("detected host platform %s", active))
	}
	for _, status := range statuses {
		if message := formatDispatchAvailabilityStatus(status); message != "" {
			parts = append(parts, message)
		}
	}
	return strings.Join(parts, "; ")
}

func formatDispatchAvailabilityStatus(status codex.AvailabilityStatus) string {
	if (status.Platform == "" || status.Platform == codex.PlatformUnknown) && status.Category == "" && strings.TrimSpace(status.Reason) == "" {
		return ""
	}
	provider := string(status.Platform)
	if provider == "" || status.Platform == codex.PlatformUnknown {
		provider = "worker"
	}
	cause := strings.TrimSpace(status.Reason)
	if cause == "" {
		cause = defaultAvailabilityCause(status)
	}
	cause = strings.TrimRight(cause, ".")
	next := dispatchAvailabilityNextAction(status)
	if next == "" {
		return fmt.Sprintf("%s provider: %s", provider, cause)
	}
	return fmt.Sprintf("%s provider: %s. Next: %s", provider, cause, next)
}

func defaultAvailabilityCause(status codex.AvailabilityStatus) string {
	switch status.Category {
	case codex.AvailabilityCategoryBinaryMissing:
		return "CLI binary was not found"
	case codex.AvailabilityCategoryAuthProbeFailed:
		return "auth probe failed; sensitive details omitted"
	case codex.AvailabilityCategoryAuthInactive:
		return "auth probe did not confirm an active login"
	case codex.AvailabilityCategoryInvalidAuthOutput:
		return "auth probe returned output Aether could not parse"
	case codex.AvailabilityCategoryCredentialsMissing:
		return "no configured credentials or environment keys were reported"
	case codex.AvailabilityCategoryProbeSkipped:
		return "auth probe was skipped for an override binary"
	case codex.AvailabilityCategoryUnsupportedProvider:
		return "worker provider override is unsupported"
	case codex.AvailabilityCategoryAvailable:
		return "available"
	default:
		return "unavailable"
	}
}

func dispatchAvailabilityNextAction(status codex.AvailabilityStatus) string {
	binary := strings.TrimSpace(status.Binary)
	if binary == "" {
		binary = defaultProviderBinary(status.Platform)
	}
	switch status.Category {
	case codex.AvailabilityCategoryBinaryMissing:
		return fmt.Sprintf("install %s or set AETHER_WORKER_PLATFORM to another authenticated provider before rerunning this Aether command.", binary)
	case codex.AvailabilityCategoryAuthProbeFailed:
		return fmt.Sprintf("check `%s` locally, fix the provider login, then rerun this Aether command.", providerProbeCommand(status.Platform, binary))
	case codex.AvailabilityCategoryAuthInactive:
		return fmt.Sprintf("sign in to %s with %s before rerunning this Aether command.", status.Platform, binary)
	case codex.AvailabilityCategoryInvalidAuthOutput:
		return fmt.Sprintf("update or repair %s so `%s` returns the expected auth status, then rerun.", binary, providerProbeCommand(status.Platform, binary))
	case codex.AvailabilityCategoryCredentialsMissing:
		return fmt.Sprintf("add %s credentials or environment keys before rerunning this Aether command.", status.Platform)
	case codex.AvailabilityCategoryProbeSkipped:
		return fmt.Sprintf("use a real %s CLI path if you need auth verification, or rerun with this override intentionally.", status.Platform)
	case codex.AvailabilityCategoryUnsupportedProvider:
		return "set AETHER_WORKER_PLATFORM to codex, claude, or opencode, or unset it to allow automatic fallback."
	case codex.AvailabilityCategoryAvailable:
		return "rerun this Aether command."
	default:
		return "install or authenticate a worker provider, then rerun this Aether command."
	}
}

func defaultProviderBinary(platform codex.Platform) string {
	switch platform {
	case codex.PlatformCodex:
		return "codex"
	case codex.PlatformClaude:
		return "claude"
	case codex.PlatformOpenCode:
		return "opencode"
	default:
		return "the provider CLI"
	}
}

func providerProbeCommand(platform codex.Platform, binary string) string {
	switch platform {
	case codex.PlatformCodex:
		return binary + " login status"
	case codex.PlatformClaude:
		return binary + " auth status --json"
	case codex.PlatformOpenCode:
		return binary + " auth list"
	default:
		return binary + " auth status"
	}
}
