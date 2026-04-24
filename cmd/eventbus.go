package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
	"github.com/spf13/cobra"
)

const eventBusDefaultTimeout = 5 * time.Second

var (
	eventTopic        string
	eventPayload      string
	eventSource       string
	eventPattern      string
	eventSince        string
	eventLimit        int
	eventDryRun       bool
	eventTimeout      time.Duration
	eventStream       bool
	eventFilter       string
	eventPollInterval time.Duration
	eventMaxEvents    int
)

// newEventBus creates an event bus backed by the current store.
func newEventBus() (*events.Bus, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	return events.NewBus(store, events.DefaultConfig()), nil
}

var eventBusPublishCmd = &cobra.Command{
	Use:   "event-bus-publish",
	Short: "Publish a typed event to the event bus",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		topic := mustGetString(cmd, "topic")
		payload := mustGetString(cmd, "payload")
		source := mustGetString(cmd, "source")
		if topic == "" || payload == "" {
			return nil
		}

		bus, err := newEventBus()
		if err != nil {
			outputErrorMessage(err.Error())
			return nil
		}

		ctx, cancel := timeoutCtxWith(cmd, eventTimeout)
		defer cancel()

		evt, err := bus.Publish(ctx, topic, json.RawMessage(payload), source)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"id":         evt.ID,
			"topic":      evt.Topic,
			"source":     evt.Source,
			"timestamp":  evt.Timestamp,
			"ttl_days":   evt.TTLDays,
			"expires_at": evt.ExpiresAt,
		})
		return nil
	},
}

var eventBusQueryCmd = &cobra.Command{
	Use:   "event-bus-query",
	Short: "Query events from the event bus matching a topic pattern",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		pattern := mustGetString(cmd, "pattern")
		if pattern == "" {
			return nil
		}

		bus, err := newEventBus()
		if err != nil {
			outputErrorMessage(err.Error())
			return nil
		}

		ctx, cancel := timeoutCtxWith(cmd, eventTimeout)
		defer cancel()

		var since time.Time
		if eventSince != "" {
			var parseErr error
			since, parseErr = time.Parse(time.RFC3339, eventSince)
			if parseErr != nil {
				outputError(1, "invalid --since format, use RFC3339", nil)
				return nil
			}
		}

		evts, err := bus.Query(ctx, pattern, since, eventLimit)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"events": evts,
			"count":  len(evts),
		})
		return nil
	},
}

var eventBusReplayCmd = &cobra.Command{
	Use:   "event-bus-replay",
	Short: "Replay persisted events for a specific topic in chronological order",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		topic := mustGetString(cmd, "topic")
		if topic == "" {
			return nil
		}

		bus, err := newEventBus()
		if err != nil {
			outputErrorMessage(err.Error())
			return nil
		}

		ctx, cancel := timeoutCtxWith(cmd, eventTimeout)
		defer cancel()

		var since time.Time
		if eventSince != "" {
			var parseErr error
			since, parseErr = time.Parse(time.RFC3339, eventSince)
			if parseErr != nil {
				outputError(1, "invalid --since format, use RFC3339", nil)
				return nil
			}
		}

		evts, err := bus.Replay(ctx, topic, since, eventLimit)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"events": evts,
			"count":  len(evts),
		})
		return nil
	},
}

var eventBusCleanupCmd = &cobra.Command{
	Use:   "event-bus-cleanup",
	Short: "Remove expired events from the event bus JSONL file",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		bus, err := newEventBus()
		if err != nil {
			outputErrorMessage(err.Error())
			return nil
		}

		ctx, cancel := timeoutCtxWith(cmd, eventTimeout)
		defer cancel()

		removed, remaining, err := bus.Cleanup(ctx, eventDryRun)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"removed":   removed,
			"remaining": remaining,
			"dry_run":   eventDryRun,
		})
		return nil
	},
}

var eventBusSubscribeCmd = &cobra.Command{
	Use:   "event-bus-subscribe",
	Short: "Stream persisted event bus events as NDJSON",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		filter := strings.TrimSpace(mustGetString(cmd, "filter"))
		if filter == "" {
			return nil
		}

		bus, err := newEventBus()
		if err != nil {
			outputErrorMessage(err.Error())
			return nil
		}

		ctx, cancel := eventBusSubscribeContext(cmd, eventStream)
		defer cancel()

		var since time.Time
		if eventSince != "" {
			var parseErr error
			since, parseErr = time.Parse(time.RFC3339, eventSince)
			if parseErr != nil {
				outputError(1, "invalid --since format, use RFC3339", nil)
				return nil
			}
		} else if eventStream {
			since = time.Now().UTC()
		}

		if !eventStream {
			evts, err := bus.Query(ctx, filter, since, eventLimit)
			if err != nil {
				outputError(1, err.Error(), nil)
				return nil
			}
			outputOK(map[string]interface{}{
				"events": evts,
				"count":  len(evts),
			})
			return nil
		}

		if eventPollInterval <= 0 {
			outputError(1, "--poll-interval must be greater than 0", nil)
			return nil
		}
		if err := streamEventBusNDJSON(ctx, bus, filter, since, eventLimit, eventPollInterval, eventMaxEvents); err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		return nil
	},
}

func streamEventBusNDJSON(ctx context.Context, bus *events.Bus, filter string, since time.Time, limit int, pollInterval time.Duration, maxEvents int) error {
	seen := map[string]struct{}{}
	written := 0
	baseQueryLimit := limit
	if baseQueryLimit <= 0 {
		baseQueryLimit = 1000
	}
	queryLimit := limit
	if queryLimit <= 0 {
		queryLimit = baseQueryLimit
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		evts, err := bus.Query(ctx, filter, since, queryLimit)
		if err != nil {
			return err
		}
		newThisPoll := 0
		for _, evt := range evts {
			if _, ok := seen[evt.ID]; ok {
				continue
			}
			seen[evt.ID] = struct{}{}
			newThisPoll++
			data, err := json.Marshal(evt)
			if err != nil {
				return fmt.Errorf("marshal event %s: %w", evt.ID, err)
			}
			fmt.Fprintln(stdout, string(data))
			written++
			if parsed, err := time.Parse(time.RFC3339, evt.Timestamp); err == nil && parsed.After(since) {
				since = parsed
			}
			if maxEvents > 0 && written >= maxEvents {
				return nil
			}
		}
		if len(evts) >= queryLimit && newThisPoll == 0 {
			queryLimit *= 2
			continue
		}
		if queryLimit > baseQueryLimit && len(evts) < queryLimit {
			queryLimit = baseQueryLimit
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func eventBusSubscribeContext(cmd *cobra.Command, stream bool) (context.Context, context.CancelFunc) {
	if !stream || cmd.Flags().Changed("timeout") {
		return timeoutCtxWith(cmd, eventTimeout)
	}
	parent := cmd.Context()
	if parent == nil {
		parent = context.Background()
	}
	return context.WithCancel(parent)
}

func init() {
	// event-bus-publish
	rootCmd.AddCommand(eventBusPublishCmd)
	eventBusPublishCmd.Flags().StringVar(&eventTopic, "topic", "", "Event topic (required)")
	eventBusPublishCmd.Flags().StringVar(&eventPayload, "payload", "", "JSON payload (required)")
	eventBusPublishCmd.Flags().StringVar(&eventSource, "source", "", "Event source")
	eventBusPublishCmd.Flags().DurationVar(&eventTimeout, "timeout", eventBusDefaultTimeout, "Timeout for event bus operation")

	// event-bus-query
	rootCmd.AddCommand(eventBusQueryCmd)
	eventBusQueryCmd.Flags().StringVar(&eventPattern, "pattern", "", "Topic pattern (supports trailing * wildcard)")
	eventBusQueryCmd.Flags().StringVar(&eventSince, "since", "", "Only return events after this RFC3339 timestamp")
	eventBusQueryCmd.Flags().IntVar(&eventLimit, "limit", 0, "Maximum events to return (0 for default)")
	eventBusQueryCmd.Flags().DurationVar(&eventTimeout, "timeout", eventBusDefaultTimeout, "Timeout for event bus operation")

	// event-bus-replay
	rootCmd.AddCommand(eventBusReplayCmd)
	eventBusReplayCmd.Flags().StringVar(&eventTopic, "topic", "", "Exact topic to replay")
	eventBusReplayCmd.Flags().StringVar(&eventSince, "since", "", "Only return events after this RFC3339 timestamp")
	eventBusReplayCmd.Flags().IntVar(&eventLimit, "limit", 0, "Maximum events to return (0 for default)")
	eventBusReplayCmd.Flags().DurationVar(&eventTimeout, "timeout", eventBusDefaultTimeout, "Timeout for event bus operation")

	// event-bus-cleanup
	rootCmd.AddCommand(eventBusCleanupCmd)
	eventBusCleanupCmd.Flags().BoolVar(&eventDryRun, "dry-run", false, "Report counts without modifying the file")
	eventBusCleanupCmd.Flags().DurationVar(&eventTimeout, "timeout", eventBusDefaultTimeout, "Timeout for event bus operation")

	// event-bus-subscribe
	rootCmd.AddCommand(eventBusSubscribeCmd)
	eventBusSubscribeCmd.Flags().StringVar(&eventFilter, "filter", "", "Event topic filter (supports trailing * wildcard)")
	eventBusSubscribeCmd.Flags().BoolVar(&eventStream, "stream", false, "Continuously stream matching events as NDJSON")
	eventBusSubscribeCmd.Flags().StringVar(&eventSince, "since", "", "Only return events after this RFC3339 timestamp")
	eventBusSubscribeCmd.Flags().IntVar(&eventLimit, "limit", 0, "Maximum events queried per poll (0 for default)")
	eventBusSubscribeCmd.Flags().DurationVar(&eventPollInterval, "poll-interval", 250*time.Millisecond, "Polling interval for --stream")
	eventBusSubscribeCmd.Flags().IntVar(&eventMaxEvents, "max-events", 0, "Stop streaming after this many events (0 for unlimited)")
	eventBusSubscribeCmd.Flags().DurationVar(&eventTimeout, "timeout", eventBusDefaultTimeout, "Timeout for event bus operation")
}
