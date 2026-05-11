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
	message := strings.TrimSpace(codex.DescribeInvokerAvailability(invoker, context.Background()))
	if message == "" || message == "worker dispatcher availability unknown" {
		return "no authenticated worker platform is available"
	}
	return message
}

func dispatchUnavailableError(invoker codex.WorkerInvoker) error {
	return fmt.Errorf("worker dispatcher is unavailable: %s", dispatchAvailabilityMessage(invoker))
}
