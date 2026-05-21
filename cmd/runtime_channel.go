package cmd

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/calcosmic/Aether/pkg/downloader"
)

type runtimeChannel string

const (
	channelStable runtimeChannel = "stable"
	channelDev    runtimeChannel = "dev"
)

func normalizeRuntimeChannel(raw string) runtimeChannel {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "stable":
		return channelStable
	case "dev":
		return channelDev
	default:
		return channelStable
	}
}

func runtimeChannelFromInvocation(arg0 string) runtimeChannel {
	base := strings.ToLower(strings.TrimSpace(filepath.Base(strings.ReplaceAll(arg0, "\\", "/"))))
	base = strings.TrimSuffix(base, filepath.Ext(base))
	if base == "aether-dev" {
		return channelDev
	}
	return channelStable
}

func resolveRuntimeChannel() runtimeChannel {
	if env := strings.TrimSpace(os.Getenv("AETHER_CHANNEL")); env != "" {
		return normalizeRuntimeChannel(env)
	}
	if len(os.Args) > 0 {
		return runtimeChannelFromInvocation(os.Args[0])
	}
	return channelStable
}

func runtimeChannelFromFlag(cmd flagGetter) runtimeChannel {
	if cmd != nil {
		if value, err := cmd.GetString("channel"); err == nil {
			if channel := normalizeRuntimeChannel(value); channel == channelDev || strings.TrimSpace(value) == "" {
				if strings.TrimSpace(value) != "" {
					return channel
				}
			}
		}
	}
	return resolveRuntimeChannel()
}

type flagGetter interface {
	GetString(name string) (string, error)
}

func defaultHubDirName(channel runtimeChannel) string {
	if channel == channelDev {
		return ".aether-dev"
	}
	return ".aether"
}

func defaultBinaryName(channel runtimeChannel) string {
	if channel == channelDev {
		return "aether-dev"
	}
	return "aether"
}

func defaultBinaryDestSubdirForChannel(channel runtimeChannel) string {
	return filepath.Join(defaultHubDirName(channel), "bin")
}

func resolveHubPathForHome(homeDir string, channel runtimeChannel) string {
	if dir := strings.TrimSpace(os.Getenv("AETHER_HUB_DIR")); dir != "" {
		return dir
	}
	return filepath.Join(homeDir, defaultHubDirName(channel))
}

func shouldSyncPlatformHomes(channel runtimeChannel, optIn bool) bool {
	return channel != channelDev || optIn
}

func alignDownloadedBinaryToChannel(result *downloader.DownloadResult, destDir string, channel runtimeChannel) (*downloader.DownloadResult, error) {
	if result == nil || channel != channelDev {
		return result, nil
	}
	desiredPath := filepath.Join(destDir, defaultBinaryName(channel)+filepath.Ext(result.Path))
	if result.Path == desiredPath {
		return result, nil
	}
	if err := os.Rename(result.Path, desiredPath); err != nil {
		return nil, err
	}
	result.Path = desiredPath
	return result, nil
}
