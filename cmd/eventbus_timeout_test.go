package cmd

import (
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestEventBusTimeoutFlag_DefaultIs5s(t *testing.T) {
	tests := []struct {
		name string
		cmd  *cobra.Command
	}{
		{"publish", eventBusPublishCmd},
		{"query", eventBusQueryCmd},
		{"replay", eventBusReplayCmd},
		{"cleanup", eventBusCleanupCmd},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := tt.cmd.Flags().Lookup("timeout")
			if flag == nil {
				t.Fatalf("expected --timeout flag on event-bus-%s", tt.name)
			}
			if flag.DefValue != "5s" {
				t.Fatalf("expected default value 5s, got %q", flag.DefValue)
			}
		})
	}
}

func TestEventBusTimeoutFlag_CustomValue(t *testing.T) {
	tests := []struct {
		name string
		cmd  *cobra.Command
	}{
		{"publish", eventBusPublishCmd},
		{"query", eventBusQueryCmd},
		{"replay", eventBusReplayCmd},
		{"cleanup", eventBusCleanupCmd},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []string{"--timeout", "10s"}
			// Add required flags for commands that need them
			switch tt.name {
			case "publish":
				args = append(args, "--topic", "test", "--payload", `"{}"`)
			case "query":
				args = append(args, "--pattern", "test")
			case "replay":
				args = append(args, "--topic", "test")
			}

			tt.cmd.SetArgs(args)
			err := tt.cmd.ParseFlags(args)
			if err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			timeout, err := tt.cmd.Flags().GetDuration("timeout")
			if err != nil {
				t.Fatalf("failed to get timeout: %v", err)
			}
			if timeout != 10*time.Second {
				t.Fatalf("expected 10s, got %v", timeout)
			}
		})
	}
}

func TestEventBusTimeoutFlag_AcceptsDurationFormats(t *testing.T) {
	tests := []struct {
		name string
		args string
		want time.Duration
	}{
		{"milliseconds", "500ms", 500 * time.Millisecond},
		{"minutes", "2m", 2 * time.Minute},
		{"seconds", "30s", 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := eventBusCleanupCmd
			args := []string{"--timeout", tt.args}
			cmd.SetArgs(args)
			err := cmd.ParseFlags(args)
			if err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			timeout, err := cmd.Flags().GetDuration("timeout")
			if err != nil {
				t.Fatalf("failed to get timeout: %v", err)
			}
			if timeout != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, timeout)
			}
		})
	}
}
